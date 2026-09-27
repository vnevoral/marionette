# Implementation block: Named release versions (git tags)

- **Phase**: 7 — Packaging and deployment (addendum after the phase was closed)
- **Requirements**: FR-01, FR-41 (version in `/api/health`), NFR-02
- **ADRs**: ADR-0002; no new ADR needed
- **Status**: Done
- **Dependencies**: Block 0023 (release archive, `VERSION` from
  `git describe`)

## Goal

Today the release version is the commit hash (`9bb21bf`), because the
repository has no tag; neither `/api/health` nor the archive name tells
which version it is and whether it is newer. After this block every
release has a `vMAJOR.MINOR.PATCH` version from a git tag, the archive
carries it in its name, and a release cannot be built from uncommitted
changes by mistake.

## Scope

- **In scope**:
  - `Makefile`:
    - `release-arm64` refuses to build a release if `VERSION` ends with
      `-dirty` (uncommitted changes) or does not match a
      `vMAJOR.MINOR.PATCH` tag exactly on `HEAD`; it prints how to create
      the tag. The development build (`make build`, `build-arm64`) stays
      unrestricted;
    - archive `bin/marionette-<version>-linux-arm64.tar.gz` (e.g.
      `marionette-v1.0.0-linux-arm64.tar.gz`); the archive content and the
      binary name inside do not change, so the installation procedure
      stays the same;
    - a `.sha256` file next to the archive for checking after transfer;
  - versioning rules in `docs/devops/ci-cd.md` (Release process section):
    PATCH = fixes, MINOR = new functionality with backward-compatible
    configuration and API, MAJOR = incompatible change to the
    configuration, API or installation; the project owner creates and
    pushes the tag; procedure
    `git tag -a vX.Y.Z -m "…" && make release-arm64`;
  - README (installation): archive name with the version, `sha256sum -c`
    check, verifying the version via `/api/health`;
  - an automated test of the release rule (a script called from
    `make test`, similar to `deploy/install_test.sh`): rejecting `-dirty`,
    rejecting a version without a tag, accepting `v1.2.3`.
- **Out of scope**:
  - automatic publishing to GitHub Releases or building the release in CI;
  - showing the version in the UI (today it is in `/api/health`);
  - changing the version in `web/package.json` (the package is private
    and does not determine the application version);
  - creating and pushing the first tag — done by the project owner.

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-09-27
- **Decision notes**: Approved by the owner ("agree with the proposals"). First version `v1.0.0` as proposed; the owner creates and pushes the tag.

## Proposed solution

- Version check in the Makefile via `git describe --tags --exact-match
  --match 'v[0-9]*.[0-9]*.[0-9]*'` and `git status --porcelain`; the logic
  lives in a small shell script `deploy/release_version.sh` (returns the
  version or an error with a hint) so that it can be tested without the
  Makefile.
- Without a tag, `release-arm64` ends with an error;
  `VERSION=… make release-arm64` (manual override) is not supported, so
  that the version always matches the tag.

## Test plan

- Script test in a temporary git repository: clean tree with tag
  `v1.2.3` → version `v1.2.3`; uncommitted change → error; commit after
  the tag → error; a tag of another form (`test`) → error.
- Manual: `make release-arm64` without a tag fails with a hint; after
  tagging it creates an archive with the version in its name and a
  `.sha256`; `/api/health` on the Pi returns the tag.
- `make verify`.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md) plus:

- the release archive carries the tag version in its name and in
  `/api/health`, and cannot be built from uncommitted changes.

## Closure

- **Status after implementation**: Done (2026-09-27)
- **Verification**:
  - `bash -n deploy/release_version.sh`, `bash -n deploy/release_version_test.sh` — OK;
  - `bash deploy/release_version_test.sh` — `release_version_test: ok`
    (temporary repository: commit without a tag → error; clean tree with
    annotated tag `v1.2.3` → `v1.2.3`; modified tracked file and new
    untracked file → error; commit after the tag → error; tags `test`,
    `v1.2`, `v1.2.3-rc1` → error; lightweight tag `v1.3.0` → `v1.3.0`;
    directory outside a repository → error; error messages contain the
    hint `git tag -a vX.Y.Z`);
  - `bash deploy/install_test.sh` — `install_test: ok`;
  - `make -n release-arm64` in a working tree with uncommitted changes
    prints the recipe, `make release-arm64` itself ends with an error with
    a hint (verified in a temporary clone); in a temporary clone with tag
    `v1.0.0`, `make -n release-arm64` even with `VERSION=hack` and
    `RELEASE_VERSION=bad` prints a build with `-X main.version=v1.0.0`,
    the archive `bin/marionette-v1.0.0-linux-arm64.tar.gz` and `.sha256`;
  - final verification: `make verify` passed; a full `make release-arm64`
    in a temporary copy of the repository (commit + test tag `v1.0.0`,
    the real repository unchanged) created
    `marionette-v1.0.0-linux-arm64.tar.gz` and `.sha256`,
    `sha256sum -c` → OK, archive content unchanged, the binary carries
    `v1.0.0`, the tree stayed clean after the build. Verification of
    `/api/health` on the Pi will be done by the owner after creating the
    real `v1.0.0` tag.
- **Deviations from the plan**:
  - the version is determined by `git tag --points-at HEAD` filtered by
    the regular expression `^v[0-9]+\.[0-9]+\.[0-9]+$` instead of
    `git describe --exact-match --match` (the glob would let through e.g.
    `v1.2.3-rc1`); with multiple tags on `HEAD` the highest version is
    taken;
  - `release-arm64` no longer depends on `build-arm64` as a prerequisite:
    it first verifies the version and then calls
    `$(MAKE) build-arm64 VERSION=<tag>`, so that the check runs before the
    build and the version in the binary matches the tag;
  - the test is wired in as a new target `make release-test` (part of
    `make test`) and as a step of the `backend` job in
    `.github/workflows/ci.yml`, because CI does not call `make test`.
- **Documentation updated**: `docs/devops/ci-cd.md` (CI backend job,
  versioning rules and release procedure), `README.md` (installation:
  archive with the version, `sha256sum -c`, verifying the version via
  `/api/health`).
