package main

import (
	"errors"
	"os"
	"path/filepath"
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
	store, err := openStore(path, fixedNow)
	if err != nil {
		t.Fatalf("openStore() error = %v", err)
	}
	if _, err := store.CreateCard(testCard("first")); err != nil {
		t.Fatalf("CreateCard() error = %v", err)
	}
	loaded, err := config.LoadFile(path)
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

	store, err := openStore(path, fixedNow)
	if err != nil {
		t.Fatalf("openStore() error = %v", err)
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
	if loaded, err := config.LoadFile(path); err != nil || len(loaded.ListCards()) != 1 {
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
	store, err := openStore(path, fixedNow)
	if err != nil {
		t.Fatalf("openStore() error = %v", err)
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
