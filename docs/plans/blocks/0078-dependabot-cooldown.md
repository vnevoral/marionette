# Implementation block: Dependabot cooldown

- **Phase**: maintenance of the public repository
- **Requirements**: NFR-08 (maintainability)
- **ADRs**: none (dependency policy in `docs/devops/ci-cd.md`)
- **Status**: Done
- **Dependencies**: none

## Goal

The project takes only dependency versions released at least two weeks
earlier, but Dependabot proposed fresh ones: PRs #23 (`happy-dom` 21.0.3)
and #31 (`@lucide/vue` 1.59.0) had to be closed by hand. Dependabot
should propose only versions that meet the rule, and the rule should be
written down.

## Scope

- **In scope**:
  - `cooldown: default-days: 14` for Go modules, npm and GitHub Actions
    in `.github/dependabot.yml`;
  - the age rule and its exception for security fixes in
    `docs/devops/ci-cd.md` ("Dependency versioning") and AGENTS.md;
  - roadmap.
- **Out of scope**:
  - updating any dependency.

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-10-20
- **Decision notes**: The owner approved adding the cooldown after the two
  Dependabot PRs were closed for being too new.

## Proposed solution

- Dependabot's `cooldown` delays version updates until the release is
  `default-days` old. Security updates are not delayed by it, which matches
  the exception for security fixes.

## Test plan

- GitHub validates `.github/dependabot.yml` after the merge; the next
  weekly run proposes no version younger than 14 days.
- `make verify` passes.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md).

## Closure

- **Status after implementation**: Done (2026-10-20)
- **Verification**: `make verify` passed. The configuration is checked
  by GitHub once it is on `main`.
- **Deviations from the plan**: none.
- **Documentation updated**: `docs/devops/ci-cd.md`, AGENTS.md, roadmap.
