# ADR-0013: Pairing code from the host over a local control socket

- **Status**: Accepted
- **Date**: 2026-10-12

## Context

ADR-0011 issues pairing codes in two ways: in the service log at startup
while no device is paired (FR-53), and in the UI of an already paired
device (FR-54). An operator who has SSH access to the host but no paired
browser at hand (a new phone, a lost laptop, cleared cookies) can only use
the recovery from FR-56: delete `devices.json` and restart the service.
That removes every other paired device as well.

ADR-0011 already treats whoever can read the service log on the host as an
authorized operator. The owner asked (2026-10-12) for a way to issue a code
for a new device from the host over SSH, without the web UI and without
losing the existing devices.

## Decision

The binary gets a `pair` subcommand that asks the running service for a
pairing code over a **local Unix domain socket**:

```bash
sudo marionette pair
```

1. `marionette` without arguments runs the service as today;
   `marionette pair` is a short-lived client; any other argument prints
   the usage and exits with status 2.
2. The service listens on a control socket next to the HTTP listener. The
   path is `MARIONETTE_CONTROL_SOCKET`, default
   `/run/marionette/control.sock`. The systemd unit creates
   `/run/marionette` with `RuntimeDirectory=marionette` and
   `RuntimeDirectoryMode=0700`; the socket itself is `0600`, owned by the
   service user. Only root (`sudo`) and the service user can connect: the
   file permissions are the authorization, consistent with ADR-0011.
3. The protocol is HTTP/1.1 over the socket with one route,
   `POST /pairing-code`, answering `201` with the code and its expiry in
   JSON. It is a separate `http.Server` with its own handler; nothing of
   the public API is reachable over the socket and the socket route is not
   reachable over TCP.
4. The code is issued by the same registry as in the UI, with every rule
   of FR-51 (10 minutes, single use, 5 failed attempts, one valid code at a
   time, memory only). It works whether or not devices are paired and
   changes no device.
5. The code is printed only by the command to its standard output. The
   service log records that a code was issued from the host, never the
   code itself.
6. With `MARIONETTE_AUTH=off` the socket answers `409` and the command
   reports that access control is disabled.
7. A stale socket file is removed at startup and the socket is removed at
   shutdown. If the default directory does not exist (the service is not
   started by the unit, e.g. in development), the service logs a warning
   and runs without the socket; an explicitly configured path that cannot
   be used stops the startup.

Approved by the owner 2026-10-12 together with block 0065.

## Considered alternatives

- **Signal (`systemctl kill -s USR1`) and a code in the log** — a few lines
  of code, but the code stays in the journal on disk, readable by the
  `adm`/`systemd-journal` groups, and the command gives no direct answer.
  Rejected by the owner in favor of the command.
- **The command writes a code into a file that the service reads** —
  codes are memory only (FR-51) and two processes writing state invites
  races; rejected.
- **A localhost-only HTTP endpoint** — any local user or process could call
  it, and NFR-12 assumes a browser; file permissions on a socket are a
  clearer authorization. Rejected.
- **Checking the peer's credentials (`SO_PEERCRED`)** — adds nothing over
  `0600` on a socket in a `0700` directory; can be added if group access
  is ever needed.

## Consequences

- New surface: a second listener (local only), a CLI mode of the binary
  and a change of the systemd unit. An upgrade must install the new unit
  (the installer already does).
- A code from the host invalidates a code shown in the UI and the other
  way around (FR-51: one valid code).
- FR-56 (delete `devices.json`) stays for revoking every device at once;
  regaining access no longer needs it.
- Development needs an explicit socket path (`make backend-run` sets one,
  `make pair` calls it).
- Documentation and agent instructions must mention the command:
  requirements, ADR-0011, architecture overview, README, SECURITY.md,
  installer hint, `marionette.default`, UX specification and pairing screen
  text, `AGENTS.md` and `.github/instructions/go-backend.instructions.md`.
