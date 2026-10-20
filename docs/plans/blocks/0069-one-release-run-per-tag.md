# Implementation block: One release run per tag

- **Phase**: after Phase 7 (release process)
- **Requirements**: FR-06
- **ADRs**: none
- **Status**: Done
- **Dependencies**: Block 0057 (release from CI)

## Goal

A tag push that starts the Release workflow more than once still ends
with one GitHub Release and only green runs. For `v1.2.4` GitHub started
two runs one second apart; the second failed in `gh release create` with
"a release with the same tag name already exists", although the release
itself was fine.

## Scope

- **In scope**:
  - `release.yml`: a `concurrency` group per tag
    (`release-${{ github.ref }}`, `cancel-in-progress: false`), so runs for
    the same tag never publish at the same time;
  - `deploy/publish_release.sh`: publishes the archive and its `.sha256`
    with `gh release create`; if the release already exists, is published
    (not a draft) and has both assets, it prints a notice and succeeds;
    any other existing release (a draft or missing assets) or a `gh` error
    fails the run;
  - `deploy/publish_release_test.sh` with a fake `gh` on `PATH`, run by
    `make test` and the `backend` CI job like `release_version_test.sh`;
  - `docs/devops/ci-cd.md` (release procedure), roadmap.
- **Out of scope**:
  - finding out why GitHub started two runs for one push (no reproducible
    cause; the workflow has a single `push: tags` trigger);
  - skipping the CI job of a duplicate run (it runs, then publishing is
    skipped);
  - tag protection rules.

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-10-20
- **Decision notes**: Backlog item approved as proposed (concurrency group
  per tag, skip an existing release instead of failing); the owner asked
  to continue with the approved plan.

## Proposed solution

`release.yml` gets a workflow-level `concurrency` and its last step calls
`deploy/publish_release.sh "$TAG"` instead of inline `gh release create`.
The script:

1. `gh release view "$TAG" --json isDraft,assets`; on "release not found"
   it runs `gh release create "$TAG" <archive> <archive>.sha256
   --verify-tag --title "Marionette $TAG" --generate-notes`;
2. on an existing published release with both asset names it prints
   `::notice::` and exits 0;
3. otherwise exits non-zero with the reason (draft, missing asset, or the
   `gh` error).

## Test plan

- `deploy/publish_release_test.sh` with a fake `gh` that records its
  arguments: no release → `create` with both files and the flags above;
  published release with both assets → no `create`, exit 0; draft
  release → failure; one asset missing → failure; another `gh view`
  error → failure, no `create`; missing archive file → failure.
- `make verify` passes.
- Manual: the next release tag is published by one green run (recorded at
  that release).

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md).

## Closure

- **Status after implementation**: Done (2026-10-20)
- **Verification**: `bash deploy/publish_release_test.sh` passed (new
  tag, published release skipped, draft, missing asset, `gh` error and
  missing file); the `--jq` format of the fake `gh` matches the real
  `gh release view` for `v1.3.1` and an unknown tag ("release not found",
  exit 1). `actionlint` v1.7.7 reports no issue in `release.yml` and
  `ci.yml`. `make verify` passed. The next release tag still has to be
  published by one green run; recorded with that release.
- **Deviations from the plan**: none.
- **Documentation updated**: `docs/devops/ci-cd.md` (CI backend job,
  release procedure), roadmap (backlog item for PrimeVue 4 support added
  at the owner's request).
