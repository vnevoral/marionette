# Implementation block: Page header actions on mobile

- **Phase**: 9 — UX redesign and shared design system (addendum after the phase was closed)
- **Requirements**: FR-29, NFR-08
- **ADRs**: no new ADR needed
- **Status**: Done
- **Dependencies**: none

## Goal

On a phone (reported from the reference Raspberry Pi deployment, about
400 px wide) the page header buttons on the dashboard (**Refresh**,
**New card**) sat on their own row but only as wide as their content, and
the **New card** label broke into two lines. The same header is used by
the card detail (**Edit card**, **Delete card**). After this block the
header actions on narrow screens fill the row equally and their labels
stay on one line; the desktop layout does not change.

## Scope

- **In scope**:
  - `PageHeader`: labels of the header actions do not wrap
    (`white-space: nowrap`); up to 48 rem the title and the actions are
    stacked and the actions row takes the full width, each action
    `flex: 1 1 0`; below 24 rem the actions stay stacked one per row as
    today. This covers every page with the shared header (dashboard,
    detail, edit, devices);
  - UX specification §5.3;
  - tests: E2E at 320 px and 412 px on the dashboard and the detail (the
    actions span the header row, both buttons are equally wide, each label
    is on one line).
- **Out of scope**:
  - buttons inside cards and panels (Run action, Check status), which
    already fill their row;
  - changing button labels, order or styles.

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-09-29
- **Decision notes**: Requested by the owner with a screenshot from a
  phone: fix the width and behavior of the main header buttons on the
  dashboard and the card detail.

## Proposed solution

The cause is that `.page-header-row` wraps, so the actions container
moves to a new line but keeps its content width; `flex: 1` on the buttons
then has no room to fill. Stacking the row as a column with
`align-items: stretch` makes the container full width, `flex: 1 1 0`
splits it equally and `nowrap` keeps each label on one line.

## Test plan

- `make e2e` (new scenario in `security-and-layout.e2e.ts`, fails without
  the fix).
- `make verify`.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md) plus:

- at 320 px and about 400 px the header actions on the dashboard and the
  detail fill the width, are equally wide and have one-line labels; the
  desktop header is unchanged.

## Closure

- **Status after implementation**: Done (2026-09-29)
- **Verification**: `make verify` passed (208 Vitest tests,
  `go test -race`, lint, `install_test`, `release_version_test`);
  `make e2e` passed (21 scenarios). The new E2E scenario failed on the
  old header (the actions were 118 px of the 288 px row at 320 px).
  Screenshots at 412 px and 1280 px checked: the phone header has two
  equal one-line buttons, the desktop header is unchanged.
- **Deviations from the plan**: none.
- **Documentation updated**: UX specification §5.3.
