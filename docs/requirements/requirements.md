# Marionette Requirements (SRS) — v0.11

> Status: **refined** (phase 1 and UX specification, 2026-09-25; extended with
> an SSE stream for live status changes, 2026-09-25; extended with two polling
> intervals and forced termination of actions, 2026-09-25; extended with run
> history persistence on graceful shutdown, 2026-09-25; status actions driven
> by an internal scheduler, 2026-09-25; extended with UX requirements FR-24 to
> FR-29 and NFR-08 to NFR-10, 2026-09-25; project and code review 2026-09-26:
> NFR-12, clarification of FR-18 and NFR-01, see
> [Decisions — review 2026-09-26](#13-decisions--project-review-2026-09-26);
> added FR-05, FR-21a and FR-22a after validation on a Raspberry Pi, 2026-09-26;
> FR-57 device renaming, 2026-09-27; FR-10a card color, 2026-09-27;
> FR-06, FR-41a and FR-42a, 2026-09-27).
> The open questions from v0.1 were decided
> with the project owner, see [Phase 1 decisions](#7-phase-1-decisions-2026-09-25),
> [Decisions — polling and termination](#9-decisions--polling-and-termination-2026-09-25)
> and [Decisions — history persistence on shutdown](#11-decisions--history-persistence-on-shutdown-2026-09-25).
> Every item has an ID for back-references from ADRs and implementation blocks.

## 1. Purpose and scope

Marionette is a self-contained deployable application that runs as a service
on the target host (typically a Raspberry Pi ARM64 with Ubuntu 24.x, or
generic Linux) and provides a web interface for defining, running and
monitoring the status of user-configured actions on that host (or on its
network surroundings).

Deployment must not require installing a runtime environment (Node.js, JVM,
Python, ...) on the target host — the whole application (backend + UI) is
distributed as a single executable file.

## 2. Actors

- **Operator** — a user accessing the web interface, running actions and
  watching the status of cards.
- **Administrator** — a user configuring action cards and actions (may be the
  same person as the operator in the MVP).
- **Host** — the machine on which Marionette runs and on which the configured
  commands are actually executed.

## 3. Functional requirements

### 3.1 Deployment and operation

- **FR-01**: The application is distributed as a single binary file
  containing the embedded web UI (no need to install Node.js/npm on the
  target).
- **FR-02**: The application runs as a system service on supported Linux
  (systemd unit; verified as reference on Ubuntu 24.x), starting at boot and
  restarting automatically after a crash.
- **FR-03**: The application listens on a configurable HTTP port and serves
  both the API and the static UI content from the same process.
- **FR-04**: Supported target platforms: linux/amd64 (development/tests) and
  linux/arm64 (Raspberry Pi).
- **FR-05** _(added 2026-09-26)_: The reference status action `ping` (FR-16)
  must be runnable as the service user in the hardened unit
  (`NoNewPrivileges=true` prevents `ping` from using the file capability
  `cap_net_raw`, so only unprivileged ICMP according to
  `net.ipv4.ping_group_range` works). The installer detects whether the host
  allows unprivileged ICMP; if not, it prints clear remediation instructions.
  It does not change the host's system settings itself. The instructions are
  also in the installation documentation.
- **FR-06** _(added 2026-09-27)_: The release archive for linux/arm64
  (version from the git tag `vMAJOR.MINOR.PATCH`, a `.sha256` file) is built
  and published by CI automatically after the tag is pushed, as a GitHub
  Release. Tags are still created only by the project owner. The release is
  not published if the tests do not pass.

### 3.2 Action cards and actions

- **FR-10**: The user can create, edit and delete an action card. A card has
  a name, a description, an icon and is visible on the dashboard.
- **FR-10a** _(added 2026-09-27)_: A card has an optional **color** for
  visual distinction on the dashboard, shown as a colored stripe on the edge
  of the card. The color is picked in the card editor from a fixed palette of
  roughly 8 options including "no color" (the default, card without a stripe)
  and black; each option has a text name in the editor. The color carries no
  status: the card status is still shown only by the status badge (FR-20,
  FR-25) and the palette does not overlap with the status colors. A card
  without a color (including one from an older version's configuration)
  behaves as it does today. The color is also visible in the card detail.
- **FR-11**: Every card has exactly one **primary action** — a definition of a
  command run on the host (command, arguments, working directory, environment
  variables, timeout). The timeout is a binding **maximum time to wait for
  the action to finish** — once it elapses, the execution engine forcibly
  ends the process (terminate/kill), the run is recorded as failed/timeout and
  must not be left hanging (see FR-19, NFR-04).
- **FR-12**: Every card may optionally have a **status action** (health
  check) — an action of the same type as the primary one, whose result
  determines the current status of the card.
- **FR-13**: Running an action captures the exit code, stdout/stderr and the
  run time. The captured combined output (stdout+stderr) is limited to
  **4 KB**; above this size the output is truncated and marked as truncated
  (the application does not fail because of it).
- **FR-14**: The evaluation of an action's result is given by the exit code
  (`0` = success, otherwise failure) and optionally by an additional rule
  matching the captured output against a regular expression (e.g. "must
  contain" / "must not contain"). The output rule is an optional extension on
  top of the exit code.
- **FR-15** _(updated 2026-09-25)_: A status action can be triggered manually
  via the API, but like the primary action only asynchronously: the API
  confirms that it was queued for running and does not wait for it to finish.
  The status action is also optionally run automatically by internal
  background processes, primarily the scheduler, at the configured
  **standard polling interval**, per card. The API status read is a separate
  synchronous read-only operation that always returns the last known status
  projection and does not run the status action. The default standard
  interval is **60 s**; polling can be turned off completely for a given
  card. Standard polling is a prerequisite for FR-15a (fast polling) —
  without standard polling enabled, fast polling is not activated.
- **FR-15a**: Immediately after the **primary action** of a card is triggered
  (manual or future scheduled run), the status action temporarily switches to
  the **fast polling interval**, default **10 s**, for a default
  **120 s** (both values configurable globally/per card). After this period
  the card returns to the standard polling interval (FR-15). If the card
  does not have standard polling enabled, fast polling is not activated
  (see FR-15).
- **FR-16**: An example reference use case: primary action = Wake-on-LAN
  packet to a MAC address; status action = `ping` to the IP/hostname of the
  target PC.
- **FR-19**: Expiry of an action's timeout (FR-11) leads to forced
  termination of the running process by the execution engine (never to
  leaving it running in the background) — protection against uncontrolled
  accumulation of unfinished processes on weak hardware (NFR-04, NFR-07).
- **FR-17**: For the primary action, the history of the last **N runs** is
  kept (default N = 20, configurable globally). The status action separately
  has the last check result for the current status and a history of actual
  status transitions only. Each transition contains the new status, the start
  and end or the duration; repeated checks with the same status do not create
  a new history record. The UI shows the current status, how long it has
  lasted and the history of changes.
- **FR-18** _(updated 2026-09-26)_: The number of actions running at the same
  time across the whole application (manual runs and polling together) is
  limited by a configurable limit, default **4**; actions over the limit wait
  in a queue. The queue is bounded (capacity `4 × limit`); the API rejects a
  run request with a full queue with status 503 and a `Retry-After` header,
  so the client is always informed and no accepted (202) action is lost. A
  request for an action that is already waiting in the queue for the same
  card and the same kind (primary/status) is not enqueued again and the API
  returns 202 idempotently. The original wording "none is lost" without a
  bounded queue is replaced by this clarification.

### 3.3 Dashboard and UI

- **FR-20**: The main screen (Dashboard) shows all action cards as a
  grid/list with the current status (color-coded: unknown/OK/error/running).
- **FR-21**: From a card, both the primary action and the status action can
  be run with a single click and the ongoing/last result can be followed.
- **FR-21a** _(added 2026-09-26)_: For the last status check, the card detail
  shows, besides the result, also the exit code, the run time and the
  captured output (FR-13, expandable on demand, with a truncation marker), so
  that the operator can find the cause of the **Problem** status without
  access to the API or the host. The dashboard stays scannable; it does not
  show the output on the card.
- **FR-22**: There is a separate screen for managing (configuring) cards and
  actions (CRUD).
- **FR-22a** _(added 2026-09-26)_: In the card editor, both the primary and
  the status action are entered as **a single command line** (e.g.
  `/usr/bin/ping -c 1 -W 2 192.168.1.10`), not as a separate command and
  repeatable argument rows. The UI splits the line into the command and
  arguments (quotes and `\` for arguments with spaces, no expansion of
  variables and globs) and shows the result of the split below the field;
  an existing action is shown back as a single line. Shell characters (`|`,
  `&`, `;`, `<`, `>`, `(`, `)`, `` ` ``, `$`) outside quotes are rejected by
  the editor with an explanation. The stored form of the action, the API and
  execution do not change (FR-11, NFR-01 a); see
  [ADR-0012](../architecture/decisions/0012-single-line-command-editor.md).
- **FR-23**: The UI is built on Vue 3 + PrimeVue (see
  [ADR-0003](../architecture/decisions/0003-vue-primevue-frontend.md)).

- **FR-24**: The UI has a unified information architecture for the operator
  and the administrator. The dashboard is for quick monitoring and running of
  actions, card management for configuration and the card detail for history
  and diagnostics. Navigation between these contexts is available from every
  main screen.
- **FR-25**: The UI uses a unified design system with named tokens for
  colors, typography, spacing, control sizes, surfaces and states. Status
  colors have the same meaning in a tag, an icon, text and any chart; status
  must not be communicated by color alone.
- **FR-26**: Names of statuses, actions, buttons, errors, confirmations and
  empty states use a unified vocabulary. The interface uses one chosen
  language across all screens; mixing languages and technical internal names
  in regular user-facing text is not acceptable.
- **FR-27**: Every asynchronous operation distinguishes the request state
  (pending, accepted, in progress, completed, failed) and the last known
  device state. The user gets local feedback, the server error and an option
  to retry loading without losing unsaved changes.
- **FR-28**: The main workflows are usable with a keyboard and on a touch
  screen. Interactive elements have a visible focus, a descriptive name and
  sufficient contrast; an icon without text has a tooltip or another
  accessible description.
- **FR-29**: The dashboard, management and detail are usable at widths from
  320 px up to desktop without horizontal scrolling, overlapping or a change
  in the meaning of controls. Information density adapts to the context: the
  dashboard is scannable, the form readable and the history comparable.

### 3.4 Configuration and persistence

- **FR-30**: All configuration (cards, actions) is stored in a single
  configuration file (JSON) on the host's disk.
- **FR-31**: At runtime the configuration is held in memory (in-memory
  store); it is written to the file on every change (create/update/delete)
  and the file is loaded at startup.
- **FR-32**: Expected data volume: a few to low tens of action cards —
  scaling to large volumes and multi-tenancy are not addressed.
- **FR-33**: A corrupted/missing configuration file at startup must not crash
  the application — it starts with an empty configuration and the error is
  logged.
- **FR-34** _(added 2026-09-26)_: The path to the configuration file can be
  set with the environment variable `MARIONETTE_CONFIG` (default
  `./marionette.json`), analogous to the already existing `MARIONETTE_ADDR`
  (default `:8080`) for the HTTP address/port. The overall graceful shutdown
  limit (FR-35) can be set with the variable `MARIONETTE_SHUTDOWN_TIMEOUT`
  (Go `time.Duration` format, default `20s`; must be shorter than the
  systemd unit's `TimeoutStopSec`, default 90 s). Logging is controlled by
  `MARIONETTE_LOG_FORMAT` (`text`/`json`, default `text`) and
  `MARIONETTE_LOG_LEVEL` (`debug`, `info`, `warn`, `error`; default `info`)
  — block 0034.
- **FR-35**: On **graceful shutdown of the application** (receiving
  SIGINT/SIGTERM and completing the graceful shutdown), the history of
  primary runs and the history of status changes (FR-17) are saved to
  disk together with the configuration. At application startup the history
  is loaded together with the configuration, if available and valid — thanks
  to this, run history is not lost on a controlled shutdown/restart (e.g. a
  binary update, a systemd service restart). On an uncontrolled termination
  (process crash, power outage, `SIGKILL`) the history since the last save is
  lost — this is an accepted risk (see
  [ADR-0004](../architecture/decisions/0004-action-card-domain-model.md)
  and NFR-03, the reason why history is not persisted on every run).
  A corrupted/missing saved history at startup behaves like FR-33 — it must
  not crash the application, the history is just not loaded (starts empty).

### 3.5 API

- **FR-40**: The backend provides a REST/JSON API for CRUD over cards/actions
  and for running actions and reading the history/last result.
- **FR-41**: `GET /api/health` (already exists) serves as the liveness
  endpoint of the process itself (distinct from the status actions of user
  cards).
- **FR-41a** _(added 2026-09-27)_: The UI shows the version of the running
  application (from `GET /api/health`) on every screen, so that the operator
  can verify the version after an upgrade without the command line.
- **FR-42**: The backend provides a Server-Sent Events stream for live card
  status changes. A connected client subscribes to the events endpoint and on
  every change of the status projection receives an event containing the
  card identifier and the new `StatusSnapshot`; repeated checks without a
  status change do not create an event. The stream sends a regular
  heartbeat, supports client reconnection, and while the stream is
  temporarily unavailable the client uses the REST read-only API as a
  fallback. The REST API remains the source for the initial screen load and
  for synchronous reads of the current projection.
- **FR-42a** _(added 2026-09-27)_: The same stream sends an event after a
  **finished run of the primary action is written** to the history (FR-17),
  with the card identifier and the written run. Running an action stays
  asynchronous (FR-15, `202`); the event tells the client that the action
  has finished without it having to reload the history repeatedly. A run
  that is not written (an action canceled on service shutdown before it
  started running, or a card deleted in the meantime) does not create an
  event. The client therefore still relies on a REST
  reload after reconnecting and on a wait timeout.

### 3.6 Access and device pairing _(accepted 2026-09-26, ADR-0011)_

Only **paired devices** (browsers) have access to the UI and the API. A
device is verified once, with a one-time code, and is not verified again
afterwards; no password is entered. The service runs in the internal
network, reachable from outside via VPN.

- **FR-50 Mandatory pairing**: All endpoints under `/api/` except
  `GET /api/health` and the pairing endpoints require a valid device token;
  without it they respond `401` with a JSON envelope. The SPA's static files
  stay public (they contain no data); on `401` the SPA shows the pairing
  screen. The SSE stream (`/api/events`) requires the token just like REST.
- **FR-51 Pairing code**: The code has 8 characters from an alphabet without
  confusable characters (Crockford Base32, ~40 bits), is valid for 10
  minutes, is single-use and is invalidated after 5 failed attempts. There
  is at most one valid code; a new code invalidates the previous one. Codes
  are held only in memory.
- **FR-52 Device token**: After a valid code and a device name (e.g.
  "Work laptop") are entered, the server issues a random token (256 bits) in
  a cookie with `HttpOnly`, `SameSite=Strict`, `Path=/`. The server stores
  only the token's hash. A device that has not been used for longer than the
  **device validity** expires; the validity is the parameter
  `MARIONETTE_DEVICE_EXPIRY_DAYS` (default 60 days, range 1–400) and the
  cookie has the same lifetime. A device in use does not expire: when it is
  used, the server renews both the cookie and the last-used time (at most
  once a day).
- **FR-53 First device**: If no device is paired, the service generates a
  pairing code at startup and writes it to the log
  (`journalctl -u marionette`); the pairing screen points the operator to
  the log. Once a paired device exists, the code is not written to the log.
- **FR-54 Additional devices**: A paired device can generate a code for
  another device in the UI (**Devices** → **Pair a new device**); the code
  and a link that pre-fills it are shown.
- **FR-55 Device management**: The UI shows a list of paired devices (name,
  when paired, when last used, a "this device" marker) and allows removing a
  device. Removal takes effect immediately for REST; an open SSE stream of a
  removed device ends at the latest at the next heartbeat (15 s). Removing
  one's own device is a logout.
- **FR-56 Access recovery**: When all devices are lost, the operator deletes
  the devices file on the host and restarts the service; a new code is then
  generated at startup according to FR-53. The procedure is in the README.
- **FR-57 Device renaming** _(added 2026-09-27)_: The device name is entered
  during pairing (FR-52) and a paired device can later change it on the
  **Devices** page, for any paired device including its own (just like
  removal, FR-55). The new name follows the same rules as during pairing
  (required, at most 64 characters after trimming whitespace); the server
  rejects an invalid name with `422` and a field error. The change is saved
  immediately to the devices file and does not affect the token, validity or
  last-used time. Names do not have to be unique.

## 4. Non-functional requirements

- **NFR-01 Security** _(changed 2026-09-26: access only from paired
  devices, section 3.6, ADR-0011; the sentence about missing authentication
  below thereby ceases to apply, the other points a–c still apply)_: The MVP
  does not implement authentication/authorization for the UI/API —
  it assumes deployment in a trusted network (home/lab network behind a
  firewall, without direct exposure to the internet). If access from outside
  is needed, that is the operator's responsibility (VPN/reverse proxy with its
  own authentication), not the application's. Nevertheless: (a) actions are
  always run via `exec.Command(name,
args...)` with structured arguments, never by composing a shell command from
  a string/user input (no shell interpolation), (b) the config store/API
  interface is designed so that an auth layer can be added later without a
  major rebuild (see roadmap phase 8), (c) _(added 2026-09-26)_ a running
  action does not inherit the whole environment of the service process; it
  gets a minimal base (`PATH`, `HOME`, `LANG`, `TZ`) and the variables
  defined in the action (`Env`), so that any sensitive variables of the
  service do not leak into user commands.
- **NFR-02 Operational simplicity**: Installation on a new host = copy one
  binary file (+ optionally a systemd unit) and run it. No external
  dependencies (DB server, runtime).
- **NFR-03 Performance**: The application is intended for low-power hardware
  (Raspberry Pi) — low memory and CPU usage, no unnecessary background
  processes.
- **NFR-04 Reliability**: A failure of a running action (e.g. a non-existent
  binary) must not affect the server or other cards.
- **NFR-05 Testability**: Domain logic (action evaluation, config store) is
  covered by unit tests independently of actual process execution
  (an abstraction over exec).
- **NFR-06 Maintainability**: No unnecessary dependencies/frameworks beyond
  what is needed (see the [ADR log](../architecture/decisions)).
- **NFR-07 Protecting weak HW under concurrency**: The number of actions
  running at the same time is limited by a configurable limit (default 4,
  see FR-18), so that polling many cards at once does not overwhelm the
  Raspberry Pi.
- **NFR-08 Visual consistency**: New and modified screens use the shared
  shell, tokens and component patterns; local ad-hoc colors, typography and
  spacing are not added without justification in the design documentation.
- **NFR-09 Accessibility**: The UI meets the basic WCAG 2.2 AA rules for
  contrast, focus, control names, keyboard operation and state changes;
  critical workflows are verified at least with a keyboard and at mobile
  width.
- **NFR-10 UX verifiability**: Every UX block has described loading, empty,
  error, success and destructive action states and, before closure, passes a
  screenshot or manual smoke test of the main workflows.
- **NFR-11 Real-time updates**: The SSE stream for status changes must not
  block the REST API or the status scheduler. A client disconnect must not
  create unbounded work or memory growth on the server; it must be possible
  to close the connection safely when the page or application is closed.

- **NFR-12 Protection against cross-site requests** _(accepted 2026-09-26)_:
  Mutating API endpoints (creating/changing/deleting a card,
  running an action) reject a request that comes from an origin other than
  the application's own UI: a request with a body must have
  `Content-Type: application/json`, and a request marked by the browser as
  `Sec-Fetch-Site: cross-site` or with an `Origin` header not matching the
  server's host is rejected (403/415). Optionally, the accepted `Host` values
  can be restricted with the variable `MARIONETTE_ALLOWED_HOSTS`. Host
  comparison does not depend on the scheme or on the default ports 80/443,
  so the protection also works behind a TLS-terminating reverse proxy
  (block 0036). Read-only endpoints and the SSE stream stay unrestricted.
  Reason: NFR-01 assumes a trusted network, not a trusted browser — a
  third-party web page opened by the operator could otherwise run any
  configured action on the host. It does not address authentication (that
  stays out of the MVP). Implementation: block 0026.
- **NFR-13 Protection of pairing and tokens** _(accepted 2026-09-26,
  ADR-0011)_: Tokens and codes are generated from `crypto/rand`, compared in
  constant time, and only the SHA-256 hash of the token is stored on disk,
  in a file with `0600` permissions separate from the card configuration.
  The token is never written to the log nor returned in a response body. The
  cookie has the `Secure` attribute if the connection uses TLS or it is
  forced by configuration (behind a TLS proxy). Authentication does not
  replace NFR-12 — the cross-site protection stays.

## 5. Constraints and assumptions

- Target environment: Linux with systemd; Ubuntu 24.x is the reference
  environment for deployment validation on Raspberry Pi ARM64 and
  linux/amd64. Windows is not a target production platform.
- Single-user / small-team deployment in a trusted network (home/lab
  network) — multi-user RBAC is not required in the MVP, but NFR-01 must
  allow future extension.
- The data (configuration) is not sensitive to a degree requiring on-disk
  encryption in the MVP; this is an assumption to be verified with the
  administrator.

## 6. Open questions

No blocking open questions for phases 2–5 (see the phase 1 decisions and
the polling/termination decisions below). Future questions (auth for
multi-user operation, scaling run history beyond N records) will be
addressed only when there is a concrete need, with a separate requirement.

Questions opened by the 2026-09-26 review were decided by the owner on the
same day, see [section 13](#13-decisions--project-review-2026-09-26).

## 7. Phase 1 decisions (2026-09-25)

The following open questions from v0.1 were decided with the project owner
and reflected in the FR/NFR above:

| Question                       | Decision                                                                                         |
| ------------------------------ | ------------------------------------------------------------------------------------------------ |
| Authentication/authorization   | None in the MVP; trusted network assumption (NFR-01).                                            |
| Status action evaluation       | Exit code + optional output rule (regex) (FR-14).                                                |
| Run history                    | Last N runs, default N = 20 (FR-17).                                                             |
| Automatic polling              | Optional, per card, default interval 60 s (FR-15).                                               |
| Command validation             | No whitelisting; only structured arguments, no shell string (NFR-01).                            |
| Action output limit            | 4 KB combined stdout+stderr, the rest is truncated (FR-13).                                      |
| Config file path / port        | `MARIONETTE_CONFIG` (default `./marionette.json`) / `MARIONETTE_ADDR` (default `:8080`) / `MARIONETTE_SHUTDOWN_TIMEOUT` (default `20s`) (FR-34). |
| Number of actions per card     | 1 primary + 1 optional status action, more actions per card are not in the MVP (see FR-11, FR-12). |
| Action execution concurrency   | Configurable limit, default 4 (FR-18, NFR-07).                                                   |

## 9. Decisions — polling and termination (2026-09-25)

| Question                             | Decision                                                                                                                                           |
| ------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| Standard polling interval            | Default **60 s** (previously 30 s), per card (FR-15).                                                                                              |
| Fast polling after the primary action | Default interval **10 s** for **120 s** from triggering the primary action, then back to standard (FR-15a).                                      |
| Fast polling without standard polling | Not activated — fast polling is only a temporary boost of standard polling and requires it to be enabled (FR-15a).                               |
| Forced termination after timeout     | The existing `TimeoutSec` (FR-11) is the binding maximum wait time; after expiry the execution engine kills the process (FR-19), no additional new parameter. |

## 11. Decisions — history persistence on shutdown (2026-09-25)

| Question                          | Decision                                                                                                                                                                                                                                                                                                                                 |
| --------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Loss of run history on restart    | The history is additionally saved to disk on a **graceful** application shutdown and loaded back at startup together with the configuration (FR-35). At runtime (between individual action runs) it is still not persisted because of SD card wear (ADR-0004). On an uncontrolled crash/outage the history since the last save is lost — an accepted risk. |

## 13. Decisions — project review (2026-09-26)

The review of the structure, process and code (2026-09-26) opened the
following questions. The project owner confirmed the proposed solutions;
these are mostly fixes and straightening out the state, not new
functionality. The implementation is in blocks 0024–0034 (roadmap,
phase 8).

| Question                       | Decision                                                                                                                                                    |
| ------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Bounded action queue vs. FR-18 | The queue is bounded, a full queue returns 503 + `Retry-After`; same card + action kind in the queue → idempotent 202 (FR-18 clarified, block 0027).        |
| Environment of run actions     | Minimal base `PATH`, `HOME`, `LANG`, `TZ` + the action's `Env`; the service's `os.Environ()` is not inherited (NFR-01 c, block 0034).                       |
| Endpoint for global settings   | `Store.UpdateSettings` is removed from the public store API (YAGNI); the concurrency limit is changed in the file and takes effect after a restart (block 0034). |
| Future of PrimeFlex            | Removed after phase 8 and replaced with an own utility layer with the same classes; Tailwind rejected, PrimeVue stays on v4 (MIT) because of the v5 license ([ADR-0010](../architecture/decisions/0010-own-layout-utilities-replace-primeflex.md), block 0035). |
| Cross-site API protection      | NFR-12 accepted; implementation in block 0026.                                                                                                              |
| Repository license             | MIT (block 0031 adds `LICENSE`).                                                                                                                            |
| Authentication (NFR-01)        | Access only from paired devices: a one-time code (the first one in the service log), then a device token in a cookie without further verification; validity 60 days of non-use, configurable; no tokens for scripts (FR-50..56, NFR-13, [ADR-0011](../architecture/decisions/0011-device-pairing-access.md), blocks 0043–0044). |

## 12. Traceability

Every implementation block in the [roadmap](../plans/roadmap.md) must
reference at least one FR/NFR from this document. New requirements are
added with the `/new-requirement` prompt and get the next free ID within the
relevant section.
