# Implementation block: Orchestration of the primary and status action

- **Phase**: 5 — REST API
- **Requirements**: FR-11, FR-12, FR-13, FR-14, FR-15a, FR-17, FR-18, FR-40
- **ADRs**: ADR-0004, ADR-0005, ADR-0006
- **Status**: Done
- **Dependencies**: Block 0010; `config.Store`, `execengine.Runner`, `StatusCheckService`, `Scheduler`

## Goal

The REST API accepts a request for the primary or a manual status action and
returns a response to the client without waiting for the process to finish.
The run itself continues on a background worker; the primary result is
stored in the history, the status result updates the projection. Reading
the status stays a separate read-only operation.

## Scope

- **In scope**:
  - `POST /api/cards/{id}/actions/primary`;
  - `POST /api/cards/{id}/actions/status/check`;
  - the primary endpoint returns `202 Accepted` after accepting a valid
    request;
  - background orchestration `Runner.Run` → `Result.ToRun("primary")` → `AppendRun`;
  - calling `Scheduler.NotifyPrimaryAction` right after the primary action is
    accepted;
  - the status endpoint validates the status action and enqueues
    `StatusCheckService.CheckNow` on the background worker;
  - both enqueue endpoints return only an enqueue confirmation.
- **Out of scope**:
  - a new process execution rule;
  - changes to the status transition model;
  - a public async job resource, job polling, cancellation and
    prioritization;
  - scheduler reconcile and lifecycle (0012).

## Approval

- **Approved by**: pending
- **Approval date**: pending
- **Decision notes**: The primary request returns acceptance, not the process
  result. A failure is recorded after completion in the `Run` as `fail` or
  `timeout`; the card status is the authoritative way to find out the
  resulting state.

## Proposed solution

The handler receives interfaces for the background execution manager, the
status checker and the scheduler notifier. The primary endpoint validates the
card and the request, hands `card.Primary` to the background worker and
calls `NotifyPrimaryAction`. The worker then calls `Runner.Run`, converts
the result via `Result.ToRun("primary")` and stores it via `AppendRun`. The
background manager offers `Wait`/`Close`; wiring it into the application
lifecycle will be completed by block 0012.

The status is not run in this handler; the read-only status handler only
returns the last stored projection. The recommended response body of the
primary endpoint is only an acceptance confirmation with `cardId` and
`actionKind: "primary"`; the result is read from the history/status endpoint.

## Test plan

- the primary endpoint returns `202` without waiting for the process to
  finish;
- the background worker handles the outcomes `ok`, `fail` and `timeout`;
- storing the primary run in the newest-first history;
- notifying the scheduler before the primary action finishes;
  - status enqueue with `ok`/`fail`, a missing status action and a
    non-existent card;
  - a runner/setup error and a persistence error;
  - verifying that background runs do not bypass the scheduler or the shared
    runner;
  - verifying that `GET /status` only reads the last snapshot and runs
    nothing.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md). Specifically:
the primary endpoint does not wait for the process, every accepted run can
be drained before shutdown and no execution bypasses the shared
concurrency-limited runner.

## Closure

- **Status after implementation**: Done
- **Verification**: `go test ./internal/server -count=1`, `go test -race ./...`
- **Documentation updated**: yes; the lifecycle wiring stays in block 0012
