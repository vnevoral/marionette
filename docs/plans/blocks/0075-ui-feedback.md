# Implementation block: toast and confirmation dialog on Reka UI

- **Phase**: maintenance of the public repository
- **Requirements**: FR-23 (v0.14), FR-25, NFR-08, NFR-09
- **ADRs**: ADR-0014 (points 1, 2 and 4)
- **Status**: Done
- **Dependencies**: Block 0074

## Goal

The third of the four ADR-0014 blocks: page-level notices and confirmation
questions are the project's own components in `web/src/ui/`, built on
Reka UI `Toast` and `AlertDialog`. After it no PrimeVue component is
rendered; the packages themselves are removed in block 0076.

## Scope

- **In scope**:
  - `MToastHost` (Reka UI `Toast`) and `MConfirmHost` (Reka UI
    `AlertDialog`), mounted once in `App.vue` in place of PrimeVue `Toast`
    and `ConfirmDialog`;
  - a small feedback service (`web/src/ui/feedback.ts`) installed by
    `main.ts`: `useToast().add(...)` and `useConfirm().require(...)`, with
    the project's own option types (tone instead of PrimeVue severity,
    Lucide icon names);
  - `useNotify`, `useUnsavedChangesGuard`, Card Detail (delete card) and
    Devices (remove device) moved to the new service; `fakeToast` and
    `fakeConfirm` provide the new service in unit tests;
  - the toast tokens in the PrimeVue preset and the PrimeVue note in
    `tokens.css` removed;
  - unit tests for both hosts, e2e (the save notice), UX spec §4 and §8,
    roadmap.
- **Out of scope**:
  - removing `primevue`, `@primeuix/*`, `primeicons`, `preset.ts` and the
    PrimeVue setup in `main.ts` and the specs (block 0076);
  - any change to the wording, the moments a notice or a question appears,
    or the default focus.

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-10-20
- **Decision notes**: Part of the ADR-0014 plan the owner accepted; the
  owner asked to continue after block 0074 was merged.

## Proposed solution

- The service is provided through Vue injection, so a test provides a fake
  and the hosts read one shared reactive state.
- Confirm: one question at a time. The dialog is an `alertdialog` with the
  title, the message, a secondary **cancel** button that has the initial
  focus (Reka UI focuses the cancel button), and the accept button in the
  danger variant for destructive questions. Escape and the cancel button
  call `reject`; `accept` closes the dialog first.
- Toast: top right, full width with the page gutter on phones, a tone
  icon, title, detail, a close button and the same 4 s life as before;
  hovering or focusing pauses the timer (Reka UI behaviour). Contrast meets
  AA with the palette colours.

## Test plan

- Unit tests: `MConfirmHost` (title, message, focus on cancel, accept,
  cancel and Escape), `MToastHost` (summary, detail, tone, close), the
  existing view tests through the new fakes.
- `make verify` and `make e2e` pass (the delete, remove-device and
  unsaved-changes dialogs by role `alertdialog`; the save notice fits a
  320 px screen and uses the palette colour).
- Visual check of the delete confirmation, the unsaved-changes question
  and the save toast at 1280 px and 375 px.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md).

## Closure

- **Status after implementation**: Done (2026-10-20)
- **Verification**: `make verify` passed (Vitest 231 tests, including the
  new `MConfirmHost` and `MToastHost` tests); `make e2e` passed (21
  tests). Visual check at 1280 px and 375 px of the delete confirmation,
  the "Card deleted" and "Card saved" toasts and the unsaved-changes
  question. The only console errors are the expected 401 responses before
  pairing.
- **Deviations from the plan**: The visible Reka UI toast has no `alert`
  role (Reka UI announces it through a hidden copy), so the e2e tests find
  a toast through the notifications region (`toast()` in
  `e2e/helpers.ts`). The accept button is a plain `MButton`, not
  `AlertDialogAction`, because closing through the action would also
  report a rejection. The unused `button` tokens left the PrimeVue preset
  with the toast tokens.
- **Documentation updated**: UX spec §4, §8.2 and §8.4, roadmap.
