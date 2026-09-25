package config

import (
	"errors"
	"fmt"
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

	if err := store.UpdateSettings(Settings{}); err == nil {
		t.Fatal("UpdateSettings() accepted invalid settings")
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

func TestStoreUpdateSettingsTrimsHistory(t *testing.T) {
	store := NewStore(validSettings())
	card, err := store.CreateCard(validCard())
	if err != nil {
		t.Fatalf("CreateCard() error = %v", err)
	}
	for index := 1; index <= 3; index++ {
		if err := store.AppendRun(card.ID, Run{ActionKind: "primary", ExitCode: index}); err != nil {
			t.Fatalf("AppendRun() error = %v", err)
		}
	}

	settings := validSettings()
	settings.HistorySize = 1
	if err := store.UpdateSettings(settings); err != nil {
		t.Fatalf("UpdateSettings() error = %v", err)
	}
	runs, err := store.GetRuns(card.ID, "primary")
	if err != nil {
		t.Fatalf("GetRuns() error = %v", err)
	}
	if len(runs) != 1 || runs[0].ExitCode != 3 {
		t.Fatalf("GetRuns() = %#v, want newest run only", runs)
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
