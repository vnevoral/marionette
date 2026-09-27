# Implementation block: UX-02 — Overview and action feedback

- **Phase**: 9 — UX redesign and shared design system
- **Requirements**: FR-20, FR-21, FR-24, FR-25, FR-26, FR-27, FR-28, FR-29, NFR-08..10
- **ADRs**: ADR-0003, ADR-0007
- **Status**: Done
- **Dependencies**: Block 0015, [UX specification](../../ux/ui-ux-specification.md)

## Goal

The Overview will use shared tokens and a clear card hierarchy. The user
tells the last known device state apart from the current request state
and sees local feedback on enqueue, polling refresh, error and an empty
list.

## Scope

- **In scope**:
  - redesign of `/` according to the Overview specification;
  - a stable page header with the card count, refresh and New card;
  - semantic status label + icon + color without relying on color alone;
  - request states `Queued`, `Running`, `Updated`, `Error`;
  - isolated status read errors on a specific card;
  - empty, loading and page-level error states;
  - responsive grid and full-width mobile controls;
  - removal of local colors and font overrides outside the tokens.
- **Out of scope**:
  - card detail and runs/status timeline;
  - a new backend or job ID API;
  - redesign of the management form;
  - localization other than English.

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-09-25
- **Decision notes**: The Overview is the first product screen converted to the design system.

## Proposed solution

Convert `HomeView.vue` to semantic tokens and shared component patterns.
Request feedback stays a local client-side projection, while the device
state will always come from `StatusSnapshot`. Status read failures are
stored by card ID without discarding the loaded list. AppShell will
provide the single `main` landmark; the view will not create a nested
shell.

## Test plan

- `npm run format`, `npm run lint`, `npm run build`.
- Browser smoke test at 320 px and on desktop without horizontal overflow.
- Manually verify empty, loading, status read error and accepted/running feedback.
- Verify keyboard focus on refresh, New card and the action buttons.

## Done criteria

The Overview uses only shared tokens, no state is color alone, a status
error of one card does not hide the other cards, and the main actions
have English, unambiguous labels.

## Closure

- **Status after implementation**: Done
- **Verification**: `npm run format`, `npm run lint`, `npm run build` — successful; 320px browser smoke test without overflow and with a single `main` landmark
- **Documentation updated**: yes; roadmap and UX specification
