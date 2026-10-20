# Implementation block: `primeicons` on the MIT 7.x line

- **Phase**: maintenance of the public repository
- **Requirements**: NFR-06 (maintainability); no product requirement
  changes
- **ADRs**: ADR-0003 (MIT PrimeTek packages only, PrimeVue v4)
- **Status**: Done
- **Dependencies**: Block 0071

## Goal

Every PrimeTek package in the UI bundle is MIT. `primeicons` 8.x
(released 2026-07-15) is under the PrimeUI License, which requires a
license key; the project had depended on `^8` since its first commit, so
the icon font in releases up to v1.3.1 came under that license.

## Scope

- **In scope**:
  - `web/package.json` and the lockfile: `primeicons` `^7.0.0` (MIT; 7.0.0
    is the only 7.x release, dist-tag `v7-stable`);
  - `.github/dependabot.yml`: ignore major versions of `primeicons`;
  - a note in ADR-0003; roadmap.
- **Out of scope**:
  - replacing the icon set (part of the planned PrimeVue replacement,
    ADR-0014);
  - anything about releases already published.

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-10-20
- **Decision notes**: Approved as a small block ahead of the PrimeVue
  replacement.

## Proposed solution

`npm install primeicons@^7.0.0` in `web/`. All 35 icon classes the UI uses
(`grep -rhoE "pi-[a-z-]+" web/src`) exist in the 7.0.0 stylesheet;
`pi-spin` is its animation class.

## Test plan

- The class check above against `primeicons` 7.0.0.
- `make verify` and `make e2e` pass.
- Every installed `primevue`, `primeicons`, `@primevue/*` and `@primeuix/*`
  package reports `MIT` in its `package.json`.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md).

## Closure

- **Status after implementation**: Done (2026-10-20)
- **Verification**: all 35 classes found in `primeicons` 7.0.0; the
  installed PrimeTek packages are all MIT (`primevue`, `@primevue/core`
  and `@primevue/icons` 4.5.5, `primeicons` 7.0.0, `@primeuix/styled`
  0.7.4, `@primeuix/styles` and `@primeuix/themes` 2.0.3,
  `@primeuix/utils` 0.6.4); `make verify` and `make e2e` passed.
- **Deviations from the plan**: none.
- **Documentation updated**: ADR-0003 (note), roadmap.
