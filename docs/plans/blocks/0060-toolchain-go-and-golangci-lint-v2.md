# Implementation block: Toolchain update — supported Go and golangci-lint v2

- **Phase**: after Phase 7 (maintenance of the public repository)
- **Requirements**: NFR-01 (security — a supported toolchain with security
  fixes), NFR-06 (maintainability)
- **ADRs**: none (no architectural change; the single Go binary of
  ADR-0002 stays)
- **Status**: Proposed
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

- **Approved by**: pending
- **Approval date**: pending
- **Decision notes**: —

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

- **Status after implementation**: —
- **Verification**: —
- **Documentation updated**: —
