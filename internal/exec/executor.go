// Package execengine executes configured actions as host processes.
package execengine

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	"marionette/internal/config"
)

// Process is the small process abstraction used by Executor.
type Process interface {
	Run() error
}

// ProcessFactory creates a process for an action.
type ProcessFactory interface {
	New(context.Context, config.Action) (Process, error)
}

// Executor runs one validated action at a time.
type Executor struct {
	factory ProcessFactory
}

// Result is the process execution result before output evaluation.
type Result struct {
	StartedAt  time.Time
	Duration   time.Duration
	ExitCode   int
	Outcome    config.RunOutcome
	ProcessErr error
}

// NewExecutor creates an executor backed by os/exec.
func NewExecutor() *Executor {
	return &Executor{factory: osProcessFactory{}}
}

// NewExecutorWithFactory creates an executor with a testable process factory.
func NewExecutorWithFactory(factory ProcessFactory) (*Executor, error) {
	if factory == nil {
		return nil, errors.New("process factory is required")
	}
	return &Executor{factory: factory}, nil
}

// Execute validates and runs one action. Process failures are represented in
// Result with RunOutcomeFail; the returned error is reserved for invalid input
// and executor setup failures.
func (executor *Executor) Execute(action config.Action) (Result, error) {
	if err := action.Validate(); err != nil {
		return Result{}, fmt.Errorf("validate action: %w", err)
	}

	startedAt := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(action.TimeoutSec)*time.Second)
	defer cancel()

	process, err := executor.factory.New(ctx, action)
	if err != nil {
		return Result{}, fmt.Errorf("create process: %w", err)
	}

	processErr := process.Run()
	result := Result{
		StartedAt:  startedAt,
		Duration:   time.Since(startedAt),
		ExitCode:   0,
		Outcome:    config.RunOutcomeOK,
		ProcessErr: processErr,
	}
	if processErr != nil {
		result.ExitCode = exitCode(processErr)
		result.Outcome = config.RunOutcomeFail
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			result.Outcome = config.RunOutcomeTimeout
		}
	}
	return result, nil
}

func exitCode(err error) int {
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		return exitError.ExitCode()
	}
	return -1
}

type osProcessFactory struct{}

func (osProcessFactory) New(ctx context.Context, action config.Action) (Process, error) {
	command := exec.CommandContext(ctx, action.Command, action.Args...)
	command.Dir = action.Dir
	command.Env = actionEnvironment(action.Env)
	return command, nil
}

func actionEnvironment(values map[string]string) []string {
	environment := os.Environ()
	for key, value := range values {
		environment = append(environment, key+"="+value)
	}
	return environment
}
