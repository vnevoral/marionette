# Implementation block: Pairing code from the host (`marionette pair`)

- **Phase**: 8 — Hardening (addendum after the phase was closed), access
- **Requirements**: FR-58 (new), FR-51, FR-53, FR-56, NFR-13
- **ADRs**: [ADR-0013](../../architecture/decisions/0013-host-pairing-command.md) (new), ADR-0011
- **Status**: Done
- **Dependencies**: Blocks 0043–0045 (device pairing, installer hint)

## Goal

An operator with SSH and `sudo` on the host runs `sudo marionette pair`
and gets a pairing code for a new device, while other devices stay paired
and without opening the web UI. After this block the command works on an
installed Raspberry Pi and in development; nothing changes for browsers.

## Scope

- **In scope**:
  - **CLI**: `cmd/marionette` dispatches on the first argument: none runs
    the service, `pair` runs the client, `help`/`-h`/`--help` prints the
    usage (exit 0), anything else prints the usage (exit 2). `pair`
    prints the code as `XXXX-XXXX` and its local expiry time, plus a hint
    to enter it on the new device. Clear messages and exit 1 when the
    socket does not exist (service not running), permission is denied
    (`run it with sudo`) or access control is off.
  - **Control socket**: new package `internal/control` with the socket
    server (one route `POST /pairing-code`, JSON
    `{"code","expiresAt"}`, `201`; `409` when access control is off;
    `405`/`404` otherwise) and the client used by the CLI. Socket `0600`,
    stale file removed at startup, socket removed at shutdown. Path from
    `MARIONETTE_CONTROL_SOCKET`, default `/run/marionette/control.sock`;
    missing default directory = warning and no socket, explicit path that
    fails = startup error.
  - **Service lifecycle**: the control server starts with the HTTP server
    and stops in the shutdown sequence as its own step
    (`stop control socket`) before the HTTP server.
  - **Log**: `pairing code issued from the host` at info level, without
    the code (NFR-13).
  - **Deployment**: `deploy/marionette.service` gets
    `RuntimeDirectory=marionette` and `RuntimeDirectoryMode=0700`;
    `deploy/marionette.default` documents `MARIONETTE_CONTROL_SOCKET`;
    `install.sh` prints `sudo marionette pair` in the pairing hint (for
    the first and for further devices).
  - **Development**: `make backend-run` / `backend-dev` (air) set
    `MARIONETTE_CONTROL_SOCKET=./marionette.sock` (git-ignored); new
    `make pair` target calls the command against it.
  - **UI text**: the pairing screen of an unpaired browser mentions
    `sudo marionette pair` next to the existing hints (vocabulary only, no
    new screen).
  - **Documentation and instructions** (all in English):
    requirements (FR-58, FR-53, FR-56, v0.12 — prepared with this
    proposal), ADR-0013 (Accepted on approval) and a pointer from ADR-0011,
    architecture overview (context diagram, components table, access
    section), README (pairing section, lost devices, environment
    variables), SECURITY.md, UX specification (pairing screen),
    `AGENTS.md` (architecture at a glance, build commands),
    `.github/instructions/go-backend.instructions.md`, roadmap.
  - **Tests**: Go unit tests for `internal/control` (code issued over the
    socket and accepted by the registry, socket mode `0600`, stale socket
    replaced, `409` with access off, unknown route and method, client
    errors for a missing socket and a refused permission where testable);
    CLI dispatch and output; an integration test in `cmd/marionette` that
    runs the service with paired devices, gets a code with the client,
    pairs a new device over HTTP and checks the existing device still
    works and the log does not contain the code; `install_test.sh`
    (unit has the runtime directory, hint mentions the command); Vitest
    for the pairing screen text; `make e2e` unchanged (pairing from the
    log stays covered).
- **Out of scope**:
  - listing, renaming or removing devices from the command line;
  - API tokens for scripts (ADR-0011, still out of scope);
  - group access to the socket or `SO_PEERCRED` checks;
  - printing a link with the code (the command does not know the public
    URL of the service).

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-10-12
- **Decision notes**: The owner chose the command over the signal
  variant and approved the details without reservations.

## Proposed solution

`internal/control.Server` wraps an `http.Server` on a `net.Listen("unix",
path)` listener with a `http.ServeMux` holding the single route; it gets
the `*access.Registry` (nil when access is off) and the logger.
`internal/control.RequestCode(ctx, path)` dials the socket with an
`http.Client` whose transport uses `DialContext` to the Unix path and
returns the code and expiry or a typed error (`ErrNotRunning`,
`ErrPermission`, `ErrAccessDisabled`). `main` keeps `run` for the service
and adds `runPair(args, stdout, stderr) int` for the command.

## Test plan

- `go test -race ./internal/control/ ./cmd/marionette/`.
- `bash deploy/install_test.sh`.
- `cd web && npx vitest run`.
- `make verify` and `make e2e`.
- Manually on the reference Raspberry Pi after the release: install the
  upgrade, run `sudo marionette pair` over SSH, pair a new browser and
  check that the existing devices still work; run the command without
  `sudo` and check the permission message.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md) plus:

- `sudo marionette pair` on the host prints a code that pairs a new device
  while the existing devices keep working;
- the code never appears in the service log;
- every document and instruction listed in the scope mentions the command
  where it describes how to pair a device.

## Closure

- **Status after implementation**: Done (2026-10-12)
- **Verification**: `make verify` passed (208 Vitest tests,
  `go test -race` including the new `internal/control` package, lint,
  `install_test`, `release_version_test`); `make e2e` passed (21
  scenarios). New tests: `internal/control` (code over the socket pairs a
  device and replaces a code from the UI, socket `0600`, stale socket
  replaced but no other file, socket removed on shutdown, `409` with access
  off, only `POST /pairing-code` routed, client errors for a missing socket
  and denied permission, the code never in the log); `cmd/marionette`
  (usage and exit codes, message without a running service, environment,
  missing default directory vs. explicit path, shutdown order with the
  control socket first, and an integration test that pairs a second device
  with the code from `marionette pair` while the first stays paired);
  `install_test.sh` (runtime directory `0700`, host command in the hints);
  Vitest and E2E for the pairing screen text. A manual smoke test with the
  built binary printed a code, showed the socket as `srw-------` and
  removed it on shutdown. Released as `v1.3.1` (the `v1.3.0` release run
  failed on a new `npm audit` advisory, fixed in the lockfile); the owner
  deployed it on the reference Raspberry Pi and verified
  `sudo marionette pair` there on 2026-10-12.
- **Deviations from the plan**: none.
- **Documentation updated**: requirements v0.12 (FR-58, FR-53, FR-56),
  ADR-0013 (Accepted), ADR-0011 pointer, architecture overview, README,
  SECURITY.md, UX specification, `deploy/marionette.default`, installer
  hint, `AGENTS.md`, `.github/instructions/go-backend.instructions.md`,
  roadmap.
