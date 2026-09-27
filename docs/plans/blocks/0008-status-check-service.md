# Implementation block: Status check service

- **Phase**: 4 — Status/health-check engine
- **Requirements**: FR-12, FR-14, FR-17, FR-19
- **ADRs**: ADR-0005, ADR-0006
- **Status**: Done

## Goal

When done, a single status action of a card can be executed
programmatically through the execution runner. The result is stored as the
last check and is reflected in the history only when it means a state
change.

## Scope

- **In scope**:
  - a service accepting a card or its status action and the card ID;
  - calling `execengine.Runner`;
  - mapping the result `ok`/`fail`/`timeout` to `StatusState`;
  - atomically handing the result to the status projection/store;
  - a manual/programmatic `CheckNow` API without the HTTP layer.
- **Out of scope**:
  - the scheduler and polling intervals (0009);
  - card CRUD and HTTP endpoints (phase 5);
  - storing every repeated check in the status history.

## Proposed solution

The new package `internal/status` gets the store and `execengine.Runner`
via the constructor. `CheckNow(cardID)` loads the card, rejects a card
without a status action, runs the status action and hands over the
`config.Run` as the last check. The store decides whether a `StatusChange`
is created.

## Test plan

- a status action with the result `ok`, `fail` and `timeout`;
- a missing status action and a non-existent card;
- repeated identical results update the last check, but not the history;
- a change of the result creates one transition with the corresponding time;
- an execution error does not terminate the service or the scheduler;
- fake runner/store and `go test -race`.

## Done criteria

See Definition of Done. Specifically: `CheckNow` is testable without a
real network or process and never writes a duplicate status change for the
same result.
