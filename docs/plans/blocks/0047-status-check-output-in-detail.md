# Implementation block: Output of the last status check in the card detail

- **Phase**: 8 — Hardening (addendum after the phase was closed)
- **Requirements**: FR-21a, FR-13, FR-17, FR-24
- **ADRs**: ADR-0007 (UX and design system); no new ADR is needed
- **Status**: Done
- **Dependencies**: Blocks 0018 (card detail), 0033 (shared components),
  0040 (visible status read failure)

## Goal

During validation on the Raspberry Pi (2026-09-26), the `ping` status
action failed with `socket: Operation not permitted` (see block 0046), but
the UI only showed the result **Fail**. The backend stores the check output
(`StatusSnapshot.lastCheck` with `exitCode`, `duration`, `output`,
`truncated`) and returns it in `GET /api/cards/{id}/status` and in SSE
events, but the UI does not display it; the operator had to read JSON from
the API. When the block is done, the card detail shows the exit code, run
duration and expandable output of the last check.

## Scope

- **In scope**:
  - `web/src/components/StatusSummary.vue`: below "Last outcome", rows
    "Exit code" and "Duration" and an expandable output (`<details>` "View
    output", `…` when `truncated`) in the same form as for runs
    in `RunTable.vue`; without output the `<details>` is not rendered;
  - a shared output presentation for `RunTable` and `StatusSummary`
    (a small `RunOutput.vue` component), so that the `pre` style is not
    duplicated;
  - UX specification §6: Summary also contains the exit code, duration and
    output of the last check;
  - component tests (Vitest): the output is shown only on demand,
    truncation is marked, a missing `lastCheck` or empty output renders
    nothing extra, the `unavailable` state (block 0040) does not change;
  - E2E (Playwright): a scenario with a failed status check verifies that
    the output can be expanded in the detail.
- **Out of scope**:
  - output or exit code on the dashboard card (FR-21a: the dashboard stays
    scannable; the operator goes to the detail via **View details**);
  - history of outputs of older status checks (the status history per
    ADR-0006 holds only state transitions, not individual runs);
  - backend, API contract or persistence changes (the data already exist).

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-09-26
- **Decision notes**: Approved by the owner ("I approve everything").

## Proposed solution

See the scope. `StatusSummary` already receives the whole
`StatusSnapshot`, so it is enough to read `status.lastCheck` (`exitCode`,
`duration`, `output`, `truncated`). Labels go through
`src/ui/vocabulary.ts` (FR-26); `<details>/<summary>` is keyboard
accessible (FR-28), `pre` wraps and at 320 px does not cause horizontal
page scroll (FR-29).

## Test plan

- Unit tests for `StatusSummary` and `RunTable` (after extracting
  `RunOutput`).
- `make e2e` — a new or extended card detail scenario.
- `make verify`.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md) plus:

- in the detail of a card with a failed status check, the operator sees the
  command output (e.g. `socket: Operation not permitted`) without access to
  the API.

## Closure

- **Status after implementation**: Done (2026-09-26)
- **Verification**: `make verify` passed; `make e2e` passed including the
  new scenario `shows the output of a failed status check in the detail`
  (status action `ls /marionette-e2e-missing`: badge Problem, exit code
  other than 0, the expanded output contains the error message).
- **Implementation**: `RunOutput.vue` (expandable output, `...` when
  truncated) is shared by `RunTable` and `StatusSummary`; `StatusSummary`
  adds **Exit code** and **Duration** rows and the output of the last
  check; new `StatusSummary.spec.ts`; the **View output** label in the
  vocabulary (`ACTIONS.viewOutput`).
- **Deviations from the plan**: none.
- **Documentation updated**: yes — UX specification §6, roadmap.
