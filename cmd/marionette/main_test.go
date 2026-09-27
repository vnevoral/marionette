package main

import (
	"bufio"
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"marionette/internal/config"
)

func fixedNow() time.Time {
	return time.Date(2026, time.September, 26, 12, 0, 0, 0, time.UTC)
}

func testCard(id string) config.ActionCard {
	return config.ActionCard{
		ID:      id,
		Name:    "Card " + id,
		Primary: config.Action{Command: "true", TimeoutSec: 1},
	}
}

func TestOpenStoreMissingFileStartsEmptyAndPersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "marionette.json")
	store, _, err := openStore(path, fixedNow, nil)
	if err != nil {
		t.Fatalf("openStore() error = %v", err)
	}
	if _, err := store.CreateCard(testCard("first")); err != nil {
		t.Fatalf("CreateCard() error = %v", err)
	}
	loaded, err := config.LoadFile(path, nil)
	if err != nil || len(loaded.ListCards()) != 1 {
		t.Fatalf("persisted config = %v cards, error = %v", len(loaded.ListCards()), err)
	}
}

func TestOpenStoreQuarantinesCorruptFile(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "marionette.json")
	original := []byte(`{"settings":{"historySize":5,"maxConcurrentActions":1},"cards":[{"id":"x"`)
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	store, readOnly, err := openStore(path, fixedNow, nil)
	if err != nil {
		t.Fatalf("openStore() error = %v", err)
	}
	if readOnly {
		t.Fatal("quarantined file must not open the store read-only")
	}
	if len(store.ListCards()) != 0 {
		t.Fatal("corrupt config produced a non-empty store")
	}
	quarantined := path + ".corrupt-20260926T120000Z"
	content, err := os.ReadFile(quarantined)
	if err != nil {
		t.Fatalf("quarantined file missing: %v", err)
	}
	if string(content) != string(original) {
		t.Fatalf("quarantined content = %q, want original", content)
	}
	if _, err := store.CreateCard(testCard("fresh")); err != nil {
		t.Fatalf("CreateCard() after quarantine error = %v", err)
	}
	if content, err := os.ReadFile(quarantined); err != nil || string(content) != string(original) {
		t.Fatalf("quarantined file changed after a save: %q, %v", content, err)
	}
	if loaded, err := config.LoadFile(path, nil); err != nil || len(loaded.ListCards()) != 1 {
		t.Fatalf("fresh config = %v, error = %v", loaded, err)
	}
}

func TestOpenStoreUnreadableFileRejectsChanges(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores file permissions")
	}
	path := filepath.Join(t.TempDir(), "marionette.json")
	if err := os.WriteFile(path, []byte(`{"settings":{"historySize":5,"maxConcurrentActions":1},"cards":[]}`), 0o000); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	store, readOnly, err := openStore(path, fixedNow, nil)
	if err != nil {
		t.Fatalf("openStore() error = %v", err)
	}
	if !readOnly {
		t.Fatal("unreadable file must open the store read-only")
	}
	var persistenceErr *config.PersistenceError
	if _, err := store.CreateCard(testCard("blocked")); !errors.As(err, &persistenceErr) {
		t.Fatalf("CreateCard() error = %v, want PersistenceError", err)
	}
	if _, exists := store.GetCard("blocked"); exists {
		t.Fatal("card stored although persistence is disabled")
	}
	if _, err := os.Stat(path + ".corrupt-20260926T120000Z"); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("unreadable file must not be quarantined")
	}
}

func TestLoadEnvironmentDefaultsAndShutdownTimeout(t *testing.T) {
	env, err := loadEnvironment(func(string) string { return "" })
	if err != nil {
		t.Fatalf("loadEnvironment() error = %v", err)
	}
	if env.ConfigPath != defaultConfigPath || env.Addr != defaultAddr || env.ShutdownTimeout != defaultShutdownTimeout {
		t.Fatalf("defaults = %#v", env)
	}

	if env.AllowedHosts != nil {
		t.Fatalf("default AllowedHosts = %v, want none", env.AllowedHosts)
	}

	values := map[string]string{
		"MARIONETTE_CONFIG":           "/tmp/x.json",
		"MARIONETTE_ADDR":             "127.0.0.1:9090",
		"MARIONETTE_SHUTDOWN_TIMEOUT": "1m30s",
		"MARIONETTE_ALLOWED_HOSTS":    " pi.local:8080, ,192.168.1.10:8080 ",
	}
	env, err = loadEnvironment(func(key string) string { return values[key] })
	if err != nil {
		t.Fatalf("loadEnvironment() error = %v", err)
	}
	if env.ConfigPath != "/tmp/x.json" || env.Addr != "127.0.0.1:9090" || env.ShutdownTimeout != 90*time.Second {
		t.Fatalf("environment = %#v", env)
	}
	if strings.Join(env.AllowedHosts, "|") != "pi.local:8080|192.168.1.10:8080" {
		t.Fatalf("AllowedHosts = %v", env.AllowedHosts)
	}
	if env.LogFormat != "text" || env.LogLevel != slog.LevelInfo {
		t.Fatalf("default logging = %q %v", env.LogFormat, env.LogLevel)
	}
	values["MARIONETTE_LOG_FORMAT"] = "JSON"
	values["MARIONETTE_LOG_LEVEL"] = "debug"
	env, err = loadEnvironment(func(key string) string { return values[key] })
	if err != nil || env.LogFormat != "json" || env.LogLevel != slog.LevelDebug {
		t.Fatalf("logging environment = %q %v, %v", env.LogFormat, env.LogLevel, err)
	}
	for key, invalid := range map[string]string{"MARIONETTE_LOG_FORMAT": "xml", "MARIONETTE_LOG_LEVEL": "loud"} {
		if _, err := loadEnvironment(func(k string) string {
			if k == key {
				return invalid
			}
			return ""
		}); err == nil {
			t.Fatalf("%s=%q was accepted", key, invalid)
		}
	}
	var buffer strings.Builder
	newLogger(&buffer, env).Debug("hello", "card", "x")
	if !strings.Contains(buffer.String(), `"card":"x"`) || !strings.Contains(buffer.String(), `"level":"DEBUG"`) {
		t.Fatalf("JSON debug log = %q", buffer.String())
	}

	for _, invalid := range []string{"soon", "0s", "-5s"} {
		if _, err := loadEnvironment(func(key string) string {
			if key == "MARIONETTE_SHUTDOWN_TIMEOUT" {
				return invalid
			}
			return ""
		}); err == nil {
			t.Fatalf("MARIONETTE_SHUTDOWN_TIMEOUT=%q was accepted", invalid)
		}
	}
}

type recordedSteps struct {
	mu    sync.Mutex
	calls []string
}

func (recorded *recordedSteps) add(name string) {
	recorded.mu.Lock()
	recorded.calls = append(recorded.calls, name)
	recorded.mu.Unlock()
}

func TestShutdownSavesHistoryBeforeQueueDrain(t *testing.T) {
	recorded := &recordedSteps{}
	dirty := true
	steps := shutdownSteps{
		stopHTTP: func(context.Context) error { recorded.add("http"); return nil },
		saveHistory: func() error {
			recorded.add("save")
			dirty = false
			return nil
		},
		closeActions: func(ctx context.Context) (int, error) {
			recorded.add("actions")
			if _, ok := ctx.Deadline(); !ok {
				t.Error("close actions must receive a bounded grace context")
			}
			dirty = true // a running action appended its run
			return 2, nil
		},
		stopScheduler: func() error { recorded.add("scheduler"); return nil },
		historyDirty:  func() bool { return dirty },
		saveDevices:   func() error { recorded.add("devices"); return nil },
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err := shutdown(ctx, steps, slog.New(slog.NewTextHandler(io.Discard, nil))); err != nil {
		t.Fatalf("shutdown() error = %v", err)
	}
	want := []string{"http", "save", "actions", "scheduler", "save", "devices"}
	if strings.Join(recorded.calls, ",") != strings.Join(want, ",") {
		t.Fatalf("shutdown order = %v, want %v", recorded.calls, want)
	}
}

func TestShutdownSkipsSecondSaveWhenCleanAndSaveWhenReadOnly(t *testing.T) {
	recorded := &recordedSteps{}
	steps := shutdownSteps{
		stopHTTP:      func(context.Context) error { recorded.add("http"); return errors.New("listener gone") },
		saveHistory:   func() error { recorded.add("save"); return nil },
		closeActions:  func(context.Context) (int, error) { recorded.add("actions"); return 0, context.DeadlineExceeded },
		stopScheduler: func() error { recorded.add("scheduler"); return nil },
		historyDirty:  func() bool { return false },
	}
	err := shutdown(context.Background(), steps, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err == nil || !strings.Contains(err.Error(), "listener gone") {
		t.Fatalf("shutdown() error = %v, want the HTTP failure reported", err)
	}
	if got := strings.Join(recorded.calls, ","); got != "http,save,actions,scheduler" {
		t.Fatalf("shutdown order = %s", got)
	}

	recorded = &recordedSteps{}
	steps.saveHistory = nil
	steps.stopHTTP = func(context.Context) error { recorded.add("http"); return nil }
	steps.historyDirty = func() bool { return true }
	if err := shutdown(context.Background(), steps, slog.New(slog.NewTextHandler(io.Discard, nil))); err != nil {
		t.Fatalf("shutdown() read-only error = %v", err)
	}
	if got := strings.Join(recorded.calls, ","); got != "http,actions,scheduler" {
		t.Fatalf("read-only shutdown order = %s, history must not be saved", got)
	}
}

func TestGraceDeadlineNeverInThePast(t *testing.T) {
	expired, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	time.Sleep(5 * time.Millisecond)
	if deadline := graceDeadline(expired); deadline.Before(time.Now().Add(-time.Second)) {
		t.Fatalf("grace deadline %s is in the past", deadline)
	}
	generous, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	deadline := graceDeadline(generous)
	if remaining := time.Until(deadline); remaining < 50*time.Second || remaining > 57*time.Second {
		t.Fatalf("grace remaining = %s, want about a minute minus the reserve", remaining)
	}
}

// TestRunShutsDownQuicklyWithOpenSSEClientAndSavesHistory is the smoke test
// of the whole lifecycle: a connected SSE client and a long-running primary
// action must not delay a shutdown with a short budget, and the run history
// must be on disk afterwards.
func TestRunShutsDownQuicklyWithOpenSSEClientAndSavesHistory(t *testing.T) {
	if _, err := os.Stat("/bin/sh"); err != nil {
		t.Skip("/bin/sh is required")
	}
	directory := t.TempDir()
	configPath := filepath.Join(directory, "marionette.json")
	marker := filepath.Join(directory, "started")
	seed := config.NewStore(config.Settings{HistorySize: 5, MaxConcurrentActions: 1})
	if _, err := seed.CreateCard(config.ActionCard{
		ID:      "long",
		Name:    "Long action",
		Primary: config.Action{Command: "/bin/sh", Args: []string{"-c", "touch " + marker + " && sleep 30"}, TimeoutSec: 60},
	}); err != nil {
		t.Fatalf("CreateCard() error = %v", err)
	}
	if err := seed.SaveFile(configPath); err != nil {
		t.Fatalf("SaveFile() error = %v", err)
	}

	listening := make(chan net.Addr, 1)
	app := application{
		env:         environment{ConfigPath: configPath, Addr: "127.0.0.1:0", ShutdownTimeout: 2 * time.Second},
		logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
		onListening: func(addr net.Addr) { listening <- addr },
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	finished := make(chan error, 1)
	go func() { finished <- app.run(ctx) }()

	var addr net.Addr
	select {
	case addr = <-listening:
	case err := <-finished:
		t.Fatalf("run() returned early: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("server did not start listening")
	}
	baseURL := "http://" + addr.String()

	events, err := http.Get(baseURL + "/api/events")
	if err != nil {
		t.Fatalf("GET /api/events error = %v", err)
	}
	defer func() { _ = events.Body.Close() }()
	eventReader := bufio.NewReader(events.Body)
	if line, err := eventReader.ReadString('\n'); err != nil || line != ": connected\n" {
		t.Fatalf("first SSE line = %q, %v", line, err)
	}
	streamClosed := make(chan error, 1)
	go func() {
		_, err := io.ReadAll(eventReader)
		streamClosed <- err
	}()

	response, err := http.Post(baseURL+"/api/cards/long/actions/primary", "application/json", nil)
	if err != nil {
		t.Fatalf("POST primary action error = %v", err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusAccepted {
		t.Fatalf("POST primary action status = %d", response.StatusCode)
	}
	waitForFile(t, marker, 5*time.Second)

	cancel()
	started := time.Now()
	select {
	case err := <-finished:
		if err != nil {
			t.Fatalf("run() error = %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("run() did not return after the context was cancelled")
	}
	// The budget is 2 s while the action sleeps 30 s and the SSE stream stays
	// open: finishing well below 30 s proves shutdown waits for neither. The
	// extra 3 s absorb scheduling delays on a loaded CI runner (block 0061).
	limit := app.env.ShutdownTimeout + 3*time.Second
	if elapsed := time.Since(started); elapsed > limit {
		t.Fatalf("shutdown took %s, want under %s", elapsed, limit)
	}
	select {
	case <-streamClosed:
	case <-time.After(time.Second):
		t.Fatal("SSE stream is still open after shutdown")
	}

	loaded, err := config.LoadFile(configPath, nil)
	if err != nil {
		t.Fatalf("LoadFile(, nil) after shutdown error = %v", err)
	}
	runs, err := loaded.GetRuns("long", "primary")
	if err != nil || len(runs) != 1 || runs[0].Outcome != config.RunOutcomeCanceled {
		t.Fatalf("persisted runs = %#v, error = %v; want one cancelled run", runs, err)
	}
}

func waitForFile(t *testing.T, path string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("file %s did not appear within %s", path, timeout)
}
