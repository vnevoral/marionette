# Implementation block: `@primevue/themes` → `@primeuix/themes`

- **Phase**: maintenance of the public repository
- **Requirements**: NFR-06 (maintainability), NFR-08 (visual consistency,
  unchanged)
- **ADRs**: ADR-0003, ADR-0010 (PrimeVue stays on v4, MIT)
- **Status**: Done
- **Dependencies**: none

## Goal

The UI theme is built from the maintained `@primeuix/themes` package
instead of the deprecated `@primevue/themes`, with no visible change.
`@primevue/themes` 4.5.4 is only a wrapper over `@primeuix/themes` 2.x
and npm marks it "no longer maintained".

## Scope

- **In scope**:
  - `web/package.json`: replace `@primevue/themes` with
    `@primeuix/themes` `^2.0.3`, the 2.x line PrimeVue 4 itself depends on
    (MIT); `package-lock.json` updated;
  - `web/src/theme/preset.ts`: import `definePreset` from
    `@primeuix/themes` and Aura from `@primeuix/themes/aura`;
  - the ADR-0003 and ADR-0010 notes that name the package;
  - `.github/dependabot.yml`: the major-version ignore moves from
    `@primevue/themes` to `@primeuix/themes`, because 3.x (`latest`) is the
    PrimeVue 5 line under the PrimeUI license, not MIT;
  - `docs/devops/ci-cd.md` if it names the package, roadmap.
- **Out of scope**:
  - `@primeuix/themes` 3.x or PrimeVue 5 (backlog item on PrimeVue 4
    support);
  - any change to the preset's tokens or the look of the UI.

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-10-20
- **Decision notes**: Backlog item approved as proposed; the owner asked
  to continue with the approved plan. Staying on the MIT 2.x line follows
  the licence discussion of the same day.

## Proposed solution

`npm uninstall @primevue/themes && npm install @primeuix/themes@^2.0.3`
in `web/`, then the two import lines. `npm ls @primeuix/themes` must show
a single deduplicated 2.x copy used by both the preset and PrimeVue.

## Test plan

- `make verify` (type-check, lint, Vitest, build) passes.
- `make e2e` passes (the theme affects every screen; the 320 px and
  contrast-sensitive scenarios run against the built binary).
- The built CSS/JS bundle size stays about the same (`vite build`
  output), as a sign the same theme code is bundled.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md).

## Closure

- **Status after implementation**: Done (2026-10-20)
- **Verification**: `npm ls @primeuix/themes` shows one 2.0.3 copy and no
  `@primevue/themes`; `make verify` passed; `make e2e` passed (21
  tests). The bundle stays the same size: CSS 35.61 kB before and after,
  JS 611.65 kB → 611.56 kB (150.00 → 150.15 kB gzip).
- **Deviations from the plan**: `@primeuix/themes` types the button
  tokens, the old wrapper did not; `vue-tsc` rejected
  `button.label.fontWeight`, which belongs under `button.root`. The preset
  now sets `button.root.label.fontWeight` (the CSS variable
  `button.label.font.weight` is the same, and "500" equals Aura's
  default, so nothing changes visually). Short notes added to ADR-0003
  and ADR-0010.
- **Documentation updated**: ADR-0003, ADR-0010 (notes), roadmap.
