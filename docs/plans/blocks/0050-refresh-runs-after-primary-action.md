# Implementation block: A primary action run appears in the detail without a page reload

- **Phase**: 8 — Hardening (addendum after the phase was closed)
- **Requirements**: FR-21, FR-17, FR-27
- **ADRs**: ADR-0008 (SSE does not change); no new ADR is needed
- **Status**: Done
- **Dependencies**: Blocks 0029, 0039 (action progress), 0049 (feedback
  in the Actions panel)

## Goal

After **Run action** in the card detail, the run often does not appear in
the **Recent runs** section. The detail loads runs once: right after
**Action accepted** (a card without a follow-up check), or as soon as a
newer status check is visible. At that moment, however, the primary action
is often still running or waiting in the queue, so the list stays unchanged
until the user reloads the page. The finding comes from a flaky E2E
scenario (block 0049, closure).

When the block is done, the detail waits for the new run and shows it as
soon as the server records it.

## Scope

- **In scope**:
  - `CardDetailView` / `useCardActivity`: after the primary action is
    accepted (`202`), runs are reloaded at the card's **fast polling
    interval** (`fastPollingIntervalSeconds`), and if the card does not
    have it set, every **2 s**, until a run appears that is
    newer than the newest run known before the click, or until the budget
    `primary.timeoutSec + 30 s` expires (queue reserve, the same as for a
    manual check, UX spec §4). The `startedAt` of the new run is compared
    with the `startedAt` of the newest run so far from the server, not with
    the browser clock (robust against time drift);
  - applies to a card without a follow-up check and also to a card where
    the check completes before the primary action finishes; the status
    history keeps loading according to the current rules;
  - waiting for the run does not block the buttons (they are released
    according to the current request state logic) and ends on leaving the
    page, changing the card or starting the action again;
  - an error reading runs during the wait is shown as today
    (**Run history is unavailable**) and the wait continues;
  - UX specification §6 (Actions, Recent runs);
  - tests: Vitest `CardDetailView` (the run appears only after several
    queries; the budget expires; leaving the page ends the wait) and the
    E2E scenario `creates, runs, checks and deletes a card` without
    flakiness.
- **Out of scope**:
  - a new SSE event for run completion (an ADR-0008 contract change;
    consider it only when a live history is needed on the dashboard too);
  - run history on the dashboard (the dashboard does not display it);
  - backend or API changes.

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-09-27
- **Decision notes**: The owner determined that the interval for querying
  runs is not a fixed 2 s but the card's fast polling interval
  (`fastPollingIntervalSeconds`); 2 s only as the default when the card
  does not have the parameter.

## Proposed solution

- `useCardActivity` gets `waitForNewRun({ after, signal, maxWaitMs })`:
  a `getRuns` loop with interval `intervalMs`, result `found | timeout |
  aborted`; on `found` it sets `runs`. `after` is the `startedAt` of the
  first (newest) run in `runs` before the request is sent (`undefined` =
  no run yet).
- `CardDetailView.runAction` for `primary`, after `202`, starts waiting for
  the run concurrently with waiting for the status (`requestAction`) and
  shares an `AbortController` with it, so that both `loadDetail` and
  `onBeforeUnmount` end it.
- Interval: `card.fastPollingIntervalSeconds * 1000`, otherwise the default
  2 s (`runPollIntervalMs(card)`); the queue reserve (30 s) is taken over
  from `useCardStatus` so that it is defined once.

## Test plan

- Vitest with fake timers: the first two `getRuns` return the old list,
  the third a new run → the run is visible and the queries stop; without a
  new run the queries stop after the budget; leaving the page and starting
  the action again cancel the previous wait; an error in one query shows
  the error and the wait continues.
- `make e2e` repeatedly (`--repeat-each 5` for `card-lifecycle`).
- `make verify`.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md) plus:

- after **Run action** in the detail, the new run appears in
  **Recent runs** at most one interval (the card's fast polling, default
  2 s) after it is recorded on the server, without a page reload.

## Closure

- **Status after implementation**: Done (2026-09-27)
- **Verification**: `make verify` passed (174 Vitest tests, `install_test`,
  `release_version_test`, golangci-lint, eslint, vue-tsc, prettier,
  `go test -race`); `make e2e` passed (17 scenarios);
  `npx playwright test e2e/card-lifecycle.e2e.ts --repeat-each 5` passed
  (30/30) — the flakiness from block 0049 did not recur.
- **Implementation**: `useCardActivity.waitForNewRun(card, previousStartedAt)`
  (queries at interval `runPollIntervalMs(card)`, deadline
  `runWaitBudgetMs(card)` as a separate timer, cancellation on a new wait,
  on `reset()` and on leaving the page), `requestAction` with an
  `onAccepted` callback, `CardDetailView` loads only the status history
  after the action result (runs are handled by the wait, so a slow older
  read does not overwrite the new list). New tests with fake timers in
  `CardDetailView.spec.ts` (2 s interval, 10 s fast polling interval,
  budget, read error, leaving the page).
- **Deviations from the plan**: the wait is owned by `useCardActivity`
  (not `CardDetailView`), so that the view stays under 300 lines; after a
  status check completes, only the history is reloaded, not the runs.
- **Documentation updated**: yes — UX specification §6, roadmap.
