# Implementation block: Application composition, reconcile and lifecycle

- **Phase**: 5 — REST API
- **Requirements**: FR-15, FR-15a, FR-17, FR-35, FR-40
- **ADRs**: ADR-0004, ADR-0006
- **Status**: Done
- **Dependencies**: Blocks 0010–0011; `cmd/marionette/main.go`, scheduler, persistence

## Goal

The application builds all services from a single configuration, starts the
scheduler before the HTTP server and live-recalculates polling workers on
CRUD changes. On graceful shutdown it first stops accepting new requests
and waits for accepted background primary and manual status actions, so
that their results are saved before exiting.

## Scope

- **In scope**:
  - construction of executor/runner/status service/scheduler/router in
    `main`;
  - a background execution manager for accepted primary and manual status
    actions;
  - starting the scheduler in the application context;
  - `Scheduler.Reconcile` for adding, removing and retiming workers;
  - calling reconcile after a successful card create/update/delete;
  - shutdown order: stop accepting HTTP, wait for background primary and
    status actions, stop the scheduler, shut down the HTTP server and call
    `SaveFileWithHistory`;
  - documenting the residual risk of unbounded waiting in
    `Scheduler.Stop`.
- **Out of scope**:
  - context-aware cancellation of the queued runner;
  - a hard shutdown deadline;
  - a settings endpoint and dynamic change of runner concurrency;
  - deployment/systemd.

## Approval

- **Approved by**: pending
- **Approval date**: pending
- **Decision notes**: The current `Scheduler.Stop` remains; it may wait longer than the five-second HTTP shutdown timeout.

## Proposed solution

`main` loads the store, sets `OnChange`, creates a shared executor and
runner, the background execution manager, the status check service, the
scheduler and the server router with dependency injection. The background
manager tracks all accepted primary and manual status actions and on
shutdown lets them finish or closes them correctly according to the chosen
execution contract.
The scheduler gets a reconcile operation that is race-safe with respect to
`Start`/`Stop` and restarts the affected worker when a card changes. The
CRUD handler calls it only after a successful store mutation.

On SIGINT/SIGTERM, accepting new HTTP requests is stopped first and active
handlers complete, then the accepted background primary and status actions
are awaited, the scheduler is stopped and finally the configuration
including history is saved. A hard bounded shutdown remains outside this
block and is a possible future block.

## Test plan

- start/stop of all composed services in `main`;
- accepting a primary and a manual status action, completing the
  background run and persisting the corresponding projection/history;
- reconcile after create/update/delete and a change of polling
  configuration;
- concurrent reconcile, polling and shutdown under `go test -race`;
- verifying the shutdown order and saving of primary and status history;
- construction errors and clean exit without a partially wired server.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md). Specifically:
the API and the scheduler use the same instances of the store, runner and
status services, and graceful shutdown does not lose history after the
workers finish.

## Closure

- **Status after implementation**: Done
- **Verification**: `go build ./...`, `go vet ./...`, `go test ./...`,
  `go test -race ./...`, `git diff --check`
- **Documentation updated**: yes; architecture and roadmap synchronized
