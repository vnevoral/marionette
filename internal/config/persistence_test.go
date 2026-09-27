package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestLoadFileMissingReturnsDefaultStore(t *testing.T) {
	store, err := LoadFile(filepath.Join(t.TempDir(), "missing.json"), nil)
	if err != nil {
		t.Fatalf("LoadFile(, nil) error = %v", err)
	}
	if len(store.ListCards()) != 0 {
		t.Fatal("LoadFile(, nil) returned cards for a missing file")
	}
	want := Settings{HistorySize: DefaultHistorySize, MaxConcurrentActions: DefaultMaxConcurrentActions}
	if got := store.GetSettings(); got != want {
		t.Fatalf("LoadFile(, nil) settings = %#v, want %#v", got, want)
	}
}

func TestLoadFileCorruptJSONReturnsEmptyStoreAndError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	store, err := LoadFile(path, nil)
	if !errors.Is(err, ErrConfigCorrupt) {
		t.Fatalf("LoadFile(, nil) error = %v, want ErrConfigCorrupt", err)
	}
	if store == nil || len(store.ListCards()) != 0 {
		t.Fatal("LoadFile(, nil) did not return an empty store for corrupt JSON")
	}
}

func TestLoadFileUnreadableReturnsUnreadableError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores file permissions")
	}
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte("{}"), 0o000); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	_, err := LoadFile(path, nil)
	if !errors.Is(err, ErrConfigUnreadable) {
		t.Fatalf("LoadFile(, nil) error = %v, want ErrConfigUnreadable", err)
	}
	if errors.Is(err, ErrConfigCorrupt) {
		t.Fatal("unreadable file must not be reported as corrupt")
	}
}

func TestQuarantineFilePreservesContentAndAvoidsCollisions(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "config.json")
	now := time.Date(2026, time.September, 26, 10, 30, 0, 0, time.UTC)
	if err := os.WriteFile(path, []byte("{broken"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	first, err := QuarantineFile(path, now)
	if err != nil {
		t.Fatalf("QuarantineFile() error = %v", err)
	}
	if first != path+".corrupt-20260926T103000Z" {
		t.Fatalf("QuarantineFile() path = %q", first)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("original path still exists after quarantine: %v", err)
	}
	content, err := os.ReadFile(first)
	if err != nil || string(content) != "{broken" {
		t.Fatalf("quarantined content = %q, error = %v", content, err)
	}

	if err := os.WriteFile(path, []byte("{again"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	second, err := QuarantineFile(path, now)
	if err != nil {
		t.Fatalf("second QuarantineFile() error = %v", err)
	}
	if second != first+"-1" {
		t.Fatalf("second QuarantineFile() path = %q, want %q", second, first+"-1")
	}
	if _, err := QuarantineFile(filepath.Join(directory, "missing.json"), now); err == nil {
		t.Fatal("QuarantineFile() succeeded for a missing file")
	}
}

func TestSaveFilePreservesExistingPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	store := NewStore(validSettings())
	if err := store.SaveFile(path); err != nil {
		t.Fatalf("SaveFile() error = %v", err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("new config file mode = %v, error = %v, want 0600", info.Mode(), err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatalf("Chmod() error = %v", err)
	}
	if err := store.SaveFileWithHistory(path); err != nil {
		t.Fatalf("SaveFileWithHistory() error = %v", err)
	}
	info, err = os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o644 {
		t.Fatalf("rewritten config file mode = %v, error = %v, want 0644", info.Mode(), err)
	}
	if _, err := os.Stat(path + ".tmp"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("temporary file left behind: %v", err)
	}
}

func TestLoadFileIgnoresStatusWithUnknownState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	card := validCard()
	card.ID = "status-card"
	data := []byte(`{"settings":{"historySize":5,"maxConcurrentActions":1},"cards":[` +
		mustJSON(t, card) + `],"status":{"status-card":{"state":"bogus","checkedAt":"2026-09-26T10:00:00Z"}}}`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	store, err := LoadFile(path, nil)
	if err != nil {
		t.Fatalf("LoadFile(, nil) error = %v, want nil (invalid runtime state is ignored)", err)
	}
	if _, exists := store.GetStatus(card.ID); exists {
		t.Fatal("status with unknown state was loaded")
	}
}

func mustJSON(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return string(data)
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

	loaded, err := LoadFile(path, nil)
	if err != nil {
		t.Fatalf("LoadFile(, nil) error = %v", err)
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

// A card colour (FR-10a) survives a save and load; a card without one is
// written without the key, as by versions that had no colours.
func TestSaveFileRoundTripKeepsCardColor(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	store := NewStore(validSettings())
	colored := validCard()
	colored.ID = "colored"
	colored.Color = "teal"
	plain := validCard()
	plain.ID = "plain"
	for _, card := range []ActionCard{colored, plain} {
		if _, err := store.CreateCard(card); err != nil {
			t.Fatalf("CreateCard(%s) error = %v", card.ID, err)
		}
	}
	if err := store.SaveFile(path); err != nil {
		t.Fatalf("SaveFile() error = %v", err)
	}

	loaded, err := LoadFile(path, nil)
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}
	if card, _ := loaded.GetCard("colored"); card.Color != "teal" {
		t.Fatalf("colored card loaded with color %q, want teal", card.Color)
	}
	if card, _ := loaded.GetCard("plain"); card.Color != "" {
		t.Fatalf("plain card loaded with color %q, want none", card.Color)
	}

	var raw struct {
		Cards []map[string]json.RawMessage `json:"cards"`
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	for _, card := range raw.Cards {
		_, hasColor := card["color"]
		if id := string(card["id"]); hasColor != (id == `"colored"`) {
			t.Fatalf("card %s written with color key = %v", id, hasColor)
		}
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

	loaded, err := LoadFile(path, nil)
	if err != nil {
		t.Fatalf("LoadFile(, nil) error = %v", err)
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

	loaded, err := LoadFile(path, nil)
	if err != nil {
		t.Fatalf("LoadFile(, nil) error = %v", err)
	}
	if len(loaded.ListCards()) != 1 {
		t.Fatalf("LoadFile(, nil) loaded %d cards, want 1", len(loaded.ListCards()))
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

	store, err := LoadFile(path, nil)
	if !errors.Is(err, ErrConfigCorrupt) {
		t.Fatalf("LoadFile(, nil) error = %v, want ErrConfigCorrupt for invalid card", err)
	}
	if store == nil || len(store.ListCards()) != 0 {
		t.Fatal("LoadFile(, nil) did not return an empty store for invalid card")
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
	if err := store.DeleteCard(created.ID); err != nil {
		t.Fatalf("DeleteCard() error = %v", err)
	}
	if calls != 3 {
		t.Fatalf("OnChange calls after mutations = %d, want 3", calls)
	}

	loaded, err := LoadFile(path, nil)
	if err != nil {
		t.Fatalf("persisted hook file could not be loaded: %v", err)
	}
	if len(loaded.ListCards()) != 0 {
		t.Fatalf("persisted file still contains %d cards after delete", len(loaded.ListCards()))
	}
}

func TestLoadFileKeepsCardOverCurrentLimits(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	legacy := validCard()
	legacy.ID = "legacy.card"
	legacy.Icon = "pi pi-Home"
	legacy.Primary.TimeoutSec = MaxTimeoutSec + 1
	file := persistedFile{Settings: validSettings(), Cards: []ActionCard{legacy}}
	data, err := json.Marshal(file)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var logs bytes.Buffer
	store, err := LoadFile(path, slog.New(slog.NewTextHandler(&logs, nil)))
	if err != nil {
		t.Fatalf("LoadFile() error = %v, want the card loaded despite the limits", err)
	}
	loaded, exists := store.GetCard(legacy.ID)
	if !exists || loaded.Primary.TimeoutSec != MaxTimeoutSec+1 {
		t.Fatalf("card not loaded as stored: %#v, %v", loaded, exists)
	}
	if !strings.Contains(logs.String(), "violates current limits") || !strings.Contains(logs.String(), legacy.ID) {
		t.Fatalf("no warning for the card over the limits: %s", logs.String())
	}
	// The API boundary still rejects it until it is corrected.
	if _, err := store.UpdateCard(legacy.ID, loaded); !errors.Is(err, ErrValidation) {
		t.Fatalf("UpdateCard() error = %v, want ErrValidation", err)
	}
}

func TestWritePersistedFileConcurrentWritersKeepFileValid(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	store := NewStore(validSettings())
	if _, err := store.CreateCard(validCard()); err != nil {
		t.Fatalf("CreateCard() error = %v", err)
	}
	const writers = 8
	var wg sync.WaitGroup
	failures := make(chan error, writers*2)
	for index := 0; index < writers; index++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			if err := store.SaveFile(path); err != nil {
				failures <- err
			}
		}()
		go func() {
			defer wg.Done()
			if err := store.SaveFileWithHistory(path); err != nil {
				failures <- err
			}
		}()
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		t.Errorf("concurrent save failed: %v", err)
	}
	if _, err := LoadFile(path, nil); err != nil {
		t.Fatalf("file after concurrent saves is not loadable: %v", err)
	}
	leftovers, _ := filepath.Glob(path + ".*.tmp")
	if len(leftovers) != 0 {
		t.Fatalf("temporary files left behind: %v", leftovers)
	}
}
