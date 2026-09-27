# Implementation block: Shared status vocabulary, components and theme preset

- **Phase**: 8 — Hardening
- **Requirements**: FR-25, FR-26, NFR-06, NFR-08
- **ADRs**: ADR-0003, ADR-0007, ADR-0009
- **Status**: Done
- **Dependencies**: Blocks 0029, 0032 (so the refactor does not collide with the fixes), 0030 (tests)

## Goal

When done, there is a single module with the vocabulary of states and
texts, shared types and components from spec §8.4, and the PrimeVue theme
is customized via `definePreset` instead of `!important` overrides.

## Scope

- **In scope**:
  - `src/ui/vocabulary.ts`: labels, `statusPresentation(state)` →
    `{label, icon, tone}`, texts for empty/error states; `StatusTone`
    defined once in `src/types.ts` together with `EnvironmentRow` and other
    shared types;
  - components `PageHeader`, `EmptyState`, `RequestState`, `ActionCard`,
    `RunTable`, `StatusTimeline` (spec §8.4); views shrink below ~250 lines;
  - global classes `.eyebrow`, `.page`, `.page-lede`, `.loading-state` in
    `tokens.css`, removing duplicates from scoped CSS;
  - `definePreset(Aura, …)` in `main.ts` for primary/formField/surface
    tokens, removing all `!important` in `tokens.css` and `:deep` overrides
    in `CardEditView.vue`; align tokens with spec §8.2 (or adjust the spec
    to reality, the owner decides);
  - distinguish the `Queued` (amber) and `Running` (blue) tones per spec §4;
    add "Last checked …/Not checked yet" and "state duration" (FR-17) where
    missing; line-clamp the description on the card;
  - remove the `env.d.ts` shim if `vue-tsc` works without it.
- **Out of scope**:
  - migration PrimeFlex → Tailwind (rejected for the MVP, see ADR-0009);
  - Pinia or another store (the size of the app does not require it);
  - new features.

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-09-26
- **Decision notes**: Approved; PrimeFlex stays per ADR-0009 and its
  version is pinned to `4.0.0` in this block. Review finding 2026-09-26
  (M4, M6, L4, L6).
  ADR-0007 explicitly requires a vocabulary; today `StatusTone` is defined
  3×.

## Proposed solution

- `src/ui/vocabulary.ts` exports `const STATUS = { ok: {label:"Healthy",
  icon:"pi pi-check-circle", tone:"success"}, … } satisfies Record<StatusState,
  Presentation>`; both `StatusBadge` and `ActionCard` read from it.
- `src/theme/preset.ts`: `definePreset(Aura, { semantic: { primary: {…},
  colorScheme: { light: { surface: {…}, formField: {…} } } } })`.
- Refactor one view at a time: `HomeView` → `ActionCard` + `EmptyState`;
  `CardDetailView` → `RunTable` + `StatusTimeline`; `CardEditView` →
  `PageHeader` + `RequestState`.

## Test plan

- Vitest: `statusPresentation` covers all `StatusState` values (type test
  `satisfies`); `ActionCard` render for each state; `StatusTimeline`
  duration calculation.
- `grep -r "!important" web/src` returns 0 lines; `grep -rn "type StatusTone"`
  returns 1 line.
- Manual screenshot smoke test of all three screens at 320 px and desktop
  (NFR-10), token contrast check (axe).
- `npm run lint`, `npm run build`, `npm test`.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md) plus:

- no `!important` in `web/src`;
- every view < 300 lines;
- UX spec §8.2/§8.4 matches the implementation.

## Closure

- **Status after implementation**: Done (2026-09-26)
- **Verification**: `make verify` passed (lint, `vue-tsc`, Prettier, Vitest 77
  tests in 14 files, `vite build`, `go test -race`, `go vet`). New tests:
  `ui/vocabulary.spec.ts` (every `StatusState` and `Run.outcome` has a
  label, icon and tone; the raw values `ok/fail/accepted/running/timeout`
  are never displayed; Queued = amber, Running = blue), `ui/format.spec.ts`
  (nanoseconds, rough span, "Last checked/Not checked yet",
  `transitionDuration` for both a finished and the current transition), `components/ActionCard.spec.ts`
  (render for ok/fail/unknown, without a status action, queued/running with
  blocked buttons, result, emits), `components/StatusTimeline.spec.ts`
  (duration of the current transition from `now`, stored duration for
  finished ones, loading/empty/error). Criteria: `grep -rn "!important"
  web/src` = 0, `grep -rn "type StatusTone" web/src` = 1, views: `HomeView`
  243, `CardEditView` 293, `CardDetailView` 294 lines. `env.d.ts` removed —
  both `vue-tsc -b` and `vite build` work without the shim. PrimeFlex pinned
  to `4.0.0` in both `package.json` and the lockfile. The manual screenshot
  smoke test (320 px / desktop) and the axe contrast check remain for the
  reference host.
- **Deviations from the plan**: (1) The `StatusTone` tones stay
  `healthy|problem|unknown|info|warning` (spec §4 "Queued amber / Running
  blue" mapped to `warning`/`info`), not `success` as in the plan — the
  names match the existing `status-*` CSS classes and the tests from 0030.
  (2) Besides the components from spec §8.4 (`PageHeader`, `EmptyState`,
  `RequestState`, `InlineError`, `ActionCard`, `ActionControls`,
  `RunTable`, `StatusTimeline`, `FormSection`, `ConnectionStatus` from
  0032), `DetailPanel`, `DetailErrorState`, `StatusSummary`,
  `CardIdentityFields`, `PollingFields` and `SaveBar` were added, so that
  all views fit under 300 lines; `UnsavedChangesDialog` is implemented as
  the composable `useUnsavedChangesGuard` on top of the shared
  `ConfirmDialog` (no second dialog component). The detail logic is in the
  composables `useCardActivity` (runs + history with generation) and
  `useTransientMessage`. (3) The vocabulary is split into
  `src/ui/vocabulary.ts` (labels, presentation, texts) and
  `src/ui/format.ts` (formatting of times and durations); shared types are
  in `src/types.ts`, `cardEditModel.ts` re-exports them. (4) The tokens in
  UX spec §8.2 were adjusted to reality (the palette fine-tuned in blocks
  0019/0020 stays), the spec now states that PrimeVue colors come
  from `src/theme/preset.ts` via `definePreset(Aura, …)`; `darkModeSelector`
  is disabled, the app has a single light scheme. (5) FR-17 "state
  duration": the detail shows "In this state for …" from the current
  history transition, `StatusTimeline` marks the current transition
  "(current)" and refreshes the elapsed time every minute.
  (6) `.primary-action-button` and `.primary-action-link` remain global
  classes (without `!important`); the soft green "primary" style cannot be
  expressed in Aura tokens without a button variant.
- **Documentation updated**: yes — UX spec §8.2 (tokens per reality,
  preset) and §8.4 (component status), `docs/architecture/overview.md`
  (SPA row), `docs/devops/testing-strategy.md` (new tests), roadmap
  (phase 8 done).
