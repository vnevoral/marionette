# Implementation block: Release from CI after pushing a tag

- **Phase**: 7 — Packaging and deployment (addendum after the phase was closed)
- **Requirements**: FR-06, FR-01, FR-04, NFR-02
- **ADRs**: ADR-0002; no new ADR needed
- **Status**: Done
- **Dependencies**: Block 0051 (`deploy/release_version.sh`,
  `make release-arm64`)

## Goal

Today the owner builds a release manually (`make release-arm64`) and
transfers the archive to the Pi from their own computer. The result thus
depends on the local environment and is not published anywhere. After this
block, pushing a `vX.Y.Z` tag is enough: CI runs the tests, builds the
archive with the same command and publishes it as a GitHub Release. The
archive can be downloaded on the Pi directly via a link.

## Scope

- **In scope**:
  - a new workflow `.github/workflows/release.yml`, triggered on push of a
    `v*.*.*` tag:
    - checkout with tags (`fetch-depth: 0`), Go and Node according to
      `go.mod` and `web/.nvmrc`;
    - `make verify` and `make e2e`; on failure the release is not
      published;
    - `make release-arm64`; `release_version.sh` itself rejects a tag of
      any other shape or a dirty tree, so the rules stay in one place;
    - a GitHub Release for the tag with the archive and `.sha256`
      (`gh release create` with `GITHUB_TOKEN`, `contents: write`
      permission only for this workflow). Release notes are generated
      from commits (`--generate-notes`);
  - `docs/devops/ci-cd.md` (Release process): tag → push → workflow; the
    manual `make release-arm64` remains as a fallback procedure;
  - README (installation and upgrade): downloading the archive from GitHub
    Releases (`curl -LO …/releases/download/vX.Y.Z/…`) including
    `.sha256`.
- **Out of scope**:
  - a release for linux/amd64 (the target is the Pi; add when needed);
  - signing archives (cosign, GPG);
  - automatic creation of tags or versions from commits;
  - pre-release versions (`-rc`), which `release_version.sh` deliberately
    rejects.

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-09-27
- **Decision notes**: Approved by the owner ("agree with everything"). The tag and its push are still done by the owner.

## Proposed solution

A separate workflow instead of modifying `ci.yml`: the release has a
different trigger, needs write permission and must not run on pull
requests. It uses the `gh` CLI, which is preinstalled on GitHub runners,
instead of a third-party action. This way no additional dependency with
access to the token is added.

## Test plan

- `actionlint`, if available, otherwise a YAML syntax check.
- Verification on the real repository: the owner pushes a tag (e.g.
  `v1.1.0`), the workflow passes and the Release contains the archive and
  `.sha256`; `sha256sum -c` after download on the Pi → OK; `/api/health`
  returns the tag.
- Negative cases in a fork or a test repository: the tag `v1.2.3-rc1`
  triggers the workflow (the `v*.*.*` filter is a glob), but
  `release_version.sh` rejects it and no Release is created; a tag on a
  commit with a failing test does not create a Release either.
- `make verify`.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md) plus:

- after pushing a `vX.Y.Z` tag, a GitHub Release with the archive and
  `.sha256` is created without any manual step, and the archive can be
  installed on the Pi according to the README.

## Closure

- **Status after implementation**: Done in the repository (2026-09-27);
  verification on a real tag will be done by the owner (the first tag
  after this block, e.g. `v1.1.0`).
- **Verification**: `actionlint` v1.7.7 (`go run`) with no findings for
  `ci.yml` and `release.yml`; `make verify` passed. Building the release
  archive was already verified by block 0051 (`make release-arm64` in a
  temporary clone with a tag); the release job runs the same target on a
  clean checkout, where `npm ci` does not change the tree (`node_modules`
  is ignored). The negative scenarios (tag `v1.2.3-rc1`, failing test)
  cannot be verified without pushing a tag. They will be verified in a
  fork when the owner needs them.
- **Verification on the real repository (owner, 2026-09-27)**:
  - tag `v1.2.0`: the e2e job of the called CI failed (pairing screen at
    320 px with a fallback font) and **no Release was created** — the
    "failing test" negative case verified in practice; fixed by block 0059;
  - tag `v1.2.1`: CI passed and the workflow published the Release with the
    archive and `.sha256` without any manual step;
  - on the reference Raspberry Pi the owner installed `v1.2.1` as an upgrade
    according to the README (data and paired devices kept), tested the
    rollback to the previous release, and runs `v1.2.1` in production.
  - The `v1.2.3-rc1` negative case remains verified only by
    `deploy/release_version_test.sh`.
- **Deviations from the plan**: instead of separate `make verify` and
  `make e2e` steps, the release workflow calls the whole `ci.yml` (new
  trigger `workflow_call`), so the release goes through exactly the same
  jobs as a push to `main` (backend, web, e2e) and the CI rules are in one
  place.
- **Documentation updated**: `docs/devops/ci-cd.md` (CI, the release
  procedure from CI and the fallback manual procedure), README (download
  from GitHub Releases, upgrade).
