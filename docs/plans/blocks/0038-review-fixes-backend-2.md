# Implementation block: Code review 2 fixes — backend (WaitDelay, lenient load, atomic write, store)

- **Phase**: 8 — Hardening (addendum after the phase was closed)
- **Requirements**: FR-11, FR-13, FR-30..33, FR-35, NFR-04, NFR-05
- **ADRs**: ADR-0004, ADR-0005
- **Status**: Done
- **Dependencies**: Blocks 0003, 0025, 0028, 0036

## Goal

After this block, a successful action whose child process holds the
output pipe is recorded as `ok`; a configuration written before the limits
were tightened is loaded instead of quarantined; concurrent configuration
writes cannot corrupt the file; a directory fsync error after a
successful `rename` does not cause an in-memory rollback; the three store
mutations share one rollback protocol.

## Scope

- **In scope** (code review findings 2026-09-26, second round):
  1. `internal/execengine/executor.go`: `exec.ErrWaitDelay` (the process
     exited 0, only an orphaned child held the pipe) is evaluated as exit 0
     according to the output rule; `ProcessErr` remains for diagnostics.
  2. `internal/config/types.go` + `persistence.go`: two-level validation.
     `ValidateEssential()` (both Action and ActionCard) checks what the
     engine needs: non-empty ID and command, `timeoutSec ≥ 1`, a valid
     output rule, non-negative polling. `Validate()` = essential + limits
     (lengths, ID characters, icon, `MaxTimeoutSec`, interval relation) and
     stays at the API boundary (`CreateCard`/`UpdateCard`). `LoadFile`
     quarantines the file only when essential validation fails; a card
     that violates only the limits is loaded with a warning
     (`card violates current limits`) and gets fixed on the first edit.
     The executor validates with `ValidateEssential()`.
  3. `writePersistedFile`: a unique temp file (`os.CreateTemp(dir,
     "<name>.*.tmp")`) while preserving permissions (chmod to the target
     file's mode); `SaveFileWithHistory` takes `persistMu`, so it is
     serialized with the `OnChange` mutation.
  4. A `syncDirectory` error after a successful `rename` is returned as
     `ErrDirectorySync`; `SaveFileWithHistory` still marks the snapshot as
     saved; `openStore` logs it in `OnChange` as a warning and does not
     roll back (the file already contains the change).
  5. `internal/config/store.go`: a common `store.mutate(operation, apply)`
     for `CreateCard`/`UpdateCard`/`DeleteCard` — lock, apply, `changes++`,
     `notifyChange`, on error undo + `changes--` + `PersistenceError`.
- **Out of scope**:
  - configuration format migration; API changes; frontend (block 0039).

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-09-26
- **Decision notes**: Approved ("fix all findings"). Item 3 changes the
  decision of block 0025 (fixed `<path>.tmp`): the reason for the fixed
  name — losing permissions with `CreateTemp` — is covered by the explicit
  `chmod` the code already has; a unique name restores protection against
  two writers.

## Proposed solution

- `executor.go`: `if errors.Is(processErr, exec.ErrWaitDelay) { … evaluate
  as exit 0 }` before the existing `switch`.
- `types.go`: `func (action Action) validate(strict bool) fieldErrors`,
  `Validate()` = `validate(true)`, `ValidateEssential()` = `validate(false)`;
  the same for `ActionCard`.
- `persistence.go`: `ErrDirectorySync`; `LoadFile` logs
  `card violates current limits` with `fields`.
- `main.go` `openStore`: `OnChange` wrapper — `ErrDirectorySync` → `Warn`,
  return nil.

## Test plan

- `TestExecutorTreatsWaitDelayAsSuccess` (a fake process returns
  `exec.ErrWaitDelay`, the output is evaluated by the rule; with a `match`
  rule and non-matching output → `fail`).
- `TestLoadFileKeepsCardOverCurrentLimits` (ID with a dot, `timeoutSec`
  7200, icon `pi pi-Home` → loaded, warning in the log), the existing
  `TestLoadFileInvalidCardReturnsErrorAndEmptyStore` (empty command →
  still `ErrConfigCorrupt`).
- `TestValidateEssentialIgnoresLimits`.
- `TestWritePersistedFileConcurrentWritersKeepFileValid` (N goroutines
  `SaveFile`/`SaveFileWithHistory`, the resulting file is valid JSON).
- `TestSaveFilePreservesExistingPermissions` stays green (chmod).
- `TestStoreKeepsMutationWhenOnlyDirectorySyncFails` (OnChange returns
  `ErrDirectorySync` → the card stays, no `PersistenceError`) — via the
  wrapper in `openStore`: a test in `cmd/marionette`.
- `make verify`.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md) plus:

- `grep -n 'changes--' internal/config/store.go` finds exactly one
  occurrence (in `mutate`);
- `docs/architecture/overview.md` describes the lenient load.

## Closure

- **Status after implementation**: Done (2026-09-26)
- **Verification**: `make verify` passed (lint, `vue-tsc`, Prettier, Vitest,
  `vite build`, `go test -race`, `go vet`). New tests:
  `TestExecutorTreatsWaitDelayAsSuccess` (the output is still evaluated by
  the rule), `TestExecutorRunsActionOverCurrentLimits` (a timeout above
  `MaxTimeoutSec` runs, zero does not), `TestLoadFileKeepsCardOverCurrentLimits`
  (ID with a dot, icon `pi pi-Home`, timeout 3601 → loaded with a warning,
  `UpdateCard` still rejects), `TestValidateEssentialIgnoresLimits` (8
  essential violations), `TestWritePersistedFileConcurrentWritersKeepFileValid`
  (8 × 2 concurrent saves, file loadable, no `*.tmp`),
  `TestPersistOnChangeSavesAndReportsWriteFailures`. The existing
  `TestLoadFileInvalidCardReturnsErrorAndEmptyStore` (empty command →
  `ErrConfigCorrupt`) and `TestSaveFilePreservesExistingPermissions` stay
  green. `grep -c 'changes--' store.go` = 1.
- **Deviations from the plan**: (1) The `ErrDirectorySync` branch in
  `persistOnChange` is not covered by a test — a directory `fsync` failure
  after a successful `rename` cannot be triggered in a test without
  injecting a filesystem; what is covered is that an ordinary write error
  passes through and is not classified as `ErrDirectorySync`. (2) The
  decision of block 0025 about the fixed name `<path>.tmp` is superseded
  by this block (the reason — losing permissions — is handled by the
  explicit `chmod`); block 0025 is not changed retroactively.
- **Documentation updated**: yes — `docs/architecture/overview.md`
  ("Protecting the configuration file" section: essential vs. full
  validation, unique temp file, `persistMu`, `ErrDirectorySync`), roadmap.
