// Package execengine executes configured actions as host processes.
package execengine

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"time"

	"marionette/internal/config"
)

// Process is the small process abstraction used by Executor.
type Process interface {
	SetOutput(io.Writer)
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
	Output     string
	Truncated  bool
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

	output := newLimitedWriter(outputLimit)
	process.SetOutput(output)
	processErr := process.Run()
	result := Result{
		StartedAt:  startedAt,
		Duration:   time.Since(startedAt),
		ExitCode:   0,
		Outcome:    config.RunOutcomeOK,
		ProcessErr: processErr,
		Output:     output.String(),
		Truncated:  output.Truncated(),
	}
	if processErr != nil {
		result.ExitCode = exitCode(processErr)
		result.Outcome = config.RunOutcomeFail
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			result.Outcome = config.RunOutcomeTimeout
		} else if result.ExitCode == 0 {
			result.Outcome, err = Evaluate(action.Rule, result.ExitCode, result.Output)
		}
	} else {
		result.Outcome, err = Evaluate(action.Rule, result.ExitCode, result.Output)
	}
	if err != nil {
		return Result{}, fmt.Errorf("evaluate action result: %w", err)
	}
	return result, nil
}

// ToRun converts an execution result into a config history record.
func (result Result) ToRun(actionKind string) config.Run {
	return config.Run{
		ActionKind: actionKind,
		StartedAt:  result.StartedAt,
		Duration:   result.Duration,
		ExitCode:   result.ExitCode,
		Output:     result.Output,
		Truncated:  result.Truncated,
		Outcome:    result.Outcome,
	}
}

// Evaluate applies the exit-code and output rule to a completed process.
func Evaluate(rule config.OutputRule, exitCode int, output string) (config.RunOutcome, error) {
	if err := rule.Validate(); err != nil {
		return "", err
	}
	if exitCode != 0 {
		return config.RunOutcomeFail, nil
	}

	ruleType := rule.Type
	if ruleType == "" {
		ruleType = config.OutputRuleExitCode
	}
	switch ruleType {
	case config.OutputRuleExitCode:
		return config.RunOutcomeOK, nil
	case config.OutputRuleMatch:
		matched, err := regexp.MatchString(rule.Pattern, output)
		if err != nil {
			return "", err
		}
		if matched {
			return config.RunOutcomeOK, nil
		}
		return config.RunOutcomeFail, nil
	case config.OutputRuleNotMatch:
		matched, err := regexp.MatchString(rule.Pattern, output)
		if err != nil {
			return "", err
		}
		if !matched {
			return config.RunOutcomeOK, nil
		}
		return config.RunOutcomeFail, nil
	default:
		return "", fmt.Errorf("unsupported output rule type %q", rule.Type)
	}
}

func exitCode(err error) int {
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		return exitError.ExitCode()
	}
	return -1
}

type osProcessFactory struct{}

type osProcess struct {
	command *exec.Cmd
}

func (osProcessFactory) New(ctx context.Context, action config.Action) (Process, error) {
	command := exec.CommandContext(ctx, action.Command, action.Args...)
	command.Dir = action.Dir
	command.Env = actionEnvironment(action.Env)
	return osProcess{command: command}, nil
}

func (process osProcess) SetOutput(writer io.Writer) {
	process.command.Stdout = writer
	process.command.Stderr = writer
}

func (process osProcess) Run() error {
	return process.command.Run()
}

func actionEnvironment(values map[string]string) []string {
	environment := os.Environ()
	for key, value := range values {
		environment = append(environment, key+"="+value)
	}
	return environment
}
