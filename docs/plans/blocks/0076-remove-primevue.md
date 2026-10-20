# Implementation block: remove PrimeVue

- **Phase**: maintenance of the public repository
- **Requirements**: FR-23 (v0.14), NFR-08
- **ADRs**: ADR-0014 (point 7), ADR-0003, ADR-0010
- **Status**: Done
- **Dependencies**: Block 0075

## Goal

The last of the four ADR-0014 blocks: no PrimeTek package is left in the
project, and the documentation describes the new UI stack.

## Scope

- **In scope**:
  - `primevue`, `@primeuix/themes` and `primeicons` uninstalled;
    `web/src/theme/preset.ts` and the PrimeVue setup in `main.ts` removed;
    the unit tests no longer install `primevue/config`;
  - Dependabot: the PrimeVue ignores replaced by major-version ignores for
    `reka-ui` and `@lucide/vue`; `docs/devops/ci-cd.md`;
  - documentation: README (stack and a new dashboard screenshot),
    AGENTS.md, CONTRIBUTING.md, `.github/instructions/vue-frontend.instructions.md`,
    `docs/architecture/overview.md`, testing strategy, requirements table,
    notes in ADR-0003, ADR-0010 and ADR-0014, roadmap.
- **Out of scope**:
  - the release itself and release notes (block 0077).

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-10-20
- **Decision notes**: The owner approved the plan for blocks 0076 and
  0077 and asked for the README dashboard screenshot to be regenerated and
  for the project documentation to be updated after every change.

## Proposed solution

- Remove the packages with `npm uninstall` so `package-lock.json` drops
  them with their transitive dependencies.
- The README screenshot is taken at 1280 × 640 CSS px with a device scale
  factor of 1.5 from a fixture of three cards (two healthy, one with a
  problem).

## Test plan

- `make verify` and `make e2e` pass; `grep` finds no PrimeVue import.
- Visual check of every screen at 1280 px and 375 px: the removed PrimeVue
  base styles do not change anything.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md).

## Closure

- **Status after implementation**: Done (2026-10-20)
- **Verification**: `make verify` passed (Vitest 231 tests); `make e2e`
  passed (21 tests); no PrimeVue, `@primeuix` or `primeicons` import or
  lockfile entry is left. The JS bundle is 114 kB gzip (162 kB before this
  block, 152 kB before ADR-0014). Visual check at 1280 px and 375 px: 12 of
  the 19 screenshots are byte-identical to those of block 0075, and the
  other 7 differ only in their timestamps. The only console errors are the
  expected 401 responses before pairing.
- **Deviations from the plan**: none.
- **Documentation updated**: README (stack, screenshot), AGENTS.md,
  CONTRIBUTING.md, Vue instructions, `docs/architecture/overview.md`,
  testing strategy, CI/CD (Dependabot), requirements table, ADR-0003,
  ADR-0010 and ADR-0014 notes, roadmap.
