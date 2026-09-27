# CI/CD

## CI (GitHub Actions, `.github/workflows/ci.yml`)

On every push and pull request:

1. **backend** — `golangci-lint` (version pinned identically to the
   devcontainer, configuration `.golangci.yml`), `go build ./...`,
   `go vet ./...`, `go test -race -count=1 ./...` on linux/amd64, `bash -n`
   and a staged test of the install script (`deploy/install_test.sh`) and a
   test of the release version rule (`deploy/release_version_test.sh`).
2. **web** — `npm ci`, `npm audit --omit=dev --audit-level=high`,
   `npm run lint` (`--max-warnings 0`), `npm run format:check`,
   `npm run build` (includes type-check via `vue-tsc`) in `web/`. The Node
   version is set by the `web/.nvmrc` file (identical to the devcontainer
   and `engines`).
3. **e2e** — `npm ci`, `npx playwright install --with-deps chromium`,
   `make e2e` (builds the binary and runs Playwright tests against it,
   block 0041); on failure the report and trace are uploaded as the
   `playwright-report` artifact.

The `ci.yml` workflow can also be called from another workflow
(`workflow_call`); the release workflow below uses it so that a release
passes the same checks.

The same suite (except the `e2e` job, locally `make e2e`) is run locally by
`make verify`; CI only mirrors it. The CI run
must pass cleanly (lint warnings fail the build, they are not suppressed)
before merging into the main branch. Dependabot (`.github/dependabot.yml`)
opens weekly PRs for Go modules, npm and GitHub Actions; major versions of
key UI dependencies are excluded (ADR-0003, ADR-0010).

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

The tag is created and pushed by the project owner. `make release-arm64`
takes the version exclusively from a tag of the form `vMAJOR.MINOR.PATCH`
exactly on `HEAD` (`deploy/release_version.sh`) and refuses to build a
release if `HEAD` has no such tag or the working tree contains uncommitted
changes (including untracked files). A manual override
`VERSION=… make release-arm64` is ignored. Development builds
(`make build`, `make build-arm64`) have no such restriction and take the
version from `git describe`.

### Procedure (release from CI, block 0057)

1. On a clean tree, create a tag: `git tag -a vX.Y.Z -m "…"`.
2. `git push origin vX.Y.Z`. Pushing a tag of the form `v*.*.*` triggers
   `.github/workflows/release.yml`:
   - the `ci` job calls the whole `ci.yml` (backend, web, e2e); if it does
     not pass, no release is created;
   - the `release` job runs `make release-arm64` on a clean checkout with
     tags. `deploy/release_version.sh` rejects a tag of a different form
     (e.g. `v1.2.3-rc1`, which the workflow filter lets through), so no
     release is created;
   - `gh release create` publishes a GitHub Release for the tag with the
     archive `marionette-vX.Y.Z-linux-arm64.tar.gz` and its `.sha256`. The
     notes are generated from the commits. Only this job has the
     `contents: write` permission.
3. On the target host, the archive and the `.sha256` are downloaded from
   GitHub Releases (README, "Installation on Linux with systemd"); the
   reference verification runs on a Raspberry Pi ARM64 with Ubuntu 24.x. In
   the directory with both files, integrity is verified with
   `sha256sum -c marionette-vX.Y.Z-linux-arm64.tar.gz.sha256`.
4. After unpacking, run `sudo ./install.sh ./marionette-linux-arm64`.
   The script installs the unit, keeps the existing configuration and
   performs `systemctl enable` + start/restart of the service.
5. The deployed version is verified via
   `curl -s http://localhost:8080/api/health` (the `version` field must
   match the tag) or in the UI footer (FR-41a).
6. An upgrade uses the same script with the new binary; the data
   (`marionette.json`, `devices.json`) and `/etc/default/marionette` are
   kept, the binary and the unit are overwritten. A rollback runs the
   installer of the previous release (previous releases stay in GitHub
   Releases). The procedure with backup, checks and a rollback across a
   MAJOR version is in the README, section "Upgrading and rolling back".
7. No step requires installing Go, Node.js or any other runtime on the
   target.

Fallback manual procedure (without GitHub or during a CI outage): after
step 1, run `make release-arm64` locally. The archive and the `.sha256` are
created in `bin/` and copied to the host manually; then continue with
step 3.

## Dependency versioning

- Go: `go.mod` — keep on the current stable Go version (see the README for
  the minimum version); raise it cautiously and test the build on arm64.
- npm (`web/`): keep PrimeVue on the stable major branch (`v4-stable`
  dist-tag), see
  [ADR-0003](../architecture/decisions/0003-vue-primevue-frontend.md).
  Before accepting a new major version of any key dependency (Vite, Vue,
  PrimeVue, TypeScript), check the changelog and write/update an ADR if it
  brings breaking changes.
- Layout utilities are the project's own `web/src/styles/layout.css`
  (14 classes, see
  [ADR-0010](../architecture/decisions/0010-own-layout-utilities-replace-primeflex.md));
  PrimeFlex was removed in block 0035. PrimeVue stays on v4 (MIT); version 5
  has a commercial license and requires a new ADR.
