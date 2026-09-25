package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestLoadFileMissingReturnsDefaultStore(t *testing.T) {
	store, err := LoadFile(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}
	if len(store.ListCards()) != 0 {
		t.Fatal("LoadFile() returned cards for a missing file")
	}
	want := Settings{HistorySize: DefaultHistorySize, MaxConcurrentActions: DefaultMaxConcurrentActions}
	if got := store.GetSettings(); got != want {
		t.Fatalf("LoadFile() settings = %#v, want %#v", got, want)
	}
}

func TestLoadFileCorruptJSONReturnsEmptyStoreAndError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	store, err := LoadFile(path)
	if err == nil {
		t.Fatal("LoadFile() returned nil error for corrupt JSON")
	}
	if store == nil || len(store.ListCards()) != 0 {
		t.Fatal("LoadFile() did not return an empty store for corrupt JSON")
	}
}

func TestSaveFileRoundTripExcludesHistory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	settings := Settings{HistorySize: 3, MaxConcurrentActions: 2}
	store := NewStore(settings)
	card := validCard()
	card.ID = "round-trip"
	created, err := store.CreateCard(card)
	if err != nil {
		t.Fatalf("CreateCard() error = %v", err)
	}
	if err := store.AppendRun(created.ID, Run{ActionKind: "primary", ExitCode: 7}); err != nil {
		t.Fatalf("AppendRun() error = %v", err)
	}
	if err := store.SaveFile(path); err != nil {
		t.Fatalf("SaveFile() error = %v", err)
	}

	loaded, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}
	if loaded.GetSettings() != settings || !reflect.DeepEqual(loaded.ListCards(), []ActionCard{created}) {
		t.Fatalf("loaded configuration differs: settings=%#v cards=%#v", loaded.GetSettings(), loaded.ListCards())
	}
	runs, err := loaded.GetRuns(created.ID, "primary")
	if err != nil {
		t.Fatalf("GetRuns() error = %v", err)
	}
	if len(runs) != 0 {
		t.Fatalf("SaveFile() persisted %d history records", len(runs))
	}

	var raw map[string]json.RawMessage
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if _, ok := raw["history"]; ok {
		t.Fatal("SaveFile() wrote a history key")
	}
}

func TestSaveFileWithHistoryRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	store := NewStore(validSettings())
	card := validCard()
	card.ID = "history-card"
	created, err := store.CreateCard(card)
	if err != nil {
		t.Fatalf("CreateCard() error = %v", err)
	}
	startedAt := time.Date(2026, time.September, 25, 12, 0, 0, 0, time.UTC)
	primary := Run{ActionKind: "primary", StartedAt: startedAt, Duration: 2 * time.Second, ExitCode: 0, Output: "ok", Outcome: RunOutcomeOK}
	statusAt := startedAt.Add(time.Second)
	status := StatusSnapshot{State: StatusStateFail, CheckedAt: statusAt, LastCheck: Run{ActionKind: "status", StartedAt: statusAt, Duration: time.Second, ExitCode: 1, Output: "down", Truncated: true, Outcome: RunOutcomeFail}}
	if err := store.AppendRun(created.ID, primary); err != nil {
		t.Fatalf("AppendRun(primary) error = %v", err)
	}
	if err := store.UpdateStatus(created.ID, status); err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}
	if err := store.SaveFileWithHistory(path); err != nil {
		t.Fatalf("SaveFileWithHistory() error = %v", err)
	}

	loaded, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}
	loadedPrimary, err := loaded.GetRuns(created.ID, "primary")
	if err != nil {
		t.Fatalf("GetRuns(primary) error = %v", err)
	}
	loadedStatus, err := loaded.GetStatusChanges(created.ID)
	if err != nil {
		t.Fatalf("GetStatusChanges() error = %v", err)
	}
	loadedSnapshot, exists := loaded.GetStatus(created.ID)
	if !exists || loadedSnapshot != status {
		t.Fatalf("loaded status snapshot = %#v, exists = %t", loadedSnapshot, exists)
	}
	if !reflect.DeepEqual(loadedPrimary, []Run{primary}) || len(loadedStatus) != 1 || loadedStatus[0].State != status.State || !loadedStatus[0].StartedAt.Equal(status.CheckedAt) {
		t.Fatalf("loaded history differs: primary=%#v status=%#v", loadedPrimary, loadedStatus)
	}
}

func TestLoadFileCorruptHistoryKeepsConfiguration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	card := validCard()
	card.ID = "history-warning"
	file := map[string]any{
		"settings": Settings{HistorySize: 2, MaxConcurrentActions: 1},
		"cards":    []ActionCard{card},
		"history":  "not an object",
	}
	data, err := json.Marshal(file)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	loaded, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}
	if len(loaded.ListCards()) != 1 {
		t.Fatalf("LoadFile() loaded %d cards, want 1", len(loaded.ListCards()))
	}
	runs, err := loaded.GetRuns(card.ID, "primary")
	if err != nil {
		t.Fatalf("GetRuns() error = %v", err)
	}
	if len(runs) != 0 {
		t.Fatalf("corrupt history loaded %d records", len(runs))
	}
}

func TestLoadFileInvalidCardReturnsErrorAndEmptyStore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	file := persistedFile{
		Settings: validSettings(),
		Cards:    []ActionCard{{ID: "invalid", Name: "missing primary command"}},
	}
	data, err := json.Marshal(file)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	store, err := LoadFile(path)
	if err == nil {
		t.Fatal("LoadFile() returned nil error for invalid card")
	}
	if store == nil || len(store.ListCards()) != 0 {
		t.Fatal("LoadFile() did not return an empty store for invalid card")
	}
}

func TestSaveFileWriteError(t *testing.T) {
	store := NewStore(validSettings())
	missingDirectory := filepath.Join(t.TempDir(), "missing", "config.json")
	if err := store.SaveFile(missingDirectory); err == nil {
		t.Fatal("SaveFile() returned nil error for missing directory")
	}
	if err := store.SaveFileWithHistory(missingDirectory); err == nil {
		t.Fatal("SaveFileWithHistory() returned nil error for missing directory")
	}
}

func TestStoreOnChangeHookPersistsMutationsButNotRuns(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	store := NewStore(validSettings())
	calls := 0
	store.OnChange = func(changed *Store) error {
		calls++
		return changed.SaveFile(path)
	}

	card := validCard()
	card.ID = "hook-card"
	created, err := store.CreateCard(card)
	if err != nil {
		t.Fatalf("CreateCard() error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("OnChange calls after create = %d, want 1", calls)
	}
	if err := store.AppendRun(created.ID, Run{ActionKind: "primary"}); err != nil {
		t.Fatalf("AppendRun() error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("OnChange calls after run = %d, want 1", calls)
	}
	if _, err := store.UpdateCard(created.ID, created); err != nil {
		t.Fatalf("UpdateCard() error = %v", err)
	}
	if err := store.UpdateSettings(validSettings()); err != nil {
		t.Fatalf("UpdateSettings() error = %v", err)
	}
	if err := store.DeleteCard(created.ID); err != nil {
		t.Fatalf("DeleteCard() error = %v", err)
	}
	if calls != 4 {
		t.Fatalf("OnChange calls after mutations = %d, want 4", calls)
	}

	if _, err := LoadFile(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("persisted hook file could not be loaded: %v", err)
	}
}
