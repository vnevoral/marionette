package execengine

import (
	"context"
	"errors"
	"fmt"

	"marionette/internal/config"
)

// Runner limits the number of actions executing concurrently.
type Runner struct {
	executor *Executor
	slots    chan struct{}
}

// NewRunner creates a concurrency-limited runner for the supplied executor.
func NewRunner(settings config.Settings, executor *Executor) (*Runner, error) {
	if err := settings.Validate(); err != nil {
		return nil, err
	}
	if executor == nil {
		return nil, errors.New("executor is required")
	}
	return &Runner{
		executor: executor,
		slots:    make(chan struct{}, settings.MaxConcurrentActions),
	}, nil
}

// Run waits for a semaphore slot and executes one action. Calls above the
// configured limit block until an earlier action completes or ctx is
// canceled; a cancellation while waiting returns an error wrapping ctx.Err()
// and no Result, because the action never started.
func (runner *Runner) Run(ctx context.Context, action config.Action) (Result, error) {
	if ctx == nil {
		return Result{}, errors.New("context is required")
	}
	select {
	case runner.slots <- struct{}{}:
	case <-ctx.Done():
		return Result{}, fmt.Errorf("wait for execution slot: %w", ctx.Err())
	}
	defer func() { <-runner.slots }()
	return runner.executor.Execute(ctx, action)
}
