# Implementation block: Card color on the Current status panel

- **Phase**: 9 — UX redesign and shared design system (addendum after the phase was closed)
- **Requirements**: FR-10a, FR-25, FR-29
- **ADRs**: no new ADR needed
- **Status**: Done
- **Dependencies**: Block 0055 (card color in the detail), block 0054 (`cardColors.ts`, tokens)

## Goal

Block 0055 put the card color on the icon tile in the detail header. On
the Raspberry Pi the owner found the 4 px stripe on a 64 px tile too small
to notice and the tile the wrong place for it. After this block the color
is a stripe along the top edge of the **Current status** panel, the same
width and technique as on the dashboard card, and the icon tile has no
color.

## Scope

- **In scope**:
  - `DetailPanel` gets an optional `color` prop; a palette color draws the
    dashboard stripe (top border and a 5 px inset shadow, 6 px in total),
    so the panel keeps its size; no color or a color outside the palette
    draws nothing (`cardColorValue`);
  - `CardDetailView` passes the card color only to the **Current status**
    panel;
  - `CardIdentityTile` loses the `color` prop and its stripe;
  - UX specification §6;
  - tests: Vitest (`DetailPanel` with a color, without a color and with an
    unknown color; `CardIdentityTile` without color; `CardDetailView`
    colors only the Current status panel), E2E (the color scenario from
    blocks 0054/0055 checks the panel instead of the tile).
- **Out of scope**:
  - coloring the other panels (Actions, Recent runs, Status history) or
    the header;
  - any change to the requirement, the palette or the API.

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-09-27
- **Decision notes**: Requested by the owner: move the color from the icon
  tile to the Current status panel so the edge is larger and more visible.

## Proposed solution

The stripe moves into the shared `DetailPanel`, with the same CSS as
`ActionCard` (`border-top-color` plus `inset 0 5px 0`), so the detail
looks like the dashboard card the operator came from. Only the Current
status panel is colored: it sits next to the status badge that the color
must not be confused with, but FR-10a keeps state in the badge and the
palette does not overlap with the status colors.

## Test plan

- `cd web && npx vitest run`.
- `make e2e`.
- `make verify`.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md) plus:

- the detail of a card with a color shows the stripe on the Current
  status panel only; the icon tile has no stripe; the detail of a card
  without a color is unchanged.

## Closure

- **Status after implementation**: Done (2026-09-27)
- **Verification**: `make verify` passed (208 Vitest tests,
  `go test -race`, lint, `install_test`, `release_version_test`);
  `make e2e` passed (20 scenarios). New tests: Vitest `DetailPanel.spec.ts`
  (stripe for a palette color; no color, empty and unknown color without a
  stripe), `CardIdentityTile.spec.ts` reduced to the icon,
  `CardDetailView` (only the Current status panel gets the color); the
  color E2E scenario checks the Current status panel (`data-color` and the
  top border color), that no other panel is colored and that the icon tile
  has no color.
- **Deviations from the plan**: none.
- **Documentation updated**: UX specification §6 (card detail).
