# Implementation block: Dependency and happy-dom security update

- **Phase**: maintenance of the public repository
- **Requirements**: NFR-08 (maintainability)
- **ADRs**: none; the happy-dom major upgrade is already on the approved roadmap backlog
- **Status**: Done
- **Dependencies**: Blocks 0078 (dependency cooldown) and 0081 (dependency update and TypeScript 6)

## Goal

Update eligible npm dependencies and apply the available high-severity security fix for the test DOM. TypeScript remains on version 6; TypeScript 7 and unrelated dependency changes are out of scope.

## Scope

- **In scope**:
  - `@lucide/vue` 1.59.0, the newest release at least two weeks old;
  - `happy-dom` 21.0.4, which fixes the reported high-severity vulnerabilities; security fixes are exempt from the cooldown.
- **Out of scope**:
  - TypeScript 7 until `vue-tsc` and `typescript-eslint` support it;
  - npm versions newer than the two-week window unless needed for a security fix;
  - unrelated moderate advisories in development-only transitive dependencies.

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-10-31
- **Decision notes**: The owner requested library and toolchain updates. The roadmap already approves adopting happy-dom 21 after the cooldown; the high-severity fix is taken immediately under the security exception.

## Proposed solution

Update the two dependency ranges and `web/package-lock.json` with npm. Do not change application behavior or toolchain configuration.

## Test plan

- `make verify`
- `npm audit --omit=dev --audit-level=high`
- Confirm the high-severity happy-dom advisory is absent; record any remaining out-of-scope development-only advisories.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md) plus successful verification above.

## Closure

- **Status after implementation**: Done
- **Verification**: `TMPDIR=/home/vscode/.cache make -C /workspaces/marionette verify` passed; `npm audit --omit=dev --audit-level=high` found zero vulnerabilities. The high-severity happy-dom advisory is fixed. Full `npm audit` still reports 18 moderate advisories in development-only transitive dependencies; `npm audit fix --dry-run` only offered incompatible forced downgrades, so these remain out of scope.
- **Deviations from the plan**: An E2E run was attempted but Chromium could not launch because `libglib-2.0.so.0` is absent from the container. E2E is not required by the Definition of Done for this dependency-only change, which does not alter a UI flow or HTTP contract.
- **Documentation updated**: roadmap.
