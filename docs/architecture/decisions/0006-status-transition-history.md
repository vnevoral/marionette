# ADR-0006: Status history as state transitions

- **Status**: Accepted
- **Date**: 2026-09-25
- **Supersedes part of**: ADR-0004 concerning storing every status run in the history

## Context

The status action may run every 10–60 seconds, but with a stable device the
status does not change. Storing every check therefore fills the history
limit quickly and does not match what is useful to the operator: when the
status changed and how long the device stayed in it.

## Decision

- The primary action history remains a history of individual runs (`Run`),
  because every execution attempt is a significant event.
- The status action has two separate projections:
  1. the last check/status for the dashboard (`last status check`), which
     may be updated on every poll;
  2. the history of status transitions (`StatusChange`), to which a new
     record is added only on an actual status change.
- `StatusChange` contains the status, the start time and the end time or
  duration. The current open status has no end time yet when written; its
  current length is computed from `now - StartedAt`. On the next
  transition the previous record is closed.
- A repeated check that returns the same status does not change the
  transition history.
- The transition history has its own limit of the last `N` changes. This
  limit is not consumed by the number of polling checks.
- The `unknown` status is used at startup without a valid previous check or
  when information is lost; a transition to `ok`/`fail` is recorded the same
  way as the opposite transition.
- On graceful shutdown, the last status projection and the transition
  history are persisted. Individual repeated status checks are not
  persisted to the history.

## Consequences

- The status history stays readable even with frequent polling.
- The dashboard can show the current status and how long it has lasted.
- The status engine must atomically update the last check and a possible
  transition; this responsibility belongs to the phase 4 implementation
  block.
- The JSON schema of `history.status` changes from `[]Run` to
  `[]StatusChange`. Migration of older files with the original status run
  format is not part of the MVP; compatibility of this historical data
  will not be addressed separately.
