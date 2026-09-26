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
	"sort"
	"syscall"
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

// processWaitDelay bounds how long Wait blocks on the output pipe after the
// process was killed. Grandchildren that inherited stdout/stderr keep the pipe
// open even though the whole process group received SIGKILL; without the delay
// a timed-out action would hold its concurrency slot until they exit.
const processWaitDelay = 2 * time.Second

// Execute validates and runs one action. The action timeout is derived from
// ctx, so canceling ctx terminates the process early and yields
// RunOutcomeCanceled, whereas an expired action timeout yields
// RunOutcomeTimeout. Process failures are represented in Result with
// RunOutcomeFail; the returned error is reserved for invalid input and
// executor setup failures.
func (executor *Executor) Execute(ctx context.Context, action config.Action) (Result, error) {
	if ctx == nil {
		return Result{}, errors.New("context is required")
	}
	if err := action.Validate(); err != nil {
		return Result{}, fmt.Errorf("validate action: %w", err)
	}

	startedAt := time.Now()
	runContext, cancel := context.WithTimeout(ctx, time.Duration(action.TimeoutSec)*time.Second)
	defer cancel()

	process, err := executor.factory.New(runContext, action)
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
		switch {
		case ctx.Err() != nil:
			result.Outcome = config.RunOutcomeCanceled
		case errors.Is(runContext.Err(), context.DeadlineExceeded):
			result.Outcome = config.RunOutcomeTimeout
		case result.ExitCode == 0:
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
	// The action runs in its own process group so that cancellation kills the
	// whole tree (shell wrappers, ssh, background children), not only the
	// direct child. See ADR-0005 and FR-19.
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.Cancel = func() error {
		return killProcessGroup(command)
	}
	command.WaitDelay = processWaitDelay
	return osProcess{command: command}, nil
}

func killProcessGroup(command *exec.Cmd) error {
	if command.Process == nil {
		return nil
	}
	err := syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
	if errors.Is(err, syscall.ESRCH) {
		return os.ErrProcessDone
	}
	return err
}

func (process osProcess) SetOutput(writer io.Writer) {
	process.command.Stdout = writer
	process.command.Stderr = writer
}

func (process osProcess) Run() error {
	return process.command.Run()
}

// baseEnvironmentKeys are the only variables an action inherits from the
// service process (NFR-01 c): enough to locate binaries and behave
// predictably, without leaking secrets from the service environment.
var baseEnvironmentKeys = []string{"PATH", "HOME", "LANG", "TZ"}

// actionEnvironment builds the process environment from the minimal base and
// the action's own variables, which take precedence over the base. Keys are
// emitted in a stable order.
func actionEnvironment(values map[string]string) []string {
	environment := make([]string, 0, len(baseEnvironmentKeys)+len(values))
	for _, key := range baseEnvironmentKeys {
		if _, overridden := values[key]; overridden {
			continue
		}
		if value, ok := os.LookupEnv(key); ok {
			environment = append(environment, key+"="+value)
		}
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		environment = append(environment, key+"="+values[key])
	}
	return environment
}
