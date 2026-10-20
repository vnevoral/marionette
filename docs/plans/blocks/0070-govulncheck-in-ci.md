# Implementation block: `govulncheck` in CI

- **Phase**: maintenance of the public repository
- **Requirements**: NFR-04, NFR-06 (reliability, maintainability); no
  product requirement changes
- **ADRs**: none (a check, not a dependency of the binary)
- **Status**: Done
- **Dependencies**: Block 0066 (Go 1.27.3, toolchain policy)

## Goal

CI reports known vulnerabilities in the Go code and the Go standard
library it calls, the way `npm audit` does for the web part. Block 0066
found an outdated Go in the dev container only by chance.

## Scope

- **In scope**:
  - a `govulncheck ./...` step in the `backend` job of `ci.yml`, run with
    `go run golang.org/x/vuln/cmd/govulncheck@v1.8.0` (pinned; with the Go
    toolchain from `setup-go`, so it checks the Go that builds the
    release);
  - `make vulncheck` with the same command for a local run; not part of
    `make verify` because it needs the online vulnerability database
    (like `npm audit`, which also runs only in CI);
  - `docs/devops/ci-cd.md` (backend job, version pinning), `AGENTS.md`
    (command list), roadmap.
- **Out of scope**:
  - a scheduled run on `main` between pushes (a new advisory shows up on
    the next pull request or release, as with `npm audit`);
  - SARIF upload or code scanning alerts.

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-10-20
- **Decision notes**: Backlog item approved as proposed; the owner asked
  to continue with the approved plan.

## Proposed solution

`ci.yml`, job `backend`, after `go test`:

```yaml
- run: go run golang.org/x/vuln/cmd/govulncheck@v1.8.0 ./...
```

`govulncheck` reports only vulnerabilities in code the module actually
calls and exits non-zero when it finds one, which fails the job. v1.8.0 is
the newest release older than two weeks (v1.9.0 is from 2026-10-11); the
version is raised together with golangci-lint.

## Test plan

- Local: `make vulncheck` with Go 1.27.3 prints "No vulnerabilities
  found." and exits 0.
- The pull request's `backend` job runs the step and passes.
- `make verify` passes.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md).

## Closure

- **Status after implementation**: Done (2026-10-20)
- **Verification**: `make vulncheck` with Go 1.27.3 printed "No
  vulnerabilities found." (database updated 2026-10-20); the pull
  request's `backend` job ran the step and passed; `make verify` passed.
- **Deviations from the plan**: none.
- **Documentation updated**: `docs/devops/ci-cd.md` (backend job, local
  suite, version pinning), `AGENTS.md`, roadmap.
