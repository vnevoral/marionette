# Implementation block: UI form controls on Reka UI

- **Phase**: maintenance of the public repository
- **Requirements**: FR-23 (v0.14), FR-28, NFR-08, NFR-09
- **ADRs**: ADR-0014 (points 1, 2 and 4)
- **Status**: Done
- **Dependencies**: Block 0073

## Goal

The second of the four ADR-0014 blocks: every form control is the
project's own component in `web/src/ui/`. Select and switch are built on
Reka UI. The toast and the confirmation dialog are still PrimeVue and move
in block 0075.

## Scope

- **In scope**:
  - `reka-ui` (MIT) 2.11.0 as a dependency;
  - `MInput`, `MTextarea` (optional auto-resize), `MNumberInput`,
    `MSelect` (Reka UI `Select`), `MSwitch` (Reka UI `Switch`), styled
    through a shared `.m-control` look in `tokens.css`;
  - every PrimeVue `InputText`, `Textarea`, `InputNumber`, `Select` and
    `ToggleSwitch` replaced in the card editor, the pairing screen and the
    Devices page;
  - preset tokens that no PrimeVue component uses any more removed;
  - unit tests, UX spec §8.4, roadmap.
- **Out of scope**:
  - `Toast` and `ConfirmDialog` (block 0075), removing PrimeVue (block
    0076);
  - any change to validation, wording or layout of the forms.

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-10-20
- **Decision notes**: Part of the ADR-0014 plan the owner accepted; the
  owner asked to continue after block 0073 was merged.

## Proposed solution

- The text and number controls are native elements. They are the root of
  their component, so attributes (id, placeholder, maxlength, aria-*) and
  `$el.focus()` reach the field directly.
- `MSelect` takes `{ label, value }` options, is named through `ariaLabel`
  or `ariaLabelledby`, and has `option` and `value` slots. Reka UI reserves
  `""` for "no selection", so the empty option (colour None) is mapped to
  an internal sentinel and back. The `value` slot receives no option for a
  value outside the list, such as an old icon.
- `MNumberInput` is a native `type="number"` field rather than Reka UI's
  `NumberField`: it gives the numeric keyboard and the stepper with less
  code, and an empty field is `undefined`, as before.

## Test plan

- Unit tests for every new control: attributes, invalid state, the empty
  option of `MSelect`, an empty number field, the switch role and toggling.
- The editor tests use `MSelect` and click the switch.
- `make verify` and `make e2e` pass (the colour test opens the select by
  its role and picks an option).
- Visual check of the editor, the opened select, the pairing screen and
  the Devices page at 1280 px and 375 px.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md).

## Closure

- **Status after implementation**: Done (2026-10-20)
- **Verification**: `make verify` passed (Vitest 225 tests); `make e2e`
  passed (21 tests). Visual check at 1280 px and 375 px of the editor (with
  a status action, and with an old icon), the opened icon select, pairing
  and devices. The only console errors are the expected 401 responses
  before pairing. The bundle grows for now (JS 152 → 162 kB gzip) because
  Reka UI and PrimeVue are both in it until block 0076.
- **Deviations from the plan**: `MNumberInput` is a native number input,
  not Reka UI `NumberField` (see Proposed solution). At the owner's
  request after the visual check, **Cancel** and **Save card** moved from
  the sticky `SaveBar` (which floated mid-page and left a gap under the
  form) to the page header actions, like Card Detail; `SaveBar` is
  removed and the e2e header-button check covers the edit page.
- **Documentation updated**: UX spec §7.2, §8.4, roadmap.
