# Implementation block: UI foundation (tokens, icons, buttons, surfaces)

- **Phase**: maintenance of the public repository
- **Requirements**: FR-10 (card icon), FR-23 (v0.14), FR-25, NFR-08
- **ADRs**: ADR-0014 (points 2, 3, 4 and 6), ADR-0004 (icon value), ADR-0007
- **Status**: Done
- **Dependencies**: ADR-0014 accepted

## Goal

The first of the four ADR-0014 blocks. After it, every icon in the UI is a
Lucide icon, buttons, badges, messages, spinners and card surfaces are the
project's own components in `web/src/ui/`, and the visual refresh (tokens,
typography, header) is in place. Form controls, the toast and the
confirmation dialog are still PrimeVue and follow in blocks 0074 and 0075.

## Scope

- **In scope**:
  - `@lucide/vue` (ISC) as a dependency;
  - `web/src/ui/icons.ts`: a registry of supported icon names → Lucide
    components; `MIcon` renders a name, an unknown non-empty name renders
    the **ghost** icon, an empty one the default card icon;
  - card icon values become plain names (`house`, `server`, …): the
    editor offers only supported names, a card with an unknown icon shows
    "This icon is no longer available. Choose a new one." in the editor and
    cannot be saved until a supported icon is chosen;
  - backend: `config.ValidIcon` accepts `^[a-z0-9]+(-[a-z0-9]+)*$` up to
    64 characters (API validation only, not `ValidateEssential`), so the
    API rejects `pi pi-<name>` with 422; `deploy/dev-fixture.json`, tests,
    the `docs/architecture/overview.md` table and an ADR-0004 note;
  - own components `MButton` (variants primary, secondary, ghost, danger;
    sizes; icon; loading; `to` renders a router link), `MSpinner`, `MTag`,
    `MMessage`, `MCard`; `StatusBadge` without PrimeVue `Tag`;
  - every PrimeVue `Button`, `Tag`, `Message`, `ProgressSpinner` and `Card`
    replaced; no `pi-*` class left in templates outside the PrimeVue
    toast/confirm options;
  - visual refresh with the same layout: tokens (palette, radii, layered
    shadows, system font stack, control heights), headings, sticky header
    with pill navigation, soft status badges;
  - fix: on a phone narrower than 24rem the stacked page-header buttons
    collapsed to a few pixels (`flex: 1 1 0` in a column);
  - unit and e2e tests, UX spec §8, roadmap.
- **Out of scope**:
  - `InputText`, `Textarea`, `InputNumber`, `Select`, `ToggleSwitch`
    (block 0074), `Toast` and `ConfirmDialog` (block 0075), removing
    PrimeVue, `@primeuix/*` and `primeicons` (block 0076);
  - migrating stored icon values (owner decision in ADR-0014);
  - dark mode.

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-10-20
- **Decision notes**: The owner accepted ADR-0014 with its four-block
  plan, allowed a more modern look as long as the layout stays, asked for
  a visual check of the result, and decided that icon codes may change: an
  unknown icon shows a ghost and the user picks a supported one in the
  editor.

## Proposed solution

- Supported card icons (name → Lucide): `monitor`, `house`, `server`,
  `cloud`, `database`, `globe`, `zap`, `settings`, `shield`, `heart`,
  `router`, `printer`, `hard-drive`, `wifi`, `tv`, `lightbulb`, `camera`,
  `power`, `network`, `cpu`. The default is `monitor`. UI icons
  (statuses, buttons, navigation) live in the same registry but are not
  offered for cards.
- `MIcon` (`name`, `size`, `spin`) is always `aria-hidden`; the meaning is
  carried by the text next to it (FR-25).
- `MButton` renders `<button>` or, with `to`, a `RouterLink`; the label is
  in `.m-button-label`; `loading` shows `MSpinner` in place of the icon and
  sets `aria-busy`.
- The client rejects an unknown icon in `validate()` (`icon` field error),
  the server returns 422 for the same value; `ValidateEssential` ignores
  the icon, so existing configuration files keep loading.

## Test plan

- Unit tests: `MIcon` (known name, unknown → ghost, empty → default),
  `MButton` (link, loading, disabled), `StatusBadge`, the editor rule for
  an unknown icon, vocabulary icons are registered names.
- Go tests: `ValidIcon` accepts names and rejects `pi pi-home`, markup,
  upper case and over-long values; `ValidateEssential` still loads
  `pi pi-Home`.
- `make verify` and `make e2e` pass (the header-button test reads
  `.m-button-label`).
- Visual check: screenshots of pairing, overview, detail, edit, new card
  and devices at 1280 px and 375 px compared with the previous UI,
  including a card with an old `pi pi-*` icon.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md).

## Closure

- **Status after implementation**: Done (2026-10-20)
- **Verification**: `make verify` passed (Vitest 219 tests, including new
  `MIcon`, `MButton`, ghost-icon and editor tests; Go tests for
  `ValidIcon`); `make e2e` passed (21 tests). Visual check at 1280 px and
  375 px of pairing, overview, detail, edit, new card, devices, a card with
  an old `pi pi-cloud` icon (ghost on the dashboard, "Unavailable icon"
  and the message in the editor) and the delete confirmation; the stacked
  header buttons on the phone keep their height. The only console errors
  are the expected 401 responses before pairing.
- **Deviations from the plan**: `reka-ui` is not added yet; nothing in
  this block needs it, so it comes with block 0074. The PrimeVue preset
  takes the new accent, focus and border colours so the remaining PrimeVue
  controls match. The "Check status" button uses the `activity` icon
  instead of a heart.
- **Documentation updated**: UX spec §8, `docs/architecture/overview.md`
  (icon limit), ADR-0004 (note), roadmap.
