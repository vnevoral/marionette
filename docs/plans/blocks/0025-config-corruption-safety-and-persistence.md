# Implementation block: Configuration protection on a corrupted file and persistence consistency

- **Phase**: 8 — Hardening
- **Requirements**: FR-30, FR-31, FR-33, FR-35, NFR-04
- **ADRs**: ADR-0004
- **Status**: Done
- **Dependencies**: Blocks 0002–0003 (store and persistence), 0010 (REST API), 0012 (composition)

## Goal

When done, the application never overwrites an existing configuration file
that failed to load, and the in-memory state stays consistent with the API
response even when writing to disk fails.

## Scope

- **In scope**:
  - `config.LoadFile` distinguishes, via sentinel errors, "file does not
    exist" from "file is unreadable/corrupted";
  - with a corrupted file, `main` renames the file to
    `<path>.corrupt-<RFC3339 time>` before the first write, logs a warning
    and only then starts with an empty store (FR-33 stays satisfied);
  - rollback of the in-memory mutation in `Store`
    (`Create/Update/Delete/UpdateStatus`) if `OnChange` returns an error;
    the API then returns 500 and the in-memory state matches the disk;
  - sentinel errors `config.ErrNotFound`, `config.ErrAlreadyExists`,
    `config.ErrValidation`; the handler maps them via `errors.Is`, not by
    comparing text;
  - atomic write: a fixed temp file name `<path>.tmp`, preserving the
    permissions of the original file (`Stat` + `Chmod`), directory `fsync`
    after `rename`;
  - `loadStatus` validates `StatusSnapshot.State` the same way as
    `UpdateStatus`.
- **Out of scope**:
  - configuration format migration, schema versioning;
  - a read-only API mode with a corrupted config (renaming was chosen, not
    blocking);
  - encryption or backups of the configuration beyond a single
    `.corrupt-*` file.

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-09-26
- **Decision notes**: Approved as a bug fix. Finding from the 2026-09-26 review (H-3, M-3, M-4, L-2,
  L-3). Reproduced: `CreateTemp` + `Rename` changes permissions `0644` to `0600`.

## Proposed solution

- `internal/config/persistence.go`: `var ErrConfigMissing`, `ErrConfigCorrupt`;
  `LoadFile` wraps errors `fmt.Errorf("…: %w", ErrConfigCorrupt)`. New
  function `QuarantineFile(path string, now time.Time) (string, error)`.
- `cmd/marionette/main.go`: `switch { case errors.Is(err, ErrConfigMissing):
  info log; case errors.Is(err, ErrConfigCorrupt): QuarantineFile + warn }`.
- `internal/config/store.go`: a mutation under `mu` saves the previous
  value and restores it after a failed `OnChange` (under the same lock or
  via the `persistMu` sequence — keep today's separation of locks).
  `ErrAlreadyExists` returns `fmt.Errorf("card %q: %w", id, ErrAlreadyExists)`.
- `internal/server/server.go`: `errors.Is(err, config.ErrAlreadyExists)` →
  409, `config.ErrNotFound` → 404, `config.ErrValidation` → 422.
- Atomic write: `writeFileAtomic(path, data)` with `path+".tmp"`,
  permissions from `os.Stat(path)` (fallback `0600`), `dir.Sync()` after
  rename.

## Test plan

- Unit tests (`internal/config`):
  - corrupted JSON → `errors.Is(err, ErrConfigCorrupt)`; nonexistent →
    `ErrConfigMissing`; EACCES → `ErrConfigCorrupt` (unreadable);
  - `QuarantineFile` renames and returns the new path; a repeated call in
    the same second does not collide (suffix or `CreateTemp` pattern);
  - `OnChange` returns an error → `Get` after `Create` returns
    `ErrNotFound`, after `Update` the original value, after `Delete` still
    returns the card;
  - a write preserves the `0644` permissions of an existing file; a new
    file is `0600`;
  - `loadStatus` with an unknown state rejects the file as corrupted.
- Unit tests (`internal/server`): 500 on persistence failure and a
  subsequent `GET` does not return the card; 409 via sentinel; rewrite the
  existing `server_test.go` test that pins "the mutation stays" to the
  new behavior.
- `main` smoke test: refactor to `run(ctx, env) error`, and a test with a
  garbled file in a temp directory verifies that `.corrupt-*` is created
  and the original content is preserved.
- `go test -race ./...`, `go vet ./...`.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md) plus:

- no code path overwrites a file that `LoadFile` failed to load;
- `strings.Contains(err.Error(), …)` does not occur in `internal/server`;
- README and `docs/architecture/overview.md` describe the `.corrupt-*`
  behavior.

## Closure

- **Status after implementation**: Done (2026-09-26)
- **Verification**: `make verify` passed (golangci-lint, eslint, vue-tsc,
  prettier, `go test -race -count=1 ./...`, build, vet). New tests (all
  ran, none skipped — permission tests are conditional on
  `os.Geteuid() != 0`):
  `TestLoadFileUnreadableReturnsUnreadableError`,
  `TestQuarantineFilePreservesContentAndAvoidsCollisions`,
  `TestSaveFilePreservesExistingPermissions` (0644 preserved, new file
  0600, no `.tmp` left behind), `TestLoadFileIgnoresStatusWithUnknownState`,
  `TestStoreRollsBackMutationsWhenPersistenceFails` (create/update/delete
  incl. restoring history and status, settings), `TestStoreErrorsAreClassifiable`,
  `TestRouterMapsValidationErrorsToUnprocessableEntity`, modified
  `TestRouterMapsPersistenceFailureToInternalServerError` (the card does
  not exist after 500, GET returns 404), `TestOpenStoreMissingFileStartsEmptyAndPersists`,
  `TestOpenStoreQuarantinesCorruptFile` (original content preserved even
  after a further save), `TestOpenStoreUnreadableFileRejectsChanges`.
- **Deviations from the plan**: (1) an unreadable file (EACCES) is not
  quarantined — its content may be fine, so `ErrConfigUnreadable` is
  distinguished and the application starts read-only (mutations return
  500 and are rolled back), while `ErrConfigCorrupt` leads to quarantine;
  (2) `loadStatus` ignores an invalid state with a warning instead of
  rejecting the file, because it is runtime state and FR-35 requires that
  corrupted runtime state brings down neither the application nor the
  configuration; (3) `main` was refactored only to `openStore(path, now)`,
  the full `run(ctx, env)` is planned in block 0027 together with
  shutdown; (4) a missing file does not return the `ErrConfigMissing`
  sentinel — the existing contract "empty store, nil" remains (FR-33);
  (5) `UpdateStatus` needs no rollback, because status is not persisted
  on change (ADR-0004); (6) validation errors are mapped to 422 already
  in this block (previously 400), in line with the block 0028 plan.
- **Documentation updated**: yes — `docs/architecture/overview.md`
  ("Protecting the configuration file" section), README (behavior of
  `.corrupt-*` and read-only mode), godoc on `LoadFile`, `QuarantineFile`,
  `PersistenceError`, `writePersistedFile`, `writeStoreError`, roadmap.
