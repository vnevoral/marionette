package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"marionette/internal/config"
	"marionette/internal/server"
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

	addr := os.Getenv("MARIONETTE_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           server.NewRouter(),
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
	}
	cancel()
	if err := store.SaveFileWithHistory(configPath); err != nil {
		log.Printf("config save failed: %v", err)
	}
}
