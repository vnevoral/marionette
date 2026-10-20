# Implementation block: hand-written release notes

- **Phase**: maintenance of the public repository
- **Requirements**: FR-06 (installation from GitHub Releases)
- **ADRs**: none (process change in the release workflow, blocks 0057 and
  0069)
- **Status**: Done
- **Dependencies**: Block 0076

## Goal

Every GitHub Release has notes written for the people who run Marionette.
Until now `gh release create --generate-notes` produced only a link to the
commit comparison (for example v1.3.1).

## Scope

- **In scope**:
  - `docs/releases/vX.Y.Z.md` per release and the format guide
    `docs/releases/README.md`;
  - `deploy/publish_release.sh` fails without a non-empty notes file and
    passes it with `--notes-file`. GitHub's generated list of changes
    follows it. A new `--check-notes` mode only checks the file;
  - `.github/workflows/release.yml`: a `notes` job runs that check before CI
    and the build;
  - `deploy/publish_release_test.sh` (missing and empty notes, the check
    mode, the new `gh` arguments);
  - notes for v2.0.0; `docs/devops/ci-cd.md` (procedure and who tags),
    README, roadmap.
- **Out of scope**:
  - the v2.0.0 tag itself (pushed after this block is merged);
  - notes for past releases.

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-10-20
- **Decision notes**: The owner asked for release notes for every release
  and agreed that a release without notes fails. The owner expects the
  agent to cut releases and write the notes.

## Proposed solution

- The check runs in the script, so the local fallback procedure enforces
  it too, and in a separate first job, so a missing file fails within
  seconds instead of after CI.
- A file with only whitespace counts as empty.

## Test plan

- `deploy/publish_release_test.sh` with the fake `gh` (part of
  `make verify`).
- `make verify` passes; the first tag after the merge (v2.0.0) checks the
  workflow end to end.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md).

## Closure

- **Status after implementation**: Done (2026-10-20)
- **Verification**: `make verify` passed, including
  `publish_release_test: ok` with the new cases. The workflow change is
  checked by the v2.0.0 release run.
- **Deviations from the plan**: none.
- **Documentation updated**: `docs/devops/ci-cd.md`, README,
  `docs/releases/`, roadmap.
