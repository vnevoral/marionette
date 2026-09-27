# Implementation block: UX-04 — Card detail, runs and status timeline

- **Phase**: 9 — UX redesign and shared design system
- **Requirements**: FR-17, FR-20, FR-21, FR-24, FR-25, FR-26, FR-27, FR-29, NFR-08..10
- **ADRs**: ADR-0003, ADR-0006, ADR-0007
- **Status**: Done
- **Dependencies**: Blocks 0015–0017, read-only API runs/status/history

## Goal

The card detail provides a diagnostic view of the last known status, the
latest runs of the primary action and the history of actual status
transitions. The user gets from Overview to the detail with one clear link.

## Scope

- **In scope**:
  - route `/cards/:id` and the `View details` link from Overview;
  - header with the name, icon, state and last check;
  - summary of the current state and the duration of the last status interval;
  - actions `Run action` and `Check status`;
  - recent primary runs with outcome, exit code, time, duration and collapsible output;
  - status transition timeline with the new state, start, end and duration;
  - loading, empty, not-found and error states;
  - responsive layout and keyboard-accessible controls.
- **Out of scope**:
  - changes to the backend API or job IDs;
  - editing the configuration directly in the detail;
  - live websocket updates;
  - a status run table separate from the last status snapshot.

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-09-25
- **Decision notes**: The detail uses the read-only API and keeps operations and configuration separate.

## Proposed solution

Extend `web/src/api.ts` with `getCard`, `getRuns` and `getStatusHistory`. Add
`CardDetailView.vue` with local loading of card/status/runs/history via
`Promise.allSettled`, so that a failure of one read-only section does not
bring down the others. Duration values from Go JSON are nanoseconds and the
UI converts them to a readable duration. Output goes in a `<details>`
element so that long output does not break scanning.

## Test plan

- `npm run format`, `npm run lint`, `npm run build`.
- Browser smoke test of the detail with empty history and with runs/status history.
- Verify `/cards/missing` as a not-found/error state.
- Verify keyboard focus on back, action buttons and the output disclosure.
- Verify the 320px layout without horizontal overflow.

## Done criteria

The detail uses the real read-only endpoints, shows the transition history as
a timeline, does not show technical raw labels as the main text, and no
section hides the whole detail because of an isolated error.

## Closure

- **Status after implementation**: Done
- **Verification**: `npm run format`, `npm run lint -- --quiet`, `npm run build` — successful; browser detail smoke test and 320px responsive test — successful
- **Documentation updated**: yes; roadmap and UX specification
