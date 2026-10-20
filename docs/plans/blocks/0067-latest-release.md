# Implementation block: Every release is the Latest release

- **Phase**: after Phase 7 (release process)
- **Requirements**: FR-06 (extended 2026-10-20)
- **ADRs**: none
- **Status**: Done
- **Dependencies**: Block 0057 (release from CI)

## Goal

A consumer installs the most recently published version without looking
up its number. The release workflow marks each new GitHub Release as
**Latest** explicitly instead of relying on GitHub's automatic choice, and
the README shows how to resolve the latest tag.

## Scope

- **In scope**:
  - `release.yml`: `gh release create --latest`;
  - FR-06 extended (requirements v0.13);
  - README "Installation on Linux with systemd": the tag of the latest
    release from the GitHub API with `curl`, `gh release download` without
    a tag; another release by setting `v` by hand;
  - `docs/devops/ci-cd.md`, roadmap.
- **Out of scope**: version-less copies of the archive for a fixed
  `releases/latest/download/...` URL (declined by the owner).

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-10-20
- **Decision notes**: Points 1 (explicit Latest) and 2 (README) approved,
  point 3 (version-less asset) declined.

## Closure

- **Status after implementation**: Done (2026-10-20)
- **Verification**: the README commands resolved `v1.3.1` from the GitHub
  API and the downloaded archive passed `sha256sum -c`; `gh release create
  --help` documents `--latest`; `release.yml` parses. The workflow change
  takes effect with the next release tag.
- **Deviations from the plan**: none.
- **Documentation updated**: requirements v0.13 (FR-06), README,
  `docs/devops/ci-cd.md`, roadmap.
