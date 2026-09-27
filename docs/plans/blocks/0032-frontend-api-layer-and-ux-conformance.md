# Implementation block: Frontend API layer and conformance with the UX specification

- **Phase**: 8 — Hardening
- **Requirements**: FR-26, FR-27, FR-28, NFR-08, NFR-09, NFR-10
- **ADRs**: ADR-0003, ADR-0007
- **Status**: Done
- **Dependencies**: Block 0030 (tests), 0028 (`fields` contract for 422)

## Goal

When done, the frontend handles network and server errors uniformly and
clearly, confirms destructive actions with a PrimeVue dialog, the action
form is fully accessible and the connection indicator reflects the actual
state.

## Scope

- **In scope**:
  - `api.ts`: fix the options spread (preserve `Accept`), `ApiError` class
    (`status`, `message`, `fields`), map `TypeError` to "Server
    unreachable", check `content-type` before `json()`, `AbortSignal`
    with a 15 s timeout, `AcceptedAction` type; move `CARD_ICON_OPTIONS` to
    `src/ui/icons.ts`; document the `duration` field as nanoseconds
    or rename it to `durationNs` after agreement with the backend (contract);
  - `ConfirmationService` + `<ConfirmDialog>` in `App.vue`; `useConfirm` for
    deleting a card (names the card, action "Delete card") and for leaving
    an unsaved form (`UnsavedChangesDialog` per spec §8.4); the delete
    button is disabled while deleting;
  - `ActionEditor.vue`: `Select` with human labels for output rules,
    `aria-label` for arguments and env, a row removal button, PrimeVue
    `Button` instead of the native `<button>`, stable row keys, a single
    path for add/remove (own child);
  - `CardEditView`: show `fields` from 422 at the specific fields, sticky
    save bar, after saving stay on the page with "Card saved" or navigate
    with a toast (UX spec §7.3 decides);
  - `AppShell`: `ConnectionStatus` bound to `useStatusEvents().connected`
    (from 0029);
  - `CardDetailView`: distinguish 404 ("Card not found") from other errors
    ("Try again"), remove the dead `sectionErrors.status`, clear transient
    messages after a while; backend `checkedAt` as `omitempty` instead of
    checking for `0001-01-01` (small type change in `config`);
  - `index.html` `lang="en"`; fix the invalid `margin: -var(...)`.
- **Out of scope**:
  - vocabulary and shared components (0033);
  - Toast for all messages (consider in 0033);
  - new screens.

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-09-26
- **Decision notes**: Approved as a bug fix. Review finding
  2026-09-26 (M1, M3, M7, M8, L1,
  L2, L3, L4). UX spec §2.1 and §7.1 contradict each other on the role of
  the dashboard vs. "Manage cards" — clarify in the spec before approval.

## Proposed solution

- `web/src/api.ts`: `export class ApiError extends Error { status; fields? }`;
  `request<T>` with `const { headers, signal, ...rest } = options ?? {}`.
- `web/src/App.vue`: `<ConfirmDialog />` + `<Toast />`; `main.ts`
  `app.use(ConfirmationService).use(ToastService)`.
- `web/src/components/ActionEditor.vue`: `defineModel<ActionDraft>()`,
  internal `rows: {id, key, value}[]` with `crypto.randomUUID()`.
- Backend: `StatusSnapshot.CheckedAt *time.Time` or `omitempty` with
  `IsZero` — a one-line change + serialization test.

## Test plan

- Vitest: `ApiError` mapping (network, 4xx JSON, 5xx HTML, 204, timeout);
  `ActionEditor` adding/removing a row, emitted values, `aria-label`
  present; `CardEditView` shows a `fields` error at the correct input.
- Manual a11y pass: using the keyboard, create a card with 2 arguments and
  1 env, delete a card via the dialog, leave a half-filled form; verify
  screen-reader names (VoiceOver/NVDA or axe DevTools).
- `npm run lint`, `npm run build`, `npm test`; Go serialization test.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md) plus:

- `window.confirm` does not occur in `web/src`;
- every input in `ActionEditor` has an accessible name (test);
- the connection indicator changes state when the backend stops (manual
  test).

## Closure

- **Status after implementation**: Done (2026-09-26)
- **Verification**: `make verify` passed (lint, `vue-tsc`, Prettier, Vitest 62
  tests in 10 files, `vite build`, `go test -race`, `go vet`). New and
  extended tests: `api.spec.ts` (`ApiError` with `status` and `fields` for 422,
  404 → `isNotFound`, 2xx without JSON, network error → "Server unreachable"
  with `status` 0, 15 s timeout with fake timers), `cardEditModel.spec.ts`
  (`fieldErrorsFromServer`: mapping `primary.command`, `status.env.HOME`,
  `rule.pattern`… to editor keys, unmapped paths into the summary; row ids),
  `components/ActionEditor.spec.ts` (adding/removing rows inside the
  component, stable ids, emitted values, human rule labels,
  error messages, **every input and button has an accessible name**, Select
  has `aria-labelledby`), `components/ConnectionStatus.spec.ts` (Live /
  Reconnecting depending on the stream, keeps the stream open),
  `views/CardEditView.spec.ts` (422 `fields` at specific fields + summary
  at the top + preserved values, confirmation when leaving a half-filled
  form via `useConfirm` with reject/accept, a clean form does not ask, a
  new card stays in the edit context after saving without another GET),
  `views/CardDetailView.spec.ts` (404 "Card not found" vs. other error "Card
  unavailable" + "Try again", deletion only after confirmation in a dialog
  that names the card; the error message disappears after 4 s). Go: `TestStatusSnapshotJSONOmitsCheckForUncheckedCard`.
  Smoke test on the real binary server: `GET …/status` of an unchecked card
  returns `{"state":"unknown"}`, after a check the full snapshot; 422 carries
  `fields`; the SPA has `lang="en"`. `window.confirm` does not occur in
  `web/src`. The manual a11y pass (screen reader/axe) and the indicator test
  when the backend stops remain for the reference host.
- **Deviations from the plan**: (1) The request timeout is implemented with
  a custom `AbortController` + `setTimeout` (not `AbortSignal.timeout`), so
  that it is testable with fake timers and distinguishable from a caller
  abort. (2) `ToastService`/`<Toast>` is not introduced — UX spec §7.3
  decided to "stay in the edit context with Card saved", so a toast has no
  use (consider in 0033). (3) After saving, a new card redirects to
  `/cards/:id/edit` (`router.replace`), so that a reload and further saves
  target the saved card; the form is not reloaded from the server (module
  variable `justSaved`, works even when the router recreates the
  component). (4) `ActionEditor` uses `defineModel` for the action,
  arguments and environment; rows get ids from the `newRowId()` counter
  instead of `crypto.randomUUID()` (deterministic in tests, no dependency
  on `crypto`). The row types `ArgumentRow` and `EnvironmentRow` live in
  `cardEditModel.ts`. (5) `StatusSnapshot` keeps `time.Time`; omitting
  `checkedAt`/`lastCheck` is handled by `MarshalJSON` (Go 1.23 has no
  `omitzero`), unmarshal is the default. The frontend type has
  `checkedAt?`/`lastCheck?`; `isNewerCheck` no longer knows about
  `0001-01-01`. (6) `duration` stays in nanoseconds (Go `time.Duration`) —
  documented in the TS types and in the API contract; renaming to
  `durationNs` would change the contract with no benefit for the only
  client. (7) `ConnectionStatus` is a standalone component in `AppShell`;
  it keeps the shared stream open for the entire lifetime of the app (one
  SSE connection for the whole SPA). (8) The contradiction between UX spec
  §2.1 and §7.1 was resolved in the spec: navigation has only **Overview**,
  cards are managed from the detail (§7.1). (9) `CARD_ICON_OPTIONS` is in
  `src/ui/icons.ts` together with `DEFAULT_CARD_ICON`. (10) The fake
  `useConfirm` for tests (`src/test/fakeConfirm.ts`) reads
  `PrimeVueConfirmSymbol` via a cast, because PrimeVue exports it only at
  runtime.
- **Documentation updated**: yes — UX spec §2.1 (navigation, connection
  indicator), `docs/architecture/overview.md` (API contract: `checkedAt`/
  `lastCheck` omitted for an unchecked card, `duration` in ns),
  `docs/devops/testing-strategy.md` (new tests and the fake for
  `useConfirm`), roadmap.
