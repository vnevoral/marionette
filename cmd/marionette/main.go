// Command marionette runs the action-card service: an HTTP API with the
// embedded dashboard, the status scheduler and the background action queue.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"marionette/internal/actions"
	"marionette/internal/config"
	"marionette/internal/events"
	"marionette/internal/execengine"
	"marionette/internal/server"
	"marionette/internal/status"
)

// version is the release identifier reported by GET /api/health. It is set at
// build time by the Makefile (-ldflags "-X main.version=...").
var version = "dev"

const (
	defaultConfigPath      = "./marionette.json"
	defaultAddr            = ":8080"
	defaultShutdownTimeout = 20 * time.Second

	httpReadHeaderTimeout = 5 * time.Second
	httpIdleTimeout       = 60 * time.Second

	// shutdownReserve is held back from the shutdown budget so that cancelled
	// action processes can be terminated (execution engine WaitDelay) and the
	// final history save still fits before the deadline.
	shutdownReserve = 3 * time.Second
)

// environment holds the process configuration read from environment
// variables (FR-34).
type environment struct {
	ConfigPath      string
	Addr            string
	ShutdownTimeout time.Duration
	// AllowedHosts restricts the Host header of mutating API requests
	// (NFR-12); empty keeps every host accepted.
	AllowedHosts []string
	// LogFormat is "text" (journald friendly, default) or "json".
	LogFormat string
	// LogLevel is the minimum level written to the log; default info.
	LogLevel slog.Level
}

// loadEnvironment reads MARIONETTE_CONFIG, MARIONETTE_ADDR,
// MARIONETTE_SHUTDOWN_TIMEOUT, MARIONETTE_ALLOWED_HOSTS, MARIONETTE_LOG_FORMAT
// and MARIONETTE_LOG_LEVEL with their defaults.
func loadEnvironment(getenv func(string) string) (environment, error) {
	env := environment{
		ConfigPath:      getenv("MARIONETTE_CONFIG"),
		Addr:            getenv("MARIONETTE_ADDR"),
		ShutdownTimeout: defaultShutdownTimeout,
		LogFormat:       "text",
		LogLevel:        slog.LevelInfo,
	}
	if env.ConfigPath == "" {
		env.ConfigPath = defaultConfigPath
	}
	if env.Addr == "" {
		env.Addr = defaultAddr
	}
	if raw := getenv("MARIONETTE_SHUTDOWN_TIMEOUT"); raw != "" {
		timeout, err := time.ParseDuration(raw)
		if err != nil {
			return environment{}, fmt.Errorf("MARIONETTE_SHUTDOWN_TIMEOUT %q: %w", raw, err)
		}
		if timeout <= 0 {
			return environment{}, fmt.Errorf("MARIONETTE_SHUTDOWN_TIMEOUT %q: must be positive", raw)
		}
		env.ShutdownTimeout = timeout
	}
	for _, host := range strings.Split(getenv("MARIONETTE_ALLOWED_HOSTS"), ",") {
		if host = strings.TrimSpace(host); host != "" {
			env.AllowedHosts = append(env.AllowedHosts, host)
		}
	}
	switch format := strings.ToLower(strings.TrimSpace(getenv("MARIONETTE_LOG_FORMAT"))); format {
	case "":
	case "text", "json":
		env.LogFormat = format
	default:
		return environment{}, fmt.Errorf("MARIONETTE_LOG_FORMAT %q: must be text or json", format)
	}
	if raw := strings.TrimSpace(getenv("MARIONETTE_LOG_LEVEL")); raw != "" {
		if err := env.LogLevel.UnmarshalText([]byte(raw)); err != nil {
			return environment{}, fmt.Errorf("MARIONETTE_LOG_LEVEL %q: must be debug, info, warn or error", raw)
		}
	}
	return env, nil
}

// newLogger builds the process logger for the configured format and level.
func newLogger(writer io.Writer, env environment) *slog.Logger {
	options := &slog.HandlerOptions{Level: env.LogLevel}
	if env.LogFormat == "json" {
		return slog.New(slog.NewJSONHandler(writer, options))
	}
	return slog.New(slog.NewTextHandler(writer, options))
}

func main() {
	env, err := loadEnvironment(os.Getenv)
	if err != nil {
		slog.New(slog.NewTextHandler(os.Stderr, nil)).Error("invalid environment", "error", err)
		os.Exit(1)
	}
	logger := newLogger(os.Stderr, env)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		// Once shutdown has started, restore default signal handling so a
		// second SIGINT/SIGTERM terminates the process immediately.
		<-ctx.Done()
		stop()
	}()

	app := application{env: env, logger: logger}
	if err := app.run(ctx); err != nil {
		logger.Error("marionette exited with an error", "error", err)
		os.Exit(1)
	}
}

// application composes the service and runs it until the context is done.
type application struct {
	env    environment
	logger *slog.Logger
	// onListening, when set, receives the bound address once the HTTP
	// listener is ready. Tests use it to discover an ephemeral port.
	onListening func(net.Addr)
}

// run starts the service, blocks until ctx is done and then performs the
// graceful shutdown sequence within env.ShutdownTimeout.
func (app application) run(ctx context.Context) error {
	store, readOnly, err := openStore(app.env.ConfigPath, time.Now, app.logger)
	if err != nil {
		return fmt.Errorf("open config store: %w", err)
	}
	broker := events.NewBroker()
	store.OnStatusChange = broker.Publish

	executor := execengine.NewExecutor()
	runner, err := execengine.NewRunner(store.GetSettings(), executor)
	if err != nil {
		return fmt.Errorf("create execution runner: %w", err)
	}
	statusService, err := status.NewStatusCheckService(store, runner)
	if err != nil {
		return fmt.Errorf("create status service: %w", err)
	}
	scheduler, err := status.NewScheduler(store, statusService)
	if err != nil {
		return fmt.Errorf("create status scheduler: %w", err)
	}
	queue, err := actions.New(store, runner, statusService, app.logger)
	if err != nil {
		return fmt.Errorf("create background action queue: %w", err)
	}
	applicationContext, cancelApplication := context.WithCancel(context.Background())
	defer cancelApplication()
	if err := scheduler.Start(applicationContext); err != nil {
		return fmt.Errorf("start status scheduler: %w", err)
	}

	handler := server.NewRouter(server.Dependencies{
		Store:        store,
		Actions:      queue,
		Notifier:     scheduler,
		Reconciler:   scheduler,
		Events:       broker,
		AllowedHosts: app.env.AllowedHosts,
		Version:      version,
		StartedAt:    time.Now(),
		Logger:       app.logger,
	})
	srv := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: httpReadHeaderTimeout,
		IdleTimeout:       httpIdleTimeout,
		BaseContext:       func(net.Listener) context.Context { return applicationContext },
	}
	// Shutdown closes the listeners first and then runs the registered
	// hooks; closing the broker ends every SSE stream so Shutdown does not
	// wait for those long-lived connections.
	srv.RegisterOnShutdown(broker.Close)

	listener, err := net.Listen("tcp", app.env.Addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", app.env.Addr, err)
	}
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- srv.Serve(listener)
	}()
	app.logger.Info("marionette listening", "addr", listener.Addr().String(), "version", version)
	if app.onListening != nil {
		app.onListening(listener.Addr())
	}

	select {
	case err := <-serveErr:
		return fmt.Errorf("HTTP server: %w", err)
	case <-ctx.Done():
	}

	app.logger.Info("shutting down; send the signal again to force exit", "timeout", app.env.ShutdownTimeout)
	shutdownContext, cancelShutdown := context.WithTimeout(context.Background(), app.env.ShutdownTimeout)
	defer cancelShutdown()
	steps := shutdownSteps{
		stopHTTP: func(ctx context.Context) error {
			if err := srv.Shutdown(ctx); err != nil {
				_ = srv.Close()
				return err
			}
			return nil
		},
		closeActions:  queue.Close,
		stopScheduler: scheduler.Stop,
		historyDirty:  store.Dirty,
	}
	if !readOnly {
		steps.saveHistory = func() error { return store.SaveFileWithHistory(app.env.ConfigPath) }
	}
	return shutdown(shutdownContext, steps, app.logger)
}

// shutdownSteps are the dependencies of the shutdown sequence, expressed as
// functions so the ordering can be tested without a running server.
type shutdownSteps struct {
	stopHTTP func(context.Context) error
	// saveHistory is nil when the store is read-only (unreadable config
	// file): the file must not be overwritten in that case.
	saveHistory   func() error
	closeActions  func(context.Context) (dropped int, err error)
	stopScheduler func() error
	historyDirty  func() bool
}

// shutdown runs the graceful shutdown sequence (FR-35): stop accepting HTTP
// requests, save the history before anything can delay the process, discard
// queued actions and let running ones finish within the remaining grace
// period, stop the scheduler and save again only if the history changed in
// the meantime. Each step is logged with its duration. Every step runs even
// when an earlier one fails; the errors are joined in the result.
func shutdown(ctx context.Context, steps shutdownSteps, logger *slog.Logger) error {
	var failures []error
	step := func(name string, run func() error) {
		started := time.Now()
		err := run()
		duration := time.Since(started).Round(time.Millisecond)
		if err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", name, err))
			logger.Error("shutdown step failed", "step", name, "duration", duration, "error", err)
			return
		}
		logger.Info("shutdown step done", "step", name, "duration", duration)
	}

	step("stop HTTP server", func() error { return steps.stopHTTP(ctx) })

	if steps.saveHistory == nil {
		logger.Warn("shutdown: history not saved, the configuration file is read-only")
	} else {
		step("save history", steps.saveHistory)
	}

	step("close action queue", func() error {
		graceContext, cancel := context.WithDeadline(context.Background(), graceDeadline(ctx))
		defer cancel()
		dropped, err := steps.closeActions(graceContext)
		if dropped > 0 {
			logger.Info("shutdown: discarded queued actions", "count", dropped)
		}
		if err != nil {
			logger.Warn("shutdown: running actions cancelled after the grace period", "error", err)
		}
		return nil
	})

	step("stop status scheduler", func() error {
		if err := steps.stopScheduler(); err != nil && !errors.Is(err, status.ErrSchedulerStopped) {
			return err
		}
		return nil
	})

	if steps.saveHistory != nil && steps.historyDirty() {
		step("save history again", steps.saveHistory)
	}
	return errors.Join(failures...)
}

// graceDeadline returns how long running actions may keep going: the
// shutdown deadline minus the reserve, but never in the past.
func graceDeadline(ctx context.Context) time.Time {
	deadline, ok := ctx.Deadline()
	if !ok {
		return time.Now().Add(defaultShutdownTimeout - shutdownReserve)
	}
	grace := deadline.Add(-shutdownReserve)
	if now := time.Now(); grace.Before(now) {
		return now
	}
	return grace
}

// openStore loads the configuration file and wires persistence. A missing file
// starts an empty store that is created on the first change. A corrupt file is
// moved aside (quarantined) before anything is written, so its content is
// never lost. An unreadable file starts an empty store with persistence
// disabled (readOnly = true): every mutation fails and is rolled back, and the
// shutdown history save is skipped, until the operator fixes the file and
// restarts the service.
func openStore(configPath string, now func() time.Time, logger *slog.Logger) (store *config.Store, readOnly bool, err error) {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	store, err = config.LoadFile(configPath, logger)
	switch {
	case err == nil:
	case errors.Is(err, config.ErrConfigCorrupt):
		quarantined, quarantineErr := config.QuarantineFile(configPath, now())
		if quarantineErr != nil {
			return nil, false, fmt.Errorf("config load failed (%v) and the file could not be quarantined: %w", err, quarantineErr)
		}
		logger.Warn("config load failed; original file quarantined, starting with an empty store", "error", err, "quarantined", quarantined)
		store = emptyStore()
	case errors.Is(err, config.ErrConfigUnreadable):
		logger.Warn("config load failed; starting read-only with an empty store, configuration changes will be rejected until the file is readable", "error", err)
		store = emptyStore()
		store.OnChange = func(*config.Store) error {
			return fmt.Errorf("configuration file %q is not readable; fix its permissions and restart the service", configPath)
		}
		return store, true, nil
	default:
		return nil, false, err
	}
	store.OnChange = func(changedStore *config.Store) error {
		return changedStore.SaveFile(configPath)
	}
	return store, false, nil
}

func emptyStore() *config.Store {
	return config.NewStore(config.Settings{
		HistorySize:          config.DefaultHistorySize,
		MaxConcurrentActions: config.DefaultMaxConcurrentActions,
	})
}
