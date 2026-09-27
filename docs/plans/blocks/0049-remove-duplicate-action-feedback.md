# Implementation block: No duplicate action feedback and no card jumping

- **Phase**: 8 — Hardening (addendum after the phase was closed)
- **Requirements**: FR-21, FR-27, FR-29, NFR-08, NFR-09
- **ADRs**: ADR-0007 (UX and design system); no new ADR needed
- **Status**: Done
- **Dependencies**: Blocks 0029, 0037, 0039 (action progress and its
  result), 0040 (the "Status could not be refreshed" note), 0042 (feedback
  channels), 0041 (E2E)

## Goal

After an action is started, the card on the dashboard shows **Queued** /
**Running** in the badge and, below the description, a feedback line with
the same text; after completion, **Updated** lingers there for about 4 s
even though the badge already shows the new status. The line appears and
disappears, so the card changes height and the whole row in the grid
jumps. The project owner (2026-09-26) rejected both the duplication and
the jumping.

After this block, both the card and the detail show only feedback that
carries information beyond the badge, and in a place that does not change
the card's height.

## Scope

- **In scope**:
  - **Removed** (text duplicates the badge): the **Queued** / **Running**
    line during the action and the **Updated** / **Status updated** result
    on the dashboard and in the detail.
  - **Kept** (information the badge does not carry):
    - **Result not available yet** — the check did not finish within the
      waiting period, the badge shows the previous status;
    - an action enqueue error (full queue 503, network, 4xx) — rule §4:
      an error never disappears without a trace;
    - **Accepted** / **Action accepted** when no check follows (a card
      without a status action or without fast polling) — otherwise the
      user would have no confirmation that the action was started.
  - **Dashboard card** (`ActionCard.vue`): the remaining messages are shown
    **instead of** the `Last checked …` / `Not checked yet` line, in the
    same position and with the same height (tone and icon by type, text
    truncated to one line with the full text in `title`/tooltip); after the
    message ends (about 4 s, duration unchanged) `Last checked …` returns
    with the new time. A card without a status action, which today has no
    `Last checked` line, has this line reserved permanently (empty outside
    the message period) so that its height does not change.
  - **Card detail** (`CardDetailView.vue`): the remaining action messages
    move from the line above the summary into the **Actions** panel below
    the buttons, into a slot with a reserved height of one line. The line
    above the summary remains only for card deletion (**Deleting…** and
    the deletion error).
  - **Accessibility** (NFR-09): request state changes are still announced
    to screen readers, via a visually hidden `aria-live="polite"` region
    (text **Queued**, **Running**, **Updated** …) that takes up no space.
    The badge remains the carrier of the status for sighted users (text +
    icon + color, FR-25).
  - UX specification §4 (feedback rules, feedback channels), §5.2 (card
    anatomy) and §6 (detail, Actions) according to this block.
  - Component tests and E2E adjustment (`card-lifecycle.e2e.ts` currently
    waits for the visible text **Updated**).
- **Out of scope**:
  - changing the logic of waiting for the check, time limits or
    `useActionRequest` (only the presentation changes);
  - toasts (**Card saved**, **Card deleted**) and form feedback;
  - badge appearance and the status vocabulary.

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-09-26
- **Decision notes**: Approved by the owner ("I approve everything"). The split of messages, display instead of `Last checked`, and use in the card detail as well confirmed 2026-09-26.

## Proposed solution

- `ActionCard.vue`: `feedback` returns a message only for a `result` that
  is not `updated`; `pending` does not enter the visible text. The
  `card-meta` line is always rendered (`Last checked …`, a message, or
  empty for a card without a status action) with a fixed height of one
  line.
- The **Status could not be refreshed** note (block 0040) remains
  separate; if it also causes a jump, that is handled outside this block.
- A shared visually hidden live region: a small component or a
  `visually-hidden` class in the global styles (if it does not exist yet).
- `RequestState.vue` remains for other uses (editing, devices).

## Test plan

- Vitest `ActionCard`: during `pending` no visible **Queued** /
  **Running** text outside the badge, but it is in the live region;
  `result.updated` adds nothing visible; **Result not available yet**, an
  error and **Accepted** are shown instead of `Last checked` and after
  expiry `Last checked` returns; a card without a status action always has
  the line.
- Vitest `CardDetailView`: action messages in the Actions panel, deletion
  above the summary.
- `make e2e`: `card-lifecycle` verifies the result by the badge instead of
  the **Updated** text; a measurement that the card height stays the same
  during and after the action (`boundingBox` before / during / after).
- `make verify`.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md) plus:

- starting an action on the dashboard does not change the height of any
  card in the grid;
- no visible feedback text repeats what the badge is currently showing.

## Closure

- **Status after implementation**: Done (2026-09-26)
- **Verification**: `make verify` passed (164 Vitest tests, `install_test`,
  golangci-lint, eslint, vue-tsc, prettier, `go test -race`); `make e2e`
  passed (16 scenarios), `card-lifecycle` measures the card height before
  the action, after **Check status** and after **Run action**.
- **Implementation**: new component `ActionNote.vue` (an idle line that
  the action result replaces in place, and a visually hidden `aria-live`
  region for phases and results); `RequestResult.quiet` for **Updated** /
  **Status updated** (`outcomeResult`); `ActionCard` shows `ActionNote`
  instead of `Last checked` with a fixed height of one line;
  `CardDetailView` moved action results into the Actions panel
  (`useTransientResult`), the line above the summary remained only for
  deletion. Removed the unused texts `FEEDBACK.queued` and
  `FEEDBACK.actionQueued`.
- **Deviations from the plan**: the idle text of the Actions panel
  ("Actions are queued asynchronously…") moved into the vocabulary
  (`FEEDBACK.actionsAsync`), because `ActionNote` displays it as a
  fallback.
- **Out-of-scope finding**: the scenario `creates, runs, checks and deletes
  a card` failed once, because after **Action accepted** the detail loads
  the runs immediately, before the primary action finishes (the Recent
  runs section stayed empty). A repeated run (3×) passed; the behavior
  dates from block 0039 and this block does not change it. Proposal: a new
  block that, after an action is accepted without a subsequent check,
  reloads the runs after it completes (e.g. after `timeoutSec` or via a run
  event).
- **Documentation updated**: yes — UX specification §4, §5.2, §6,
  roadmap.
