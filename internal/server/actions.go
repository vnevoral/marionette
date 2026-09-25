package server

import (
	"errors"
	"sync"

	"marionette/internal/config"
	execengine "marionette/internal/exec"
)

var ErrActionQueueClosed = errors.New("action queue is closed")

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

	mu     sync.Mutex
	closed bool
	group  sync.WaitGroup
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
	return &BackgroundActions{store: store, runner: runner, statusChecker: statusChecker}, nil
}

// EnqueuePrimary schedules a primary action without waiting for process completion.
func (actions *BackgroundActions) EnqueuePrimary(cardID string, action config.Action) error {
	if err := actions.begin(); err != nil {
		return err
	}
	go func() {
		defer actions.group.Done()
		result, err := actions.runner.Run(action)
		if err != nil {
			return
		}
		_ = actions.store.AppendRun(cardID, result.ToRun("primary"))
	}()
	return nil
}

// EnqueueStatus schedules a manual status action without waiting for its result.
func (actions *BackgroundActions) EnqueueStatus(cardID string) error {
	if err := actions.begin(); err != nil {
		return err
	}
	go func() {
		defer actions.group.Done()
		_, _ = actions.statusChecker.CheckNow(cardID)
	}()
	return nil
}

// Close rejects new actions and waits for accepted actions to finish.
func (actions *BackgroundActions) Close() {
	actions.mu.Lock()
	actions.closed = true
	actions.mu.Unlock()
	actions.Wait()
}

// Wait waits for all actions accepted before Close.
func (actions *BackgroundActions) Wait() {
	actions.group.Wait()
}

func (actions *BackgroundActions) begin() error {
	actions.mu.Lock()
	defer actions.mu.Unlock()
	if actions.closed {
		return ErrActionQueueClosed
	}
	actions.group.Add(1)
	return nil
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
