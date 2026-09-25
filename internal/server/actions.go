package server

import (
	"errors"
	"log"
	"sync"

	"marionette/internal/config"
	execengine "marionette/internal/exec"
)

var ErrActionQueueClosed = errors.New("action queue is closed")
var ErrActionQueueFull = errors.New("action queue is full")

// ActionRunner executes one configured action through the shared concurrency limit.
type ActionRunner interface {
	Run(config.Action) (execengine.Result, error)
}

// StatusChecker runs one status action and updates the status projection.
type StatusChecker interface {
	CheckNow(string) (config.StatusSnapshot, error)
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

	mu      sync.Mutex
	closed  bool
	jobs    chan func()
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
	actions := &BackgroundActions{
		store:         store,
		runner:        runner,
		statusChecker: statusChecker,
		jobs:          make(chan func(), queueCapacity),
	}
	for range workers {
		actions.group.Add(1)
		go actions.runWorker()
	}
	return actions, nil
}

// EnqueuePrimary schedules a primary action without waiting for process completion.
func (actions *BackgroundActions) EnqueuePrimary(cardID string, action config.Action) error {
	return actions.enqueue(func() {
		result, err := actions.runner.Run(action)
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
	return actions.enqueue(func() {
		if _, err := actions.statusChecker.CheckNow(cardID); err != nil {
			log.Printf("status action %q failed: %v", cardID, err)
		}
	})
}

// Close rejects new actions and waits for accepted actions to finish.
func (actions *BackgroundActions) Close() {
	actions.mu.Lock()
	actions.closed = true
	close(actions.jobs)
	actions.mu.Unlock()
	actions.Wait()
	actions.group.Wait()
}

// Wait waits for all actions accepted before Close.
func (actions *BackgroundActions) Wait() {
	actions.pending.Wait()
}

func (actions *BackgroundActions) enqueue(job func()) error {
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
		job()
		actions.pending.Done()
	}
}

type actionQueueDependencies struct {
	queue    ActionQueue
	notifier PrimaryActionNotifier
}

type acceptedAction struct {
	CardID     string `json:"cardId"`
	ActionKind string `json:"actionKind"`
	Status     string `json:"status"`
}
