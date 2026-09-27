# Implementation block: Code review fixes 3

- **Phase**: 8 — Hardening (addendum after the phase was closed)
- **Requirements**: FR-22a, FR-21, FR-17
- **ADRs**: ADR-0012 (addendum 2026-09-27)
- **Status**: Done
- **Dependencies**: Blocks 0048, 0050

## Goal

Fix two findings from `/code-review` of commits `9bb21bf..75552ea`:

1. **Multi-line argument in the command editor** (`commandLine.ts`,
   `ActionEditor.vue`). The review reported that the parser rejects a line
   break even inside quotes. Verification showed that the parser accepts a
   line break inside quotes and `parse(format(…))` preserves it; the real
   bug is in the field: `<input type="text">` silently strips the line
   break from the value (`sh -c 'echo a⏎echo b'` is shown as
   `sh -c 'echo aecho b'`) and the first edit of the line would change the
   script without warning.
2. **Waiting for a new run without a baseline** (`CardDetailView.vue`,
   `useCardActivity.ts`). When the run list was still loading or its read
   had failed at the time of clicking **Run action**, the baseline was "no
   run", so the first successful read mistook an old run for a new one and
   the wait ended; the new run appeared only after a page reload.

## Scope

- **In scope**: fix both findings, tests, addendum to ADR-0012 and UX
  specification §7.2.
- **Out of scope**: grammar changes (no new escape sequences), API changes.

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-09-27
- **Decision notes**: Instruction "fix all findings" after `/code-review`.

## Proposed solution

1. `ActionEditor` displays a command line with a line break in a PrimeVue
   `Textarea` (auto-resize) instead of `InputText`; once the field has
   switched, it stays multi-line (no element swap and no loss of focus when
   the line break is deleted). A line break outside quotes remains an
   error.
2. `useCardActivity` tracks whether the run list matches the server (the
   last read for the card succeeded), and `runBaseline()` returns either
   the newest known `startedAt` or `null`. With `null`, the wait does not
   consider any run to be new and refreshes the list for the whole budget
   (`timeoutSec + 30 s`), so the new run always appears.

## Test plan

- Vitest: `ActionEditor` (multi-line argument in a `textarea`, preview,
  the field stays a `textarea`), `commandLine` (line break inside quotes,
  join-back with `\n` and `\r\n`), `cardEditModel` (saving without changes
  preserves a multi-line script), `CardDetailView` (failed and in-progress
  first read of runs: an old run is not considered new, the new one
  appears, reading stops after the budget).
- `make verify`, `make e2e`.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md).

## Closure

- **Status after implementation**: Done (2026-09-27)
- **Verification**: `make verify` passed (179 Vitest tests, `install_test`,
  `release_version_test`, lint, types, `go test -race`); `make e2e` passed
  (17 scenarios). The new tests fail without the fixes (an old run would
  end the wait; `input` would show the line without the line break).
- **Deviations from the plan**: finding 1 was fixed differently than the
  review suggested (the parser was correct, the bug was in the field).
- **Documentation updated**: yes — ADR-0012 (addendum), UX
  specification §7.2, roadmap.
