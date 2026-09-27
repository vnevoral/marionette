# Implementation block: Repository hygiene, lint and CI

- **Phase**: 8 — Hardening
- **Requirements**: NFR-05, NFR-06
- **ADRs**: ADR-0002, ADR-0003
- **Status**: Done
- **Dependencies**: none (can be implemented first)

## Goal

Once done, the validation commands are unified across the Makefile, CI, DoD
and prompts, lint actually fails on findings, generated files are not
versioned and `go test ./...` does not test `node_modules`.

## Scope

- **In scope**:
  - `web/go.mod` with `module marionette-web-ignore` (no dependencies), so
    that `./...` does not reach into `web/node_modules`; verify `go list ./...`;
  - `tsconfig.node.json`: `noEmit: true`, `tsBuildInfoFile` into
    `node_modules/.tmp`; `git rm --cached web/vite.config.js
    web/vite.config.d.ts web/tsconfig.node.tsbuildinfo`; add
    `web/*.tsbuildinfo`, `web/vite.config.js`, `web/vite.config.d.ts` to
    `.gitignore`;
  - ESLint: `eslint-config-prettier` as the last entry, remove
    `vue/html-indent`, `lint` script `eslint . --max-warnings 0`, new
    `format:check` (`prettier --check .`) and `lint:types`
    (`vue-tsc --noEmit -p tsconfig.app.json`); fix the `tone` prop in
    `StatusBadge.vue` (required + default);
  - `.golangci.yml` with the default set + `gofumpt`, `errcheck`, `unused`,
    `staticcheck`, `govet`, `revive` (exported doc); fix the three current
    findings (unused `actionQueueDependencies`, unchecked
    `scheduler.Stop()` in tests, gofumpt formatting of `actions.go`);
  - Makefile `test` = `go test -race ./...` + `npm test` (after 0030),
    `lint` = `golangci-lint run ./...` + `npm run lint` + `format:check`;
    `verify` = `build + lint + test`;
  - CI: `make lint` in both jobs, `go test -race`, `npm audit --omit=dev
    --audit-level=high`; add `.github/dependabot.yml` (gomod, npm,
    github-actions, weekly);
  - unify the Node version: `web/.nvmrc` = 22, `engines.node >=22`,
    devcontainer feature `node: 22`;
  - `.editorconfig` (tab, LF, trailing whitespace) — the devcontainer already
    recommends the extension;
  - `CLAUDE.md` in the root with a single line pointing to `AGENTS.md`
    (Claude Code outside the VS Code integration does not load AGENTS.md
    automatically);
  - AGENTS.md and `go-backend.instructions.md`: `gofumpt` instead of `gofmt`
    (the devcontainer already uses it), a reference to `make verify` as the
    single validation command; the DoD, workflow and the `implement-block`
    prompt reference `make verify` instead of their own command lists;
  - move `marionette.json` in the root to `deploy/dev-fixture.json`
    (without the `status`/`history` sections), the Makefile/air/launch.json
    copy it to the ignored `./marionette.json` on first run;
  - `LICENSE` (MIT, decided 2026-09-26) and `CODEOWNERS`.
- **Out of scope**:
  - a release workflow to GitHub Releases (block 0023 / a separate block);
  - application code changes beyond fixing lint findings.

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-09-26
- **Decision notes**: Approved as a straightening out of the repository
  state; MIT license. Finding of the 2026-09-26 review (L-1, H4, M5, section
  "Structure and agent setup"). Recommended to implement as the first block
  of phase 8.

## Proposed solution

The changes are configuration only; the only code changes are three lint
fixes and `tone?` in `StatusBadge`. `make verify` becomes the single place
where validation is defined, the other documents reference it (the workflow
forbids a parallel methodology, so the command lists are centralized).

## Test plan

- `go list ./...` returns no package under `web/`;
- `git status` after `npm run build` and `make backend-dev` is clean;
- `npm run lint` fails on one artificially added warning; after `npm run
  format` the warning count is 0;
- `golangci-lint run ./...` passes without findings;
- the CI run on a PR is green; Dependabot opens its first PR.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md) plus:

- `make verify` is documented in the README, AGENTS.md, the DoD and the
  prompts;
- no generated file is in `git ls-files`;
- the Node version is the same in the devcontainer, CI and `.nvmrc`.

## Closure

- **Status after implementation**: Done (2026-09-26)
- **Verification**: `make verify` (UI build, `golangci-lint run ./...`,
  `npm run lint --max-warnings 0`, `npm run lint:types`, `npm run
  format:check`, `go test -race -count=1 ./...`, `go build ./...`,
  `go vet ./...`) passed in 12 s without findings; `go list ./...` returns
  only packages under `cmd/` and `internal/` (none from `web/node_modules`);
  after `npm run build` no `vite.config.js`, `vite.config.d.ts` or
  `tsconfig.node.tsbuildinfo` is created; `git check-ignore marionette.json`
  confirms the local dev configuration is ignored; negative test — an
  artificially added file with an unused variable breaks `npm run lint`
  (exit 1), after removal exit 0; `npm audit --omit=dev --audit-level=high`
  without vulnerabilities; `git diff --check` clean. The number of ESLint
  warnings dropped from 176 to 0 without reformatting the `.vue` files
  (solved by `eslint-config-prettier`).
- **Not verified locally**: a green CI run on a PR and the first Dependabot
  PR (they require a push to GitHub); the devcontainer with Node 22 requires
  a container rebuild (Node 20 still runs locally, `engines` only warns).
- **Deviations from the plan**: `make test` so far runs only the Go tests —
  `npm test` will be added by block 0030; the revive rule `exported` runs
  with the stuttering check disabled (`StatusCheckService`), the renaming is
  handled by block 0034; missing godoc comments on exported errors and types
  were added (originally L-12 for 0034), because lint requires them.
- **Documentation updated**: yes — `AGENTS.md`, `CLAUDE.md`,
  `.github/instructions/*`, `.github/prompts/implement-block.prompt.md`,
  `docs/devops/definition-of-done.md`, `development-workflow.md`, `ci-cd.md`,
  `testing-strategy.md`, `README.md`, roadmap.
