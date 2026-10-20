# CI/CD

## CI (GitHub Actions, `.github/workflows/ci.yml`)

On every push and pull request:

1. **backend** — `golangci-lint` (version pinned identically to the
   devcontainer, configuration `.golangci.yml`), `go build ./...`,
   `go vet ./...`, `go test -race -count=1 ./...` on linux/amd64,
   `govulncheck ./...` (known vulnerabilities in called Go code and the
   standard library, block 0070; locally `make vulncheck`), `bash -n`
   and a staged test of the install script (`deploy/install_test.sh`) and a
   test of the release version rule (`deploy/release_version_test.sh`) and
   of release publishing (`deploy/publish_release_test.sh`).
2. **web** — `npm ci`, `npm audit --omit=dev --audit-level=high`,
   `npm run lint` (`--max-warnings 0`), `npm run format:check`,
   `npm run build` (includes type-check via `vue-tsc`) in `web/`. The Node
   version is set by the `web/.nvmrc` file (identical to the devcontainer
   and `engines`).
3. **e2e** — `npm ci`, `npx playwright install --with-deps chromium`,
   `make e2e` (builds the binary and runs Playwright tests against it,
   block 0041); on failure the report and trace are uploaded as the
   `playwright-report` artifact.

The workflow runs with a read-only `GITHUB_TOKEN` (`permissions: contents:
read`), including for pull requests from forks.

The `ci.yml` workflow can also be called from another workflow
(`workflow_call`); the release workflow below uses it so that a release
passes the same checks.

The same suite (except the `e2e` job, locally `make e2e`, and the online
checks `npm audit` and `govulncheck`, locally `make vulncheck`) is run locally by
`make verify`; CI only mirrors it. The CI run
must pass cleanly (lint warnings fail the build, they are not suppressed)
before merging into the main branch.

### Pull requests and `main`

Changes reach `main` only through pull requests (block 0068). The
"Protect main" ruleset on the default branch:

- requires a pull request (0 approvals, the project has a single
  maintainer) and the `backend`, `web` and `e2e` checks;
- forbids deleting `main` and force pushes;
- lets the admin role bypass the rules only within a pull request (an
  emergency merge, for example during a CI outage); direct pushes are
  refused for everyone.

Pull requests are squash-merged (merge commits and rebase merges are
disabled): the commit subject is the pull request title, the body the
branch's commit messages, and the branch is deleted after the merge.

Dependabot (`.github/dependabot.yml`)
opens weekly PRs for Go modules, npm and GitHub Actions. It proposes only
versions released at least 14 days earlier (`cooldown`, see "Dependency
versioning"), and major versions of key UI dependencies are excluded
(ADR-0003, ADR-0014).

## Release process (verified on Raspberry Pi 2026-09-26, phase 7)

### Release versioning (block 0051)

Every release has a `vMAJOR.MINOR.PATCH` version from a git tag (the first
release is `v1.0.0`):

- **PATCH** — bug fixes without changing the behavior of configuration, API
  and installation;
- **MINOR** — new functionality that is backward compatible (including new
  optional configuration items and new API endpoints);
- **MAJOR** — an incompatible change to the configuration, API or
  installation procedure.

The tag is created and pushed by the project owner, or by the coding agent
when the owner asks for a release; the agent then also writes the release
notes. `make release-arm64`
takes the version exclusively from a tag of the form `vMAJOR.MINOR.PATCH`
exactly on `HEAD` (`deploy/release_version.sh`) and refuses to build a
release if `HEAD` has no such tag or the working tree contains uncommitted
changes (including untracked files). A manual override
`VERSION=… make release-arm64` is ignored. Development builds
(`make build`, `make build-arm64`) have no such restriction and take the
version from `git describe`.

### Procedure (release from CI, block 0057)

1. Write the release notes `docs/releases/vX.Y.Z.md` (format in
   [docs/releases/README.md](../releases/README.md)) and merge them through
   a pull request with the last changes of the release.
2. After the last pull request is merged, update `main`
   (`git switch main && git pull`) and on a clean tree create a tag:
   `git tag -a vX.Y.Z -m "…"`. Tags are not covered by the branch ruleset.
3. `git push origin vX.Y.Z`. Pushing a tag of the form `v*.*.*` triggers
   `.github/workflows/release.yml`:
   - the `notes` job fails at once if `docs/releases/vX.Y.Z.md` is missing
     or empty, before CI and the build (block 0077);
   - the `ci` job calls the whole `ci.yml` (backend, web, e2e); if it does
     not pass, no release is created;
   - the `release` job runs `make release-arm64` on a clean checkout with
     tags. `deploy/release_version.sh` rejects a tag of a different form
     (e.g. `v1.2.3-rc1`, which the workflow filter lets through), so no
     release is created;
   - `deploy/publish_release.sh` publishes a GitHub Release for the tag
     with the archive `marionette-vX.Y.Z-linux-arm64.tar.gz` and its
     `.sha256` (`gh release create`). Runs for the same tag are queued in
     one `concurrency` group, and a run that finds the release already
     published with both assets succeeds without changing it; a draft or a
     release missing an asset fails the run (block 0069).
     GitHub marks the newest release as **Latest** automatically; the
     README installs that release (FR-06). The notes are the file
     `docs/releases/vX.Y.Z.md`, followed by GitHub's generated list of
     changes. Only this job has the `contents: write` permission.
4. On the target host, the archive and the `.sha256` are downloaded from
   GitHub Releases (README, "Installation on Linux with systemd"); the
   reference verification runs on a Raspberry Pi ARM64 with Ubuntu 24.x. In
   the directory with both files, integrity is verified with
   `sha256sum -c marionette-vX.Y.Z-linux-arm64.tar.gz.sha256`.
5. After unpacking, run `sudo ./install.sh ./marionette-linux-arm64`.
   The script installs the unit, keeps the existing configuration and
   performs `systemctl enable` + start/restart of the service.
6. The deployed version is verified via
   `curl -s http://localhost:8080/api/health` (the `version` field must
   match the tag) or in the UI footer (FR-41a).
7. An upgrade uses the same script with the new binary; the data
   (`marionette.json`, `devices.json`) and `/etc/default/marionette` are
   kept, the binary and the unit are overwritten. A rollback runs the
   installer of the previous release (previous releases stay in GitHub
   Releases). The procedure with backup, checks and a rollback across a
   MAJOR version is in the README, section "Upgrading and rolling back".
8. No step requires installing Go, Node.js or any other runtime on the
   target.

Fallback manual procedure (without GitHub or during a CI outage): after
step 2, run `make release-arm64` locally. The archive and the `.sha256` are
created in `bin/` and copied to the host manually; then continue with
step 4.

## Dependency versioning

- Age rule: a new dependency version (Go module, npm package, GitHub
  Action, tool) is taken only when it was released at least two weeks
  earlier, so regressions and compromised releases have time to surface.
  Dependabot enforces it with `cooldown: default-days: 14`; manual updates
  follow it too (check `npm view <package> time`). Security fixes are the
  exception: Dependabot security updates ignore the cooldown, and a fix
  for a known vulnerability may be taken at once (block 0078).
- Go: `go.mod` sets the version (currently Go 1.27, also stated in the
  README); CI and the release build use the latest patch release of it
  (`setup-go` with `go-version-file` and `check-latest: true`; without it
  the runner's cached patch release is used). The dev container pins the same
  version with its checksum (`.devcontainer/Dockerfile`, `GO_VERSION`).
  Stay on a Go release that still receives security fixes (the two newest
  major releases); raise it cautiously and test the build on arm64
  (block 0060).
- golangci-lint: v2, pinned to the same version in the dev container and
  in `ci.yml` (`golangci/golangci-lint-action`); configuration in
  `.golangci.yml`.
- govulncheck: pinned in `ci.yml` and the `vulncheck` target of the
  `Makefile` (`golang.org/x/vuln/cmd/govulncheck@vX.Y.Z`, run with
  `go run`); raise it together with golangci-lint.
- npm (`web/`): Dependabot ignores major versions of Vue, Vite,
  TypeScript, `reka-ui` and `@lucide/vue`. Before accepting a new major
  version of any of them, check the changelog and the licence and
  write/update an ADR if it brings breaking changes (ADR-0003, ADR-0014).
- Layout utilities are the project's own `web/src/styles/layout.css`
  (14 classes, see
  [ADR-0010](../architecture/decisions/0010-own-layout-utilities-replace-primeflex.md));
  PrimeFlex was removed in block 0035 and PrimeVue, `@primeuix/*` and
  `primeicons` in block 0076
  ([ADR-0014](../architecture/decisions/0014-replace-primevue-with-reka-ui.md)).
