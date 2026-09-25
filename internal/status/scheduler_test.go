package status

import (
	"context"
	"sync"
	"testing"
	"time"

	"marionette/internal/config"
)

type fakeChecker struct {
	calls chan string
}

func (checker *fakeChecker) CheckNow(cardID string) (config.StatusSnapshot, error) {
	checker.calls <- cardID
	return config.StatusSnapshot{State: config.StatusStateOK}, nil
}

type fakeClock struct {
	mu        sync.Mutex
	now       time.Time
	timers    []*fakeTimer
	ready     chan struct{}
	durations chan time.Duration
}

type fakeTimer struct {
	clock  *fakeClock
	c      chan time.Time
	due    time.Time
	active bool
}

func newFakeClock() *fakeClock {
	return &fakeClock{
		now:       time.Date(2026, time.September, 25, 12, 0, 0, 0, time.UTC),
		ready:     make(chan struct{}),
		durations: make(chan time.Duration, 8),
	}
}

func (clock *fakeClock) Now() time.Time {
	clock.mu.Lock()
	defer clock.mu.Unlock()
	return clock.now
}

func (clock *fakeClock) NewTimer(duration time.Duration) schedulerTimer {
	clock.mu.Lock()
	defer clock.mu.Unlock()
	timer := &fakeTimer{clock: clock, c: make(chan time.Time, 1), due: clock.now.Add(duration), active: true}
	clock.timers = append(clock.timers, timer)
	select {
	case <-clock.ready:
	default:
		close(clock.ready)
	}
	clock.durations <- duration
	return timer
}

func (timer *fakeTimer) Stop() bool {
	timer.clock.mu.Lock()
	defer timer.clock.mu.Unlock()
	wasActive := timer.active
	timer.active = false
	return wasActive
}

func (timer *fakeTimer) Reset(duration time.Duration) bool {
	timer.clock.mu.Lock()
	defer timer.clock.mu.Unlock()
	timer.due = timer.clock.now.Add(duration)
	timer.active = true
	timer.clock.durations <- duration
	return true
}

func (timer *fakeTimer) Chan() <-chan time.Time {
	return timer.c
}

func (clock *fakeClock) Advance(duration time.Duration) {
	clock.mu.Lock()
	clock.now = clock.now.Add(duration)
	for _, timer := range clock.timers {
		if timer.active && !clock.now.Before(timer.due) {
			timer.active = false
			timer.c <- clock.now
		}
	}
	clock.mu.Unlock()
}

func schedulerTestCard(t *testing.T, store *config.Store, id string, polling, fast, window int) {
	t.Helper()
	_, err := store.CreateCard(config.ActionCard{
		ID:                         id,
		Name:                       id,
		Primary:                    config.Action{Command: "primary", TimeoutSec: 1},
		Status:                     &config.Action{Command: "status", TimeoutSec: 1},
		PollingIntervalSeconds:     polling,
		FastPollingIntervalSeconds: fast,
		FastPollingWindowSeconds:   window,
	})
	if err != nil {
		t.Fatalf("CreateCard() error = %v", err)
	}
}

func newTestScheduler(t *testing.T, cardID string, polling, fast, window int) (*Scheduler, *fakeClock, *fakeChecker, context.CancelFunc) {
	t.Helper()
	store := config.NewStore(config.Settings{HistorySize: 5, MaxConcurrentActions: 1})
	schedulerTestCard(t, store, cardID, polling, fast, window)
	checker := &fakeChecker{calls: make(chan string, 8)}
	scheduler, err := NewScheduler(store, checker)
	if err != nil {
		t.Fatalf("NewScheduler() error = %v", err)
	}
	clock := newFakeClock()
	scheduler.clock = clock
	ctx, cancel := context.WithCancel(context.Background())
	if err := scheduler.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	return scheduler, clock, checker, cancel
}

func waitForTimer(t *testing.T, clock *fakeClock) {
	t.Helper()
	select {
	case <-clock.ready:
	case <-time.After(time.Second):
		t.Fatal("scheduler did not create a timer")
	}
}

func TestSchedulerRunsStandardPolling(t *testing.T) {
	scheduler, clock, checker, cancel := newTestScheduler(t, "standard", 10, 2, 5)
	defer cancel()
	defer scheduler.Stop()
	waitForTimer(t, clock)
	<-clock.durations
	clock.Advance(9 * time.Second)
	select {
	case <-checker.calls:
		t.Fatal("polling ran before the standard interval")
	default:
	}
	clock.Advance(time.Second)
	select {
	case cardID := <-checker.calls:
		if cardID != "standard" {
			t.Fatalf("checked card = %q", cardID)
		}
	case <-time.After(time.Second):
		t.Fatal("standard polling did not run")
	}
}

func TestSchedulerUsesFastPollingAndReturnsToStandard(t *testing.T) {
	scheduler, clock, checker, cancel := newTestScheduler(t, "fast", 10, 2, 3)
	defer cancel()
	defer scheduler.Stop()
	waitForTimer(t, clock)
	<-clock.durations
	if err := scheduler.NotifyPrimaryAction("fast"); err != nil {
		t.Fatalf("NotifyPrimaryAction() error = %v", err)
	}
	if duration := <-clock.durations; duration != 2*time.Second {
		t.Fatalf("fast timer duration = %v", duration)
	}
	clock.Advance(2 * time.Second)
	select {
	case <-checker.calls:
	case <-time.After(time.Second):
		t.Fatal("fast polling did not run")
	}
	if duration := <-clock.durations; duration != 2*time.Second {
		t.Fatalf("timer remained at duration %v, want 2s", duration)
	}
	clock.Advance(2 * time.Second)
	select {
	case <-checker.calls:
	case <-time.After(time.Second):
		t.Fatal("second fast polling did not run")
	}
	if duration := <-clock.durations; duration != 10*time.Second {
		t.Fatalf("timer duration after fast window = %v, want 10s", duration)
	}
}

func TestSchedulerIgnoresFastPollingWhenStandardPollingIsDisabled(t *testing.T) {
	scheduler, clock, checker, cancel := newTestScheduler(t, "disabled", 0, 1, 3)
	defer cancel()
	if err := scheduler.NotifyPrimaryAction("disabled"); err != nil {
		t.Fatalf("NotifyPrimaryAction() error = %v", err)
	}
	select {
	case <-clock.ready:
		t.Fatal("disabled polling created a timer")
	case <-checker.calls:
		t.Fatal("disabled polling ran a check")
	case <-time.After(20 * time.Millisecond):
	}
	if err := scheduler.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}

func TestSchedulerStopCancelsPolling(t *testing.T) {
	scheduler, clock, checker, cancel := newTestScheduler(t, "stoppable", 1, 0, 0)
	defer cancel()
	waitForTimer(t, clock)
	<-clock.durations
	if err := scheduler.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	clock.Advance(time.Hour)
	select {
	case <-checker.calls:
		t.Fatal("polling ran after scheduler stop")
	default:
	}
}

func TestSchedulerReconcileAddsRestartsAndRemovesWorkers(t *testing.T) {
	store := config.NewStore(config.Settings{HistorySize: 5, MaxConcurrentActions: 1})
	checker := &fakeChecker{calls: make(chan string, 8)}
	scheduler, err := NewScheduler(store, checker)
	if err != nil {
		t.Fatalf("NewScheduler() error = %v", err)
	}
	clock := newFakeClock()
	scheduler.clock = clock
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := scheduler.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	schedulerTestCard(t, store, "reconcile", 10, 2, 3)
	if err := scheduler.Reconcile(); err != nil {
		t.Fatalf("Reconcile(add) error = %v", err)
	}
	waitForTimer(t, clock)
	if duration := <-clock.durations; duration != 10*time.Second {
		t.Fatalf("initial timer duration = %v", duration)
	}

	card, ok := store.GetCard("reconcile")
	if !ok {
		t.Fatal("reconcile card was not stored")
	}
	card.PollingIntervalSeconds = 4
	if _, err := store.UpdateCard(card.ID, card); err != nil {
		t.Fatalf("UpdateCard() error = %v", err)
	}
	if err := scheduler.Reconcile(); err != nil {
		t.Fatalf("Reconcile(update) error = %v", err)
	}
	if duration := <-clock.durations; duration != 4*time.Second {
		t.Fatalf("restarted timer duration = %v", duration)
	}

	if err := store.DeleteCard(card.ID); err != nil {
		t.Fatalf("DeleteCard() error = %v", err)
	}
	if err := scheduler.Reconcile(); err != nil {
		t.Fatalf("Reconcile(delete) error = %v", err)
	}
	clock.Advance(time.Minute)
	select {
	case cardID := <-checker.calls:
		t.Fatalf("removed worker checked card %q", cardID)
	default:
	}
	if err := scheduler.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}

func TestSchedulerRestartClearsFastPollingWindow(t *testing.T) {
	scheduler, clock, _, cancel := newTestScheduler(t, "restart", 10, 2, 30)
	defer cancel()
	waitForTimer(t, clock)
	<-clock.durations
	if err := scheduler.NotifyPrimaryAction("restart"); err != nil {
		t.Fatalf("NotifyPrimaryAction() error = %v", err)
	}
	if duration := <-clock.durations; duration != 2*time.Second {
		t.Fatalf("fast timer duration = %v", duration)
	}
	if err := scheduler.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if err := scheduler.Start(context.Background()); err != nil {
		t.Fatalf("restart Start() error = %v", err)
	}
	waitForTimer(t, clock)
	if duration := <-clock.durations; duration != 10*time.Second {
		t.Fatalf("restart timer duration = %v, want standard interval", duration)
	}
	if err := scheduler.Stop(); err != nil {
		t.Fatalf("final Stop() error = %v", err)
	}
}
