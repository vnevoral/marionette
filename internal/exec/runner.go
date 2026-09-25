package execengine

import (
	"errors"

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
// configured limit block until an earlier action completes.
func (runner *Runner) Run(action config.Action) (Result, error) {
	runner.slots <- struct{}{}
	defer func() { <-runner.slots }()
	return runner.executor.Execute(action)
}
