# Implementation block: Repository cleanup

- **Phase**: maintenance of the public repository
- **Requirements**: NFR-08 (maintainability)
- **ADRs**: none
- **Status**: Done
- **Dependencies**: 0076 (PrimeVue removed)

## Goal

After the move to Reka UI the repository should hold only what the
project and its maintenance need: no versioned build output, no unused
code paths left from earlier designs, no stale statements in the
documentation and working dev tooling.

## Scope

- **In scope**:
  - the UI build output `internal/webui/dist` is no longer versioned; a
    binary built without `make ui-build` serves `placeholder.html`, and
    `make clean` empties the directory;
  - stale `.gitignore` entries and ESLint ignores for files that no longer
    exist;
  - unused UI code: the `info` and `error` toast tones, the `info` tone of
    `MMessage`, unused tokens and vocabulary entries, and unused Go
    constants;
  - the VS Code launch configuration (server-ready pattern, control
    socket) and the golangci-lint config comment;
  - stale or inaccurate statements in the README, AGENTS.md,
    CONTRIBUTING.md, the architecture overview, the UX specification, the
    glossary, the CI/CD and testing documents, the docs index and the
    roadmap backlog.
- **Out of scope**:
  - completed blocks and accepted ADRs, which record the state at their
    time;
  - local ignored files (`marionette.json`, `devices.json`, build and test
    output);
  - behaviour changes.

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-10-20
- **Decision notes**: The owner asked to clean up the project folder so
  that no stale or invalid files, documents or temporary files remain.

## Proposed solution

- `//go:embed all:dist` keeps working with an empty directory that holds
  only a versioned `.gitkeep`; `webui.Dist()` answers `index.html` with the
  embedded placeholder when the build output is missing. Vite empties the
  directory on build, so a small plugin writes `.gitkeep` back.
- Toasts are success notices only (UX spec §4) and `MMessage` is always an
  error alert, so the other variants and their styles go.

## Test plan

- `internal/webui` tests: the placeholder without a UI build and the built
  `index.html` with one.
- `make clean` leaves only `.gitkeep`; `make verify` restores it after the
  build.
- `make verify` and `make e2e` pass; toasts and inline errors look as
  before.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md).

## Closure

- **Status after implementation**: Done (2026-10-20)
- **Verification**: `make verify` and `make e2e` passed; `make clean`
  leaves only `.gitkeep`.
- **Deviations from the plan**: none.
- **Documentation updated**: README, AGENTS.md, CONTRIBUTING.md,
  `.github/instructions/vue-frontend.instructions.md`, `docs/README.md`,
  architecture overview, UX specification, glossary, `ci-cd.md`,
  `testing-strategy.md`, roadmap.
