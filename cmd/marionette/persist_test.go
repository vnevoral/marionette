package main

import (
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"marionette/internal/config"
)

func TestPersistOnChangeSavesAndReportsWriteFailures(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	dir := t.TempDir()
	store := config.NewStore(config.Settings{HistorySize: 5, MaxConcurrentActions: 1})
	path := filepath.Join(dir, "marionette.json")
	if err := persistOnChange(path, logger)(store); err != nil {
		t.Fatalf("persistOnChange() error = %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("configuration not written: %v", err)
	}

	missing := filepath.Join(dir, "missing", "marionette.json")
	if err := persistOnChange(missing, logger)(store); err == nil || errors.Is(err, config.ErrDirectorySync) {
		t.Fatalf("persistOnChange() error = %v, want a write failure that is not a directory sync error", err)
	}
}
