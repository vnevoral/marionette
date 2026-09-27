# Implementation block: Visible status read failure (UX spec §4)

- **Phase**: 8 — Hardening (addendum after the phase was closed)
- **Requirements**: FR-21, FR-25, FR-26, NFR-08
- **ADRs**: ADR-0007
- **Status**: Done
- **Dependencies**: Blocks 0033, 0037, 0039

## Goal

When done, both the dashboard and the card detail show that the status
could not be refreshed: the badge switches to **Unknown** and the card shows
the text "Status could not be refreshed", while the last known check time
stays visible. This makes the code match UX spec §4 ("A status read failure
keeps the card visible and marks its data as unknown"), which it does not
satisfy today: `useCardStatus.failed` is set but never displayed, and since
block 0037 the dashboard does not track the failure state at all.

## Scope

- **In scope**:
  - `vocabulary.ts`: `FEEDBACK.statusUnavailable`;
  - `ActionCard.vue`: prop `statusUnavailable`; badge **Unknown** (unless a
    request is running), a line with the text below the check time;
  - `StatusSummary.vue`: prop `unavailable`; a note above the list;
  - `HomeView.vue`: `statusUnavailable` per card — set when `listCards`
    fails in polling mode (for all cards with a status action) and on a
    REST read error while waiting for an action; cleared by every
    successfully received snapshot for the card and by a new dashboard
    load;
  - `CardDetailView.vue`: the badge and `StatusSummary` read
    `cardStatus.failed`.
- **Out of scope**: changing polling behavior, Toast, a global banner.

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-09-26
- **Decision notes**: Aligning the code with UX spec §4 (the documentation
  is the source of truth, AGENTS.md); the deviation recorded in the Closure
  of block 0037. The instruction "continue" after the second code review.

## Proposed solution

- `ActionCard.badge`: `pending` → REQUEST, `statusUnavailable` →
  `STATUS.unknown`, otherwise the state. The `.card-unavailable` line has
  no live region (UX spec §9: without spamming assistive technology).
- `HomeView.applySnapshot` first clears the flag (the read succeeded), then
  applies `supersedes`.

## Test plan

- `ActionCard.spec.ts`: `statusUnavailable` → Unknown badge and text; while
  a request is running, Running takes precedence.
- `HomeView.spec.ts`: a `listCards` failure in polling mode marks the card
  with a status action (not the card without one); a successful next
  refresh clears the flag.
- `CardDetailView.spec.ts`: a `getStatus` failure on a polling tick →
  Unknown and text; the next successful read → the original state.
- `make verify`.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md).

## Closure

- **Status after implementation**: Done (2026-09-26)
- **Verification**: `make verify` passed; Vitest 93 → 96 tests. New:
  `ActionCard.spec.ts` (Unknown + text, the last check time stays, Running
  takes precedence, a card without a status action does not show the
  text), `HomeView.spec.ts` (a `listCards` failure in polling mode marks
  Printer, not Lamp; the next successful refresh returns Healthy),
  `CardDetailView.spec.ts` (a `getStatus` failure on a polling tick →
  Unknown and a note in the summary, the next successful read → Healthy).
  Text contrast of `--color-warning-strong` (#94652d) on `--color-surface`
  4.82:1, on white 5.05:1 (WCAG 2.2 AA).
- **Deviations from the plan**: none.
- **Documentation updated**: yes — UX spec §4 (the form of the marking and
  when it disappears), roadmap.
