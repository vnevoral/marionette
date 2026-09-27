# Implementation block: Execution engine — process group termination and context propagation

- **Phase**: 8 — Hardening
- **Requirements**: FR-11, FR-19, FR-18, NFR-04, NFR-07
- **ADRs**: ADR-0005
- **Status**: Done
- **Dependencies**: Blocks 0004–0006 (execution engine), 0009 (scheduler), 0011 (orchestration)

## Goal

When done, an action timeout reliably terminates the whole tree of
processes started by the action (not just the direct child), the
concurrency slot is released at the latest shortly after the timeout, and
canceling the caller's context (scheduler worker, shutdown) interrupts
both waiting for a slot and the running action.

## Scope

- **In scope**:
  - starting the process in its own process group (`Setpgid`) and a
    `cmd.Cancel` that kills the whole group when the context expires
    (`kill(-pgid, SIGKILL)`);
  - `cmd.WaitDelay`, so that `Wait()` does not wait for the stdout/stderr
    pipe held by surviving grandchildren to close;
  - the signatures `Executor.Execute(ctx, action)` and
    `Runner.Run(ctx, action)` with the caller's context; the action timeout
    is derived with `context.WithTimeout(ctx, …)`; waiting for a semaphore
    slot reacts to `ctx.Done()`;
  - the scheduler worker and `BackgroundActions` pass their context, so
    that `Reconcile`/`Stop` do not wait for the full `TimeoutSec` to run
    out;
  - the upper bound of `TimeoutSec` is handled by block 0028, here it is
    only documented;
  - an integration test with a process that starts a child and holds the
    pipe.
- **Out of scope**:
  - Windows support (not a target platform, `syscall.SysProcAttr` is
    Linux);
  - changing the format of `RunRecord` or the API contract;
  - changing the order of shutdown steps (block 0027);
  - queue limits and deduplication (FR-18 refined, handled by block 0027).

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-09-26
- **Decision notes**: Approved as a bug fix (not new functionality).
  Review finding 2026-09-26 (H-1, M-1). Reproduced
  outside the repo: `sh -c "sleep 30 & echo started; wait"` with a 1 s
  timeout returned from `Run()` only after 30 s and the child survived.
  ADR-0005 promises that the process will not keep running; the block adds
  a note to ADR-0005 about the process group as the mechanism.

## Proposed solution

- `internal/exec/executor.go`, `osProcessFactory.New`: set
  `command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}`,
  `command.Cancel = func() error { return syscall.Kill(-command.Process.Pid,
  syscall.SIGKILL) }`, `command.WaitDelay = 2 * time.Second`. The
  `WaitDelay` value is a package constant with a comment.
- `Executor.Execute(ctx context.Context, action config.Action) RunRecord`:
  inner timeout `context.WithTimeout(ctx, action.Timeout())`; distinguish
  `RunOutcomeTimeout` (the action timeout expired) from `RunOutcomeCanceled`
  (canceled by the caller) — the new outcome must be reflected in
  `config.RunOutcome` and its validation; the UI shows it as "Canceled".
- `Runner.Run(ctx, action)`: `select { case r.slots <- struct{}{}: case
  <-ctx.Done(): return canceled }`.
- `internal/status/scheduler.go`: the worker passes `worker.ctx` to
  `CheckNow`; the `statusChecker` interface gets a context.
  `Service.CheckNow(ctx, cardID)`.
- `internal/server/actions.go`: a job gets the `BackgroundActions`
  context, which `Close()` cancels (the final behavior of Close is handled
  by 0027, here only passing the ctx).
- The `ProcessFactory`/`process` interfaces in tests get a context without
  changing the behavior of the fake implementations.

## Test plan

- Unit tests (`internal/exec`):
  - `TestExecutorKillsProcessGroupOnTimeout`: a real
    `sh -c "sleep 30 & wait"` with a 1 s timeout; `Execute` returns within
    3 s, outcome `timeout`, and `kill(-pgid, 0)` after the return reports
    `ESRCH` (the group does not exist);
  - `TestExecutorCanceledByCaller`: canceling ctx before the timeout
    returns `canceled`, not `timeout`;
  - `TestRunnerWaitForSlotHonoursContext`: full semaphore + canceled ctx →
    immediate return without blocking a slot;
  - adjust existing semaphore and timeout tests to the new signature.
- Unit tests (`internal/status`): `Reconcile` of a removed card with a
  worker in the middle of a long action (fake runner blocking on ctx)
  finishes within 100 ms.
- `go test -race ./...`, `go vet ./...`, `golangci-lint run ./...`.
- Manual verification: a card with the primary action
  `sh -c "sleep 60 & wait"` and a 5 s timeout; after 5 s the slot is free
  (another card starts) and `ps` shows no orphaned `sleep`.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md) plus:

- the test with a child holding the pipe passes under `-race` within 3 s;
- `RunOutcomeCanceled` is documented in ADR-0005 and in the types;
- ADR-0005 contains a note about the process group and `WaitDelay`.

## Closure

- **Status after implementation**: Done (2026-09-26)
- **Verification**: `make verify` passed (golangci-lint, eslint, vue-tsc,
  prettier, `go test -race -count=1 ./...`, build, vet). New tests:
  `TestExecutorKillsProcessGroupOnTimeout` (a real `sh -c "echo $$; sleep 30
  & wait"`, 1 s timeout → return after 1.0 s, outcome `timeout`,
  `kill(-pgid, 0)` reports ESRCH),
  `TestExecutorDoesNotHoldSlotForDetachedGrandchild` (`setsid sleep 6`
  holds the pipe → return after 3.0 s thanks to `WaitDelay`),
  `TestExecutorCanceledByCaller`, `TestExecutorRejectsNilContext`,
  `TestRunnerWaitForSlotHonoursContext`, `TestCheckNowCanceledDoesNotUpdateStatus`,
  `TestSchedulerReconcileCancelsInFlightCheck` (Reconcile of a deleted card
  with a blocked check finishes in < 100 ms), `TestBackgroundActionsCloseCancelsRunningJob`
  (incl. an idempotent second `Close`). Manual end-to-end on the real
  binary: card `sh -c "echo $$; sleep 60 & wait"` with a 3 s timeout → 202,
  after 3.0 s a record `outcome: timeout`, nothing remained in the process
  group nor among processes `sleep 60`.
- **Deviations from the plan**: `BackgroundActions.Close()` now cancels
  the job context immediately (running actions end as `canceled` during
  shutdown and are written to history); the protective window for running
  actions to finish will be added by block 0027 per its plan. The
  `ErrStatusCheckCanceled` error lives in the `status` package;
  `internal/server` does not recognize it via `errors.Is` because of
  layering, but via `ctx.Err()`. The `setsid` test is skipped if the
  binary is not available. The upper bound of `TimeoutSec` remains with
  block 0028.
- **Documentation updated**: yes — ADR-0005 (note about the process group,
  `WaitDelay` and `RunOutcomeCanceled`), godoc on `Execute`, `Run`,
  `CheckNow`, `Close`; the frontend shows `canceled` as "Canceled"; roadmap.
