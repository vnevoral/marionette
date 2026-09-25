package execengine

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"marionette/internal/config"
)

type fakeFactory struct {
	process Process
	action  config.Action
}

func (factory *fakeFactory) New(_ context.Context, action config.Action) (Process, error) {
	factory.action = action
	return factory.process, nil
}

type fakeProcess struct {
	err    error
	stdout string
	stderr string
	writer io.Writer
}

func (process *fakeProcess) SetOutput(writer io.Writer) {
	process.writer = writer
}

func (process *fakeProcess) Run() error {
	if process.writer != nil {
		_, _ = io.WriteString(process.writer, process.stdout)
		_, _ = io.WriteString(process.writer, process.stderr)
	}
	return process.err
}

func validAction() config.Action {
	return config.Action{
		Command:    "echo",
		TimeoutSec: 1,
		Rule:       config.OutputRule{Type: config.OutputRuleExitCode},
	}
}

func TestExecutorRunsActionAndCapturesTiming(t *testing.T) {
	factory := &fakeFactory{process: &fakeProcess{}}
	executor, err := NewExecutorWithFactory(factory)
	if err != nil {
		t.Fatalf("NewExecutorWithFactory() error = %v", err)
	}

	action := validAction()
	action.Args = []string{"hello"}
	action.Dir = filepath.Dir(t.TempDir())
	action.Env = map[string]string{"MARIONETTE_TEST": "yes"}
	result, err := executor.Execute(action)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Outcome != config.RunOutcomeOK || result.ExitCode != 0 || result.ProcessErr != nil {
		t.Fatalf("Execute() result = %#v", result)
	}
	if result.StartedAt.IsZero() || result.Duration < 0 {
		t.Fatalf("Execute() timing = started %v, duration %v", result.StartedAt, result.Duration)
	}
	if factory.action.Command != action.Command || len(factory.action.Args) != 1 || factory.action.Args[0] != "hello" || factory.action.Dir != action.Dir || factory.action.Env["MARIONETTE_TEST"] != "yes" {
		t.Fatalf("factory received action = %#v", factory.action)
	}
}

func TestExecutorReturnsFailureForProcessError(t *testing.T) {
	processError := errors.New("process failed")
	executor, err := NewExecutorWithFactory(&fakeFactory{process: &fakeProcess{err: processError}})
	if err != nil {
		t.Fatalf("NewExecutorWithFactory() error = %v", err)
	}

	result, err := executor.Execute(validAction())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Outcome != config.RunOutcomeFail || result.ExitCode != -1 || !errors.Is(result.ProcessErr, processError) {
		t.Fatalf("Execute() result = %#v", result)
	}
}

func TestExecutorRejectsInvalidAction(t *testing.T) {
	executor, err := NewExecutorWithFactory(&fakeFactory{process: &fakeProcess{}})
	if err != nil {
		t.Fatalf("NewExecutorWithFactory() error = %v", err)
	}

	if _, err := executor.Execute(config.Action{}); err == nil {
		t.Fatal("Execute() accepted an invalid action")
	}
}

func TestExecutorRejectsNilFactory(t *testing.T) {
	if _, err := NewExecutorWithFactory(nil); err == nil {
		t.Fatal("NewExecutorWithFactory(nil) returned nil error")
	}
}

func TestExecutorCapturesCombinedOutputAtLimit(t *testing.T) {
	process := &fakeProcess{
		stdout: strings.Repeat("a", outputLimit),
	}
	executor, err := NewExecutorWithFactory(&fakeFactory{process: process})
	if err != nil {
		t.Fatalf("NewExecutorWithFactory() error = %v", err)
	}

	result, err := executor.Execute(validAction())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(result.Output) != outputLimit || result.Truncated {
		t.Fatalf("exact-limit result = len(%d), truncated(%t)", len(result.Output), result.Truncated)
	}

	process.stdout = strings.Repeat("b", outputLimit)
	process.stderr = "stderr-after-limit"
	result, err = executor.Execute(validAction())
	if err != nil {
		t.Fatalf("Execute() second error = %v", err)
	}
	if len(result.Output) != outputLimit || !result.Truncated {
		t.Fatalf("over-limit result = len(%d), truncated(%t)", len(result.Output), result.Truncated)
	}
}

func TestExecutorEvaluatesOutputRules(t *testing.T) {
	tests := []struct {
		name   string
		rule   config.OutputRule
		output string
		want   config.RunOutcome
	}{
		{name: "exit code", rule: config.OutputRule{Type: config.OutputRuleExitCode}, output: "anything", want: config.RunOutcomeOK},
		{name: "match success", rule: config.OutputRule{Type: config.OutputRuleMatch, Pattern: "ready"}, output: "device ready", want: config.RunOutcomeOK},
		{name: "match failure", rule: config.OutputRule{Type: config.OutputRuleMatch, Pattern: "ready"}, output: "device down", want: config.RunOutcomeFail},
		{name: "not match success", rule: config.OutputRule{Type: config.OutputRuleNotMatch, Pattern: "error"}, output: "device ready", want: config.RunOutcomeOK},
		{name: "not match failure", rule: config.OutputRule{Type: config.OutputRuleNotMatch, Pattern: "error"}, output: "device error", want: config.RunOutcomeFail},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := Evaluate(test.rule, 0, test.output)
			if err != nil || got != test.want {
				t.Fatalf("Evaluate() = %q, error = %v, want %q", got, err, test.want)
			}
		})
	}
}

func TestEvaluateNonZeroExitAlwaysFails(t *testing.T) {
	got, err := Evaluate(config.OutputRule{Type: config.OutputRuleMatch, Pattern: "ready"}, 1, "ready")
	if err != nil || got != config.RunOutcomeFail {
		t.Fatalf("Evaluate() = %q, error = %v, want fail", got, err)
	}
}

func TestResultToRun(t *testing.T) {
	startedAt := time.Now()
	result := Result{
		StartedAt: startedAt,
		Duration:  2 * time.Second,
		ExitCode:  0,
		Outcome:   config.RunOutcomeOK,
		Output:    "ready",
		Truncated: true,
	}
	run := result.ToRun("primary")
	if run.ActionKind != "primary" || !run.StartedAt.Equal(startedAt) || run.Duration != result.Duration || run.Output != result.Output || !run.Truncated || run.Outcome != result.Outcome {
		t.Fatalf("ToRun() = %#v", run)
	}
}

func TestExecutorTimeoutTerminatesProcess(t *testing.T) {
	executor := NewExecutor()
	action := validAction()
	action.Command = "sleep"
	action.Args = []string{"2"}
	action.TimeoutSec = 1

	started := time.Now()
	result, err := executor.Execute(action)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Outcome != config.RunOutcomeTimeout {
		t.Fatalf("Execute() outcome = %q, want timeout; result = %#v", result.Outcome, result)
	}
	if time.Since(started) >= 2*time.Second {
		t.Fatalf("timeout took too long: %v", time.Since(started))
	}
}

func TestExitCodeFromExecError(t *testing.T) {
	command := exec.Command("sh", "-c", "exit 7")
	err := command.Run()
	if got := exitCode(err); got != 7 {
		t.Fatalf("exitCode() = %d, want 7", got)
	}
}
