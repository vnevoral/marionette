# ADR-0004: Action card domain model and JSON configuration

- **Status**: Accepted (status history refined in ADR-0006)
- **Date**: 2026-09-25 (updated after the requirements were refined to v0.2, v0.3 and v0.4)

## Context

We need a domain model for action cards (FR-10 to FR-19) and a way to persist
them (FR-30 to FR-35) that is simple enough for a handful up to a few dozen
cards and does not require running a database server on the Raspberry Pi.
After the requirements were refined (see [requirements.md §7](../../requirements/requirements.md#7-phase-1-decisions-2026-09-25),
[§9](../../requirements/requirements.md#9-decisions--polling-and-termination-2026-09-25)
and [§11](../../requirements/requirements.md#11-decisions--history-persistence-on-shutdown-2026-09-25))
the domain model needs to be extended with: an output evaluation rule
(FR-14), run history (FR-17) including its persistence on controlled
shutdown (FR-35), a global concurrency limit (FR-18/NFR-07), two polling
intervals per card — standard and temporarily fast after the primary action
(FR-15, FR-15a) — and explicit forced termination of an action after a
timeout (FR-19).

## Decision

- Domain entities:
  - `Action` — command, arguments (`[]string`, never a shell string), working
    directory, env, `TimeoutSec` and `OutputRule` for evaluating the result.
    `TimeoutSec` is a binding maximum run time — the execution engine
    (phase 3) forcibly terminates the process once it elapses
    (`SIGKILL`/`cmd.Process.Kill()` via `context.WithTimeout`), never leaving
    it running in the background (FR-19).
  - `OutputRule` — `Type: "exit_code" | "match" | "not_match"` (default is
    just `exit_code`), for `match`/`not_match` also `Pattern` (a regular
    expression applied to the captured, possibly truncated, output).
  - `ActionCard` — id, name, description, icon, `PrimaryAction`, optional
    `StatusAction`, and three optional polling parameters (relevant only if
    the card has a `StatusAction`): - `PollingIntervalSeconds` (0/nil = standard polling off; default
    recommended value `60`) — the regular interval while the fast window
    is not active. - `FastPollingIntervalSeconds` (default `10`) — the interval used
    during the fast window. - `FastPollingWindowSeconds` (default `120`) — how long after the card's
    **primary action** is invoked `FastPollingIntervalSeconds` applies,
    before polling returns to `PollingIntervalSeconds`.
    The fast window is activated **only** if the card has `PollingIntervalSeconds
    > 0`(standard polling on) — without it`FastPollingIntervalSeconds`/
`FastPollingWindowSeconds` have no effect (FR-15a).
  - `Run` — a single execution of the primary or status action: exit code,
    captured output (max 4 KB, with a `Truncated` flag), start/end time,
    derived result (`ok`/`fail`/`timeout`) according to `OutputRule`
    (`timeout` if the engine forcibly terminated the action per
    `TimeoutSec`). Status runs serve as the latest check, not as a long-term
    history of every poll.
  - `StatusChange` — an actual transition of the status action's state, with
    the new state, start and end or duration; repeated checks without a
    change are not stored in the history (ADR-0006).
- The config store (`internal/config`, the exact name is to be specified by
  the implementation block) is an in-memory structure holding **two separate
  things**:
  1. **Configuration** (persisted to JSON): a list of `ActionCard` and global
     `Settings{ HistorySize int (default 20), MaxConcurrentActions int
(default 4) }`. It is persisted synchronously on every mutating operation
     (create/update/delete of a card or a change of `Settings`) — writing to
     a temporary file and an atomic rename (`rename`), path per
     `MARIONETTE_CONFIG` (default `./marionette.json`, see FR-34).
  2. **Primary run history** (`Run`, ring buffer of size
     `Settings.HistorySize`) and **status change history** (`StatusChange`,
     a separate limit of the latest changes): at runtime they are held
     **only in memory** and are not written to disk continuously. The latest
     status check is held as a current projection outside the history. The
     reason is that polling every 10–60 s produces many identical checks
     with no informational value.
     **Exception (FR-35)**: on a **controlled shutdown** of the application
     (graceful shutdown after SIGINT/SIGTERM) the current contents of the
     history are saved once into the same configuration file (a new
     top-level key `history`, see below) and loaded together with the
     configuration on the next start. When the application ends in an
     uncontrolled way (crash, `SIGKILL`, power outage), the last saved
     history stays stale/missing — an accepted risk, not a safeguard for
     every single run.
     JSON structure of the file: `{"settings": Settings, "cards": []ActionCard,
"history": {cardID: {"primary": []Run, "status": []StatusChange}}}` — the
     `history` key is optional (older/hand-made files without it are loaded
     with an empty history).
- Action execution is separate from the config store (a separate "execution
  engine" package, phase 3) — the config store does not know the details of
  `os/exec`, it only holds definitions and accepts writes of results (`Run`)
  into the given action's history.
- Concurrency: the execution engine holds a global semaphore (buffered
  channel) of size `Settings.MaxConcurrentActions`; both manual execution and
  scheduled polling request a slot from the same semaphore — actions over the
  limit wait in a queue (FR-18), they are not dropped.
- Polling: a separate scheduler (phase 4) keeps, for each card with an active
  `PollingIntervalSeconds`, a timer that runs the `StatusAction` through the
  execution engine (and its semaphore), updates the latest status projection
  and, on an actual state change, writes a `StatusChange` to the history.
  After each run of the **primary** action the scheduler switches the card's
  timer to `FastPollingIntervalSeconds` and schedules a return to
  `PollingIntervalSeconds` after `FastPollingWindowSeconds` (implementation
  detail — a simple time flag "until when the fast window applies" on the
  card in the scheduler, not in the persisted configuration).

## Considered alternatives

- SQLite file — rejected for the MVP, unnecessary complexity (schema/
  migrations) for dozens of card records; run history is also handled
  in-memory, so the need for persistent storage for time series goes away.
- Storing each card in its own file — rejected, it makes atomicity and
  clarity harder (FR-30 requires a single file).
- Persisting run history on **every** `AppendRun` — rejected (see above, SD
  card wear with polling every 10–60 s). Instead the history is saved only
  once on controlled shutdown (FR-35) — a compromise between durability
  across a regular restart/update and the number of disk writes.
- Periodic continuous writing of the history (e.g. once a minute) —
  rejected for the MVP as unnecessary complexity; can be considered later in
  a separate ADR if controlled shutdown turns out to be insufficient (e.g.
  frequent uncontrolled crashes in production).

## Consequences

- No external dependencies on a database or ORM.
- After a **controlled** restart/shutdown (systemd `stop`/`restart`, binary
  update) the run history is preserved — it is loaded from the `history` key
  of the configuration file. After an **uncontrolled** termination (crash,
  power outage, `SIGKILL`) the history is only as old as the last successful
  controlled shutdown — runs in between (including the card's last displayed
  state) are lost and the UI must be able to display an "unknown" state until
  a new health check runs (ideally, invoke one status check right after
  startup for cards with polling — to be specified by the phase 4
  implementation block).
- `cmd/marionette` must catch `SIGINT`/`SIGTERM`, perform a graceful
  shutdown of the HTTP server and only then save the store (config +
  history) — in this order, so that no write happens concurrently with
  still-running handlers.
- `Settings` (history size, concurrency limit) are part of the persisted
  configuration and can be changed through the same API/UI as cards.
- This ADR is an input for the phase 2 implementation block(s) in the
  roadmap.

## Addendum 2026-09-26 — Value limits (block 0028)

Domain validation (`ActionCard.Validate`, `Action.Validate`) enforces size
limits so that a single card stays small for both the JSON file and the UI
and so that a single action cannot occupy a slot permanently: ID
`^[A-Za-z0-9_-]{1,64}$`, name ≤ 120, description ≤ 2000, icon
`pi pi-<name>` ≤ 64, command ≤ 512, arguments ≤ 64 × 1024, working
directory ≤ 1024, environment ≤ 64 entries (key
`^[A-Za-z_][A-Za-z0-9_]*$` ≤ 128, value ≤ 4096), timeout 1–3600 s.
Validation returns all errors at once as a `ValidationError` with a
`Fields` map (JSON path → reason); the API exposes it as 422 with a `fields`
field. The exact values are the `config.Max*` constants and the table in
`docs/architecture/overview.md`.
