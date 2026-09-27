package actions

import (
	"context"
	"errors"
	"testing"
	"time"

	"marionette/internal/config"
	"marionette/internal/events"
	"marionette/internal/execengine"
)

func createServerCard(t *testing.T, store *config.Store, id string) config.ActionCard {
	t.Helper()
	card, err := store.CreateCard(config.ActionCard{
		ID:      id,
		Name:    "Queue card",
		Primary: config.Action{Command: "primary", TimeoutSec: 1},
		Status:  &config.Action{Command: "status", TimeoutSec: 1},
	})
	if err != nil {
		t.Fatalf("CreateCard() error = %v", err)
	}
	return card
}

type fakeBackgroundRunner struct {
	called chan config.Action
	result execengine.Result
}

func (runner *fakeBackgroundRunner) Run(_ context.Context, action config.Action) (execengine.Result, error) {
	runner.called <- action
	return runner.result, nil
}

type fakeStatusChecker struct {
	called chan string
}

func (checker *fakeStatusChecker) CheckNow(_ context.Context, cardID string) (config.StatusSnapshot, error) {
	checker.called <- cardID
	return config.StatusSnapshot{State: config.StatusStateOK}, nil
}

func TestQueueRunAndDrainAcceptedWork(t *testing.T) {
	store := config.NewStore(config.Settings{HistorySize: 5, MaxConcurrentActions: 1})
	card := createServerCard(t, store, "background")
	runner := &fakeBackgroundRunner{
		called: make(chan config.Action, 1),
		result: execengine.Result{Outcome: config.RunOutcomeOK},
	}
	checker := &fakeStatusChecker{called: make(chan string, 1)}
	actions, err := New(store, runner, checker, nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := actions.EnqueuePrimary(card.ID, card.Primary); err != nil {
		t.Fatalf("EnqueuePrimary() error = %v", err)
	}
	if err := actions.EnqueueStatus(card.ID); err != nil {
		t.Fatalf("EnqueueStatus() error = %v", err)
	}
	actions.Wait()

	select {
	case <-runner.called:
	default:
		t.Fatal("primary action was not executed")
	}
	select {
	case cardID := <-checker.called:
		if cardID != card.ID {
			t.Fatalf("status card ID = %q", cardID)
		}
	default:
		t.Fatal("status action was not executed")
	}
	runs, err := store.GetRuns(card.ID, "primary")
	if err != nil || len(runs) != 1 || runs[0].Outcome != config.RunOutcomeOK {
		t.Fatalf("primary runs = %#v, error = %v", runs, err)
	}
	if _, err := actions.Close(context.Background()); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := actions.EnqueuePrimary(card.ID, card.Primary); !errors.Is(err, ErrQueueClosed) {
		t.Fatalf("enqueue after close error = %v", err)
	}
}

// queue → store → broker, as wired in cmd/marionette: a finished primary
// action reaches event subscribers as a recorded run (FR-42a).
func TestFinishedPrimaryActionIsPublishedAsRecordedRun(t *testing.T) {
	store := config.NewStore(config.Settings{HistorySize: 5, MaxConcurrentActions: 1})
	card := createServerCard(t, store, "published")
	broker := events.NewBroker()
	store.OnRunAppended = broker.PublishRun
	subscriber, unsubscribe := broker.Subscribe()
	defer unsubscribe()
	runner := &fakeBackgroundRunner{
		called: make(chan config.Action, 1),
		result: execengine.Result{Outcome: config.RunOutcomeFail, ExitCode: 2},
	}
	actions, err := New(store, runner, &fakeStatusChecker{called: make(chan string, 1)}, nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() { _, _ = actions.Close(context.Background()) }()
	if err := actions.EnqueuePrimary(card.ID, card.Primary); err != nil {
		t.Fatalf("EnqueuePrimary() error = %v", err)
	}

	select {
	case event := <-subscriber:
		if event.CardID != card.ID || event.Run == nil || event.Run.ActionKind != KindPrimary ||
			event.Run.Outcome != config.RunOutcomeFail || event.Run.ExitCode != 2 {
			t.Fatalf("event = %#v", event)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no run.recorded event for the finished primary action")
	}
}

type blockingBackgroundRunner struct {
	started chan struct{}
}

func (runner *blockingBackgroundRunner) Run(ctx context.Context, _ config.Action) (execengine.Result, error) {
	close(runner.started)
	<-ctx.Done()
	return execengine.Result{Outcome: config.RunOutcomeCanceled}, nil
}

func TestQueueCloseCancelsRunningJob(t *testing.T) {
	store := config.NewStore(config.Settings{HistorySize: 5, MaxConcurrentActions: 1})
	card := createServerCard(t, store, "cancel-on-close")
	runner := &blockingBackgroundRunner{started: make(chan struct{})}
	actions, err := New(store, runner, &fakeStatusChecker{called: make(chan string, 1)}, nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := actions.EnqueuePrimary(card.ID, card.Primary); err != nil {
		t.Fatalf("EnqueuePrimary() error = %v", err)
	}
	<-runner.started

	closed := make(chan error, 1)
	go func() {
		graceContext, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		_, err := actions.Close(graceContext)
		closed <- err
	}()
	select {
	case err := <-closed:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("Close() error = %v, want deadline exceeded because the running job was cancelled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Close() did not cancel the running job")
	}
	runs, err := store.GetRuns(card.ID, "primary")
	if err != nil || len(runs) != 1 || runs[0].Outcome != config.RunOutcomeCanceled {
		t.Fatalf("primary runs after cancel = %#v, error = %v", runs, err)
	}
}

func TestQueueCloseIdempotent(t *testing.T) {
	store := config.NewStore(config.Settings{HistorySize: 5, MaxConcurrentActions: 1})
	actions, err := New(store, &fakeBackgroundRunner{called: make(chan config.Action, 1)}, &fakeStatusChecker{called: make(chan string, 1)}, nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	for range 3 {
		dropped, err := actions.Close(context.Background())
		if dropped != 0 || err != nil {
			t.Fatalf("Close() = %d, %v", dropped, err)
		}
	}
}

// releasingRunner blocks every run until release is closed or the context is
// cancelled, and reports each start so tests can observe queue progress.
type releasingRunner struct {
	started chan config.Action
	release chan struct{}
}

func (runner *releasingRunner) Run(ctx context.Context, action config.Action) (execengine.Result, error) {
	runner.started <- action
	select {
	case <-runner.release:
		return execengine.Result{Outcome: config.RunOutcomeOK}, nil
	case <-ctx.Done():
		return execengine.Result{Outcome: config.RunOutcomeCanceled}, nil
	}
}

func TestQueueCloseDropsQueued(t *testing.T) {
	store := config.NewStore(config.Settings{HistorySize: 5, MaxConcurrentActions: 1})
	runner := &releasingRunner{started: make(chan config.Action, 16), release: make(chan struct{})}
	actions, err := New(store, runner, &fakeStatusChecker{called: make(chan string, 16)}, nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	running := createServerCard(t, store, "running")
	if err := actions.EnqueuePrimary(running.ID, running.Primary); err != nil {
		t.Fatalf("EnqueuePrimary() error = %v", err)
	}
	<-runner.started
	for index := range 4 {
		card := createServerCard(t, store, "queued-"+string(rune('a'+index)))
		if err := actions.EnqueuePrimary(card.ID, card.Primary); err != nil {
			t.Fatalf("EnqueuePrimary(%s) error = %v", card.ID, err)
		}
	}
	if queued := actions.Queued(); queued != 4 {
		t.Fatalf("Queued() = %d, want 4", queued)
	}

	graceContext, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	started := time.Now()
	dropped, err := actions.Close(graceContext)
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("Close() took %s", elapsed)
	}
	if dropped != 4 || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Close() = %d, %v; want 4 dropped and deadline exceeded", dropped, err)
	}
	if len(runner.started) != 0 {
		t.Fatalf("%d queued job(s) were started after Close()", len(runner.started))
	}
	actions.Wait() // dropped and cancelled jobs are all accounted for
	for _, id := range []string{"queued-a", "queued-b", "queued-c", "queued-d"} {
		if runs, err := store.GetRuns(id, "primary"); err != nil || len(runs) != 0 {
			t.Fatalf("dropped job %s left runs %#v, error = %v", id, runs, err)
		}
	}
}

func TestQueueCloseLetsRunningJobFinishWithinGrace(t *testing.T) {
	store := config.NewStore(config.Settings{HistorySize: 5, MaxConcurrentActions: 1})
	runner := &releasingRunner{started: make(chan config.Action, 1), release: make(chan struct{})}
	actions, err := New(store, runner, &fakeStatusChecker{called: make(chan string, 1)}, nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	card := createServerCard(t, store, "finishing")
	if err := actions.EnqueuePrimary(card.ID, card.Primary); err != nil {
		t.Fatalf("EnqueuePrimary() error = %v", err)
	}
	<-runner.started
	go func() {
		time.Sleep(20 * time.Millisecond)
		close(runner.release)
	}()
	graceContext, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if dropped, err := actions.Close(graceContext); dropped != 0 || err != nil {
		t.Fatalf("Close() = %d, %v", dropped, err)
	}
	runs, err := store.GetRuns(card.ID, "primary")
	if err != nil || len(runs) != 1 || runs[0].Outcome != config.RunOutcomeOK {
		t.Fatalf("primary runs after graceful close = %#v, error = %v", runs, err)
	}
}

func TestEnqueueDeduplicatesWaitingJob(t *testing.T) {
	store := config.NewStore(config.Settings{HistorySize: 5, MaxConcurrentActions: 1})
	runner := &releasingRunner{started: make(chan config.Action, 8), release: make(chan struct{})}
	actions, err := New(store, runner, &fakeStatusChecker{called: make(chan string, 8)}, nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() {
		close(runner.release)
		_, _ = actions.Close(context.Background())
	}()
	blocker := createServerCard(t, store, "blocker")
	if err := actions.EnqueuePrimary(blocker.ID, blocker.Primary); err != nil {
		t.Fatalf("EnqueuePrimary(blocker) error = %v", err)
	}
	<-runner.started

	card := createServerCard(t, store, "dedup")
	if err := actions.EnqueuePrimary(card.ID, card.Primary); err != nil {
		t.Fatalf("first EnqueuePrimary() error = %v", err)
	}
	if err := actions.EnqueuePrimary(card.ID, card.Primary); !errors.Is(err, ErrAlreadyQueued) {
		t.Fatalf("second EnqueuePrimary() error = %v, want ErrAlreadyQueued", err)
	}
	if err := actions.EnqueueStatus(card.ID); err != nil {
		t.Fatalf("EnqueueStatus() for another kind error = %v", err)
	}
	if queued := actions.Queued(); queued != 2 {
		t.Fatalf("Queued() = %d, want 2 (primary + status)", queued)
	}

	// Let the blocker finish; the deduplicated primary job starts and its key
	// is released, so the same request is queued again while it runs.
	runner.release <- struct{}{}
	if started := <-runner.started; started.Command != card.Primary.Command {
		t.Fatalf("started action = %#v, want the deduplicated primary", started)
	}
	if err := actions.EnqueuePrimary(card.ID, card.Primary); err != nil {
		t.Fatalf("EnqueuePrimary() while running error = %v", err)
	}
}

func TestEnqueueReportsFullQueueWithRetryAfter(t *testing.T) {
	store := config.NewStore(config.Settings{HistorySize: 5, MaxConcurrentActions: 1})
	runner := &releasingRunner{started: make(chan config.Action, 1), release: make(chan struct{})}
	actions, err := New(store, runner, &fakeStatusChecker{called: make(chan string, 1)}, nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() {
		close(runner.release)
		_, _ = actions.Close(context.Background())
	}()
	blocker := createServerCard(t, store, "blocker")
	if err := actions.EnqueuePrimary(blocker.ID, blocker.Primary); err != nil {
		t.Fatalf("EnqueuePrimary(blocker) error = %v", err)
	}
	<-runner.started
	for index := range 4 {
		card := createServerCard(t, store, "fill-"+string(rune('a'+index)))
		if err := actions.EnqueuePrimary(card.ID, card.Primary); err != nil {
			t.Fatalf("EnqueuePrimary(%s) error = %v", card.ID, err)
		}
	}
	overflow := createServerCard(t, store, "overflow")
	err = actions.EnqueuePrimary(overflow.ID, overflow.Primary)
	var full *QueueFullError
	if !errors.As(err, &full) || !errors.Is(err, ErrQueueFull) {
		t.Fatalf("EnqueuePrimary(overflow) error = %v, want QueueFullError", err)
	}
	if full.RetryAfter < time.Second {
		t.Fatalf("RetryAfter = %s, want at least 1s", full.RetryAfter)
	}
}
