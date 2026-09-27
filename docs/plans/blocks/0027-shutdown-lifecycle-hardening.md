# Implementation block: Graceful shutdown — step order, SSE and the action queue

- **Phase**: 8 — Hardening
- **Requirements**: FR-35, FR-18, NFR-03, NFR-04, NFR-11
- **ADRs**: ADR-0004, ADR-0008
- **Status**: Done
- **Dependencies**: Block 0024 (context through the Runner), block 0012 (lifecycle), block 0022 (SSE)

## Goal

When done, graceful shutdown runs deterministically and quickly: history
is saved before systemd could kill the process, SSE clients disconnect
immediately and the action queue drops jobs that have not started instead
of waiting for them to finish.

## Scope

- **In scope**:
  - `StatusEventBroker.Close()` closes all subscribers and the `events`
    handler ends without waiting for `srv.Close()`;
  - `http.Server` gets an `IdleTimeout` and a `BaseContext` derived from
    the application context;
  - new order in `main`: stop accepting HTTP → save config + history
    (first pass) → `BackgroundActions.Close()` (drops waiting jobs,
    finishes running ones with a short limit) → `scheduler.Stop()` → final
    history save (second pass, only if it changed since the first);
  - `BackgroundActions.Close()` is idempotent (`sync.Once`), logs the
    number of dropped jobs;
  - remove `requestTracker` or document why it exists (the implementation
    decides, based on whether `srv.Shutdown` covers the need);
  - queue behavior per the refined FR-18: full queue → 503 with a
    `Retry-After` header (value = estimate from the queue length, min. 1 s);
    a request for a card + action kind that is already waiting in the queue
    is not enqueued again and returns 202 idempotently (deduplication only
    for waiting jobs, not running ones);
  - a configurable overall shutdown limit `MARIONETTE_SHUTDOWN_TIMEOUT`
    (default 20 s, less than systemd `TimeoutStopSec` 90 s) — document it
    in `deploy/marionette.default`.
- **Out of scope**:
  - changing the format of the saved history;
  - interrupting an already running action process in any way other than
    canceling the context (handled by 0024);
  - persisting history while running (ADR-0004 deliberately does not do
    that).

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-09-26
- **Decision notes**: Approved as a fix and a straightening of the state.
  Review finding 2026-09-26 (M-2, M-5, M-8, L-9, L-10); FR-18 refined
  the same day (section 13 of the requirements).
  Today every shutdown with an open dashboard takes the full 5 s and history
  is saved only after the queue drains, which violates FR-35 for long
  actions.

## Proposed solution

- `internal/server/status_events.go`: `done chan struct{}` in the broker;
  `Close()` closes it under the lock and disconnects subscribers; the
  `events` handler does a `select` on `r.Context().Done()`, `broker.done`,
  heartbeat and events.
- `cmd/marionette/main.go`: extract `run(ctx, env, logger) error`;
  the shutdown sequence as a separate function `shutdown(ctx, deps)` with
  individual steps and a log of each step's duration.
- `internal/server/actions.go` (or a new package per 0034):
  `Close()` closes the queue, empties unprocessed jobs, cancels the context
  of running ones and waits on the `WaitGroup` for at most
  `shutdownTimeout`.
- `config.Store.SaveFileWithHistory` called twice is cheap (small JSON),
  the second call only if `store` reports a change since the last save
  (a simple `dirty` flag under `persistMu`).

## Test plan

- Unit tests:
  - `TestBrokerCloseEndsEventsHandler`: an `httptest` client connected to
    `/api/events`, `Close()` → the handler returns within 100 ms;
  - `TestBackgroundActionsCloseDropsQueued`: 10 jobs, 1 worker blocking on
    ctx, `Close()` → returns within the limit, dropped jobs counted;
  - `TestBackgroundActionsCloseIdempotent`: double `Close()` without panic;
  - `TestEnqueueDeduplicatesWaitingJob`: a second enqueue of the same card
    and kind returns 202 without a new job; after the job starts, a new
    enqueue is enqueued again;
  - handler test: full queue → 503 and `Retry-After` ≥ 1;
  - `TestShutdownSavesHistoryBeforeQueueDrain`: a fake store records the
    order of `SaveFileWithHistory` vs `Close()` calls.
- Integration smoke test of `run()`: start, POST an action with a long
  timeout, SIGTERM → the process ends within 3 s and the file contains the
  history.
- `go test -race ./...`, `go vet ./...`.
- Manual verification on the reference host: `systemctl restart marionette`
  with an open dashboard takes < 3 s; the journal shows the shutdown steps.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md) plus:

- shutdown with a connected SSE client does not wait for the 5 s timeout;
- history is on disk before waiting for the queue and the scheduler;
- `MARIONETTE_SHUTDOWN_TIMEOUT` is documented in the README and FR-34 is
  extended with this variable.

## Closure

- **Status after implementation**: Done (2026-09-26)
- **Verification**: `make verify` passed (golangci-lint, eslint, vue-tsc, prettier,
  `go test -race -count=1 ./...`, build, vet). New and modified tests:
  `TestBrokerCloseEndsEventsHandler` (real HTTP client, the stream ends
  within 100 ms), `TestStatusEventsEndpointEndsImmediatelyWhenBrokerClosed`,
  `TestBackgroundActionsCloseDropsQueued` (4 waiting dropped, the running
  one canceled after a 50 ms grace), `TestBackgroundActionsCloseIdempotent`,
  `TestBackgroundActionsCloseLetsRunningJobFinishWithinGrace`,
  `TestEnqueueDeduplicatesWaitingJob` (same card + kind → `202`, a different
  kind is enqueued, after the job starts a new request is enqueued again),
  `TestEnqueueReportsFullQueueWithRetryAfter`,
  `TestRouterMapsQueueErrorsToAcceptedOrServiceUnavailable` (`202`/`503` +
  `Retry-After`), `TestStoreDirtyTracksChangesSinceHistorySave`,
  `TestLoadEnvironmentDefaultsAndShutdownTimeout`,
  `TestShutdownSavesHistoryBeforeQueueDrain` (order http → save → actions →
  scheduler → save), `TestShutdownSkipsSecondSaveWhenCleanAndSaveWhenReadOnly`,
  `TestGraceDeadlineNeverInThePast` and the integration smoke test
  `TestRunShutsDownQuicklyWithOpenSSEClientAndSavesHistory` (a running
  `sleep 30`, a connected SSE client, 2 s limit → `run()` ends within 3 s,
  the stream is closed, the file contains the run with the result
  `canceled`, the process group no longer exists). Manual verification on
  the reference host (`systemctl restart` with an open dashboard < 3 s)
  remains to be done together with block 0023.
- **Deviations from the plan**: (1) `requestTracker` removed —
  `http.Server.Shutdown` itself waits for active handlers to finish, the
  tracker was redundant; (2) `BackgroundActions.Close(ctx)` takes a
  context instead of an internal `shutdownTimeout` and returns the number
  of dropped jobs; running jobs get a grace period (limit minus a 3 s
  reserve) and only then are canceled, so that short actions finish and
  get written to history; (3) Close idempotence is implemented with a flag
  under the lock, not `sync.Once`, because of the return value; (4) the
  queue is a slice under a lock instead of a channel, because a channel
  allows neither deduplication nor dropping waiting jobs; (5) `Retry-After`
  is computed as `ceil(waiting / workers) × 1 s`, min. 1 s — action
  durations are not known in advance; (6) additionally fixed an impact of
  block 0025: in read-only mode history is not saved during shutdown, so
  that an unreadable file is not overwritten with an empty configuration
  (`openStore` returns a `readOnly` flag); (7) a second signal during
  shutdown terminates the process immediately (`signal.NotifyContext` +
  restoring the default handler); (8) `http.Server` got `IdleTimeout` 60 s
  and `BaseContext` from the application context.
- **Documentation updated**: yes — `docs/architecture/overview.md`
  (section "Graceful shutdown"), requirements FR-34 and the
  decision table (`MARIONETTE_SHUTDOWN_TIMEOUT`), ADR-0008 (addendum on
  `Close()`), README (variable table, shutdown behavior),
  `deploy/marionette.default`, godoc (`shutdown`, `BackgroundActions.Close`,
  `StatusEventBroker.Close`, `Store.Dirty`), roadmap.
