# Implementation block: Code review fixes — frontend (waiting for a check, dashboard polling, API)

- **Phase**: 8 — Hardening (addendum after the phase was closed)
- **Requirements**: FR-21, FR-22, FR-26, FR-42, NFR-03, NFR-11
- **ADRs**: ADR-0007
- **Status**: Done
- **Dependencies**: Blocks 0029, 0032, 0033

## Goal

When done, the frontend waits for a new check only when the backend actually
schedules it; the dashboard in polling mode does not discard the results of
other cards and loads states with a single request; errors for `args[N]` are
shown at the field; the request timeout also covers reading the body;
shared constants live in one place.

## Scope

- **In scope** (code review findings 2026-09-26):
  1. `useCardStatus.ts`: `expectsFollowUpCheck(card, action)` — status
     action always (the check is enqueued directly), primary action only
     when the card has a status action and `pollingIntervalSeconds`,
     `fastPollingIntervalSeconds` and `fastPollingWindowSeconds` are all
     > 0 (mirrors `Scheduler.NotifyPrimaryAction`); `waitBudgetMs(card)`
     returns the window in ms, default `defaultMaxWaitMs`. Without a check,
     both `HomeView` and `CardDetailView` report **Accepted** immediately
     and release the buttons.
  2. `cardEditModel.ts`: paths `primary.args[N]`/`status.args[N]` are
     mapped to the error of the Arguments block.
  3. `api.ts`: the `REQUEST_TIMEOUT_MS` timeout applies until the response
     body has been read.
  4. `HomeView.vue`: instead of the global `statusRefreshGeneration`, the
     refresh result is applied per card only when it carries a newer
     `checkedAt` (`isNewerCheck`), or when the card has no snapshot yet.
  5. `HomeView.vue`: `onRefresh` (polling mode) and the refresh after load
     use a single `listCards()` and its `currentStatus` instead of N ×
     `getStatus`; a `listCards` failure marks the monitored cards as
     `statusErrors`, the cards stay displayed.
  6. Remove duplicate constants: `resultVisibleMs` → `messageVisibleMs`
     from `useTransientMessage`, `defaultFastPollingWindowSeconds` → item 1.
  7. UX spec §4 and §6: waiting for a check only for cards with automatic
     checks; polling mode reads the card list.
- **Out of scope**:
  - backend changes (e.g. enqueueing a status check after the primary
    action for cards without polling);
  - Toast / other presentation of results;
  - E2E tests.

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-09-26
- **Decision notes**: Approved as a state fix ("plan and fix all the
  findings"). The waiting rule is intentionally a frontend mirror of the
  backend; extending the 202 response with a "check scheduled" flag would
  be cleaner, but it is an API contract change outside the scope of the
  fix.

## Proposed solution

- `useCardStatus.ts`: `export function expectsFollowUpCheck(card: ActionCard,
  action: ActionKind): boolean`, `export function waitBudgetMs(card:
  ActionCard): number`.
- `HomeView.vue`: `refreshStatuses()` → `listCards()`; `applySnapshot(id,
  snapshot)` writes only a newer check; `onSnapshot` from
  `waitForNewerStatus` goes the same way; generations disappear.
- `api.ts`: `clearTimeout` after `response.json()`; an abort while reading
  the body is mapped to "Request timed out".
- `cardEditModel.ts`: `rest[0]` starting with `args` → key `args`.

## Test plan

- `useCardStatus.spec.ts`: `expectsFollowUpCheck` for a status action,
  primary with polling, primary without polling, card without a status
  action; `waitBudgetMs`.
- `HomeView.spec.ts`: primary action on a card without polling →
  **Accepted** immediately, buttons released, `getStatus` is not called;
  polling mode calls `listCards`, not `getStatus`; an older snapshot from a
  refresh does not overwrite a newer one from SSE.
- `CardDetailView.spec.ts`: primary action without polling → **Accepted**.
- `cardEditModel.spec.ts`: `primary.args[2]` → `primaryArgs`.
- `api.spec.ts`: a body that is not fully read within `REQUEST_TIMEOUT_MS`
  ends with "Request timed out".
- `make verify`.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md) plus:

- `grep -n "4000\|= 120" web/src/views/*.vue` finds no local constants;
- `HomeView.vue` does not contain `statusRefreshGeneration`.

## Closure

- **Status after implementation**: Done (2026-09-26)
- **Verification**: `make verify` passed; Vitest 77 → 85 tests. New tests:
  `useCardStatus.spec.ts` (`supersedes`, `expectsFollowUpCheck` for a
  status action / primary with polling / without polling / without a
  status action, `waitBudgetMs`), `HomeView.spec.ts` (Accepted immediately
  for a card without polling and without calling `getStatus`; polling mode
  calls only `listCards` and applies its `currentStatus`; a slow refresh
  does not overwrite a newer snapshot from the stream),
  `CardDetailView.spec.ts` (Accepted without waiting, buttons released),
  `cardEditModel.spec.ts` (`primary.args[2]` → `primaryArgs`, the second
  message is not overwritten), `api.spec.ts` (a body that is not fully read
  within `REQUEST_TIMEOUT_MS` ends with "Request timed out"). Criterion:
  `grep -n "4000\|= 120" web/src/views/*.vue` and
  `grep statusRefreshGeneration` find nothing; `HomeView.vue` has 229
  lines.
- **Deviations from the plan**: (1) instead of `isNewerCheck` (mere
  inequality of `checkedAt`), overwriting is decided by a new function
  `supersedes(next, current)` — it compares `checkedAt` times, so an older
  snapshot from a slow REST read never overwrites a newer one from the
  stream; an unchecked snapshot does not overwrite a checked one. SSE
  events go the same way. (2) `statusErrors` in `HomeView` was state that
  was written but never displayed (the template does not pass it to
  `ActionCard`); it was removed instead of being rewired to `listCards`.
  The dashboard therefore does not visualize a status read error in any
  way — the card keeps showing the last known state and `Last checked …`.
  If UX spec §4 ("marks its data as unknown") is to apply to the dashboard
  as well, that is a separate UX change (`ActionCard` prop), not a review
  fix. (3) After the initial `listCards`, a second round of `getStatus` is
  no longer sent — the list already carries `currentStatus`.
- **Documentation updated**: yes — UX spec §4 (waiting only when a check
  follows, polling mode reads the card list, a snapshot is not overwritten
  by an older one) and §6 (Action accepted without waiting), roadmap.
