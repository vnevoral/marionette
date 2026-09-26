package execengine

import (
	"context"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"marionette/internal/config"
)

type runnerProcessFactory struct {
	active      atomic.Int32
	maxActive   atomic.Int32
	started     chan struct{}
	release     chan struct{}
	processErr  error
	waitContext bool
}

func (factory *runnerProcessFactory) New(ctx context.Context, _ config.Action) (Process, error) {
	return &runnerProcess{
		factory:     factory,
		ctx:         ctx,
		processErr:  factory.processErr,
		waitContext: factory.waitContext,
	}, nil
}

type runnerProcess struct {
	factory     *runnerProcessFactory
	ctx         context.Context
	processErr  error
	waitContext bool
	writer      io.Writer
}

func (process *runnerProcess) SetOutput(writer io.Writer) {
	process.writer = writer
}

func (process *runnerProcess) Run() error {
	active := process.factory.active.Add(1)
	for {
		currentMax := process.factory.maxActive.Load()
		if active <= currentMax || process.factory.maxActive.CompareAndSwap(currentMax, active) {
			break
		}
	}
	defer process.factory.active.Add(-1)
	if process.factory.started != nil {
		process.factory.started <- struct{}{}
	}
	if process.waitContext {
		<-process.ctx.Done()
		return process.ctx.Err()
	}
	if process.factory.release != nil {
		<-process.factory.release
	}
	return process.processErr
}

func newRunner(t *testing.T, settings config.Settings, factory ProcessFactory) *Runner {
	t.Helper()
	executor, err := NewExecutorWithFactory(factory)
	if err != nil {
		t.Fatalf("NewExecutorWithFactory() error = %v", err)
	}
	runner, err := NewRunner(settings, executor)
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}
	return runner
}

func validRunnerSettings() config.Settings {
	return config.Settings{
		HistorySize:          config.DefaultHistorySize,
		MaxConcurrentActions: config.DefaultMaxConcurrentActions,
	}
}

func TestNewRunnerRejectsInvalidSettingsAndExecutor(t *testing.T) {
	executor := NewExecutor()
	if _, err := NewRunner(config.Settings{}, executor); err == nil {
		t.Fatal("NewRunner() accepted invalid settings")
	}
	if _, err := NewRunner(validRunnerSettings(), nil); err == nil {
		t.Fatal("NewRunner() accepted nil executor")
	}
}

func TestRunnerLimitOneQueuesSecondAction(t *testing.T) {
	factory := &runnerProcessFactory{started: make(chan struct{}, 2), release: make(chan struct{})}
	settings := validRunnerSettings()
	settings.MaxConcurrentActions = 1
	runner := newRunner(t, settings, factory)

	firstDone := make(chan struct{})
	go func() {
		_, _ = runner.Run(context.Background(), validAction())
		close(firstDone)
	}()
	<-factory.started

	secondDone := make(chan struct{})
	go func() {
		_, _ = runner.Run(context.Background(), validAction())
		close(secondDone)
	}()
	select {
	case <-factory.started:
		t.Fatal("second action started while the first occupied the only slot")
	case <-time.After(20 * time.Millisecond):
	}

	close(factory.release)
	<-firstDone
	<-secondDone
	if factory.maxActive.Load() != 1 {
		t.Fatalf("max concurrent actions = %d, want 1", factory.maxActive.Load())
	}
}

func TestRunnerReleasesSlotAfterError(t *testing.T) {
	factory := &runnerProcessFactory{processErr: errors.New("failed")}
	settings := validRunnerSettings()
	settings.MaxConcurrentActions = 1
	runner := newRunner(t, settings, factory)

	for range 2 {
		result, err := runner.Run(context.Background(), validAction())
		if err != nil || result.Outcome != config.RunOutcomeFail {
			t.Fatalf("Run() result = %#v, error = %v", result, err)
		}
	}
}

func TestRunnerReleasesSlotAfterTimeout(t *testing.T) {
	factory := &runnerProcessFactory{started: make(chan struct{}, 2), waitContext: true}
	settings := validRunnerSettings()
	settings.MaxConcurrentActions = 1
	runner := newRunner(t, settings, factory)
	action := validAction()
	action.TimeoutSec = 1

	firstDone := make(chan Result)
	go func() {
		result, _ := runner.Run(context.Background(), action)
		firstDone <- result
	}()
	<-factory.started
	first := <-firstDone
	if first.Outcome != config.RunOutcomeTimeout {
		t.Fatalf("first timeout result = %#v", first)
	}

	secondDone := make(chan Result)
	go func() {
		result, _ := runner.Run(context.Background(), action)
		secondDone <- result
	}()
	select {
	case second := <-secondDone:
		if second.Outcome != config.RunOutcomeTimeout {
			t.Fatalf("second timeout result = %#v", second)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("second action did not acquire the released slot")
	}
}

func TestRunnerDoesNotLoseConcurrentCalls(t *testing.T) {
	factory := &runnerProcessFactory{}
	settings := validRunnerSettings()
	settings.MaxConcurrentActions = 4
	runner := newRunner(t, settings, factory)

	const calls = 32
	var waitGroup sync.WaitGroup
	waitGroup.Add(calls)
	for range calls {
		go func() {
			defer waitGroup.Done()
			if _, err := runner.Run(context.Background(), validAction()); err != nil {
				t.Errorf("Run() error = %v", err)
			}
		}()
	}
	waitGroup.Wait()
	if factory.maxActive.Load() > int32(settings.MaxConcurrentActions) {
		t.Fatalf("max concurrent actions = %d, want <= %d", factory.maxActive.Load(), settings.MaxConcurrentActions)
	}
}

func TestRunnerWaitForSlotHonoursContext(t *testing.T) {
	factory := &runnerProcessFactory{started: make(chan struct{}, 1), release: make(chan struct{})}
	settings := validRunnerSettings()
	settings.MaxConcurrentActions = 1
	runner := newRunner(t, settings, factory)

	firstDone := make(chan struct{})
	go func() {
		_, _ = runner.Run(context.Background(), validAction())
		close(firstDone)
	}()
	<-factory.started

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	started := time.Now()
	_, err := runner.Run(ctx, validAction())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() with canceled context error = %v, want context.Canceled", err)
	}
	if time.Since(started) > 100*time.Millisecond {
		t.Fatalf("Run() blocked %v on the semaphore despite a canceled context", time.Since(started))
	}

	close(factory.release)
	<-firstDone
	if factory.maxActive.Load() != 1 {
		t.Fatalf("max concurrent actions = %d, want 1", factory.maxActive.Load())
	}
}
