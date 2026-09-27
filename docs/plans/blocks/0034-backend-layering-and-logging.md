# Implementation block: Backend layering, structured logging and minor cleanups

- **Phase**: 8 — Hardening
- **Requirements**: NFR-03, NFR-05, NFR-06
- **ADRs**: ADR-0005, ADR-0008
- **Status**: Done
- **Dependencies**: Blocks 0024, 0025, 0027 (so that the refactor is done on top of the fixed code)

## Goal

After this block, `internal/server` is a pure HTTP layer, application
services (action queue, event broker) have their own packages, logging
uses `log/slog` with levels and card context, and the scheduler has a
unified card selection logic.

## Scope

- **In scope**:
  - move `BackgroundActions` to `internal/actions`, `StatusEventBroker`
    to `internal/events`; `NewRouterWithDependencies` accepts interfaces
    instead of mutating `Store.OnStatusChange` (composition in `main`);
  - remove the variadic `NewRouter(stores ...*config.Store)`;
  - `log/slog` with `TextHandler` (readable in journald) or `JSONHandler`
    depending on `MARIONETTE_LOG_FORMAT`; level via `MARIONETTE_LOG_LEVEL`;
    attributes `card`, `action`, `outcome`, `duration`;
  - scheduler: a common `desiredCards()` for both `Start` and `Reconcile`,
    rename field `context` → `ctx`, `running=false` after the parent
    context is canceled;
  - remove or document `cloneStatusSnapshot`;
  - godoc for all exported errors and types (`revive` from 0031 will
    enforce it);
  - rename directory `internal/exec` → `internal/execengine` so the
    directory name matches the package (update the ADR-0005 reference);
  - `actionEnvironment`: a minimal base environment (`PATH`, `HOME`,
    `LANG`, `TZ`) + `action.Env` instead of inheriting the whole
    `os.Environ()`; document in ADR-0005 (NFR-01 c already accepted;
    impact on users: service variables are not available to actions);
  - `UpdateSettings` is removed from the store's public API including
    tests (decided 2026-09-26, YAGNI); the README will state that the
    concurrency limit is changed in the configuration file and takes
    effect after a restart.
- **Out of scope**:
  - API behavior changes other than the `settings` decision;
  - metrics/Prometheus endpoint (a separate requirement, if any).

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-09-26
- **Decision notes**: Approved; the minimal action environment is
  anchored in NFR-01 (c), the `settings` endpoint is not introduced.
  Finding of the 2026-09-26 review (M-6, L-4, L-5, L-6, L-11, L-12,
  L-13, L-14).

## Proposed solution

- `internal/actions/queue.go`: `type Queue` with `Enqueue(ctx, job) error`,
  `Close(ctx) error`; a `Runner` interface defined on the consumer side.
- `internal/events/broker.go`: `Broker` with `Publish`, `Subscribe`, `Close`;
  `internal/server/status_events.go` keeps only the SSE writing.
- `cmd/marionette/main.go`: `store.OnStatusChange = broker.Publish`,
  `server.NewRouter(server.Dependencies{Store, Queue, Broker, Logger})`.
- `slog`: the logger is passed explicitly via `Dependencies`, no global
  state; tests use `slog.New(slog.NewTextHandler(io.Discard, nil))`.

## Test plan

- Existing tests pass after moving packages without a behavior change
  (imports only);
- test that `NewRouter` does not mutate the passed store;
- test of `desiredCards()` for cards without a status action and without
  polling;
- test of the minimal action environment: the `env` command returns only
  the expected keys;
- `go test -race ./...`, `golangci-lint run ./...`, `go vet ./...`.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md) plus:

- `internal/server` does not import `sync.WaitGroup` and holds no
  goroutines outside HTTP handlers;
- `log.Printf` does not occur in `cmd`/`internal`;
- `docs/architecture/overview.md` matches the new packages.

## Closure

- **Status after implementation**: Done (2026-09-26)
- **Verification**: `make verify` passed (golangci-lint, eslint, vue-tsc, prettier,
  `go test -race -count=1 ./...`, build, vet). Existing queue and broker
  tests moved without a behavior change to `internal/actions/queue_test.go`
  and `internal/events/broker_test.go` (imports and names only). New tests:
  `TestNewRouterDoesNotMutateStoreAndStreamsBrokerEvents` (the router
  does not set `Store.OnStatusChange`; after wiring in the composition
  the SSE event arrives), `TestSchedulerDesiredCardsRequireStatusActionAndPolling`,
  `TestSchedulerStopsWhenParentContextIsCanceled` (after the parent is
  canceled `Reconcile`/`Stop` return `ErrSchedulerStopped`, `Start` can
  be repeated), `TestActionEnvironmentIsMinimalAndOverridable`,
  `TestExecutorRunsActionWithMinimalEnvironment` (real `env`: only
  `PATH`/`HOME`/`LANG`/`TZ` + `Action.Env`, a service variable does not
  leak), `TestBrokerCloseIsIdempotentAndDropsSubscribers`, extended
  `TestLoadEnvironmentDefaultsAndShutdownTimeout` (`MARIONETTE_LOG_FORMAT`,
  `MARIONETTE_LOG_LEVEL`, JSON output with attributes). Criteria:
  `internal/server` does not import `sync` and starts no goroutines
  (verified with grep), `log.Printf` does not occur in `cmd`/`internal`,
  `git grep internal/exec"` is empty.
- **Deviations from the plan**: (1) the queue kept `EnqueuePrimary`/
  `EnqueueStatus` instead of a generic `Enqueue(ctx, job)` — deduplication
  needs the card and the kind, and the existing tests thus stayed
  unchanged; (2) `config.LoadFile(path, logger)` accepts the logger
  explicitly (nil = discard) instead of returning warnings as data;
  (3) after the parent context is canceled the scheduler switches to
  `running=false` via a watching goroutine (`watchParent`); `Stop()` then
  returns `ErrSchedulerStopped`, which shutdown already tolerates;
  (4) `cloneStatusSnapshot` and the unused `cloneHistory` removed
  (`StatusSnapshot` is a value type without references, documented at
  `OnStatusChange`); (5) `Dependencies.Logger` is optional (nil =
  discard) so that handler tests do not have to pass a logger;
  (6) `slog.DiscardHandler` is not in Go 1.23, so `TextHandler` over
  `io.Discard` is used; (7) on error `main` logs via `slog` and calls
  `os.Exit(1)` instead of `log.Fatalf`; (8) `MARIONETTE_LOG_LEVEL` is
  parsed via `slog.Level.UnmarshalText`, so it also accepts forms like
  `INFO+2`.
- **Documentation updated**: yes — AGENTS.md and README (package
  layout, logging variables, `settings` change only in the file +
  restart, minimal action environment), `docs/architecture/overview.md`
  (component table, "Composition and logging" section), ADR-0005
  (directory rename, minimal environment), ADR-0008 (`events` package),
  requirements FR-34 (logging variables), `deploy/marionette.default`,
  roadmap.
