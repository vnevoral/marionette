# Implementation block: Installing the latest release

- **Phase**: after Phase 7 (release process)
- **Requirements**: FR-06 (extended 2026-10-20)
- **ADRs**: none
- **Status**: Done
- **Dependencies**: Block 0057 (release from CI)

## Goal

A consumer installs the most recently published version without looking
up its number. GitHub already marks the newest release as **Latest**
automatically, so the release workflow stays as it is and the README shows
how to resolve the tag of the latest release.

## Scope

- **In scope**:
  - FR-06 extended (requirements v0.13);
  - README "Installation on Linux with systemd": the tag of the latest
    release from the GitHub API with `curl`, `gh release download` without
    a tag; another release by setting `v` by hand;
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
- **Verification**: the README commands resolved `v1.3.1` (marked Latest
  by GitHub) from the GitHub API and the downloaded archive passed
  `sha256sum -c`.
- **Deviations from the plan**: `--latest` in `release.yml` was added in
  `e6bf2d8` and removed again in the follow-up commit.
- **Documentation updated**: requirements v0.13 (FR-06), README,
  `docs/devops/ci-cd.md`, roadmap.
