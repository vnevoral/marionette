# Implementation block: UX-05 — Visual consistency, accessibility and responsive audit

- **Phase**: 9 — UX redesign and shared design system
- **Requirements**: FR-24..FR-29, NFR-08..NFR-10
- **ADRs**: ADR-0007
- **Status**: Done
- **Dependencies**: Blocks 0015–0020

## Goal

Finish the UX phase so that Overview, Manage cards and Card detail use one
visual and interaction system. Manage cards must not feel like a separate
dark or technical product; all screens must share the same tokens,
typography, surfaces, spacing, status colors and responsive behavior.

## Scope

- unify the page header, content width, vertical rhythm and main actions;
- unify panel/card surfaces, border, radius, shadow and form controls;
- remove local colors, fonts and layout rules that bypass the tokens;
- move Manage cards to the same green-first visual language as Overview;
- unify semantic status/request feedback and status contrast;
- add consistent keyboard focus, labels, live regions and error feedback;
- verify responsive behavior at 320 px, mobile, tablet and desktop;
- add a visual smoke/regression check of the main workflows;
- update the UX specification and the roadmap after the block is closed.

## Out of scope

- a new API contract or a change of the domain model;
- authentication and authorization;
- a change of the PrimeVue major version or adding another UI framework;
- a change of the product palette beyond unifying the existing tokens.

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-09-25
- **Decision notes**: Phase 9 continues by unifying the whole application; it is not only a dashboard redesign.

## Test plan

- `npm run format`, `npm run lint -- --quiet`, `npm run build`;
- browser smoke test of Overview, Manage and Detail at 320 px, 390 px, tablet and desktop;
- verify no horizontal overflow and consistent computed font/background/border tokens;
- verify keyboard tab order, focus visibility and accessible names;
- verify empty, loading, error, save, delete and action feedback states.

## Done criteria

All three main screens use the same page/header/panel/form language,
Manage cards visually belongs to the same application as Overview, no
regular state is communicated by color alone and the responsive/accessibility
smoke tests pass.

## Completion evidence

- `npm run format`, `npm run lint -- --quiet` and `npm run build` in `web/`
  passed.
- A browser smoke test of the form verified field-level validation,
  `aria-invalid`, the unsaved-changes confirmation on navigation and a
  working tab order.
- A responsive check at widths of 320, 390, 768 and 1440 px revealed no
  horizontal overflow or content collisions.
- Loading, error, save feedback and the regular empty-state workflows of the
  main screens were verified as well.
