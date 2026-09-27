# Implementation block: Action state on the dashboard and shared status tracking

- **Phase**: 8 — Hardening
- **Requirements**: FR-20, FR-21, FR-27, FR-42, NFR-03, NFR-11
- **ADRs**: ADR-0007, ADR-0008
- **Status**: Done
- **Dependencies**: Blocks 0013, 0016, 0018, 0022

## Goal

After this block, card buttons are released again once an action
finishes, the card detail updates after running an action just like the
dashboard, and the SSE/polling logic lives in one shared composable
instead of inside a single view.

## Scope

- **In scope**:
  - `HomeView`: the `requests[card.id]` entry is cleared after
    `success`/`error` (the result stays visible for a short time via a
    separate `lastResult` state); `disabled`/`loading` derived only from
    `queued|running`;
  - new composable `useStatusEvents()` — a singleton `EventSource` shared
    between views, `connected` state, reconnect, REST fallback polling,
    proper `close()` after the last subscriber;
  - new composable `useCardStatus(cardId)` — per-card snapshot, waiting
    for a new `checkedAt` with an `AbortController`, cancellation on
    unmount;
  - `CardDetailView`: after the action is accepted (202) it uses
    `useCardStatus`; after a status change it refetches runs + history;
    buttons blocked during the run; "Action queued" is replaced by the
    result once finished;
  - `CardDetailView` and `CardEditView`: `watch(() => route.params.id, …,
    { immediate: true })`, normalization of a `string[]` parameter,
    ignoring stale responses after an id change;
  - `AcceptedAction` type for the enqueue endpoints' response in `api.ts`.
- **Out of scope**:
  - visual changes to cards, status vocabulary (block 0033);
  - replacing `window.confirm` (block 0032);
  - test infrastructure (block 0030) — tests are written in this block,
    though, so 0030 must be done first.

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-09-26
- **Decision notes**: Approved as a bug fix. Finding of the 2026-09-26 review (H1, H2, M2, M9).
  Verified in code: after the first action `HomeView.vue` leaves the
  buttons permanently disabled and the spinner runs until a remount.

## Proposed solution

- `web/src/composables/useStatusEvents.ts`: a module variable with the
  `EventSource` and a subscriber counter; `subscribe(handler)` returns
  `unsubscribe`; on `error` it switches `mode = "polling"` and starts a
  5 s interval; on `open` it refetches all statuses and stops the interval.
- `web/src/composables/useCardStatus.ts`: `ref<StatusSnapshot|null>`,
  `waitForNewer(previousCheckedAt, { signal, maxWaitMs })` — primarily
  waits for an SSE event, with a REST fallback every 2 s only in polling
  mode.
- `HomeView.vue`: `requests` type narrowed to `queued|running`; `lastResult`
  with a 4 s `setTimeout` for showing "Updated"/"Result not available yet".
- `CardDetailView.vue`: `runAction` → `enqueue*` → `waitForNewer` →
  `Promise.allSettled([loadRuns(), loadHistory()])`.

## Test plan

- Vitest + Vue Test Utils (infrastructure from 0030):
  - `useStatusEvents`: mock `EventSource`; two subscribers share one
    connection; it closes after both unsubscribe; `error` → polling;
    `open` → stop polling;
  - `useCardStatus.waitForNewer`: returns after an event with a newer
    `checkedAt`; `abort` ends waiting without writing; the same
    `checkedAt` is not considered new;
  - `HomeView.runAction`: after success the buttons are enabled and
    without a spinner (H1 regression test); enqueue error → enabled +
    error message;
  - `CardDetailView`: after an action runs/history are refetched; a
    change of `route.params.id` reloads the detail.
- `npm run lint`, `npm run build`, `npm test`.
- Manual smoke test: run → check → run on the same card without a reload;
  the card detail shows the new run without F5.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md) plus:

- there is no direct `new EventSource` in `web/src/views`;
- the H1 and H2 regression tests pass in CI;
- UX spec §4 and §6 updated if the behavior was refined.

## Closure

- **Status after implementation**: Done (2026-09-26)
- **Verification**: `make verify` passed (lint, `vue-tsc`, Prettier, Vitest 45
  tests in 7 files, `vite build`, `go test -race`, `go vet`). New tests:
  `composables/useStatusEvents.spec.ts` (two subscribers share one
  connection, close after the last unsubscribe, polling until `open`,
  `open` → one-off refresh and stop polling, `error` → polling, fallback
  without `EventSource`), `composables/useCardStatus.spec.ts`
  (`isNewerCheck`, `waitForNewerStatus`: event with a newer `checkedAt`,
  REST poll, REST error without ending the wait, `timeout`, `abort`
  without writing; `useCardStatus`: filter by card, release with the
  scope, refresh only when `enabled`, ignoring a stale response after an
  id change), `views/HomeView.spec.ts` (H1 regression: after the action
  finishes the buttons are enabled and without a spinner, the result
  disappears after 4 s, enqueue error → enabled + message, card without a
  status action → "Accepted", REST polling only while the stream is not
  running), `views/CardDetailView.spec.ts` (H2 regression: buttons
  blocked until a new `checkedAt`, then refetch runs + history and
  "Status updated"; enqueue error; a change of `route.params.id` reloads
  the detail and a stale response is ignored; card without a status
  action; card not found). The H1 regression tests were verified to fail
  on the original `HomeView.vue`. There is no direct `new EventSource` in
  `web/src/views` (the only one is in `api.ts`). A manual smoke test in
  the browser (run → check → run without a reload) remains to be verified
  on the reference host.
- **Deviations from the plan**: (1) `waitForNewer` queries REST every 2 s
  **always**, not only in polling mode — the backend publishes
  `status.changed` only when the status changes (`Store.UpdateStatus`),
  so a check with the same result changes `checkedAt` without an event;
  the SSE event only shortens the wait. (2) `waitForNewer` returns
  `"updated" | "timeout" | "aborted"` instead of a boolean, so the view
  writes nothing after an abort (unmount, id change). (3) A testable
  factory `createStatusEvents(connect, intervalMs)` was added; the
  singleton `useStatusEvents()` builds on it, and a subscription
  registered inside a component scope is released via `onScopeDispose`.
  The subscriber also receives `onRefresh` (polling tick / reconnect) so
  the view decides what to reload. (4) The mode is derived from
  `connected` (`live`/`polling`); polling runs even before the first
  `open`, just like the former `HomeView`. (5) `HomeView` waits with its
  own snapshot map via `waitForNewerStatus` (a function shared with
  `useCardStatus`), because one composable per card would not make sense
  in a list. (6) For a card without a status action `CardDetailView`
  reports "Action accepted" and loads runs/history right after the 202 —
  without a status action there is no completion signal. (7) Helper
  `router/params.ts::singleParam` normalizes `string | string[]`;
  `CardEditView` watches `[route.name, id]` because the same component
  serves both `new` and `edit`. (8) `AcceptedAction` is the return type
  of `enqueuePrimary`/`enqueueStatus`. (9) The shared fake `EventSource`
  for tests is in `web/src/test/fakeEventSource.ts`.
- **Documentation updated**: yes — UX spec §4 (result after completion,
  releasing buttons, SSE + REST fallback) and §6 (Actions in the detail),
  `docs/devops/testing-strategy.md` (composables and view tests),
  `docs/architecture/overview.md` (SPA row), roadmap.
