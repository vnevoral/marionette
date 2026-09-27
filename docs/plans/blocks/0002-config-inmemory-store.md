# Implementation block: In-memory config store (card CRUD + primary run history)

- **Phase**: 2 — Domain model + config store (in-memory + JSON persistence)
- **Requirements**: FR-10, FR-17, FR-18, FR-31, FR-32, NFR-05, NFR-07
- **ADRs**: ADR-0004
- **Status**: Done

## Goal

When done, there is a thread-safe in-memory `Store` in `internal/config`
providing CRUD over `ActionCard` and `Settings` and storing the run
history of the primary action (`Run`) as a ring buffer limited by
`Settings.HistorySize`. Everything purely in memory — no reading/writing to
disk (that is handled by block 0003). The last status check and the
history of status transitions are a separate ADR-0006 contract, which will
be completed in phase 4.

## Scope

- **In scope**:
  - `Store` struct with `sync.RWMutex`, holds `Settings` and the map
    `map[string]ActionCard` (key = `ActionCard.ID`).
  - `NewStore(settings Settings) *Store` — creates an empty store (no cards).
  - `ListCards() []ActionCard`, `GetCard(id string) (ActionCard, bool)`.
  - `CreateCard(card ActionCard) (ActionCard, error)` — generates the `ID`
    (if empty), validates (`card.Validate()` from block 0001), rejects a
    duplicate ID.
  - `UpdateCard(id string, card ActionCard) (ActionCard, error)`,
    `DeleteCard(id string) error` — error `ErrNotFound` if the card does
    not exist.
  - `GetSettings() Settings`, `UpdateSettings(s Settings) error` —
    validates; if `HistorySize` decreases, the existing history is trimmed
    to the new limit (drops the oldest records).
  - `AppendRun(cardID string, run Run) error` — adds a primary action run
    to the ring buffer and keeps at most `Settings.HistorySize` latest
    records.
  - `GetRuns(cardID string, actionKind string) ([]Run, error)` — returns a
    copy of the primary run history; status transitions are handled by
    separate phase 4 logic per ADR-0006.
  - All return values are copies (no shared mutable structures between
    calls), so that the caller cannot bypass the mutex.
- **Out of scope**: JSON persistence and atomic write (block 0003),
  executing commands/evaluating `OutputRule` (phase 3 — this block only
  stores finished `Run` records, it does not create them), HTTP API
  (phase 5).

## Proposed solution

New file `internal/config/store.go`. Internal state:

```go
type Store struct {
    mu       sync.RWMutex
    settings Settings
    cards    map[string]ActionCard
    history  map[string]map[string][]Run // cardID -> actionKind -> primary run history
}
```

The ring buffer can be implemented simply as a slice trimmed to
`Settings.HistorySize` on every `AppendRun` (for dozens of cards and a low
N a circular index is not needed).

## Test plan

- CRUD: create/get/update/delete happy path + `ErrNotFound` on
  update/delete of a non-existent card, duplicate ID on create.
- `AppendRun`/`GetRuns`: filling beyond the `HistorySize` limit trims the
  oldest primary runs.
- `UpdateSettings` with a decreasing `HistorySize` trims the existing
  history of all cards.
- Concurrent access: a test with `go test -race` running CRUD and
  `AppendRun` from multiple goroutines simultaneously without race
  conditions.
- Slices/structs returned from `ListCards`/`GetRuns` are not affected by a
  later mutation of the store (copies, not references to internal state).

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md). Specifically:
`go test -race ./...` passes cleanly for `internal/config`.
