package status

import (
	"errors"
	"testing"
	"time"

	"marionette/internal/config"
	execengine "marionette/internal/exec"
)

type fakeRunner struct {
	result execengine.Result
	err    error
	action config.Action
}

func (runner *fakeRunner) Run(action config.Action) (execengine.Result, error) {
	runner.action = action
	return runner.result, runner.err
}

func testStatusCard(t *testing.T, store *config.Store) config.ActionCard {
	t.Helper()
	card := config.ActionCard{
		ID:      "card-status",
		Name:    "Status card",
		Primary: config.Action{Command: "primary", TimeoutSec: 1},
		Status:  &config.Action{Command: "status", Args: []string{"--check"}, TimeoutSec: 3},
	}
	created, err := store.CreateCard(card)
	if err != nil {
		t.Fatalf("CreateCard() error = %v", err)
	}
	return created
}

func TestCheckNowMapsOutcomeAndStoresSnapshot(t *testing.T) {
	checkedAt := time.Date(2026, time.September, 25, 12, 0, 0, 0, time.UTC)
	runner := &fakeRunner{result: execengine.Result{
		StartedAt: checkedAt,
		Duration:  2 * time.Second,
		Outcome:   config.RunOutcomeTimeout,
	}}
	store := config.NewStore(config.Settings{HistorySize: 5, MaxConcurrentActions: 1})
	card := testStatusCard(t, store)
	service, err := NewStatusCheckService(store, runner)
	if err != nil {
		t.Fatalf("NewCheckService() error = %v", err)
	}

	snapshot, err := service.CheckNow(card.ID)
	if err != nil {
		t.Fatalf("CheckNow() error = %v", err)
	}
	if snapshot.State != config.StatusStateFail || snapshot.CheckedAt != checkedAt.Add(2*time.Second) {
		t.Fatalf("snapshot = %#v", snapshot)
	}
	if runner.action.Command != "status" || runner.action.Args[0] != "--check" {
		t.Fatalf("runner action = %#v", runner.action)
	}
	stored, ok := store.GetStatus(card.ID)
	if !ok || stored.LastCheck.ActionKind != "status" || stored.State != config.StatusStateFail {
		t.Fatalf("stored status = %#v, found = %v", stored, ok)
	}
}

func TestCheckNowDoesNotDuplicateRepeatedState(t *testing.T) {
	checkedAt := time.Date(2026, time.September, 25, 12, 0, 0, 0, time.UTC)
	runner := &fakeRunner{result: execengine.Result{StartedAt: checkedAt, Outcome: config.RunOutcomeOK}}
	store := config.NewStore(config.Settings{HistorySize: 5, MaxConcurrentActions: 1})
	card := testStatusCard(t, store)
	service, _ := NewStatusCheckService(store, runner)

	if _, err := service.CheckNow(card.ID); err != nil {
		t.Fatalf("first CheckNow() error = %v", err)
	}
	runner.result.StartedAt = checkedAt.Add(time.Minute)
	if _, err := service.CheckNow(card.ID); err != nil {
		t.Fatalf("second CheckNow() error = %v", err)
	}
	changes, err := store.GetStatusChanges(card.ID)
	if err != nil || len(changes) != 1 {
		t.Fatalf("status changes = %#v, error = %v", changes, err)
	}
	stored, _ := store.GetStatus(card.ID)
	if stored.CheckedAt != checkedAt.Add(time.Minute) {
		t.Fatalf("latest status checked at = %v", stored.CheckedAt)
	}
}

func TestCheckNowRejectsMissingCardAndStatusAction(t *testing.T) {
	store := config.NewStore(config.Settings{HistorySize: 5, MaxConcurrentActions: 1})
	service, err := NewStatusCheckService(store, &fakeRunner{})
	if err != nil {
		t.Fatalf("NewCheckService() error = %v", err)
	}
	if _, err := service.CheckNow("missing"); !errors.Is(err, config.ErrNotFound) {
		t.Fatalf("missing card error = %v", err)
	}
	card := config.ActionCard{ID: "without-status", Name: "No status", Primary: config.Action{Command: "primary", TimeoutSec: 1}}
	if _, err := store.CreateCard(card); err != nil {
		t.Fatalf("CreateCard() error = %v", err)
	}
	if _, err := service.CheckNow(card.ID); !errors.Is(err, ErrStatusActionMissing) {
		t.Fatalf("missing status error = %v", err)
	}
}

func TestCheckNowDoesNotUpdateOnRunnerError(t *testing.T) {
	runner := &fakeRunner{err: errors.New("runner failed")}
	store := config.NewStore(config.Settings{HistorySize: 5, MaxConcurrentActions: 1})
	card := testStatusCard(t, store)
	service, _ := NewStatusCheckService(store, runner)

	if _, err := service.CheckNow(card.ID); err == nil {
		t.Fatal("CheckNow() succeeded despite runner error")
	}
	if _, ok := store.GetStatus(card.ID); ok {
		t.Fatal("CheckNow() stored status after runner error")
	}
}
