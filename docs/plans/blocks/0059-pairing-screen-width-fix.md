# Implementation block: Pairing screen at 320 px with a fallback font

- **Phase**: 8 — Hardening (fix after the phase was closed)
- **Requirements**: FR-29, FR-06
- **ADRs**: no new ADR is needed
- **Status**: Done
- **Dependencies**: Blocks 0043/0044 (pairing screen), 0057 (release from CI)

## Goal

The first release from CI (tag `v1.2.0`, 2026-09-27) was not published. In
the `e2e` job, the scenario "at 320 px › the pairing screen and a shown
code fit the screen" failed: the `/pair` screen was 2 px wider than 320 px.
On the GitHub runner, DejaVu Sans / DejaVu Sans Mono are used as the
fallback font, which are wider than the fonts in the devcontainer, where
the test passed. After the fix, the pairing screen fits into 320 px
regardless of the fallback font.

## Scope

- **In scope**: `PairView.vue`: the `.pair-panel`, `.pair-form` and
  `.pair-field` grids have `grid-template-columns: minmax(0, 1fr)`, so the
  intrinsic width of the code field (monospaced font 1.25rem with letter
  spacing) does not widen the column.
- **Out of scope**: adding fonts to the application or CI; changing the
  existing E2E test (the test correctly revealed the bug).

## Approval

- **Approved by**: project owner (fix for the release workflow failure that
  the owner handed over for resolution on 2026-09-27)
- **Approval date**: 2026-09-27
- **Decision notes**: Bug fix without a behavior change; the block was
  written retroactively together with the implementation.

## Closure

- **Status after implementation**: Done (2026-09-27)
- **Verification**:
  - The bug was reproduced locally after installing DejaVu fonts
    (`fc-match sans-serif` → DejaVu Sans): `/pair` overflowed by 2 px.
    The form, the labels and both fields were wider than 320 px, all with
    a width of 281 px. The same overflow also occurred with `AppShell`
    before block 0056, so the bug had existed earlier and only showed up
    with the runner's fonts.
  - After the fix, the overflow is 0 px on `/`, `/devices` (also with a
    shown code), `/cards/new/edit` and `/pair`.
  - `make e2e` with DejaVu fonts passed (20 scenarios); `make verify`
    passed (206 Vitest tests).
  - Note: one `make verify` run on a heavily loaded machine (load
    average around 20) failed in the timing test
    `TestRunShutsDownQuicklyWithOpenSSEClientAndSavesHistory` (3.45 s
    against a 3 s limit). It passed five times on its own and in another
    full run. Unrelated to this fix.
- **Deviations from the plan**: none.
- **Documentation updated**: roadmap.
