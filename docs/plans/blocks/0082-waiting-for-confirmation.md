# Implementation block: Waiting for confirmation of a primary action

- **Phase**: maintenance and extension after v2.0.0
- **Requirements**: FR-15a (updated), FR-15b, FR-42
- **ADRs**: [ADR-0016](../../architecture/decisions/0016-waiting-for-confirmation.md)
- **Status**: Done
- **Dependencies**: blocks 0009 (polling scheduler), 0050 and 0058 (action feedback)

## Goal

After **Run action** the card waits for a status check to confirm the
action instead of showing Problem after the first failed check; failed
checks during the fast polling window are not recorded.

## Scope

- **In scope**:
  - `config`: the `waiting` state and `waitingUntil`, the wait in the
    store (begin, extend, end, the rule in `UpdateStatus`), not persisted;
  - `status`: the scheduler pauses checks while the primary action runs,
    starts the window when it finishes, times the last check to the end
    of the window and ends the wait for a card that stops polling;
  - `actions`: the queue reports every finished primary action;
  - web: the **Waiting** state (timer icon, blue), **Run action** ends with
    the request and is announced as **Waiting for confirmation**, the
    detail re-reads the status history when a wait ends;
  - requirements v0.15, ADR-0016, UX specification, architecture overview,
    README, glossary, roadmap.
- **Out of scope**:
  - waiting for cards without fast polling;
  - disabling **Run action** while the card waits (a repeated run opens a
    new window).

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-10-21
- **Decision notes**: The owner chose the name Waiting and the rule that
  failed checks are not recorded until a check confirms the action or the
  window ends.

## Test plan

- Store: a failed check while waiting is not recorded, a healthy one ends
  the wait, a check after the window closes is recorded, ending a wait
  with and without a check, nothing persisted.
- Scheduler: checks pause until the action finishes, the window starts
  then, the last check is due at its end, a failed action ends the wait,
  a card that stops polling stops waiting.
- Queue: the outcome of a finished primary action is reported.
- Vitest: the request flow and both views with the Waiting state.
- e2e: a confirmed card goes Waiting → Healthy; an unconfirmed card stays
  Waiting for the window, then Problem with no status change recorded.
- `make verify` and `make e2e` pass.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md).

## Closure

- **Status after implementation**: Done (2026-10-21)
- **Verification**: `make verify` and `make e2e` passed.
- **Deviations from the plan**: none.
- **Documentation updated**: requirements v0.15, ADR-0016, UX
  specification, architecture overview, README, glossary, roadmap.
