package status

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"time"

	"marionette/internal/config"
)

// primaryQueueMargin is the time a triggered primary action may spend in the
// action queue before it starts. With the action's timeout and the fast
// polling window it bounds the wait for an action that never reports back.
const primaryQueueMargin = 30 * time.Second

// Errors returned by Scheduler lifecycle methods.
var (
	ErrSchedulerRunning = errors.New("status scheduler is already running")
	ErrSchedulerStopped = errors.New("status scheduler is not running")
)

type statusChecker interface {
	CheckNow(context.Context, string) (config.StatusSnapshot, error)
}

type schedulerTimer interface {
	Stop() bool
	Reset(time.Duration) bool
	Chan() <-chan time.Time
}

type schedulerClock interface {
	Now() time.Time
	NewTimer(time.Duration) schedulerTimer
}

type realSchedulerClock struct{}

func (realSchedulerClock) Now() time.Time {
	return time.Now()
}

func (realSchedulerClock) NewTimer(duration time.Duration) schedulerTimer {
	return realSchedulerTimer{timer: time.NewTimer(duration)}
}

type realSchedulerTimer struct {
	timer *time.Timer
}

func (timer realSchedulerTimer) Stop() bool {
	return timer.timer.Stop()
}

func (timer realSchedulerTimer) Reset(duration time.Duration) bool {
	return timer.timer.Reset(duration)
}

func (timer realSchedulerTimer) Chan() <-chan time.Time {
	return timer.timer.C
}

type pollWorker struct {
	wake   chan struct{}
	cancel context.CancelFunc
	card   config.ActionCard
	done   chan struct{}
}

// Scheduler runs status checks for cards with standard polling enabled.
type Scheduler struct {
	store   *config.Store
	checker statusChecker
	clock   schedulerClock

	mu        sync.Mutex
	running   bool
	ctx       context.Context
	cancel    context.CancelFunc
	workers   map[string]*pollWorker
	fastUntil map[string]time.Time
	// awaiting holds the cards whose triggered primary action has not
	// finished yet; their checks wait for it (FR-15b).
	awaiting    map[string]bool
	waitGroup   sync.WaitGroup
	reconcileMu sync.Mutex
}

// NewScheduler creates a stopped status polling scheduler.
func NewScheduler(store *config.Store, checker statusChecker) (*Scheduler, error) {
	if store == nil {
		return nil, errors.New("config store is required")
	}
	if checker == nil {
		return nil, errors.New("status checker is required")
	}
	return &Scheduler{
		store:     store,
		checker:   checker,
		clock:     realSchedulerClock{},
		workers:   make(map[string]*pollWorker),
		fastUntil: make(map[string]time.Time),
		awaiting:  make(map[string]bool),
	}, nil
}

// Start begins workers for the currently configured polling cards.
func (scheduler *Scheduler) Start(parent context.Context) error {
	if parent == nil {
		return errors.New("parent context is required")
	}

	scheduler.mu.Lock()
	if scheduler.running {
		scheduler.mu.Unlock()
		return ErrSchedulerRunning
	}
	ctx, cancel := context.WithCancel(parent)
	scheduler.running = true
	scheduler.ctx = ctx
	scheduler.cancel = cancel
	scheduler.workers = make(map[string]*pollWorker)
	for _, card := range scheduler.desiredCards() {
		scheduler.startWorkerLocked(ctx, card)
	}
	scheduler.mu.Unlock()
	go scheduler.watchParent(ctx)
	return nil
}

// watchParent marks the scheduler stopped once the context it was started
// with ends, so that a cancelled parent context (application shutdown) leaves
// the scheduler restartable and Stop reports ErrSchedulerStopped instead of
// pretending to stop already-finished workers.
func (scheduler *Scheduler) watchParent(ctx context.Context) {
	<-ctx.Done()
	scheduler.mu.Lock()
	defer scheduler.mu.Unlock()
	if scheduler.ctx != ctx {
		return // Stop already ran, or a new Start replaced this run
	}
	scheduler.running = false
	scheduler.ctx = nil
	scheduler.cancel = nil
}

// desiredCards returns the cards that need a polling worker: those with a
// status action and standard polling enabled. Start and Reconcile share it so
// both apply the same rule.
func (scheduler *Scheduler) desiredCards() map[string]config.ActionCard {
	desired := make(map[string]config.ActionCard)
	for _, card := range scheduler.store.ListCards() {
		if card.PollingIntervalSeconds > 0 && card.Status != nil {
			desired[card.ID] = card
		}
	}
	return desired
}

// Stop cancels all polling workers and waits until they have returned.
func (scheduler *Scheduler) Stop() error {
	scheduler.mu.Lock()
	if !scheduler.running {
		scheduler.mu.Unlock()
		return ErrSchedulerStopped
	}
	cancel := scheduler.cancel
	scheduler.running = false
	scheduler.ctx = nil
	scheduler.cancel = nil
	scheduler.mu.Unlock()

	cancel()
	scheduler.waitGroup.Wait()
	scheduler.mu.Lock()
	scheduler.workers = make(map[string]*pollWorker)
	scheduler.fastUntil = make(map[string]time.Time)
	scheduler.awaiting = make(map[string]bool)
	scheduler.mu.Unlock()
	return nil
}

// Reconcile updates polling workers to match the current cards in the store.
func (scheduler *Scheduler) Reconcile() error {
	scheduler.reconcileMu.Lock()
	defer scheduler.reconcileMu.Unlock()

	scheduler.mu.Lock()
	if !scheduler.running {
		scheduler.mu.Unlock()
		return ErrSchedulerStopped
	}
	current := make(map[string]config.ActionCard)
	for id, worker := range scheduler.workers {
		current[id] = worker.card
	}
	desired := scheduler.desiredCards()
	stoppedWorkers := make([]*pollWorker, 0)
	// Cards that no longer poll get no confirming check, so their wait ends.
	unpolled := make([]string, 0)
	for id, worker := range scheduler.workers {
		card, exists := desired[id]
		if exists && reflect.DeepEqual(current[id], card) {
			continue
		}
		worker.cancel()
		stoppedWorkers = append(stoppedWorkers, worker)
		delete(scheduler.workers, id)
		if !exists {
			delete(scheduler.fastUntil, id)
			delete(scheduler.awaiting, id)
			unpolled = append(unpolled, id)
		}
	}
	scheduler.mu.Unlock()
	for _, worker := range stoppedWorkers {
		<-worker.done
	}
	for _, id := range unpolled {
		_ = scheduler.store.EndWaiting(id)
	}
	scheduler.mu.Lock()
	if !scheduler.running {
		scheduler.mu.Unlock()
		return ErrSchedulerStopped
	}
	ctx := scheduler.ctx
	for id, card := range desired {
		if _, exists := scheduler.workers[id]; !exists {
			scheduler.startWorkerLocked(ctx, card)
		}
	}
	scheduler.mu.Unlock()
	return nil
}

// NotifyPrimaryAction is called when a primary action of a card is accepted.
// For a card with standard polling and fast polling configured, the card
// waits for confirmation (FR-15b): its checks pause until the action has
// finished (PrimaryActionFinished), at most for the action's timeout, the
// queue margin and the fast polling window.
func (scheduler *Scheduler) NotifyPrimaryAction(cardID string) error {
	card, exists := scheduler.store.GetCard(cardID)
	if !exists {
		return config.ErrNotFound
	}
	if !confirmsPrimaryAction(card) {
		return nil
	}

	bound := time.Duration(card.Primary.TimeoutSec)*time.Second + primaryQueueMargin + fastWindow(card)
	until := scheduler.clock.Now().Add(bound)
	scheduler.mu.Lock()
	scheduler.fastUntil[cardID] = until
	scheduler.awaiting[cardID] = true
	worker := scheduler.workers[cardID]
	scheduler.mu.Unlock()
	if err := scheduler.store.BeginWaiting(cardID, until); err != nil {
		return err
	}
	scheduler.wake(worker)
	return nil
}

// PrimaryActionFinished starts fast polling and the confirmation window of
// a card whose primary action was notified and has now finished (FR-15a,
// FR-15b). An action that failed or timed out has nothing to confirm: the
// wait ends at once and the card returns to its standard interval.
func (scheduler *Scheduler) PrimaryActionFinished(cardID string, outcome config.RunOutcome) {
	card, exists := scheduler.store.GetCard(cardID)
	if !exists || !confirmsPrimaryAction(card) {
		return
	}
	scheduler.mu.Lock()
	if !scheduler.awaiting[cardID] {
		scheduler.mu.Unlock()
		return
	}
	delete(scheduler.awaiting, cardID)
	ok := outcome == config.RunOutcomeOK
	until := scheduler.clock.Now().Add(fastWindow(card))
	if ok {
		scheduler.fastUntil[cardID] = until
	} else {
		delete(scheduler.fastUntil, cardID)
	}
	worker := scheduler.workers[cardID]
	scheduler.mu.Unlock()
	if ok {
		_ = scheduler.store.ExtendWaiting(cardID, until)
	} else {
		_ = scheduler.store.EndWaiting(cardID)
	}
	scheduler.wake(worker)
}

// confirmsPrimaryAction reports whether a primary action of the card is
// followed by fast polling and a confirmation window.
func confirmsPrimaryAction(card config.ActionCard) bool {
	return card.Status != nil && card.PollingIntervalSeconds > 0 && card.FastPollingIntervalSeconds > 0 && card.FastPollingWindowSeconds > 0
}

func fastWindow(card config.ActionCard) time.Duration {
	return time.Duration(card.FastPollingWindowSeconds) * time.Second
}

func (scheduler *Scheduler) wake(worker *pollWorker) {
	if worker == nil {
		return
	}
	select {
	case worker.wake <- struct{}{}:
	default:
	}
}

// awaitingPrimary reports whether the card's checks wait for its primary
// action. Past the bound set by NotifyPrimaryAction the action is no longer
// waited for.
func (scheduler *Scheduler) awaitingPrimary(cardID string) bool {
	scheduler.mu.Lock()
	defer scheduler.mu.Unlock()
	if !scheduler.awaiting[cardID] {
		return false
	}
	if scheduler.clock.Now().Before(scheduler.fastUntil[cardID]) {
		return true
	}
	delete(scheduler.awaiting, cardID)
	return false
}

func (scheduler *Scheduler) runWorker(ctx context.Context, cardID string, worker *pollWorker) {
	defer scheduler.waitGroup.Done()
	defer close(worker.done)
	card, exists := scheduler.store.GetCard(cardID)
	if !exists || card.PollingIntervalSeconds <= 0 || card.Status == nil {
		return
	}
	_, _ = scheduler.checker.CheckNow(ctx, cardID)
	timer := scheduler.clock.NewTimer(scheduler.nextInterval(card))
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-worker.wake:
			scheduler.resetTimer(timer, scheduler.nextInterval(card))
		case <-timer.Chan():
			if scheduler.awaitingPrimary(cardID) {
				scheduler.resetTimer(timer, scheduler.nextInterval(card))
				continue
			}
			_, _ = scheduler.checker.CheckNow(ctx, cardID)
			// A check that failed to run cannot close an expired window.
			if until, waiting := scheduler.store.WaitingUntil(cardID); waiting && !scheduler.clock.Now().Before(until) {
				_ = scheduler.store.EndWaiting(cardID)
			}
			card, exists = scheduler.store.GetCard(cardID)
			if !exists || card.PollingIntervalSeconds <= 0 || card.Status == nil {
				return
			}
			scheduler.resetTimer(timer, scheduler.nextInterval(card))
		}
	}
}

func (scheduler *Scheduler) startWorkerLocked(parent context.Context, card config.ActionCard) {
	workerContext, cancel := context.WithCancel(parent)
	worker := &pollWorker{wake: make(chan struct{}, 1), cancel: cancel, card: card, done: make(chan struct{})}
	scheduler.workers[card.ID] = worker
	scheduler.waitGroup.Add(1)
	go scheduler.runWorker(workerContext, card.ID, worker)
}

// nextInterval is the time to the card's next check: the fast or standard
// interval, at most up to the end of its confirmation window so the last
// check closes it on time. While the card waits for its primary action the
// timer only guards the bound of that wait.
func (scheduler *Scheduler) nextInterval(card config.ActionCard) time.Duration {
	interval := card.PollingIntervalSeconds
	scheduler.mu.Lock()
	fastUntil := scheduler.fastUntil[card.ID]
	awaiting := scheduler.awaiting[card.ID]
	now := scheduler.clock.Now()
	scheduler.mu.Unlock()
	if awaiting {
		return max(fastUntil.Sub(now), 0)
	}
	if card.PollingIntervalSeconds > 0 && card.FastPollingIntervalSeconds > 0 && now.Before(fastUntil) {
		interval = card.FastPollingIntervalSeconds
	}
	next := time.Duration(interval) * time.Second
	if until, waiting := scheduler.store.WaitingUntil(card.ID); waiting {
		next = min(next, max(until.Sub(now), 0))
	}
	return next
}

func (scheduler *Scheduler) resetTimer(timer schedulerTimer, duration time.Duration) {
	if !timer.Stop() {
		select {
		case <-timer.Chan():
		default:
		}
	}
	timer.Reset(duration)
}
