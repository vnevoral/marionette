# Implementation block: Card color stripe

- **Phase**: 8 — Hardening (addendum after the phase was closed), UI area
- **Requirements**: FR-10a, FR-10, FR-20, FR-25, FR-28, FR-29
- **ADRs**: ADR-0004 (the card domain model gains an optional field;
  persistence does not change). No new ADR is needed: the architecture
  does not change, only a new optional card field is added
- **Status**: Done
- **Dependencies**: Blocks 0019/0020 and 0033 (design tokens, card on the
  dashboard), 0049 (stable card height)

## Goal

On a dashboard with many cards, cards are told apart only by icon and
name. After this block, each card can be assigned a color from a fixed
palette in the editor, and the card on the dashboard shows it as a thick
colored top edge. Cards can then be grouped by meaning (e.g. PC, network,
servers) and found by eye faster.

## Scope

- **In scope**:
  - `internal/config`: field `ActionCard.Color string` (`json:"color,omitempty"`)
    holding the name of a palette color (not a hex value), so the shade can
    be adjusted in the UI without a configuration migration. Validation in
    `Validate`: empty or one of the names in `CardColors` → otherwise an
    error on field `color` (API `422`). An empty value means no color. A
    missing field in an older configuration also means no color;
  - the card API (`GET`/`POST`/`PUT`) carries the field without changing
    the contract of the other fields; card field table in
    `docs/architecture/overview.md`;
  - palette: 8 options, see the proposed solution. Hex values as tokens
    `--card-color-<name>` in `web/src/styles/tokens.css`. The list of names
    lives in the UI (`web/src/ui/cardColors.ts`) and in Go, and a test
    checks that they match;
  - `ActionCard.vue` (dashboard): a card with a color has a 6 px top edge
    in that color; without a color it looks as it does today. The stripe
    is decorative and does not change the card height or the grid layout
    (FR-29, block 0049): it is drawn as `box-shadow: inset` and a top
    border color, which take no space in the layout;
  - `CardIdentityFields.vue`: field **Color** (`Select` with a color swatch
    and a text name, same pattern as the icon picker, FR-28), default
    **None**. `cardEditModel` carries the color into the form and back;
    a server `color` error is shown at the field;
  - UX specification §5.2 (card anatomy), §7.2 (Card identity) and §8.2
    (palette tokens and the rule that the card color carries no status);
  - `deploy/dev-fixture.json`: one card with a color;
  - tests: Go (`Validate` accepts an empty value and all palette names,
    rejects an unknown value and hex; JSON without the field loads;
    `omitempty` on save), Vitest (`ActionCard` with and without a color,
    `cardEditModel` color round-trip, `CardEditView` saves the selected
    color and shows a `color` error, UI palette matches Go via a shared
    list in the test), E2E (`card-lifecycle`: setting a color in the
    editor, the stripe is visible on the dashboard also after a page
    reload; a card with a color has the same height as the same card
    without one).
- **Out of scope**:
  - arbitrary color (color picker, hex input);
  - color in the card detail and other screens (can be added later);
  - filtering or sorting the dashboard by color;
  - choosing the stripe position per card (the position is the same for
    all cards so the grid looks consistent).

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-09-27
- **Decision notes**: Approved by the owner ("agree"): top edge, palette
  as proposed (no red and green). The owner requires the same card height
  with and without the stripe; added to the proposal and done criteria.

## Proposed solution

**Position**: a thick top edge (6 px) across the full card width, rounded
to match the card corners. Why not the side: the top edge follows the
card's top bar (icon + status badge), takes no content width at 320 px,
and in a multi-column grid the color is visible even at a quick glance
across a row.

**Uniform height**: the card today has a 1 px top border. For a card with
a color it is colored and complemented by an inner shadow `inset 0 5px 0`
in the same color, 6 px in total. Neither the border nor the shadow
changes the box dimensions, so a card with or without a color has the
same height and content padding.
A side stripe is technically just as simple. If the owner prefers it,
only the CSS changes.

**Palette (proposal)**:

| Name (UI) | Value in JSON | Shade |
|---|---|---|
| None | *(empty)* | no stripe (white/transparent) |
| Black | `black` | `#26332f` (text color) |
| Blue | `blue` | `#3b6fb6` |
| Teal | `teal` | `#2a9d8f` |
| Purple | `purple` | `#7b5ea7` |
| Pink | `pink` | `#c95b8f` |
| Orange | `orange` | `#e07b39` |
| Yellow | `yellow` | `#e0b53a` |

Red and green are intentionally not in the palette. On the dashboard they
already mean the **Problem** and **Healthy** statuses (FR-25), so a red
stripe on a healthy card would be confusing. Orange and yellow are more
saturated and warmer than the status `warning` (`#ad7e3f`), and the
stripe is in a different place than the text badge. The exact shades
will be tuned during implementation for contrast against `--color-canvas`
and the card surface.

**Compatibility and version**: the field is optional, so a configuration
from v1.0.0 loads unchanged. An older version ignores the unknown field
on load but drops it on the next configuration save. After a rollback to
v1.0.0 the card colors are therefore lost; the cards themselves remain.
This is backward-compatible new functionality, hence a MINOR version
(`v1.1.0`).

## Test plan

- `go test -race ./internal/config ./internal/server`.
- `cd web && npx vitest run`.
- `make e2e` (extending `card-lifecycle.e2e.ts`).
- Manual check of the dashboard at 320 px, tablet and desktop with cards
  with and without a color (same card height, no layout shift).
- `make verify`.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md) plus:

- a color selected in the editor is saved to the configuration, survives
  a service restart and is visible on the dashboard as a stripe; a card
  without a color looks as it did before the block;
- cards with and without a color have the same height (verified by E2E
  comparing the height of two cards with the same content).

## Closure

- **Status after implementation**: Done (2026-09-27)
- **Verification**: `make verify` passed (183 Vitest tests, `go test -race`,
  golangci-lint, eslint, vue-tsc, prettier, `install_test`,
  `release_version_test`); `make e2e` passed (18 scenarios including the
  new "colours a card with a stripe that keeps the card height": color
  selected in the editor, stripe after a dashboard reload, same height of
  a card with and without a color and the same heading padding). Visual
  check of the dashboard with all 8 options and the expanded picker in
  the editor (Playwright screenshots, 1280 px).
  New tests: Go `TestActionCardValidateLimits` (empty, known, unknown,
  hex and uppercase), `TestSaveFileRoundTripKeepsCardColor` (the color
  survives save and load; a card without a color is written without the
  key), `TestCardColorsMatchTheUIPalette` (Go list = UI list),
  `TestValidateEssentialIgnoresLimits` (an unknown color from the file
  loads), `TestRouterReturnsValidationFields` (`422` with `fields.color`);
  Vitest `ActionCard` (stripe only for a palette color),
  `cardEditModel` (fingerprint, mapping of the `color` error),
  `CardEditView` (a saved color is shown, a new one is saved, a `422`
  error at the field).
- **Deviations from the plan**:
  - the card carries a `data-color` attribute for tests and style
    debugging;
  - an unknown color loaded from the file (hand-edited configuration) is
    not shown on the dashboard, and the editor converts it to **None**
    when loading the card (`knownCardColor`), so the card can be saved
    and the unknown value is thereby removed (finding from the
    post-implementation code review: originally the editor showed an
    empty field and saving failed with `422`);
  - the form keeps "no color" as an empty string so that loading another
    card does not inherit the previous card's color.
- **Documentation updated**: UX specification §5.2, §7.2, §8.2; field
  limits table in `docs/architecture/overview.md`; `deploy/dev-fixture.json`.
