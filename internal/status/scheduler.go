package status

import (
	"context"
	"errors"
	"sync"
	"time"

	"marionette/internal/config"
)

var (
	ErrSchedulerRunning = errors.New("status scheduler is already running")
	ErrSchedulerStopped = errors.New("status scheduler is not running")
)

type statusChecker interface {
	CheckNow(string) (config.StatusSnapshot, error)
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
	wake chan struct{}
}

// Scheduler runs status checks for cards with standard polling enabled.
type Scheduler struct {
	store   *config.Store
	checker statusChecker
	clock   schedulerClock

	mu        sync.Mutex
	running   bool
	cancel    context.CancelFunc
	workers   map[string]*pollWorker
	fastUntil map[string]time.Time
	waitGroup sync.WaitGroup
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
	scheduler.cancel = cancel
	scheduler.workers = make(map[string]*pollWorker)
	for _, card := range scheduler.store.ListCards() {
		if card.PollingIntervalSeconds <= 0 || card.Status == nil {
			continue
		}
		worker := &pollWorker{wake: make(chan struct{}, 1)}
		scheduler.workers[card.ID] = worker
		scheduler.waitGroup.Add(1)
		go scheduler.runWorker(ctx, card.ID, worker)
	}
	scheduler.mu.Unlock()
	return nil
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
	scheduler.cancel = nil
	scheduler.mu.Unlock()

	cancel()
	scheduler.waitGroup.Wait()
	return nil
}

// NotifyPrimaryAction activates fast polling for a card with standard polling enabled.
func (scheduler *Scheduler) NotifyPrimaryAction(cardID string) error {
	card, exists := scheduler.store.GetCard(cardID)
	if !exists {
		return config.ErrNotFound
	}
	if card.PollingIntervalSeconds <= 0 || card.FastPollingIntervalSeconds <= 0 || card.FastPollingWindowSeconds <= 0 {
		return nil
	}

	scheduler.mu.Lock()
	scheduler.fastUntil[cardID] = scheduler.clock.Now().Add(time.Duration(card.FastPollingWindowSeconds) * time.Second)
	worker := scheduler.workers[cardID]
	scheduler.mu.Unlock()
	if worker != nil {
		select {
		case worker.wake <- struct{}{}:
		default:
		}
	}
	return nil
}

func (scheduler *Scheduler) runWorker(ctx context.Context, cardID string, worker *pollWorker) {
	defer scheduler.waitGroup.Done()
	card, exists := scheduler.store.GetCard(cardID)
	if !exists || card.PollingIntervalSeconds <= 0 || card.Status == nil {
		return
	}
	timer := scheduler.clock.NewTimer(scheduler.nextInterval(card))
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-worker.wake:
			scheduler.resetTimer(timer, scheduler.nextInterval(card))
		case <-timer.Chan():
			_, _ = scheduler.checker.CheckNow(cardID)
			card, exists = scheduler.store.GetCard(cardID)
			if !exists || card.PollingIntervalSeconds <= 0 || card.Status == nil {
				return
			}
			scheduler.resetTimer(timer, scheduler.nextInterval(card))
		}
	}
}

func (scheduler *Scheduler) nextInterval(card config.ActionCard) time.Duration {
	interval := card.PollingIntervalSeconds
	scheduler.mu.Lock()
	fastUntil := scheduler.fastUntil[card.ID]
	now := scheduler.clock.Now()
	scheduler.mu.Unlock()
	if card.PollingIntervalSeconds > 0 && card.FastPollingIntervalSeconds > 0 && now.Before(fastUntil) {
		interval = card.FastPollingIntervalSeconds
	}
	return time.Duration(interval) * time.Second
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
