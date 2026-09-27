# Implementation block: Status projection and transition history

- **Phase**: 4 — Status/health-check engine
- **Requirements**: FR-12, FR-17, FR-20, FR-35
- **ADRs**: ADR-0006
- **Status**: Done

## Goal

After completion, the config layer can keep the last status check separate
from the history of status changes. A repeated check of the same status
does not grow the history; on a change, the previous status is closed and
its duration can be computed.

## Scope

- **In scope**:
  - `StatusState` (`unknown`, `ok`, `fail`) and `StatusChange` with the
    new state, `StartedAt`, closing/duration;
  - the last status projection including the last `Run` and the check
    time;
  - a thread-safe Store API for reading the projection and atomically
    processing a new check;
  - a separate limit for the transition history and trimming of the
    oldest changes;
  - deep copies of return values and JSON tags;
  - updating the persistence snapshot for the last status and
    transitions.
- **Out of scope**:
  - running the status command (0008);
  - timers and polling (0009);
  - HTTP API, dashboard and migration of the old `history.status` format.

## Proposed solution

Extend `internal/config` with `StatusState`, `StatusChange` and a projection
of the last check. On a new check the store compares the previous state; the
same state only replaces the last check, whereas a change closes the
previous interval and inserts a new open interval. The history returns
copies ordered from the newest change.

## Test plan

- the first check creates an initial change from `unknown`;
- repeated `ok`/`fail` checks do not create another change;
- `ok -> fail -> ok` closes the intervals with the correct duration;
- trimming the history keeps the last N transitions;
- concurrent checks of one card cause neither a race nor an inconsistent
  interval;
- JSON round-trip of the last projection and the change history.

## Done criteria

See Definition of Done. Specifically: `go test -race ./...` passes and the
tests prove that the number of status checks does not affect the number of
historical changes.
