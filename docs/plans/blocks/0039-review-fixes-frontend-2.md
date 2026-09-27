# Implementation block: Code review 2 fixes — frontend (baseline from 202, shared action progress, snapshot guard)

- **Phase**: 8 — Hardening (addendum after the phase was closed)
- **Requirements**: FR-21, FR-22, FR-26, FR-42, NFR-08
- **ADRs**: ADR-0003, ADR-0007
- **Status**: Done
- **Dependencies**: Blocks 0029, 0033, 0037

## Goal

After completion, waiting for a new check starts from the moment the server
accepted the request, the action progress (queued → enqueue → running →
wait → result) exists in the code only once, the snapshot on the detail
does not overwrite a newer one with an older one, the waiting budget for a
manual check matches the status action's timeout, and the "New card" /
"Edit card" links are PrimeVue buttons.

## Scope

- **In scope** (code review findings 2026-09-26, second round):
  1. API: the 202 response from `POST …/actions/primary|status/check`
     carries the `checkedAt` of the last known check at the moment of
     acceptance (the server reads it before enqueueing);
     `AcceptedAction.checkedAt?: string`. The frontend uses this value as
     the baseline for `waitForNewerStatus` instead of the value captured
     before the request.
  2. New composable `useActionRequest.ts`: `requestAction(card, action,
     { signal, onPhase, onSnapshot, onError }) → ActionOutcome`
     (`accepted | updated | timeout | aborted | failed(message)`) and
     `outcomeResult(outcome, labels)`; both `HomeView` and
     `CardDetailView` use it, their own `runAction` only maps the state
     and the vocabulary.
  3. `useCardStatus`: `apply`/`set` respect `supersedes`; `set(undefined)`
     still clears; `waitForNewer` removed from `CardStatus` (progress is
     handled by `requestAction`).
  4. `waitBudgetMs(card, action)`: status action → `(status.timeoutSec +
     30) × 1000` (reserve for the queue), primary → fast window or
     `defaultMaxWaitMs`.
  5. `.primary-action-link` removed from `tokens.css`; "New card" and
     "Edit card" are `<Button asChild>` with a `RouterLink` inside and the
     class `primary-action-button` (AGENTS.md: PrimeVue components instead
     of custom UI elements).
  6. UX spec §4: baseline from request acceptance; waiting budget for a
     manual check.
- **Out of scope**: Toast, E2E, further refactoring of views.

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-09-26
- **Decision notes**: Approved ("fix all findings"). Extending the 202
  with `checkedAt` is the only API contract change; it is backward
  compatible (the field is optional, a client without it behaves as
  before).

## Proposed solution

- `server.go`: `acceptedAction.CheckedAt *time.Time
  `json:"checkedAt,omitempty"`` from `store.GetStatus(cardID)` before
  enqueue.
- `useActionRequest.ts` (see scope); `HomeView.runAction` ~15 lines,
  `CardDetailView.runAction` ~20 lines.
- `useCardStatus.ts`: `apply(id, next)` → `if (supersedes(next,
  snapshot.value ?? undefined))`; `set(next)` → `undefined` clears,
  otherwise like `apply`.

## Test plan

- `useActionRequest.spec.ts`: accepted without a check; updated via the
  stream; timeout; failed (enqueue reject); aborted; baseline from 202 —
  an SSE event with the same `checkedAt` as in the 202 does not resolve
  the wait, a newer one does.
- `useCardStatus.spec.ts`: `set`/stream does not overwrite a newer
  snapshot with an older one.
- `HomeView.spec.ts`, `CardDetailView.spec.ts`: existing scenarios +
  baseline from 202 (H1/H2 mock returns `checkedAt`).
- `server_test.go`: 202 carries `checkedAt` after `UpdateStatus`, without
  it the field is absent.
- `make verify`.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md) plus:

- `grep -rn 'primary-action-link' web/src` is empty;
- `grep -c 'waitForNewerStatus' web/src/views/*.vue` = 0.

## Closure

- **Status after implementation**: Done (2026-09-26)
- **Verification**: `make verify` passed; Vitest 85 → 93 tests. New:
  `useActionRequest.spec.ts` (accepted without waiting; baseline from
  202 — an event with the same or an older `checkedAt` does not resolve
  the wait, a newer one does; a manual check expires after
  `timeoutSec + 30 s`; failed/aborted; `outcomeResult`),
  `useCardStatus.spec.ts` (`set`/`apply`/stream do not overwrite a newer
  snapshot, `fail` only for the watched card, `waitBudgetMs` per action,
  `isNewerCheck` rejects an older check), `HomeView.spec.ts` (a check
  known to the server at acceptance is not the result),
  `CardDetailView.spec.ts` (a loaded card with an older check does not
  overwrite a newer event). `server_test.go`: 202 without `checkedAt` for
  an unchecked card, with `checkedAt` after `UpdateStatus`. Criteria:
  `grep primary-action-link web/src` = 0, `waitForNewerStatus` in views
  = 0; `HomeView.vue` 206 lines, `CardDetailView.vue` 279.
- **Deviations from the plan**: (1) `isNewerCheck` compares the order of
  times (`>`) instead of mere inequality — without it, an event with a
  `checkedAt` older than the baseline from the 202 would end the wait; a
  test revealed this. (2) Instead of `waitForNewer`, `CardStatus` exposes
  `apply(id, snapshot)` and `fail(id)`, so that the action progress in
  `requestAction` writes only for the card it was started for, even after
  a route change. (3) The manual check budget has no lower bound of
  `defaultMaxWaitMs`: with `timeoutSec` 5 s the wait is 35 s, which
  matches "the result did not arrive" more accurately than 120 s.
- **Documentation updated**: yes — UX spec §4 (baseline from 202, waiting
  budgets), the `acceptedAction` comment in `server.go` and
  `AcceptedAction` in `api.ts`, roadmap.
