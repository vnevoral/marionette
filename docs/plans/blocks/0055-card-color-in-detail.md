# Implementation block: Card color in the detail

- **Phase**: 8 — Hardening (addendum after the phase was closed), UI area
- **Requirements**: FR-10a, FR-25, FR-29
- **ADRs**: no new ADR needed
- **Status**: Done
- **Dependencies**: Block 0054 (card color, `cardColors.ts`, tokens)

## Goal

Today only the dashboard shows the card color. But one click takes you
from the dashboard to the detail, and without the color the operator
loses the cue about which card they are on. After this block the card
detail shows the same color as the dashboard.

## Scope

- **In scope**:
  - `CardDetailView`: the icon tile in the header (`detail-icon`, the
    `identity` slot in `PageHeader`) of a card with a color gets a 4 px
    stripe in the card color along the top edge of the tile. The same
    technique as on the dashboard is used (`box-shadow: inset` and the
    border color), so the header dimensions do not change. A card without
    a color or with a color outside the palette looks as it does today
    (`cardColorValue`);
  - to keep `CardDetailView.vue` under 300 lines (298 today), the icon
    tile is extracted into a small component `CardIdentityTile.vue`
    (icon + color). The dashboard does not use it, because there the
    stripe belongs to the whole card;
  - UX specification §6 (detail header);
  - tests: Vitest (`CardIdentityTile` with a color, without a color and
    with an unknown color; `CardDetailView` passes the card color), E2E
    (the detail of a card with a color has the stripe; extending the
    scenario from block 0054).
- **Out of scope**:
  - the color on the card edit page (it is visible there in the **Color**
    picker);
  - a colored background or border of the whole detail header (the header
    is not a card and a colored area would compete with the status
    badge).

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-09-27
- **Decision notes**: Approved by the owner ("agree with everything"): stripe on the icon tile as proposed.

## Proposed solution

A stripe on the icon tile, not on the whole header: the tile is the only
"card-like" element in the detail header, so the color is tied to the
same place as the icon and does not change the layout at 320 px. The
alternative, a left border of the whole header, is equally simple if the
owner prefers it.

## Test plan

- `cd web && npx vitest run`.
- `make e2e`.
- `make verify`.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md) plus:

- the detail of a card with a color shows the same color as the
  dashboard; the detail of a card without a color is unchanged.

## Closure

- **Status after implementation**: Done (2026-09-27)
- **Verification**: `make verify` passed (202 Vitest tests,
  `go test -race`, lint, `install_test`, `release_version_test`);
  `make e2e` passed (20 scenarios). New tests: Vitest
  `CardIdentityTile.spec.ts` (stripe for a palette color; no color, empty
  and unknown color without a stripe, default icon), `CardDetailView`
  (the header passes the card icon and color); the color E2E scenario
  from block 0054 extended to the detail (`.identity-tile` with
  `data-color`).
- **Deviations from the plan**: none. The `CardIdentityTile.vue`
  component also took over the tile styling; `CardDetailView.vue` has
  285 lines.
- **Documentation updated**: UX specification §6 (detail header).
