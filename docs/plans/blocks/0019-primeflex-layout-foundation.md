# Implementation block: PrimeFlex layout foundation

- **Phase**: 9 — UX redesign and shared design system
- **Requirements**: FR-25, FR-29, NFR-08, NFR-09
- **ADRs**: ADR-0003, ADR-0007
- **Status**: Done
- **Dependencies**: Blocks 0015–0018, PrimeVue 4

## Goal

The project uses PrimeFlex as the standard utility layer for layout, grid,
flex, spacing and responsive breakpoints. Custom CSS remains only for
product compositions and tokens, not for repeated general margin/padding
rules.

## Scope

- **In scope**:
  - installing `primeflex@4`;
  - loading `primeflex/primeflex.css` globally;
  - documenting the boundary between PrimeFlex layout utilities and our own tokens;
  - keeping the PrimeVue theme and the green-first semantic tokens.
- **Out of scope**:
  - a complete rewrite of all existing views to utility classes;
  - changing the PrimeVue major version;
  - changing the color palette or the API.

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-09-25
- **Decision notes**: PrimeFlex is the mandatory layout utility layer for further UX blocks.

## Proposed solution

PrimeFlex is loaded globally in `main.ts`. New and refactored views use
utility classes for grid/flex/spacing/responsive layout. CSS tokens remain
for semantic colors, typography, radius, focus and product components.

## Test plan

- `npm install` completes without a dependency conflict.
- `npm run format`, `npm run lint -- --quiet`, `npm run build`.
- A browser smoke test verifies that the PrimeFlex CSS is available and the application loads.

## Done criteria

PrimeFlex is available in the bundle, loaded globally and documented; no new
layout change adds another utility framework.

## Closure

- **Status after implementation**: Done
- **Verification**: `npm install primeflex`, `npm run format`, `npm run lint -- --quiet`, `npm run build` — successful
- **Documentation updated**: yes; roadmap and ADR-0003
