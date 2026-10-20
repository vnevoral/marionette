# Implementation block: Dependency update and TypeScript 6

- **Phase**: maintenance of the public repository
- **Requirements**: NFR-08 (maintainability)
- **ADRs**: [ADR-0015](../../architecture/decisions/0015-typescript-6.md)
- **Status**: Done
- **Dependencies**: 0078 (Dependabot cooldown)

## Goal

Update the libraries and tools to the newest versions that are at least
two weeks old (`docs/devops/ci-cd.md`).

## Findings before the update

On 2026-10-20 (versions released by 2026-10-06 qualify):

- Go has no module dependencies; Go 1.27.3, golangci-lint v2.15.0,
  gofumpt v0.12.0 and air v1.67.4 are the newest releases.
- npm: every package is on its newest version except three:
  - `typescript` 5.9.3: 6.0.3 and 7.0.2 qualify (ADR-0015);
  - `happy-dom` 21 (21.0.0 from 2026-10-11) and `@lucide/vue` 1.60.0
    (2026-10-20) are too new;
- govulncheck v1.9.0 (2026-10-11) is too new; v1.8.0 stays.
- `npm audit` reports no vulnerabilities.

## Scope

- **In scope**:
  - TypeScript 6.0.3 and the configuration changes from ADR-0015;
  - the roadmap backlog and the state of the reference Raspberry Pi
    (the owner upgraded it to v2.0.0).
- **Out of scope**:
  - TypeScript 7 (ADR-0015), `happy-dom` 21, `@lucide/vue` 1.60 and
    govulncheck v1.9.0 until they qualify; Dependabot proposes the last
    three once they are two weeks old.

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-10-20
- **Decision notes**: The owner asked to upgrade the libraries.

## Test plan

- `make verify` and `make e2e` pass; `npm audit` reports nothing.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md).

## Closure

- **Status after implementation**: Done (2026-10-20)
- **Verification**: `make verify` (231 Vitest tests) and `make e2e`
  (21 tests) passed; `npm audit --omit=dev` found no vulnerabilities; the
  bundle is unchanged (339 kB).
- **Deviations from the plan**: none.
- **Documentation updated**: ADR-0015, `.github/dependabot.yml` comment,
  roadmap.
