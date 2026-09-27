# ADR-0007: Unified UI design system and UX model

- **Status**: Accepted
- **Date**: 2026-09-25

## Context

The first MVP screens use local CSS, different layouts and inconsistent
naming. The dashboard and card management therefore do not feel like a
single application, and the user has a hard time distinguishing the
configuration, the action request and the last known state. Requirements
FR-24 to FR-29 and NFR-08 to NFR-10 call for a common foundation before the
UI is extended further.

## Decision

Marionette gets a shared UI shell and a small project design system on top of
Vue 3 and PrimeVue 4. PrimeVue remains the source of the basic components;
the project layer defines CSS tokens and patterns for layout, typography,
semantic status colors, action states, forms, tables/histories, errors and
confirmation of destructive actions.

Every main flow uses the same feedback model: `idle`, `loading`,
`accepted`, `running`, `success`, `error` and `stale/unknown`, where the last
known device status is visually separated from the state of the HTTP
request. Texts, labels and state names are centralized in a single
vocabulary. The default UI language is English (`en-US`); localization into
Czech is not part of this phase.

## Considered alternatives

- A custom component set outside PrimeVue — rejected; it increases
  maintenance and bypasses the accepted ADR-0003.
- Mere local adjustments of individual views — rejected; it does not solve
  consistency and leads to further deviations.
- Switching to a different UI library or Tailwind — deferred; there is no
  need to change the technology base as long as PrimeVue 4 covers the
  component needs.

## Consequences

- Layout, tokens and state patterns will be shared between the dashboard,
  management and the future card detail.
- Visual changes will be verified at desktop and mobile width and across the
  main workflows, which increases implementation time but reduces UX
  regressions.
- No new runtime dependency or PrimeVue major version change is introduced.
- The default UI language is English; any further localization is a future
  extension.
