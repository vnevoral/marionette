# Implementation block: SSE event after a primary action run is recorded

- **Phase**: 8 — Hardening (addendum after the phase was closed)
- **Requirements**: FR-42a, FR-42, FR-17, FR-15, FR-27
- **ADRs**: ADR-0008, addendum "2026-09-27: `run.recorded` event"
  (accepted 2026-09-27)
- **Status**: Done
- **Dependencies**: Blocks 0034 (`internal/events`), 0050 (waiting for a
  run in the detail)

## Goal

Running an action is asynchronous: `POST …/actions/primary` only enqueues
the action and returns `202`. Today the client does not learn when the
process finishes and the run is written to the history. Since block 0050
the detail therefore repeatedly loads `/runs` at the fast polling
interval until it sees the new run. After this block the server sends an
event over the existing SSE stream once the run is recorded, and the
detail shows the run immediately, without repeated requests.

Flow after the block:

1. `POST` → `202` (action in the queue) — unchanged;
2. the worker picks up the queue, the process runs (at most until the
   timeout) — unchanged;
3. the worker records the run (`Store.AppendRun`) → **new**
   `run.recorded` event with the card and the recorded run;
4. a detail waiting for the run adds it to **Recent runs** and stops
   waiting.

## Scope

- **In scope**:
  - `internal/events`: the broker broadcasts two kinds of events (status
    change and recorded run). The existing `status.changed` behavior does
    not change. The broker still never blocks the producer. A slow
    subscriber loses the oldest events, which for runs means the client
    may not get notified about a run. The fallback mechanism below covers
    this;
  - `internal/config`: a `Store.OnRunAppended` hook (analogous to
    `OnStatusChange`), called after a successful `AppendRun` outside the
    store lock; wired to the broker in `cmd/marionette` (composition root,
    ADR-0008 addendum from block 0034);
  - `internal/server` (`/api/events`): the event
    `event: run.recorded`, `data: {"cardId": "…", "run": {…}}`, where `run`
    has the same shape as an item of `GET /api/cards/{id}/runs`;
  - applies only to the **primary action**. Status checks already have
    `status.changed` (only when the status changes) and the detail loads
    the latest check based on it;
  - `web/src/api.ts` + `useStatusEvents`: a `run.recorded` listener
    (shape validation as for `status.changed`);
  - `useCardActivity.waitForNewRun`: in addition to requests it reacts to
    the event for its card. A run newer than the starting point ends it
    immediately with `found` and is put at the top of the list. Periodic
    requests remain only as a fallback:
    - when SSE is not connected (`EventSource` unavailable or
      disconnected), requests run as today;
    - when SSE is connected, the first request comes only after the action
      time (`timeoutSec`) has elapsed and then at today's interval up to
      the budget. This covers a lost event as well as a run recorded
      during a connection outage;
    - after an SSE reconnect the runs are loaded once over REST (ADR-0008:
      reconnect synchronization over REST);
  - **dashboard** (`HomeView`, `ActionCard`): `run.recorded` for a card on
    the dashboard, whoever ran the action, is reflected in the card's note
    line (`ActionNote`, block 0049), because the badge does not show the
    run result:
    - a failure stays visible until another action or check is run on
      the card, a newer run arrives or the page is reloaded: **Action
      failed · exit 2**, **Action timed out**, **Action canceled**. The
      operator must not miss a run error, so it does not disappear after
      4 s like other results;
    - a success on a card **without a status action** briefly shows
      **Action finished** (replaces **Accepted**); on a card with a status
      action the success is only announced to the screen reader, because
      the badge shows the status after the check;
    - the card height is not affected (the line is reserved, block 0049);
  - UX specification §4 and §5.2 (card note), §6 (Recent runs), ADR-0008 addendum, the event table
    in `docs/architecture/overview.md`;
  - tests: Go (the hook is called only after a successful write and with
    a copy of the run; the broker delivers both kinds; the SSE handler
    writes `run.recorded` in the correct format; queue → store → broker
    in an integration test with a fake runner), Vitest (the event ends
    waiting without another `getRuns`; an event for another card or an
    older run is ignored; without SSE requests work as today; lost event
    → the fallback request finds the run), E2E (`card-lifecycle`: the run
    appears in the detail sooner than the fallback interval; on the
    dashboard a card with a failing action shows **Action failed ·
    exit N** and a card without a status action **Action finished**;
    existing scenarios unchanged). Vitest `HomeView`: a failure lasts
    until the next action, a success without a status action is brief,
    with a status action it is silent.
- **Out of scope**:
  - an event when the action is enqueued or started (`queued`,
    `started`): the UI currently shows the request state based on `202`
    and the status badge; add it once a real "running" state from the
    worker is needed;
  - event history and `Last-Event-ID` replay (ADR-0008: MVP without an
    event log).

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-09-27
- **Decision notes**: Approved by the owner including the run result on the dashboard, which was originally out of scope ("include everything in 0058 right away"). ADR-0008 addendum accepted.

## Proposed solution

The name is `run.recorded`, not `action.finished`: the event is created
once the run is recorded in the history, so the client can show it right
away and does not need to load anything more. A run that is not recorded
(an action interrupted when the service shuts down, a runner error)
creates no event, which is why the waiting time budget from block 0050
remains.

Fallback requests with SSE connected start only after `timeoutSec`.
Asking earlier makes no sense, because by then the event would have
arrived if the run had finished. On weak hardware this reduces requests
while the action is running, which is the main benefit of the block
besides faster display.

## Test plan

- `go test -race ./internal/...`.
- `cd web && npx vitest run`.
- `make e2e` and `npx playwright test e2e/card-lifecycle.e2e.ts
  --repeat-each 5` (stability as in block 0050).
- `make verify`.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md) plus:

- with SSE connected, a new run appears in the detail within 1 s of being
  recorded on the server without repeated requests to `/runs` while the
  action is running;
- without SSE or with a lost event, the run appears at the latest via the
  fallback request, as after block 0050.

## Closure

- **Status after implementation**: Done (2026-09-27)
- **Verification**: `make verify` passed (202 Vitest tests,
  `go test -race`, lint, `install_test`, `release_version_test`);
  `make e2e` passed (20 scenarios); `npx playwright test
  e2e/card-lifecycle.e2e.ts --repeat-each 5` passed (41/41).
  New tests: Go `TestStoreRunAppendedCallbackRunsOutsideTheLock`
  (hook after a successful write, reading the store from the hook without
  a deadlock, no hook for an unknown card),
  `TestBrokerPublishesRunsInTheSameSequence` (shared `id` sequence, copy
  of the run), `TestNewRouterDoesNotMutateStoreAndStreamsBrokerEvents`
  extended with `run.recorded` (`id`/`event`/`data` format),
  `TestFinishedPrimaryActionIsPublishedAsRecordedRun` (queue → store →
  broker); Vitest `api.spec.ts` (parsing `run.recorded`, discarding
  malformed ones), `useStatusEvents.spec.ts` (dispatching `onRun`),
  `CardDetailView.spec.ts` (the event ends waiting without requests while
  the action runs; an event for another card is ignored and the fallback
  reads after the action timeout; a stream outage postpones the first
  request; a reconnect loads the runs; a run from another device is
  added), `HomeView.spec.ts` (a failure lasts until the next action;
  timeout and canceled; success briefly, silent only with a follow-up
  check; a run recorded before the `202` response does not overwrite
  **Accepted**; unknown card); E2E "reports a failed run on the dashboard
  and shows the run in the detail at once" (Action failed · exit 3 even
  after 4.5 s; the run in the detail within 3 s, i.e. sooner than the
  first fallback request at 5 s).
- **Deviations from the plan**:
  - after the event the detail does not insert the run into the list
    itself but loads `/runs` once: the list thus matches the server
    exactly, including trimming to N runs, while repeated requests during
    the action run are gone;
  - without a starting point (the list was loading or failed) waiting is
    ended by any `run.recorded` of the card, because a REST read cannot
    recognize a new run;
  - dashboard: success is "silent" only where a check really follows the
    action (`expectsFollowUpCheck`), not on every card with a status
    action. A card with a status action but without automatic checks
    would otherwise show nothing about the result (a finding from E2E);
  - dashboard: a run recorded even before the `202` response (fast
    commands) takes precedence over the request result
    (**Accepted**/**Updated**);
  - clarification of FR-42a and ADR-0008: an action canceled when the
    service shuts down before it got a slot to run is not recorded, nor
    is the run of a card deleted during the run. A run interrupted while
    running is recorded as `canceled`;
  - `StatusListener.onStatus` is now optional (a listener may subscribe
    only to runs or only to the connection state).
- **Documentation updated**: ADR-0008 (addendum 2026-09-27),
  requirements FR-42a, UX specification §4 and §6,
  `docs/architecture/overview.md` (SSE event table, composition, broker).
- **Fixes after code review (2026-09-27)**:
  - the dashboard took any run recorded during a pending request as the
    result of that request and discarded the request's own result. A run
    started elsewhere thus hid **Result not available yet** during
    **Check status** and, on an enqueue error, hid the error message too.
    A run is now attributed only to a pending **primary** action. An
    enqueue error is always shown. **Result not available yet** gives way
    only to a failed run (`keepsRunNote`). New tests in `HomeView.spec.ts`
    (status check with a foreign run, enqueue error with a recorded run,
    failed run versus waiting timeout); the first two fail on the
    original logic;
  - detail: when the event loaded a new run even before the `202`
    response, waiting needlessly kept requesting up to the budget. It now
    checks the already loaded list at the start and ends immediately
    (test in `CardDetailView.spec.ts`);
  - verification: `make verify` (206 Vitest tests) and `make e2e`
    (20 scenarios) passed.
