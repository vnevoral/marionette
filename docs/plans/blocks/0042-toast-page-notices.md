# Implementation block: Toast for page-level notices

- **Phase**: 8 — Hardening (addendum after the phase was closed)
- **Requirements**: FR-26, FR-27, NFR-08, NFR-09
- **ADRs**: ADR-0003, ADR-0007
- **Status**: Done
- **Dependencies**: Blocks 0032, 0033, 0041

## Goal

Once complete, the UI distinguishes two feedback channels: inline at the
card, field or page, and a toast for the result of an operation that leaves
or replaces the page. Saving a card shows the toast **Card saved**.
Deleting a card, which so far gave no feedback after returning to the
overview, shows the toast **Card deleted** with the card's name.

## Scope

- **In scope**:
  - `ToastService` in `main.ts`, a single `<Toast>` in `App.vue` (top right,
    on a phone full width minus 16 px margins);
  - `src/composables/useNotify.ts` (`success(summary, detail?)`, display
    duration `messageVisibleMs` shared with inline messages);
  - `CardEditView`: toast instead of an inline `notice`; `CardDetailView`:
    toast after a successful delete, a delete error stays inline;
  - theme preset: success toast in palette colors (AA contrast);
  - `src/test/fakeToast.ts`, unit and E2E tests;
  - UX spec §4 (feedback channels), §7.3, §8.4.
- **Out of scope**:
  - action results on cards and in the detail (they stay inline, UX spec
    §4, §5.2);
  - error toasts (an error is never shown only in a toast);
  - info/warn variants of the preset (not used).

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-09-26
- **Decision notes**: Instruction "sort out that toast". The decision on
  scope (only notices that survive navigation, not a replacement for inline
  feedback) is justified in UX spec §4: several cards can report a result at
  the same time and errors belong to the field or the card.

## Proposed solution

See scope. `useNotify` wraps `useToast` so that views know neither the
PrimeVue API nor the display duration. The toast after a delete is added
only after navigating to the overview.

## Test plan

- `CardEditView.spec.ts`: after creating a card exactly one toast
  `Card saved` (success, detail = name, 4000 ms), no inline message.
- `CardDetailView.spec.ts`: after a confirmed delete a toast `Card deleted`
  with the name; when the delete fails, the detail stays, the error is
  inline, no toast.
- E2E: toast after saving and deleting (`role=alert`), at 320 px it fits on
  the screen, does not cause horizontal scrolling and has a palette color.
- `make verify`, `make e2e`.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md).

## Closure

- **Status after implementation**: Done (2026-09-26)
- **Verification**: `make verify` passed (Vitest 96 → 97), `make e2e` 7 → 8
  tests, passed. Success toast contrast: Aura default `green.600` on
  `green.50` 3.15:1 (below AA), now `primary.600` (#397254) on accent-soft
  (#e6f1e8) 4.88:1, detail ink 11.34:1.
- **Deviations from the plan**: none. During implementation it turned out
  that the default Aura toast colors meet neither AA nor the palette; the
  override in the preset is therefore part of the block.
- **Documentation updated**: yes — UX spec §4 (feedback channels),
  §7.3 (save, delete), §8.4 (shared Toast, `useNotify`), roadmap.
