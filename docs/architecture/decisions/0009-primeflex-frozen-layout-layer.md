# ADR-0009: PrimeFlex as a frozen layout layer for the MVP

- **Status**: Superseded by ADR-0010
- **Date**: 2026-09-26

## Context

Blocks 0019 and 0020 introduced PrimeFlex 4 as the mandatory utility layer
for layout (grid, flex, spacing) alongside PrimeVue components and our own
design tokens. The 2026-09-26 project review pointed out that upstream
PrimeFlex is frozen at version 4.0.0 (February 2025), PrimeTek no longer
develops it and recommends Tailwind CSS with the `tailwindcss-primeui`
plugin as a replacement. PrimeFlex also pulls in the complete CSS without
tree-shaking (~370 kB of CSS in the bundle before gzip together with the
Aura theme). The project needs to decide whether to keep building on a
frozen library into phase 8 and beyond, or to migrate.

## Decision

PrimeFlex 4.0.0 remains Marionette's layout layer for the duration of the
MVP as a **frozen but sufficient** dependency. No further utility library
is introduced and there is no migration to Tailwind. Conditions:

- the version is pinned exactly (`4.0.0`, without `^`) and Dependabot does
  not update it;
- new screens use only the subset of classes already used in the project
  (grid, flex, gap, spacing, display, text alignment) — the scope is
  documented in the UX specification;
- all visual customization of PrimeVue components goes through
  `definePreset` and design tokens (block 0033), not through `!important`
  or PrimeFlex overrides;
- the decision is reopened if (a) PrimeVue 5 stops being compatible with
  PrimeFlex and the project wants to move to PrimeVue 5 (see ADR-0003),
  or (b) the CSS bundle size starts measurably slowing down loading on
  the Raspberry Pi (NFR-03).

## Considered alternatives

- **Migrating to Tailwind CSS + `tailwindcss-primeui`** — actively
  maintained, tree-shaking, PrimeTek's official direction. Rejected for
  the MVP: it requires rewriting all three views and components,
  introduces a PostCSS build step and a new class vocabulary with no user
  benefit; it fits as a separate phase after stabilization (phase 8).
- **Removing the utility layer, only scoped CSS + tokens** — fewest
  dependencies, but repeats layout code across views and goes against
  blocks 0019/0020, which removed exactly that duplicated scoped CSS.
  Rejected.
- **Keep PrimeFlex without restrictions** — risk of silently growing the
  dependency on a frozen library. Rejected in favor of the conditional
  freeze above.

## Consequences

- No extra work in phase 8; blocks 0029, 0032 and 0033 build on the
  existing layout.
- The CSS bundle stays larger than it would be with tree-shaking; for a
  LAN deployment and a handful of cards this is acceptable (NFR-03
  measured on the reference host).
- An upgrade to PrimeVue 5 will require revisiting this ADR together with
  ADR-0003.
- `docs/devops/ci-cd.md` (dependency versioning) and the UX specification
  list PrimeFlex as a pinned dependency with a limited set of classes.

> Superseded 2026-09-26: PrimeFlex was removed and replaced by our own
> utility layer, see [ADR-0010](0010-own-layout-utilities-replace-primeflex.md).
