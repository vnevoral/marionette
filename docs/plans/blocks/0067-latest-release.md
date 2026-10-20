# Implementation block: Installing the latest release

- **Phase**: after Phase 7 (release process)
- **Requirements**: FR-06 (extended 2026-10-20)
- **ADRs**: none
- **Status**: Done
- **Dependencies**: Block 0057 (release from CI)

## Goal

A consumer installs the most recently published version without looking
up its number. GitHub already marks the newest release as **Latest**
automatically, so the release workflow stays as it is and the README
installs the latest release with `gh release download` without a tag.

## Scope

- **In scope**:
  - FR-06 extended (requirements v0.13);
  - README "Installation on Linux with systemd": "Installing the latest
    release" with `gh release download` without a tag and file patterns
    (no version anywhere); "Installing a specific release, or without the
    GitHub CLI" keeps the `curl` steps with `v` set by hand;
  - `docs/devops/ci-cd.md`, roadmap.
- **Out of scope**:
  - forcing `--latest` in `release.yml` (first added, then removed at the
    owner's request: GitHub's automatic choice is enough);
  - version-less copies of the archive for a fixed
    `releases/latest/download/...` URL (declined by the owner).

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-10-20
- **Decision notes**: Only the README part; GitHub's automatic Latest
  marking is kept.

## Closure

- **Status after implementation**: Done (2026-10-20)
- **Verification**: the README steps for the latest release, run in an
  empty directory up to the installer, downloaded `v1.3.1` (marked Latest
  by GitHub), passed `sha256sum -c` and unpacked `install.sh` and
  `marionette-linux-arm64`.
- **Deviations from the plan**: `--latest` in `release.yml` was added in
  `e6bf2d8` and removed again in `1abb035`. A first README version resolved
  the tag from the GitHub API with `curl`; the owner preferred no version
  lookup at all, so the README uses `gh` (a plain `curl` download of a
  `releases/latest/download/...` URL would need a version-less archive
  name, which was declined).
- **Documentation updated**: requirements v0.13 (FR-06), README,
  `docs/devops/ci-cd.md`, roadmap.
