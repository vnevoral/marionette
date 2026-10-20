# Implementation block: Changes reach `main` through pull requests

- **Phase**: maintenance of the public repository
- **Requirements**: NFR-05, NFR-06 (testability, maintainability); no
  product requirement changes
- **ADRs**: none (repository process, not architecture)
- **Status**: Approved
- **Dependencies**: none; roadmap backlog item 1 (decided 2026-10-20)

## Goal

Every change, including the owner's and an AI agent's, reaches `main`
through a pull request whose three CI checks (`backend`, `web`, `e2e`)
passed. Today the "Protect main" ruleset requires the checks, but the
repository admin role bypasses it `always`, so direct pushes to `main`
skip CI before the change lands.

## Scope

- **In scope**:
  - the "Protect main" ruleset (GitHub settings, applied with `gh api`):
    - add a `pull_request` rule with 0 required approvals (the owner is
      the only maintainer and cannot approve their own pull request);
    - change the admin bypass from `always` to `pull_request`, so that
      direct pushes are refused while the owner can still merge a pull
      request in an emergency (for example a CI outage);
    - keep the existing rules (no deletion, no force push, the three
      required checks);
  - merge methods: allow only squash merge, with the pull request title as
    the commit message (keeps the linear `type: summary` history);
    automatic deletion of the merged branch stays on;
  - documentation of the flow: `AGENTS.md` (agents work on a branch, push
    it and open a pull request; they never push to `main`),
    `docs/devops/development-workflow.md` (implementation and closure go
    through a pull request), `docs/devops/ci-cd.md` (ruleset, merge
    method; a release tag is created on an up-to-date `main` after the
    merge), `CONTRIBUTING.md` (the same rules apply to the maintainer);
  - roadmap: remove backlog item 1 and link this block.
- **Out of scope**:
  - required reviews or CODEOWNERS approval (single maintainer);
  - "require branches to be up to date" (`strict` checks): with squash
    merge and a single maintainer it only adds rebases;
  - tag protection or changes to the release workflow (backlog item 2);
  - merge queues.

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-10-20
- **Decision notes**: Squash merge only; the admin bypass stays for pull
  requests only.

## Proposed solution

1. Update the ruleset with `gh api -X PUT
   repos/vnevoral/marionette/rulesets/24077459`, keeping `deletion`,
   `non_fast_forward` and `required_status_checks`, adding
   `{"type": "pull_request", "parameters":
   {"required_approving_review_count": 0, ...}}` and setting the
   `RepositoryRole` 5 (admin) bypass to `"bypass_mode": "pull_request"`.
2. `gh repo edit --enable-merge-commit=false --enable-rebase-merge=false
   --enable-squash-merge` and set the squash commit title to the pull
   request title (`squash_merge_commit_title=PR_TITLE`,
   `squash_merge_commit_message=PR_BODY` or `BLANK`).
3. Update the documents listed in the scope; the block itself is merged
   through a pull request under the new rules.

## Test plan

- `gh api repos/vnevoral/marionette/rules/branches/main` lists the
  `pull_request` rule and the required checks; the ruleset shows the admin
  bypass as `pull_request`.
- A direct `git push origin main` with a commit is refused by GitHub
  (tested with a throwaway commit that is reset locally afterwards; the
  refusal means it never reaches `main`).
- The pull request for this block shows the three required checks and
  offers only squash merge; after the merge `main` has one commit with the
  pull request title.
- `make verify` passes (documentation-only change in the tree).

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md) plus:

- the ruleset and merge settings above are in place and verified;
- no document still describes pushing directly to `main`.

## Closure

- **Status after implementation**:
- **Verification**:
- **Documentation updated**:
