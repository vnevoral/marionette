# Implementation block: Toolchain update — supported Go and golangci-lint v2

- **Phase**: after Phase 7 (maintenance of the public repository)
- **Requirements**: NFR-01 (security — a supported toolchain with security
  fixes), NFR-06 (maintainability)
- **ADRs**: none (no architectural change; the single Go binary of
  ADR-0002 stays)
- **Status**: Done
- **Dependencies**: 0031 (lint and CI hygiene), 0057 (release from CI)

## Goal

The project builds, lints and releases with a Go version that still receives
security fixes, and linting uses golangci-lint v2, so the Dependabot update of
`golangci/golangci-lint-action` (PR #3, v6 → v9) can be merged. The
application behavior does not change.

## Scope

In scope:

- Raise `go.mod` from `go 1.23` to the current stable Go release (1.26 at the
  time of writing). Go 1.23 is out of support; release binaries are built with
  the version from `go.mod` (`setup-go` with `go-version-file`), so the
  standard library in the shipped binary gets no security fixes today.
- Dev container: base image `mcr.microsoft.com/devcontainers/go` with the
  same Go version; `air` and `gofumpt` at versions that build with it.
- golangci-lint v2 (pinned, identical in the dev container and CI):
  - migrate `.golangci.yml` to the v2 format (`golangci-lint migrate`), keeping
    the same linters and settings (`errcheck`, `govet`, `ineffassign`,
    `revive` with `disableStutteringCheck`, `staticcheck`, `unused`;
    `gofumpt` as a formatter with `module-path: marionette`);
  - the dev container installs the prebuilt v2 binary (the official install
    script with a pinned version) instead of `go install`, because v2 requires
    a newer Go to build than the project may use;
  - CI uses `golangci/golangci-lint-action@v9` with the same pinned version;
    this supersedes Dependabot PR #3, which is closed after the merge.
- Fix new findings reported by the newer linters and Go vet, if any, without
  changing behavior.
- Update the README (minimum Go version), `CONTRIBUTING.md`, `ci-cd.md`
  (pinned lint version, dependency versioning) and the roadmap.

Out of scope:

- New linters or stricter settings than today.
- Upgrading the frontend toolchain (Node, TypeScript, Vite).
- Changes of application behavior, API or configuration.

## Approval

- **Approved by**: project owner ("pokračuj" after the proposal of blocks
  0060 and 0061)
- **Approval date**: 2026-09-27
- **Decision notes**: implemented after 0061.

## Proposed solution

1. `go mod edit -go=1.26` (and `toolchain` only if needed); `go mod tidy`.
2. `.devcontainer/Dockerfile`: `FROM mcr.microsoft.com/devcontainers/go:1-1.26-bookworm`;
   install golangci-lint via
   `curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $(go env GOPATH)/bin vX.Y.Z`.
3. `golangci-lint migrate` on `.golangci.yml`, review the result so that the
   enabled linters and the `revive` rule stay the same; the header comment
   states the pinned v2 version.
4. `ci.yml`: `golangci/golangci-lint-action@v9`, `version: vX.Y.Z`.
5. Run `make verify` and `make e2e`; fix findings.
6. Documentation as listed in the scope.

## Test plan

- `make verify` passes in the rebuilt dev container (Go build, vet, race
  tests, golangci-lint v2, web lint and tests).
- `make e2e` passes.
- `make release-arm64` on a tagged test commit builds the archive (checked
  locally without pushing a tag); the binary starts and `/api/health`
  answers.
- CI on the pull request passes all three jobs with the new lint action.
- Manual: after the next release, the Raspberry Pi upgrade works as before.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md), plus:

- the same Go and golangci-lint versions in `go.mod`, the dev container and
  CI;
- Dependabot PR #3 closed as superseded.

## Closure

- **Status after implementation**: Done (2026-09-27).
- **Result**:
  - `go.mod` (and the `web/go.mod` stub) at `go 1.27`; CI and the release
    build take the latest 1.27 patch release via `go-version-file`;
  - dev container: the `devcontainers/go` image has no Go newer than 1.24,
    so the Dockerfile keeps `1-1.24-bookworm` as the base and replaces its
    Go with the official `go1.27.1` archive verified against the go.dev
    checksum (amd64 and arm64 hosts); `air` v1.67.4, `gofumpt` v0.12.0
    (the version bundled in golangci-lint v2.14.0), golangci-lint v2.14.0
    as a prebuilt binary;
  - `.golangci.yml` in the v2 format: `default: standard` (errcheck, govet,
    ineffassign, staticcheck, unused), `revive` with the same `exported`
    rule, `gofumpt` as a formatter; the migration boilerplate exclusions
    were dropped;
  - `ci.yml`: `golangci/golangci-lint-action@v9` with `version: v2.14.0`.
- **Verification**:
  - `golangci-lint config verify` and `golangci-lint run ./...` → 0 issues;
  - `make verify` passed on Go 1.27.1 (race tests, lint, 206 Vitest
    tests); `make e2e` passed (20 scenarios);
  - `make release-arm64` in a temporary clone with a test tag `v9.9.9`
    built the archive, `sha256sum -c` OK, the binary is `go1.27.1`,
    `GOARCH=arm64`, `main.version=v9.9.9`;
  - `actionlint` v1.7.12 with no findings;
  - the Dockerfile could not be built here (no Docker in the dev
    container); its steps were run by hand in the container (archive
    download and checksum, `go install` of the tools, the pinned
    golangci-lint install script) and the shell syntax was checked.
- **Deviations from the plan**:
  - Go **1.27** instead of the 1.26 named in the proposal: 1.27 is the
    current stable release (the proposal was wrong about "current"), it
    gets security fixes longer, and golangci-lint v2.14.0 is built with
    it;
  - the base image stays on 1.24 with Go replaced (see above), because no
    newer `devcontainers/go` tag exists;
  - on Go 1.27 `TestRunRequiresAPairedDevice` failed intermittently (2 of
    60 runs: "stop HTTP server: context deadline exceeded"). The test left
    three HTTP response bodies unclosed, so their connections were still
    busy when the server shut down. The test now reads and closes every
    body (`expectStatus`); 150 of 150 runs passed. Production behavior is
    unchanged.
- **Documentation updated**: README, CONTRIBUTING, ci-cd (dependency
  versioning), roadmap.
