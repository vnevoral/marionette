# ADR-0003: Vue 3 + PrimeVue for the frontend

- **Status**: Accepted
- **Date**: 2026-09-25

## Context

We need a UI library for a dashboard with cards, forms for configuring
actions and status indicators (FR-20 to FR-23), without having to build our
own component set from scratch, and with a reasonable size of the resulting
bundle for deployment on weaker hardware.

## Decision

The frontend is Vue 3 (Composition API, `<script setup>`) with the component
library **PrimeVue v4** (the Aura theme from `@primevue/themes`), the layout
utility layer **PrimeFlex v4** and the `primeicons` icons. Routing is handled
by `vue-router`. The build tool is Vite.

The PrimeVue major version is kept on the stable `v4` branch (npm dist-tag
`v4-stable`) — higher major versions are not adopted automatically, but only
after reviewing the changelog/migration and updating this ADR.

## Considered alternatives

- Own minimalist components without a library — rejected, unnecessary
  development and maintenance costs for an admin/dashboard UI.
- Vuetify / Element Plus — not categorically rejected, but PrimeVue offers a
  wide set of components (cards, data tables, form elements, badge/tag for
  states) and good Vue 3 + TS support; chosen as the project's default.
- Tailwind + headless UI — rejected for the MVP because of the higher cost
  of building an own design system; can be considered later as a complement
  to PrimeVue (unstyled mode) if the need arises.

## Consequences

- All new UI components should preferably be composed from PrimeVue elements
  (`Card`, `Button`, `Tag`/`Badge`, `DataTable`, `Dialog`, form inputs)
  instead of writing own HTML/CSS from scratch.
- General layout, grid, flex, spacing and responsive breakpoints are handled
  via PrimeFlex; own CSS is used for product compositions and design tokens.
- An upgrade to PrimeVue v5 (or another future major version) requires a new
  ADR that evaluates the breaking changes and license/package changes (v5
  changes the dependency structure to `@primeuix/*` packages).

> Added 2026-09-26: the decision to keep PrimeFlex as a frozen layout layer
> for the MVP is in [ADR-0009](0009-primeflex-frozen-layout-layer.md).

> Added 2026-09-26 (block 0035): PrimeVue 5.0.x moved from MIT to the
> commercial PrimeUI License with a license key; the project stays on the v4
> branch (MIT, dist-tag `v4-stable`). PrimeFlex was replaced with an own
> utility layer ([ADR-0010](0010-own-layout-utilities-replace-primeflex.md));
> the sentence about PrimeFlex in the Decision and Consequences above is
> thereby superseded.
