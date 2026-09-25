package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"marionette/internal/config"
	execengine "marionette/internal/exec"
	"marionette/internal/server"
	"marionette/internal/status"
)

func main() {
	configPath := os.Getenv("MARIONETTE_CONFIG")
	if configPath == "" {
		configPath = "./marionette.json"
	}
	store, err := config.LoadFile(configPath)
	if err != nil {
		log.Printf("config load failed: %v; starting with an empty store", err)
		store = config.NewStore(config.Settings{
			HistorySize:          config.DefaultHistorySize,
			MaxConcurrentActions: config.DefaultMaxConcurrentActions,
		})
	}
	store.OnChange = func(changedStore *config.Store) error {
		return changedStore.SaveFile(configPath)
	}
	statusEvents := server.NewStatusEventBroker()

	executor := execengine.NewExecutor()
	runner, err := execengine.NewRunner(store.GetSettings(), executor)
	if err != nil {
		log.Fatalf("create execution runner: %v", err)
	}
	statusService, err := status.NewStatusCheckService(store, runner)
	if err != nil {
		log.Fatalf("create status service: %v", err)
	}
	scheduler, err := status.NewScheduler(store, statusService)
	if err != nil {
		log.Fatalf("create status scheduler: %v", err)
	}
	backgroundActions, err := server.NewBackgroundActions(store, runner, statusService)
	if err != nil {
		log.Fatalf("create background action queue: %v", err)
	}
	applicationContext, cancelApplication := context.WithCancel(context.Background())
	defer cancelApplication()
	if err := scheduler.Start(applicationContext); err != nil {
		log.Fatalf("start status scheduler: %v", err)
	}

	addr := os.Getenv("MARIONETTE_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	handler := server.NewRequestTracker(server.NewRouterWithDependencies(server.RouterDependencies{
		Store:        store,
		Actions:      backgroundActions,
		Notifier:     scheduler,
		Reconciler:   scheduler,
		StatusEvents: statusEvents,
	}))
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("marionette listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("shutting down")
	shutdownContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := srv.Shutdown(shutdownContext); err != nil {
		log.Printf("HTTP server shutdown failed: %v", err)
		_ = srv.Close()
	}
	cancel()
	handler.Wait()
	backgroundActions.Close()
	if err := scheduler.Stop(); err != nil && !errors.Is(err, status.ErrSchedulerStopped) {
		log.Printf("status scheduler shutdown failed: %v", err)
	}
	if err := store.SaveFileWithHistory(configPath); err != nil {
		log.Printf("config save failed: %v", err)
	}
}
