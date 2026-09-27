# ADR-0010: Our own layout utilities instead of PrimeFlex

- **Status**: Accepted
- **Date**: 2026-09-26

## Context

ADR-0009 kept PrimeFlex 4.0.0 as a frozen layout layer for the MVP, with
the decision to be reopened after Phase 8. Phase 8 is closed and the
analysis of 2026-09-26 showed:

- PrimeFlex 4.0.0 (February 2025) is the last version; PrimeTek no longer
  develops it and recommends Tailwind CSS with the `tailwindcss-primeui`
  plugin.
- The Marionette code uses **14 utility classes** in roughly 60 occurrences
  (`grid`, `col-12`, `md:col-4`, `md:col-6`, `lg:col-4`, `flex`,
  `flex-column`, `flex-wrap`, `align-items-center`,
  `justify-content-between`, `gap-2`, `gap-3`, `p-2`, `mt-4`).
- PrimeFlex adds 446 kB of CSS without tree-shaking; the whole CSS bundle
  is 368 kB minified.
- PrimeVue 5.0.0 (July 2026) and 5.0.1 (August 2026) moved from MIT to the
  commercial **PrimeUI License** with an offline license key
  (`@primeui/license-manager`); the free Community license applies only to
  organizations under USD 1 million in revenue, fewer than 5 developers and
  fewer than 10 employees, and requires yearly renewal. PrimeVue 4.5.5
  remains MIT under the `v4-stable` dist-tag. Future compatibility of
  PrimeFlex with PrimeVue 5, which ADR-0009 listed as a trigger, is
  therefore not relevant.

## Decision

PrimeFlex is removed from the project. The layout utilities the code uses
are defined by our own file `web/src/styles/layout.css` with **the same
class names and the same values** (including the 768 px and 992 px
breakpoints), so the templates remain unchanged and the visual result is
identical. New utility classes are added only to this file and only when a
view actually needs them; there is no generated "just in case" set.

PrimeVue stays on the **v4 (MIT)** branch; upgrading to PrimeVue 5 would
mean a license key in the build and, for deployments outside the Community
criteria, a paid per-developer license, which makes no sense for a small
self-hosted tool. This decision supplements ADR-0003.

## Considered alternatives

- **Tailwind CSS + `tailwindcss-primeui`** — actively maintained, the
  official PrimeTek direction. Rejected: for 14 classes it brings a new
  build plugin, a new class vocabulary and a rewrite of all templates
  without any user benefit. It can be considered if the frontend grows
  significantly.
- **Keep PrimeFlex frozen (ADR-0009)** — functional, but permanently
  carries 446 kB of dead CSS and an unmaintained dependency. Rejected.
- **Upgrade to PrimeVue 5** — changes the license and requires a license
  key. Rejected for the MVP and further phases until the licensing model or
  the project's needs change.

## Consequences

- The `primeflex` dependency disappears from `package.json`, the lockfile
  and the Dependabot ignore list; the CSS bundle shrinks by hundreds of kB.
- `layout.css` is under the project's control: changes to values or
  breakpoints are deliberate and visible in review.
- Dependabot still ignores major versions of `primevue` and
  `@primevue/themes`; moving to PrimeVue 5 requires a new ADR with a
  licensing assessment.
- ADR-0009 is superseded by this ADR; `docs/devops/ci-cd.md`, requirements
  section 13 and the UX specification refer to this ADR.
