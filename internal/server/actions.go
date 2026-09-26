package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"marionette/internal/config"
	execengine "marionette/internal/exec"
)

// Errors returned by BackgroundActions when a job cannot be accepted.
var (
	ErrActionQueueClosed = errors.New("action queue is closed")
	ErrActionQueueFull   = errors.New("action queue is full")
	// ErrActionAlreadyQueued reports that the same card and action kind is
	// already waiting in the queue, so the request was satisfied by the
	// existing job (FR-18). Callers treat it as an accepted request.
	ErrActionAlreadyQueued = errors.New("action is already queued")
)

// retryAfterPerQueuedJob is the crude per-job estimate used to derive the
// Retry-After hint when the queue is full. Action durations are unknown up
// front, so the hint only scales with the queue depth per worker.
const retryAfterPerQueuedJob = time.Second

// QueueFullError wraps ErrActionQueueFull with a suggested retry delay.
type QueueFullError struct {
	RetryAfter time.Duration
}

func (err *QueueFullError) Error() string {
	return fmt.Sprintf("%v (retry after %s)", ErrActionQueueFull, err.RetryAfter)
}

func (err *QueueFullError) Unwrap() error {
	return ErrActionQueueFull
}

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

// ActionQueue accepts actions for background execution.
type ActionQueue interface {
	EnqueuePrimary(string, config.Action) error
	EnqueueStatus(string) error
}

type jobKey struct {
	cardID string
	kind   string
}

type queuedJob struct {
	key jobKey
	run func(context.Context)
}

// BackgroundActions runs accepted primary and manual status actions
// asynchronously with a bounded queue. Jobs for the same card and action kind
// are queued at most once while waiting (FR-18).
type BackgroundActions struct {
	runner        ActionRunner
	statusChecker StatusChecker
	store         *config.Store

	workers  int
	capacity int

	// ctx is shared by running jobs; cancel terminates their processes.
	ctx    context.Context
	cancel context.CancelFunc
	// wake carries one token per queued job so an idle worker picks it up.
	wake chan struct{}

	mu      sync.Mutex
	closed  bool
	queue   []queuedJob
	waiting map[jobKey]struct{}
	running int
	// drained is closed by Close once the queue is empty and no job runs.
	drained chan struct{}

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
	capacity := workers * 4
	ctx, cancel := context.WithCancel(context.Background())
	actions := &BackgroundActions{
		store:         store,
		runner:        runner,
		statusChecker: statusChecker,
		workers:       workers,
		capacity:      capacity,
		ctx:           ctx,
		cancel:        cancel,
		wake:          make(chan struct{}, capacity),
		waiting:       make(map[jobKey]struct{}),
		drained:       make(chan struct{}),
	}
	for range workers {
		actions.group.Add(1)
		go actions.runWorker()
	}
	return actions, nil
}

// EnqueuePrimary schedules a primary action without waiting for process completion.
func (actions *BackgroundActions) EnqueuePrimary(cardID string, action config.Action) error {
	return actions.enqueue(jobKey{cardID: cardID, kind: "primary"}, func(ctx context.Context) {
		result, err := actions.runner.Run(ctx, action)
		if err != nil {
			if ctx.Err() == nil {
				log.Printf("primary action %q failed: %v", cardID, err)
			}
			return
		}
		if err := actions.store.AppendRun(cardID, result.ToRun("primary")); err != nil {
			log.Printf("save primary action result for %q: %v", cardID, err)
		}
	})
}

// EnqueueStatus schedules a manual status action without waiting for its result.
func (actions *BackgroundActions) EnqueueStatus(cardID string) error {
	return actions.enqueue(jobKey{cardID: cardID, kind: "status"}, func(ctx context.Context) {
		if _, err := actions.statusChecker.CheckNow(ctx, cardID); err != nil && ctx.Err() == nil {
			log.Printf("status action %q failed: %v", cardID, err)
		}
	})
}

// Close stops accepting jobs, discards every job still waiting in the queue
// and lets running jobs finish until ctx is done. When ctx expires first, the
// running jobs are cancelled (their processes are terminated by the execution
// engine) and Close waits for the workers to return. It returns the number of
// discarded jobs and ctx's error when running jobs had to be cancelled. Close
// is idempotent; later calls return immediately.
func (actions *BackgroundActions) Close(ctx context.Context) (int, error) {
	actions.mu.Lock()
	if actions.closed {
		actions.mu.Unlock()
		return 0, nil
	}
	actions.closed = true
	dropped := len(actions.queue)
	for range actions.queue {
		actions.pending.Done()
	}
	actions.queue = nil
	actions.waiting = make(map[jobKey]struct{})
	if actions.running == 0 {
		close(actions.drained)
	}
	actions.mu.Unlock()

	var err error
	select {
	case <-actions.drained:
	case <-ctx.Done():
		err = ctx.Err()
	}
	actions.cancel()
	actions.group.Wait()
	return dropped, err
}

// Wait blocks until every accepted job has finished or been discarded.
func (actions *BackgroundActions) Wait() {
	actions.pending.Wait()
}

// Queued reports how many jobs are waiting for a worker.
func (actions *BackgroundActions) Queued() int {
	actions.mu.Lock()
	defer actions.mu.Unlock()
	return len(actions.queue)
}

func (actions *BackgroundActions) enqueue(key jobKey, run func(context.Context)) error {
	actions.mu.Lock()
	defer actions.mu.Unlock()
	if actions.closed {
		return ErrActionQueueClosed
	}
	if _, waiting := actions.waiting[key]; waiting {
		return ErrActionAlreadyQueued
	}
	if len(actions.queue) >= actions.capacity {
		return &QueueFullError{RetryAfter: actions.retryAfterLocked()}
	}
	actions.pending.Add(1)
	actions.queue = append(actions.queue, queuedJob{key: key, run: run})
	actions.waiting[key] = struct{}{}
	actions.wake <- struct{}{}
	return nil
}

// retryAfterLocked estimates how long a client should wait before retrying:
// one time slice per queued job divided among the workers, at least one second.
func (actions *BackgroundActions) retryAfterLocked() time.Duration {
	slices := (len(actions.queue) + actions.workers - 1) / actions.workers
	return max(time.Duration(slices)*retryAfterPerQueuedJob, time.Second)
}

func (actions *BackgroundActions) runWorker() {
	defer actions.group.Done()
	for {
		select {
		case <-actions.ctx.Done():
			return
		case <-actions.wake:
		}
		job, ok := actions.take()
		if !ok {
			continue
		}
		job.run(actions.ctx)
		actions.finish()
	}
}

// take removes the oldest queued job and marks it running. The job's key is
// released from the deduplication set here, so a new request for the same
// card and kind is queued again while the job runs.
func (actions *BackgroundActions) take() (queuedJob, bool) {
	actions.mu.Lock()
	defer actions.mu.Unlock()
	if len(actions.queue) == 0 {
		return queuedJob{}, false
	}
	job := actions.queue[0]
	actions.queue[0] = queuedJob{}
	actions.queue = actions.queue[1:]
	delete(actions.waiting, job.key)
	actions.running++
	return job, true
}

func (actions *BackgroundActions) finish() {
	actions.mu.Lock()
	actions.running--
	if actions.closed && actions.running == 0 {
		close(actions.drained)
	}
	actions.mu.Unlock()
	actions.pending.Done()
}

type acceptedAction struct {
	CardID     string `json:"cardId"`
	ActionKind string `json:"actionKind"`
	Status     string `json:"status"`
}
