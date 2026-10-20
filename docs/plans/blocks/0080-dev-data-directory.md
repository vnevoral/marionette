# Implementation block: Dev data directory

- **Phase**: maintenance of the public repository
- **Requirements**: NFR-08 (maintainability)
- **ADRs**: none
- **Status**: Done
- **Dependencies**: 0079 (repository cleanup)

## Goal

The dev backend wrote its data into the repository root: `marionette.json`,
`devices.json` and `marionette.sock`. They are git-ignored but clutter the
root. All local dev data should live in one ignored directory.

## Scope

- **In scope**:
  - `.dev/` for the dev configuration, the paired devices and the control
    socket in the Makefile, `.air.toml` and `.vscode/launch.json`;
    `make dev-config` creates the directory;
  - one `/.dev/` entry in `.gitignore` instead of three files;
  - README and AGENTS.md.
- **Out of scope**:
  - the service defaults and the production paths under
    `/var/lib/marionette`;
  - the e2e data, which stays in a temporary directory.

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-10-20
- **Decision notes**: The owner did not want the dev data files in the
  repository root and approved the `.dev/` directory.

## Proposed solution

- `devices.json` already defaults to the directory of `MARIONETTE_CONFIG`,
  so only the configuration and control socket paths change.

## Test plan

- `make backend-run` creates and uses `.dev/marionette.json`,
  `.dev/devices.json` and `.dev/marionette.sock`; `make pair` reaches the
  backend.
- `make verify` passes.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md).

## Closure

- **Status after implementation**: Done (2026-10-20)
- **Verification**: `make backend-run` served from `.dev/` and `make pair`
  printed a code over `.dev/marionette.sock`; `make verify` passed.
- **Deviations from the plan**: none.
- **Documentation updated**: README, AGENTS.md, roadmap.
