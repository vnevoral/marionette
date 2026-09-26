package server

import (
	"context"
	"errors"
	"log"
	"sync"

	"marionette/internal/config"
	execengine "marionette/internal/exec"
)

// Errors returned by BackgroundActions when a job cannot be accepted.
var (
	ErrActionQueueClosed = errors.New("action queue is closed")
	ErrActionQueueFull   = errors.New("action queue is full")
)

// ActionRunner executes one configured action through the shared concurrency limit.
type ActionRunner interface {
	Run(context.Context, config.Action) (execengine.Result, error)
}

// StatusChecker runs one status action and updates the status projection.
type StatusChecker interface {
	CheckNow(context.Context, string) (config.StatusSnapshot, error)
}

// PrimaryActionNotifier activates fast polling after a primary action is accepted.
type PrimaryActionNotifier interface {
	NotifyPrimaryAction(string) error
}

// CardReconciler updates background workers after card configuration changes.
type CardReconciler interface {
	Reconcile() error
}

// ActionQueue accepts actions for background execution and can drain them during shutdown.
type ActionQueue interface {
	EnqueuePrimary(string, config.Action) error
	EnqueueStatus(string) error
	Wait()
}

// BackgroundActions runs accepted primary and manual status actions asynchronously.
type BackgroundActions struct {
	runner        ActionRunner
	statusChecker StatusChecker
	store         *config.Store

	ctx    context.Context
	cancel context.CancelFunc

	mu      sync.Mutex
	closed  bool
	jobs    chan func(context.Context)
	group   sync.WaitGroup
	pending sync.WaitGroup
}

// NewBackgroundActions creates an asynchronous action queue.
func NewBackgroundActions(store *config.Store, runner ActionRunner, statusChecker StatusChecker) (*BackgroundActions, error) {
	if store == nil {
		return nil, errors.New("config store is required")
	}
	if runner == nil {
		return nil, errors.New("action runner is required")
	}
	if statusChecker == nil {
		return nil, errors.New("status checker is required")
	}
	workers := store.GetSettings().MaxConcurrentActions
	queueCapacity := workers * 4
	ctx, cancel := context.WithCancel(context.Background())
	actions := &BackgroundActions{
		store:         store,
		runner:        runner,
		statusChecker: statusChecker,
		ctx:           ctx,
		cancel:        cancel,
		jobs:          make(chan func(context.Context), queueCapacity),
	}
	for range workers {
		actions.group.Add(1)
		go actions.runWorker()
	}
	return actions, nil
}

// EnqueuePrimary schedules a primary action without waiting for process completion.
func (actions *BackgroundActions) EnqueuePrimary(cardID string, action config.Action) error {
	return actions.enqueue(func(ctx context.Context) {
		result, err := actions.runner.Run(ctx, action)
		if err != nil {
			log.Printf("primary action %q failed: %v", cardID, err)
			return
		}
		if err := actions.store.AppendRun(cardID, result.ToRun("primary")); err != nil {
			log.Printf("save primary action result for %q: %v", cardID, err)
		}
	})
}

// EnqueueStatus schedules a manual status action without waiting for its result.
func (actions *BackgroundActions) EnqueueStatus(cardID string) error {
	return actions.enqueue(func(ctx context.Context) {
		if _, err := actions.statusChecker.CheckNow(ctx, cardID); err != nil && ctx.Err() == nil {
			log.Printf("status action %q failed: %v", cardID, err)
		}
	})
}

// Close rejects new actions, cancels the context shared by accepted actions so
// that running processes are terminated, and waits for the workers to return.
// Block 0027 refines this into a graceful drain with a bounded grace period.
func (actions *BackgroundActions) Close() {
	actions.mu.Lock()
	if actions.closed {
		actions.mu.Unlock()
		return
	}
	actions.closed = true
	close(actions.jobs)
	actions.mu.Unlock()
	actions.cancel()
	actions.Wait()
	actions.group.Wait()
}

// Wait waits for all actions accepted before Close.
func (actions *BackgroundActions) Wait() {
	actions.pending.Wait()
}

func (actions *BackgroundActions) enqueue(job func(context.Context)) error {
	actions.mu.Lock()
	defer actions.mu.Unlock()
	if actions.closed {
		return ErrActionQueueClosed
	}
	actions.pending.Add(1)
	select {
	case actions.jobs <- job:
		return nil
	default:
		actions.pending.Done()
		return ErrActionQueueFull
	}
}

func (actions *BackgroundActions) runWorker() {
	defer actions.group.Done()
	for job := range actions.jobs {
		job(actions.ctx)
		actions.pending.Done()
	}
}

type acceptedAction struct {
	CardID     string `json:"cardId"`
	ActionKind string `json:"actionKind"`
	Status     string `json:"status"`
}
