# Implementation block: Dependency and toolchain update (October 2026)

- **Phase**: maintenance of the public repository
- **Requirements**: NFR-04, NFR-05 (maintainability, testability)
- **ADRs**: none (no new major version of a key dependency)
- **Status**: Done
- **Dependencies**: Block 0060 (toolchain policy)

## Goal

Bring the toolchain and the npm dependencies up to date and supersede the
open Dependabot pull requests #13–#17, without changing behavior.

## Findings before the update

- `govulncheck` with the dev container's Go 1.27.1 reported 13 standard
  library vulnerabilities (11 reachable, mostly `net/http`), all fixed in
  Go 1.27.2. CI and the release build already use the latest 1.27 patch
  release: `v1.3.1` was built with Go 1.27.2, and `govulncheck` with Go
  1.27.2 and 1.27.3 reports no vulnerabilities, so the deployed binary is
  not affected.
- `npm audit`: no vulnerabilities. Newer versions within the declared
  ranges for 13 packages, including the five open Dependabot PRs.
- golangci-lint v2.15.0 is available (v2.14.0 pinned); it reports one new
  `errcheck` finding in a test.

## Scope

- **In scope**:
  - dev container: Go 1.27.3 (checksums from go.dev), golangci-lint
    v2.15.0 (gofumpt stays v0.12.0, the version bundled in v2.15.0);
  - CI: golangci-lint v2.15.0; `setup-go` with `check-latest: true` in
    `ci.yml` and `release.yml` (found after the first push: without it CI
    used the runner's cached Go 1.27.2, not 1.27.3). On 2026-10-20 the
    `actions/go-versions` manifest that `setup-go` reads ends at 1.27.2,
    so CI still builds with 1.27.2 (no known vulnerabilities) and picks
    up 1.27.3 once it is published;
  - `web/`: `npm update` within the ranges and the ranges raised to the
    installed versions (vue 3.5.44, vue-router 5.4.0, vite 8.3.4, vitest
    5.0.5, eslint 10.13.0, Playwright 1.64.1, ...);
  - the `errcheck` finding: `TestRunShutsDownQuicklyWithOpenSSEClientAndSavesHistory`
    signals the end of the stream by closing a channel instead of
    discarding an error.
- **Out of scope** (each needs its own decision):
  - TypeScript 7 (major; ci-cd.md asks for a changelog check and an ADR);
  - PrimeVue 5 (commercial license, ADR-0003/ADR-0010);
  - happy-dom 21 (major of a test dependency);
  - `@primevue/themes` is deprecated in favor of `@primeuix/themes`.

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-10-20
- **Decision notes**: Requested by the owner: update the project and its
  libraries after Dependabot findings.

## Closure

- **Status after implementation**: Done (2026-10-20)
- **Verification**: with Go 1.27.3 and golangci-lint v2.15.0, `make verify`
  passed (208 Vitest tests, `go test -race`, lint, `install_test`,
  `release_version_test`) and `make e2e` passed (21 scenarios);
  `govulncheck ./...` and `npm audit` report nothing. The Vite chunk size
  warning (index chunk about 610 kB) predates the update.
- **Deviations from the plan**: none.
- **Documentation updated**: roadmap, `docs/devops/ci-cd.md`.
