# Implementation block: PrimeFlex view migration

- **Phase**: 9 — UX redesign and shared design system
- **Requirements**: FR-25, FR-29, NFR-08, NFR-09
- **ADRs**: ADR-0003, ADR-0007
- **Status**: Done
- **Dependencies**: Block 0019, UX-02, UX-03, UX-04

## Goal

Overview, Manage cards and Card detail use PrimeFlex for general layout,
grid, flex, spacing and responsive breakpoints. The views' own CSS stops
repeating general layout rules.

## Scope

- **In scope**:
  - page headers, action groups and panel spacing via PrimeFlex;
  - the Overview card grid via responsive PrimeFlex columns;
  - the Manage list/editor layout via responsive columns;
  - Card detail summary/panels and action groups via PrimeFlex;
  - keeping the green-first tokens and the visual appearance.
- **Out of scope**:
  - changes to the API or the component hierarchy;
  - a new layout framework;
  - removing CSS that handles the genuinely product-specific appearance of
    components.

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-09-25
- **Decision notes**: PrimeFlex is the standard layout utility layer.

## Proposed solution

Use `grid`, `col-*`, `flex`, `align-items-*`, `justify-content-*`, `gap-*`,
`p-*`, `m-*` and responsive variants. Keep local CSS for colors, border,
shadow, typography hierarchy, card rows and semantic feedback.

## Test plan

- `npm run format`, `npm run lint -- --quiet`, `npm run build`.
- Browser smoke test of Overview, Manage and Detail on desktop and at 320 px.
- Verify the absence of horizontal overflow and that keyboard focus is kept.

## Done criteria

General layout CSS for grid/flex/spacing is not duplicated between views;
responsive behavior is defined by PrimeFlex classes and all three screens
visually keep the same shell.

## Closure

- **Status after implementation**: Done
- **Verification**: `npm run format`, `npm run lint -- --quiet`, `npm run build` — successful; browser smoke test Overview/Manage/Detail desktop + 320px without overflow
- **Documentation updated**: yes; roadmap
