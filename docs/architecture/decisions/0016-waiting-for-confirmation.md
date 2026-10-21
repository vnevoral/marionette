# ADR-0016: Waiting for a status check to confirm a primary action

- **Status**: Accepted
- **Date**: 2026-10-21

## Context

After **Run action** the scheduler switches to fast polling (FR-15a). The
first fast check usually runs before the action has taken effect: a PC
woken by Wake-on-LAN needs a minute to answer `ping`. That check fails,
the card shows **Problem** and a transition to Problem is recorded in the
status history, although the action is still being confirmed. A few checks
later the card becomes Healthy. The dashboard showed **Running** only in
the browser that started the action, and only until that first check.

The owner asked (2026-10-21) for an intermediate state, named **Waiting**
since an action does not always start something, that lasts until a check
confirms the action or the fast polling window ends, with failed checks in
between neither shown nor recorded.

## Decision

- The status projection gets a fourth state, `waiting` (FR-15b), with a
  `waitingUntil` time. It is a runtime state of the projection: the store
  keeps the last confirmed snapshot separately, so `waiting` is never
  written to the status history or to the persisted file.
- When a primary action of a card with fast polling is accepted, the
  scheduler puts the card into `waiting` and pauses its checks until the
  action finishes, at most for the action's timeout, a 30 s queue margin
  and the fast polling window. The action queue reports every finished
  primary action to the scheduler.
- When the action has finished successfully, the fast polling window
  starts and the wait lasts until its end. A failed or timed-out action
  ends the wait at once and starts no fast polling: a timeout is an error,
  not a possible success.
- While the card waits, a check that is not healthy only updates the
  latest check (`checkedAt`, `lastCheck`), with no event. A healthy check,
  or a check that ends after the window has closed, ends the wait and is
  stored as usual. The scheduler times the last check to the end of the
  window; if it cannot run, the window is closed with the latest check.
- Entering and leaving `waiting` publish `status.changed`, so every client
  shows it; the UI shows **Waiting** (timer icon, blue) and no longer waits
  for a check in the browser after **Run action**.

## Considered alternatives

- **Only in the browser** (extend the pending **Running** until the window
  ends) — rejected: other devices and a reload would still show Problem,
  and the failed checks would still be recorded.
- **A `waiting` flag next to an unchanged `state`** — rejected: clients
  that show `state` would keep showing Problem during the wait.
- **Starting the window when the action is accepted** (as before) —
  rejected: a slow action or a long queue would use up the window before
  the action has done anything.

## Consequences

- API clients see a new `state` value, `waiting`, and the optional
  `waitingUntil`; this is an additive change (minor version).
- Checks no longer run concurrently with the card's own primary action.
- A card that never confirms shows Problem only after the window, so a
  real failure is reported up to one window later than before.
- Cards without fast polling behave as before.
