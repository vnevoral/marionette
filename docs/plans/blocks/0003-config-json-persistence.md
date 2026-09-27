# Implementation block: JSON persistence of configuration and history, and loading at startup

- **Phase**: 2 — Domain model + config store (in-memory + JSON persistence)
- **Requirements**: FR-30, FR-31, FR-33, FR-34, FR-35
- **ADRs**: ADR-0004
- **Status**: Done

## Goal

When done, the configuration (`Settings` + the list of `ActionCard`) is
persisted to a single JSON file on every mutating operation and loaded
from it at application startup. The file path is configurable via
`MARIONETTE_CONFIG`. In addition: on a **graceful shutdown** of the
application (SIGINT/SIGTERM) the current run history (FR-35) is also saved
once to the same file and loaded together with the configuration on the
next start. `cmd/marionette` creates/loads the store at startup and saves
it on shutdown.

## Scope

- **In scope**:
  - File format: `{"settings": Settings, "cards": []ActionCard,
"history": {cardID: {"primary": []Run, "status": []StatusChange}}}` — the
    `history` key is optional when reading (if missing, history is empty).
  - `LoadFile(path string) (*Store, error)` — if the file does not exist,
    returns an empty `Store` with default `Settings` (`DefaultHistorySize`,
    `DefaultMaxConcurrentActions`) and a `nil` error (FR-33: a missing
    file is not an error). If the file exists but contains invalid JSON or
    does not validate (`ActionCard.Validate()`), returns an empty `Store`
    and an error (the caller — `cmd/marionette` — logs the error but
    continues starting, per FR-33). If `settings`/`cards` are valid but
    `history` is missing or corrupted, loading continues with an empty
    history (only a warning is logged, not an error, see FR-35).
  - `(*Store) SaveFile(path string) error` — saves **only** `settings` +
    `cards` (called after every mutating operation, see the persist hook
    below).
  - `(*Store) SaveFileWithHistory(path string) error` — saves `settings` +
    `cards` + `history`; called exclusively from the graceful-shutdown
    path in `cmd/marionette`, not after regular mutations nor after
    `AppendRun`.
  - Both methods write atomically: to a temporary file in the same
    directory (`*.tmp`) and `os.Rename` to the target path.
  - The `Store` from block 0002 is extended with an optional "persist
    hook": after every successful mutation (`CreateCard`, `UpdateCard`,
    `DeleteCard`, `UpdateSettings`) the store calls a configured function
    to save via `SaveFile` (e.g. `Store.OnChange func(*Store) error`, set
    after `LoadFile`) — `AppendRun` does **not** call this hook (history
    is not persisted continuously, only on shutdown, see ADR-0004).
  - `cmd/marionette/main.go`: reading `MARIONETTE_CONFIG` (default
    `./marionette.json`), calling `config.LoadFile`, logging the error on
    a corrupted file/history, wiring the store with the persist hook to
    the same path. In the existing `SIGINT`/`SIGTERM` handling (after the
    HTTP server stops, `srv.Shutdown`) `store.SaveFileWithHistory` is also
    called before the process exits.
- **Out of scope**: HTTP endpoints for CRUD (phase 5) — for now the store
  is only created internally in `main.go`, without being wired to
  `internal/server` (adding a variable/field for later use is fine, but
  without routing).

## Proposed solution

A new file `internal/config/persistence.go` with `LoadFile`/`SaveFile`.
Atomic write: `os.CreateTemp(dir, "marionette-*.json")`, write, `Sync`,
`Close`, `os.Rename(tmp.Name(), path)`.

## Test plan

- `LoadFile` on a nonexistent path → empty store, no error.
- `LoadFile` on corrupted JSON → error + empty store (no panic).
- Round-trip `SaveFile` → `LoadFile`: cards and settings identical,
  history empty (because `SaveFile` does not save history).
  - Round-trip `SaveFileWithHistory` → `LoadFile`: cards, settings,
    primary run history and status transition history identical.
- `LoadFile` on a file with valid `settings`/`cards` but corrupted
  `history` → the configuration loads, history stays empty, a warning is
  logged (not a fatal error).
- Writing to a file without permission / to a nonexistent directory →
  a readable error, the application does not call `panic`.
- `MARIONETTE_CONFIG` override: a test at the `main.go` level is not
  necessary (simple env read), a unit test on `LoadFile`/`SaveFile`/
  `SaveFileWithHistory` with an explicit path in `t.TempDir()` is enough.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md). Specifically:
manual verification that `make backend-run` with an empty/missing
`marionette.json` starts without crashing, that `Ctrl+C` (SIGINT) saves
the file incl. history and that a restart loads the history back — until
the API exists (phase 5), history can only get into the store via a
test/temporary call to `AppendRun`, not end-to-end through the UI.
