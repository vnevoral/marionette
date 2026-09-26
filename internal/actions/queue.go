// Package actions runs accepted primary and manual status actions in the
// background with a bounded queue (FR-18) and a graceful drain (FR-35).
package actions

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"marionette/internal/config"
	"marionette/internal/execengine"
)

// Errors returned by Queue when a job cannot be accepted.
var (
	// ErrQueueClosed is returned once Close has been called.
	ErrQueueClosed = errors.New("action queue is closed")
	// ErrQueueFull is returned when no queue slot is free; the returned
	// error also carries a *QueueFullError with a retry hint.
	ErrQueueFull = errors.New("action queue is full")
	// ErrAlreadyQueued reports that the same card and action kind is already
	// waiting in the queue, so the request was satisfied by the existing job
	// (FR-18). Callers treat it as an accepted request.
	ErrAlreadyQueued = errors.New("action is already queued")
)

// retryAfterPerQueuedJob is the crude per-job estimate used to derive the
// Retry-After hint when the queue is full. Action durations are unknown up
// front, so the hint only scales with the queue depth per worker.
const retryAfterPerQueuedJob = time.Second

// QueueFullError wraps ErrQueueFull with a suggested retry delay.
type QueueFullError struct {
	RetryAfter time.Duration
}

func (err *QueueFullError) Error() string {
	return fmt.Sprintf("%v (retry after %s)", ErrQueueFull, err.RetryAfter)
}

// Unwrap makes errors.Is(err, ErrQueueFull) true.
func (err *QueueFullError) Unwrap() error {
	return ErrQueueFull
}

// Runner executes one configured action through the shared concurrency limit.
type Runner interface {
	Run(context.Context, config.Action) (execengine.Result, error)
}

// StatusChecker runs one status action and updates the status projection.
type StatusChecker interface {
	CheckNow(context.Context, string) (config.StatusSnapshot, error)
}

// Kind names the two kinds of jobs the queue accepts.
const (
	KindPrimary = "primary"
	KindStatus  = "status"
)

type jobKey struct {
	cardID string
	kind   string
}

type queuedJob struct {
	key jobKey
	run func(context.Context)
}

// Queue runs accepted jobs asynchronously with MaxConcurrentActions workers
// and a queue of 4 × that size. Jobs for the same card and kind are queued at
// most once while waiting.
type Queue struct {
	runner        Runner
	statusChecker StatusChecker
	store         *config.Store
	logger        *slog.Logger

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

// New creates a queue and starts its workers. A nil logger discards logs.
func New(store *config.Store, runner Runner, statusChecker StatusChecker, logger *slog.Logger) (*Queue, error) {
	if store == nil {
		return nil, errors.New("config store is required")
	}
	if runner == nil {
		return nil, errors.New("action runner is required")
	}
	if statusChecker == nil {
		return nil, errors.New("status checker is required")
	}
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	workers := store.GetSettings().MaxConcurrentActions
	capacity := workers * 4
	ctx, cancel := context.WithCancel(context.Background())
	queue := &Queue{
		store:         store,
		runner:        runner,
		statusChecker: statusChecker,
		logger:        logger,
		workers:       workers,
		capacity:      capacity,
		ctx:           ctx,
		cancel:        cancel,
		wake:          make(chan struct{}, capacity),
		waiting:       make(map[jobKey]struct{}),
		drained:       make(chan struct{}),
	}
	for range workers {
		queue.group.Add(1)
		go queue.runWorker()
	}
	return queue, nil
}

// EnqueuePrimary schedules a primary action without waiting for process completion.
func (queue *Queue) EnqueuePrimary(cardID string, action config.Action) error {
	return queue.enqueue(jobKey{cardID: cardID, kind: KindPrimary}, func(ctx context.Context) {
		logger := queue.logger.With("card", cardID, "action", KindPrimary)
		result, err := queue.runner.Run(ctx, action)
		if err != nil {
			if ctx.Err() == nil {
				logger.Error("primary action failed", "error", err)
			}
			return
		}
		logger.Info("primary action finished", "outcome", result.Outcome, "duration", result.Duration, "exitCode", result.ExitCode)
		if err := queue.store.AppendRun(cardID, result.ToRun(KindPrimary)); err != nil {
			logger.Error("save primary action result", "error", err)
		}
	})
}

// EnqueueStatus schedules a manual status action without waiting for its result.
func (queue *Queue) EnqueueStatus(cardID string) error {
	return queue.enqueue(jobKey{cardID: cardID, kind: KindStatus}, func(ctx context.Context) {
		logger := queue.logger.With("card", cardID, "action", KindStatus)
		snapshot, err := queue.statusChecker.CheckNow(ctx, cardID)
		if err != nil {
			if ctx.Err() == nil {
				logger.Error("status action failed", "error", err)
			}
			return
		}
		logger.Info("status action finished", "outcome", snapshot.LastCheck.Outcome, "state", snapshot.State, "duration", snapshot.LastCheck.Duration)
	})
}

// Close stops accepting jobs, discards every job still waiting in the queue
// and lets running jobs finish until ctx is done. When ctx expires first, the
// running jobs are cancelled (their processes are terminated by the execution
// engine) and Close waits for the workers to return. It returns the number of
// discarded jobs and ctx's error when running jobs had to be cancelled. Close
// is idempotent; later calls return immediately.
func (queue *Queue) Close(ctx context.Context) (int, error) {
	queue.mu.Lock()
	if queue.closed {
		queue.mu.Unlock()
		return 0, nil
	}
	queue.closed = true
	dropped := len(queue.queue)
	for range queue.queue {
		queue.pending.Done()
	}
	queue.queue = nil
	queue.waiting = make(map[jobKey]struct{})
	if queue.running == 0 {
		close(queue.drained)
	}
	queue.mu.Unlock()
	if dropped > 0 {
		queue.logger.Warn("discarded queued actions on close", "count", dropped)
	}

	var err error
	select {
	case <-queue.drained:
	case <-ctx.Done():
		err = ctx.Err()
		queue.logger.Warn("cancelling running actions after the grace period", "error", err)
	}
	queue.cancel()
	queue.group.Wait()
	return dropped, err
}

// Wait blocks until every accepted job has finished or been discarded.
func (queue *Queue) Wait() {
	queue.pending.Wait()
}

// Queued reports how many jobs are waiting for a worker.
func (queue *Queue) Queued() int {
	queue.mu.Lock()
	defer queue.mu.Unlock()
	return len(queue.queue)
}

func (queue *Queue) enqueue(key jobKey, run func(context.Context)) error {
	queue.mu.Lock()
	defer queue.mu.Unlock()
	if queue.closed {
		return ErrQueueClosed
	}
	if _, waiting := queue.waiting[key]; waiting {
		return ErrAlreadyQueued
	}
	if len(queue.queue) >= queue.capacity {
		return &QueueFullError{RetryAfter: queue.retryAfterLocked()}
	}
	queue.pending.Add(1)
	queue.queue = append(queue.queue, queuedJob{key: key, run: run})
	queue.waiting[key] = struct{}{}
	queue.wake <- struct{}{}
	return nil
}

// retryAfterLocked estimates how long a client should wait before retrying:
// one time slice per queued job divided among the workers, at least one second.
func (queue *Queue) retryAfterLocked() time.Duration {
	slices := (len(queue.queue) + queue.workers - 1) / queue.workers
	return max(time.Duration(slices)*retryAfterPerQueuedJob, time.Second)
}

func (queue *Queue) runWorker() {
	defer queue.group.Done()
	for {
		select {
		case <-queue.ctx.Done():
			return
		case <-queue.wake:
		}
		job, ok := queue.take()
		if !ok {
			continue
		}
		job.run(queue.ctx)
		queue.finish()
	}
}

// take removes the oldest queued job and marks it running. The job's key is
// released from the deduplication set here, so a new request for the same
// card and kind is queued again while the job runs.
func (queue *Queue) take() (queuedJob, bool) {
	queue.mu.Lock()
	defer queue.mu.Unlock()
	if len(queue.queue) == 0 {
		return queuedJob{}, false
	}
	job := queue.queue[0]
	queue.queue[0] = queuedJob{}
	queue.queue = queue.queue[1:]
	delete(queue.waiting, job.key)
	queue.running++
	return job, true
}

func (queue *Queue) finish() {
	queue.mu.Lock()
	queue.running--
	if queue.closed && queue.running == 0 {
		close(queue.drained)
	}
	queue.mu.Unlock()
	queue.pending.Done()
}
