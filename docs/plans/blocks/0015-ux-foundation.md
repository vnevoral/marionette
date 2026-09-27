# Implementation block: UX-01 — Design tokens and AppShell

- **Phase**: 9 — UX redesign and shared design system
- **Requirements**: FR-24, FR-25, FR-26, FR-28, FR-29, NFR-08, NFR-09
- **ADRs**: ADR-0003, ADR-0007
- **Status**: Done
- **Dependencies**: Phase 6, ADR-0007, [UX specification](../../ux/ui-ux-specification.md)

## Goal

The application gets a unified visual foundation: semantic design tokens,
PrimeVue theme mapping, typography, global focus/spacing rules and a shared
AppShell with Overview / Manage cards navigation. The existing content of
the screens stays functionally unchanged; converting them step by step is
in UX-02 and UX-03.

## Scope

- **In scope**:
  - central CSS tokens for canvas, surface, text, border, accent and states;
  - common spacing, radius, focus and body typography rules;
  - PrimeVue Aura semantic overrides without changing the major version;
  - `AppShell` with the application name, active navigation and a responsive layout;
  - navigation to `/` and `/manage` accessible from the keyboard;
  - global English UI labels for the shell.
- **Out of scope**:
  - redesign of the dashboard card content;
  - redesign of the management form;
  - card detail, runs, status timeline and new API;
  - changing PrimeVue, the router or adding a runtime dependency.

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-09-25
- **Decision notes**: The first implementation slice of the UX phase; English is the only UI language and the palette is quiet botanical green-first.

## Proposed solution

Add `web/src/styles/tokens.css` and load it in `main.ts`. AppShell will be
a shared component wrapping `RouterView`; the active route is highlighted
via `RouterLink`. The PrimeVue configuration gets a semantic color/content
surface mapping matching the tokens. Global CSS must not use local hex
values for semantic state colors.

## Test plan

- `npm run format`, `npm run lint`, `npm run build`.
- Manually verify keyboard navigation, the active route and a 320 px / desktop viewport.
- Verify that the original `/` and `/manage` load inside the shell without changing API behavior.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md). Specifically:
both existing screens use the same shell, no navigation disappears on
mobile and the tokens are defined in one place.

## Closure

- **Status after implementation**: Done
- **Verification**: `npm run format`, `npm run lint`, `npm run build` — successful; 320px browser smoke test without horizontal overflow
- **Documentation updated**: yes; roadmap and UX specification
