package config

import (
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestStoreCRUD(t *testing.T) {
	store := NewStore(validSettings())
	card := validCard()
	card.ID = ""
	card.Primary.Args = []string{"hello"}
	card.Primary.Env = map[string]string{"MODE": "test"}

	created, err := store.CreateCard(card)
	if err != nil {
		t.Fatalf("CreateCard() error = %v", err)
	}
	if created.ID == "" {
		t.Fatal("CreateCard() returned an empty generated ID")
	}
	if _, ok := store.GetCard(created.ID); !ok {
		t.Fatal("GetCard() did not find created card")
	}

	if _, err := store.CreateCard(created); err == nil {
		t.Fatal("CreateCard() accepted a duplicate ID")
	}

	created.Name = "Updated card"
	updated, err := store.UpdateCard(created.ID, created)
	if err != nil {
		t.Fatalf("UpdateCard() error = %v", err)
	}
	if updated.Name != "Updated card" || updated.ID != created.ID {
		t.Fatalf("UpdateCard() returned %#v", updated)
	}

	if _, err := store.UpdateCard("missing", validCard()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("UpdateCard() error = %v, want ErrNotFound", err)
	}
	if err := store.DeleteCard("missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("DeleteCard() error = %v, want ErrNotFound", err)
	}
	if err := store.DeleteCard(created.ID); err != nil {
		t.Fatalf("DeleteCard() error = %v", err)
	}
	if _, ok := store.GetCard(created.ID); ok {
		t.Fatal("GetCard() found deleted card")
	}
}

func TestStoreCopiesCards(t *testing.T) {
	store := NewStore(validSettings())
	card := validCard()
	card.Primary.Args = []string{"original"}
	card.Primary.Env = map[string]string{"MODE": "original"}
	status := validAction()
	card.Status = &status

	created, err := store.CreateCard(card)
	if err != nil {
		t.Fatalf("CreateCard() error = %v", err)
	}
	if created.Primary.Env["MODE"] != "original" {
		t.Fatalf("CreateCard() returned env = %v", created.Primary.Env)
	}
	card.Primary.Args[0] = "caller mutation"
	card.Primary.Env["MODE"] = "caller mutation"
	card.Status.Command = "caller mutation"

	stored, ok := store.GetCard(created.ID)
	if !ok {
		t.Fatal("GetCard() did not find created card")
	}
	if stored.Primary.Args[0] != "original" {
		t.Fatalf("store args changed through input mutation: %v", stored.Primary.Args)
	}
	if stored.Primary.Env["MODE"] != "original" {
		t.Fatalf("store env changed through input mutation: %v", stored.Primary.Env)
	}
	if stored.Status.Command != "echo" {
		t.Fatalf("store status changed through input mutation: %q", stored.Status.Command)
	}

	stored.Primary.Args[0] = "returned mutation"
	stored.Primary.Env["MODE"] = "returned mutation"
	stored.Status.Command = "returned mutation"
	listed := store.ListCards()
	if len(listed) != 1 {
		t.Fatalf("ListCards() length = %d, want 1", len(listed))
	}
	if listed[0].Primary.Args[0] != "original" {
		t.Fatalf("store args changed through returned card mutation: %v", listed[0].Primary.Args)
	}
	if listed[0].Primary.Env["MODE"] != "original" {
		t.Fatalf("store env changed through returned card mutation: %v", listed[0].Primary.Env)
	}
	if listed[0].Status.Command != "echo" {
		t.Fatalf("store status changed through returned card mutation: %q", listed[0].Status.Command)
	}
}

func TestStoreRunHistory(t *testing.T) {
	settings := validSettings()
	settings.HistorySize = 2
	store := NewStore(settings)
	card, err := store.CreateCard(validCard())
	if err != nil {
		t.Fatalf("CreateCard() error = %v", err)
	}

	for index := 1; index <= 3; index++ {
		run := Run{ActionKind: "primary", ExitCode: index}
		if err := store.AppendRun(card.ID, run); err != nil {
			t.Fatalf("AppendRun() error = %v", err)
		}
	}
	if err := store.AppendRun(card.ID, Run{ActionKind: "status", ExitCode: 99}); err != nil {
		t.Fatalf("AppendRun(status) error = %v", err)
	}

	primary, err := store.GetRuns(card.ID, "primary")
	if err != nil {
		t.Fatalf("GetRuns(primary) error = %v", err)
	}
	if len(primary) != 2 || primary[0].ExitCode != 3 || primary[1].ExitCode != 2 {
		t.Fatalf("GetRuns(primary) = %#v, want newest-first [3 2]", primary)
	}
	primary[0].ExitCode = 100
	again, err := store.GetRuns(card.ID, "primary")
	if err != nil {
		t.Fatalf("GetRuns(primary) second call error = %v", err)
	}
	if again[0].ExitCode != 3 {
		t.Fatal("GetRuns() returned a slice backed by store state")
	}

	status, err := store.GetRuns(card.ID, "status")
	if err != nil || len(status) != 1 || status[0].ExitCode != 99 {
		t.Fatalf("GetRuns(status) = %#v, error = %v", status, err)
	}
	missingKind, err := store.GetRuns(card.ID, "unknown")
	if err != nil || len(missingKind) != 0 {
		t.Fatalf("GetRuns(unknown) = %#v, error = %v", missingKind, err)
	}
	if err := store.AppendRun("missing", Run{ActionKind: "primary"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("AppendRun(missing) error = %v, want ErrNotFound", err)
	}
	if _, err := store.GetRuns("missing", "primary"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetRuns(missing) error = %v, want ErrNotFound", err)
	}
}

func TestStoreConcurrentAccess(t *testing.T) {
	store := NewStore(validSettings())
	card, err := store.CreateCard(validCard())
	if err != nil {
		t.Fatalf("CreateCard() error = %v", err)
	}

	var waitGroup sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		worker := worker
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			for iteration := 0; iteration < 50; iteration++ {
				_ = store.AppendRun(card.ID, Run{ActionKind: "primary", ExitCode: worker})
				_, _ = store.GetCard(card.ID)
				_ = store.ListCards()
				_, _ = store.GetRuns(card.ID, "primary")
			}
		}()
	}
	waitGroup.Wait()

	runs, err := store.GetRuns(card.ID, "primary")
	if err != nil {
		t.Fatalf("GetRuns() error = %v", err)
	}
	if len(runs) != DefaultHistorySize {
		t.Fatalf("GetRuns() length = %d, want %d", len(runs), DefaultHistorySize)
	}
}

func TestStoreSerializesOnChangeHooks(t *testing.T) {
	store := NewStore(validSettings())
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	secondStarted := make(chan struct{}, 1)
	var calls int
	var callsMu sync.Mutex
	store.OnChange = func(*Store) error {
		callsMu.Lock()
		calls++
		callNumber := calls
		callsMu.Unlock()
		if callNumber == 1 {
			close(firstStarted)
			<-releaseFirst
			return nil
		}
		secondStarted <- struct{}{}
		return nil
	}

	firstDone := make(chan struct{})
	go func() {
		card := validCard()
		card.ID = "first"
		_, _ = store.CreateCard(card)
		close(firstDone)
	}()
	<-firstStarted

	secondDone := make(chan struct{})
	go func() {
		card := validCard()
		card.ID = "second"
		_, _ = store.CreateCard(card)
		close(secondDone)
	}()

	select {
	case <-secondStarted:
		t.Fatal("second OnChange callback started before the first completed")
	case <-time.After(20 * time.Millisecond):
	}
	close(releaseFirst)
	<-firstDone
	<-secondDone

	select {
	case <-secondStarted:
	case <-time.After(time.Second):
		t.Fatal("second OnChange callback did not complete")
	}
}

func TestNewStoreDefaultsInvalidSettings(t *testing.T) {
	store := NewStore(Settings{})
	settings := store.GetSettings()
	want := validSettings()
	if settings != want {
		t.Fatalf("NewStore(Settings{}) settings = %#v, want %#v", settings, want)
	}
}

func TestStoreListCardsSorted(t *testing.T) {
	store := NewStore(validSettings())
	for _, id := range []string{"card-b", "card-a"} {
		card := validCard()
		card.ID = id
		if _, err := store.CreateCard(card); err != nil {
			t.Fatalf("CreateCard(%s) error = %v", id, err)
		}
	}
	cards := store.ListCards()
	if len(cards) != 2 || cards[0].ID != "card-a" || cards[1].ID != "card-b" {
		t.Fatalf("ListCards() = %#v", cards)
	}
}

func TestStoreDeleteRemovesHistory(t *testing.T) {
	store := NewStore(validSettings())
	card, err := store.CreateCard(validCard())
	if err != nil {
		t.Fatalf("CreateCard() error = %v", err)
	}
	if err := store.AppendRun(card.ID, Run{ActionKind: "primary"}); err != nil {
		t.Fatalf("AppendRun() error = %v", err)
	}
	if err := store.DeleteCard(card.ID); err != nil {
		t.Fatalf("DeleteCard() error = %v", err)
	}
	if _, err := store.GetRuns(card.ID, "primary"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetRuns() error = %v, want ErrNotFound", err)
	}
}

func ExampleStore_GetRuns() {
	store := NewStore(validSettings())
	card, _ := store.CreateCard(validCard())
	_ = store.AppendRun(card.ID, Run{ActionKind: "primary", ExitCode: 0})
	runs, _ := store.GetRuns(card.ID, "primary")
	fmt.Println(len(runs), runs[0].ExitCode)
	// Output: 1 0
}

func TestStoreStatusTransitionsIgnoreRepeatedChecks(t *testing.T) {
	settings := validSettings()
	settings.HistorySize = 3
	store := NewStore(settings)
	card, err := store.CreateCard(validCard())
	if err != nil {
		t.Fatalf("CreateCard() error = %v", err)
	}
	firstAt := time.Date(2026, time.September, 25, 12, 0, 0, 0, time.UTC)
	check := func(state StatusState, checkedAt time.Time) {
		t.Helper()
		err := store.UpdateStatus(card.ID, StatusSnapshot{
			State:     state,
			CheckedAt: checkedAt,
			LastCheck: Run{ActionKind: "status", StartedAt: checkedAt, Outcome: RunOutcomeOK},
		})
		if err != nil {
			t.Fatalf("UpdateStatus(%q) error = %v", state, err)
		}
	}

	check(StatusStateOK, firstAt)
	check(StatusStateOK, firstAt.Add(5*time.Second))
	changes, err := store.GetStatusChanges(card.ID)
	if err != nil || len(changes) != 1 || changes[0].State != StatusStateOK || changes[0].EndedAt != nil {
		t.Fatalf("repeated status changes = %#v, error = %v", changes, err)
	}

	check(StatusStateFail, firstAt.Add(10*time.Second))
	check(StatusStateOK, firstAt.Add(25*time.Second))
	changes, err = store.GetStatusChanges(card.ID)
	if err != nil || len(changes) != 3 {
		t.Fatalf("status changes = %#v, error = %v", changes, err)
	}
	if changes[0].State != StatusStateOK || changes[0].StartedAt != firstAt.Add(25*time.Second) || changes[0].EndedAt != nil {
		t.Fatalf("current status change = %#v", changes[0])
	}
	if changes[1].State != StatusStateFail || changes[1].Duration != 15*time.Second {
		t.Fatalf("fail status duration = %#v", changes[1])
	}
	if changes[2].State != StatusStateOK || changes[2].Duration != 10*time.Second {
		t.Fatalf("ok status duration = %#v", changes[2])
	}

	snapshot, ok := store.GetStatus(card.ID)
	if !ok || snapshot.State != StatusStateOK || !snapshot.CheckedAt.Equal(firstAt.Add(25*time.Second)) {
		t.Fatalf("status snapshot = %#v, exists = %t", snapshot, ok)
	}
	changes[1].EndedAt = nil
	again, _ := store.GetStatusChanges(card.ID)
	if again[1].EndedAt == nil {
		t.Fatal("GetStatusChanges() returned shared EndedAt pointer")
	}
}

func TestStoreStatusHistoryTrimsWithSettings(t *testing.T) {
	settings := validSettings()
	settings.HistorySize = 2
	store := NewStore(settings)
	card, err := store.CreateCard(validCard())
	if err != nil {
		t.Fatalf("CreateCard() error = %v", err)
	}
	base := time.Date(2026, time.September, 25, 12, 0, 0, 0, time.UTC)
	for index, state := range []StatusState{StatusStateOK, StatusStateFail, StatusStateOK} {
		if err := store.UpdateStatus(card.ID, StatusSnapshot{State: state, CheckedAt: base.Add(time.Duration(index) * time.Second)}); err != nil {
			t.Fatalf("UpdateStatus() error = %v", err)
		}
	}
	changes, err := store.GetStatusChanges(card.ID)
	if err != nil || len(changes) != 2 {
		t.Fatalf("trimmed status changes = %#v, error = %v", changes, err)
	}
}

func TestStoreStatusChangeCallbackOnlyRunsForTransitions(t *testing.T) {
	store := NewStore(validSettings())
	card, err := store.CreateCard(validCard())
	if err != nil {
		t.Fatalf("CreateCard() error = %v", err)
	}
	callbacks := make(chan StatusState, 2)
	store.OnStatusChange = func(_ string, snapshot StatusSnapshot) {
		callbacks <- snapshot.State
	}
	checkedAt := time.Date(2026, time.September, 25, 12, 0, 0, 0, time.UTC)
	for _, state := range []StatusState{StatusStateOK, StatusStateOK, StatusStateFail} {
		if err := store.UpdateStatus(card.ID, StatusSnapshot{State: state, CheckedAt: checkedAt}); err != nil {
			t.Fatalf("UpdateStatus(%q) error = %v", state, err)
		}
		checkedAt = checkedAt.Add(time.Second)
	}

	if first := <-callbacks; first != StatusStateOK {
		t.Fatalf("first callback state = %q", first)
	}
	if second := <-callbacks; second != StatusStateFail {
		t.Fatalf("second callback state = %q", second)
	}
	select {
	case extra := <-callbacks:
		t.Fatalf("unexpected extra callback state = %q", extra)
	default:
	}
}

func TestStoreRollsBackMutationsWhenPersistenceFails(t *testing.T) {
	store := NewStore(validSettings())
	card := validCard()
	card.ID = "kept"
	if _, err := store.CreateCard(card); err != nil {
		t.Fatalf("CreateCard() error = %v", err)
	}
	if err := store.AppendRun(card.ID, Run{ActionKind: "primary", Outcome: RunOutcomeOK}); err != nil {
		t.Fatalf("AppendRun() error = %v", err)
	}
	if err := store.UpdateStatus(card.ID, StatusSnapshot{State: StatusStateOK}); err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}
	failing := errors.New("disk full")
	store.OnChange = func(*Store) error { return failing }
	var persistenceErr *PersistenceError

	extra := validCard()
	extra.ID = "extra"
	if _, err := store.CreateCard(extra); !errors.As(err, &persistenceErr) || !errors.Is(err, failing) {
		t.Fatalf("CreateCard() error = %v, want PersistenceError wrapping the cause", err)
	}
	if _, exists := store.GetCard(extra.ID); exists {
		t.Fatal("card created despite persistence failure")
	}

	renamed := card
	renamed.Name = "renamed"
	if _, err := store.UpdateCard(card.ID, renamed); !errors.As(err, &persistenceErr) {
		t.Fatalf("UpdateCard() error = %v", err)
	}
	if got, _ := store.GetCard(card.ID); got.Name != card.Name {
		t.Fatalf("card updated despite persistence failure: %q", got.Name)
	}

	if err := store.DeleteCard(card.ID); !errors.As(err, &persistenceErr) {
		t.Fatalf("DeleteCard() error = %v", err)
	}
	if _, exists := store.GetCard(card.ID); !exists {
		t.Fatal("card deleted despite persistence failure")
	}
	if runs, err := store.GetRuns(card.ID, "primary"); err != nil || len(runs) != 1 {
		t.Fatalf("run history not restored after failed delete: %v, %v", runs, err)
	}
	if snapshot, exists := store.GetStatus(card.ID); !exists || snapshot.State != StatusStateOK {
		t.Fatalf("status not restored after failed delete: %#v, %v", snapshot, exists)
	}
	if changes, err := store.GetStatusChanges(card.ID); err != nil || len(changes) != 1 {
		t.Fatalf("status history not restored after failed delete: %v, %v", changes, err)
	}
}

func TestStoreErrorsAreClassifiable(t *testing.T) {
	store := NewStore(validSettings())
	card := validCard()
	card.ID = "dup"
	if _, err := store.CreateCard(card); err != nil {
		t.Fatalf("CreateCard() error = %v", err)
	}
	if _, err := store.CreateCard(card); !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("duplicate CreateCard() error = %v, want ErrAlreadyExists", err)
	}
	if _, err := store.CreateCard(ActionCard{ID: "invalid"}); !errors.Is(err, ErrValidation) {
		t.Fatalf("invalid CreateCard() error = %v, want ErrValidation", err)
	}
	if _, err := store.UpdateCard("dup", ActionCard{}); !errors.Is(err, ErrValidation) {
		t.Fatalf("invalid UpdateCard() error = %v, want ErrValidation", err)
	}
	if _, err := store.UpdateCard("missing", card); !errors.Is(err, ErrNotFound) {
		t.Fatalf("UpdateCard() error = %v, want ErrNotFound", err)
	}
	if err := store.UpdateStatus("dup", StatusSnapshot{State: "bogus"}); !errors.Is(err, ErrValidation) {
		t.Fatalf("UpdateStatus() error = %v, want ErrValidation", err)
	}
}

func TestStoreDirtyTracksChangesSinceHistorySave(t *testing.T) {
	store := NewStore(Settings{HistorySize: 5, MaxConcurrentActions: 1})
	if store.Dirty() {
		t.Fatal("new store must not be dirty")
	}
	card, err := store.CreateCard(ActionCard{ID: "dirty", Name: "Dirty", Primary: Action{Command: "true", TimeoutSec: 1}, Status: &Action{Command: "true", TimeoutSec: 1}})
	if err != nil {
		t.Fatalf("CreateCard() error = %v", err)
	}
	path := filepath.Join(t.TempDir(), "marionette.json")
	if err := store.SaveFileWithHistory(path); err != nil {
		t.Fatalf("SaveFileWithHistory() error = %v", err)
	}
	if store.Dirty() {
		t.Fatal("store is dirty right after SaveFileWithHistory")
	}
	if err := store.AppendRun(card.ID, Run{ActionKind: "primary", Outcome: RunOutcomeOK, StartedAt: time.Now()}); err != nil {
		t.Fatalf("AppendRun() error = %v", err)
	}
	if !store.Dirty() {
		t.Fatal("AppendRun did not mark the store dirty")
	}
	if err := store.SaveFileWithHistory(path); err != nil {
		t.Fatalf("second SaveFileWithHistory() error = %v", err)
	}
	if err := store.UpdateStatus(card.ID, StatusSnapshot{State: StatusStateOK}); err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}
	if !store.Dirty() {
		t.Fatal("UpdateStatus did not mark the store dirty")
	}
	if err := store.SaveFile(path); err != nil {
		t.Fatalf("SaveFile() error = %v", err)
	}
	if !store.Dirty() {
		t.Fatal("SaveFile without history must not clear the dirty flag")
	}
	loaded, err := LoadFile(path, nil)
	if err != nil {
		t.Fatalf("LoadFile(, nil) error = %v", err)
	}
	if loaded.Dirty() {
		t.Fatal("freshly loaded store must not be dirty")
	}
}

func TestStoreRollbackKeepsDirtyHonest(t *testing.T) {
	store := NewStore(Settings{HistorySize: 5, MaxConcurrentActions: 1})
	card := validCard()
	card.ID = "kept"
	if _, err := store.CreateCard(card); err != nil {
		t.Fatalf("CreateCard() error = %v", err)
	}
	if err := store.SaveFileWithHistory(filepath.Join(t.TempDir(), "marionette.json")); err != nil {
		t.Fatalf("SaveFileWithHistory() error = %v", err)
	}
	store.OnChange = func(*Store) error { return errors.New("disk full") }

	extra := validCard()
	extra.ID = "extra"
	if _, err := store.CreateCard(extra); err == nil {
		t.Fatal("CreateCard() succeeded despite failing OnChange")
	}
	renamed := card
	renamed.Name = "renamed"
	if _, err := store.UpdateCard(card.ID, renamed); err == nil {
		t.Fatal("UpdateCard() succeeded despite failing OnChange")
	}
	if err := store.DeleteCard(card.ID); err == nil {
		t.Fatal("DeleteCard() succeeded despite failing OnChange")
	}
	if store.Dirty() {
		t.Fatal("store reports unsaved changes although every mutation was rolled back")
	}
}
