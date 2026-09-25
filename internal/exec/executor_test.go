package execengine

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
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
	err error
}

func (process fakeProcess) Run() error {
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
	factory := &fakeFactory{process: fakeProcess{}}
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
	executor, err := NewExecutorWithFactory(&fakeFactory{process: fakeProcess{err: processError}})
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
	executor, err := NewExecutorWithFactory(&fakeFactory{process: fakeProcess{}})
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
