# Implementation block: Device pairing — backend

- **Phase**: 8 — Hardening ("auth/access control" item)
- **Requirements**: FR-50..FR-56, NFR-01, NFR-12, NFR-13
- **ADRs**: ADR-0011
- **Status**: Done
- **Dependencies**: Blocks 0025, 0026, 0038; block 0044 (frontend)
  follows

## Goal

When done, the backend requires a device token on all API endpoints
except health and pairing, can issue a token for a one-time code, manages
the device list and, on a start with no devices, writes a pairing code
to the log. The SPA does not have a pairing screen yet in this block
(block 0044); until then access can be turned off with
`MARIONETTE_AUTH=off`.

## Scope

- **In scope**:
  - package `internal/access`: `Devices` (loading/saving `devices.json`,
    adding, removing, token verification, `lastSeen`, expiry after the
    validity period elapses), `Pairing` (a single valid code, 10 min,
    5 attempts), generating tokens and codes from `crypto/rand`, SHA-256
    hash, `subtle.ConstantTimeCompare`;
  - shared atomic file write (`internal/fsutil.WriteFileAtomic`)
    extracted from `config` and used by both packages;
  - `requireDevice` middleware in `internal/server`: cookie
    `marionette_device` → device in the request context; without it `401`
    with the envelope `{"error": "...", "code": "pairing_required"}`;
    cookie renewal at most once a day;
  - endpoints:
    - `GET /api/session` → `200 {device}` or `401` with
      `"bootstrap": true` when no device is paired;
    - `POST /api/pairing` `{code, name}` → `201 {device}` + `Set-Cookie`;
      invalid/expired code → `400` with a single generic message;
    - `POST /api/pairing/code` → `201 {code, expiresAt}` (see Deviations);
    - `GET /api/devices` → list with a `current` flag;
    - `DELETE /api/devices/{id}` → `204`; for the own device it deletes
      the cookie;
  - SSE: on every heartbeat verifies that the device still exists,
    otherwise ends the stream;
  - `cmd/marionette`: `MARIONETTE_AUTH` (`on` default / `off` with a
    warning), `MARIONETTE_COOKIE_SECURE` (`auto` default = based on TLS,
    `always`, `never`), `MARIONETTE_DEVICES` (default `devices.json` next
    to `MARIONETTE_CONFIG`), `MARIONETTE_DEVICE_EXPIRY_DAYS` (default 60,
    1–400); bootstrap code to the log on a start with no devices;
    saving `lastSeen` on shutdown;
  - documentation: overview (Access section), README (pairing, recovery,
    curl, variables), `deploy/marionette.default`, requirements (NFR-01
    final wording, section 13), ADR-0011 → Accepted.
- **Out of scope**: UI (block 0044), bearer/API tokens for scripts,
  passkeys, mTLS, multi-user roles.

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-09-26
- **Decision notes**: Owner's request 2026-09-26: internal network +
  VPN, one-time device verification, no repeated password. Approved with
  a change: validity 60 days instead of 180, and configurable.

## Proposed solution

- `devices.json`: `{"devices":[{"id","name","tokenHash","pairedAt",
"lastSeenAt"}]}`, permissions `0600`; a corrupted file is quarantined the
  same way as the configuration (`.corrupt-<time>`) and the service starts
  with no devices → bootstrap code.
- `lastSeenAt` is updated in memory on every request, and written to disk
  only on cookie renewal (≤ once a day per device) and on shutdown —
  because of SD card wear (same principle as ADR-0004).
- Middleware order: `requireSameOrigin` (NFR-12) → `requireDevice` → mux.
  Public: `GET /api/health`, `GET /api/session`, `POST /api/pairing`,
  everything outside `/api/`.
- Failed pairing attempts are logged (without the code) at `warn` level.

## Test plan

- `internal/access`: generation (length, alphabet), code expiry, one-time
  use, 5 attempts, a new code revokes the old one, token verification,
  hash on disk (the token is not in the file), 0600 permissions, expiry
  after the validity period elapses, quarantine of a corrupted file,
  concurrent access (`-race`).
- `internal/server`: 401 on every protected route (route table as for
  NFR-12), public routes without a token, pairing sets the cookie with
  the correct attributes (`Secure` based on mode), cookie renewal,
  removing the own device deletes the cookie, SSE ends after the device
  is removed.
- `cmd/marionette`: bootstrap code in the log only with no devices;
  `MARIONETTE_AUTH=off` logs a warning and leaves the API open; invalid
  variable values → startup error.
- `make verify`; `make e2e` with `MARIONETTE_AUTH=off` (until block 0044).

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md) plus:

- `grep` of the log and `devices.json` after the tests contains no token;
- ADR-0011 is `Accepted`, NFR-01 has its final wording.

## Closure

- **Status after implementation**: Done (2026-09-26)
- **Verification**: `make verify` passed (`golangci-lint`, Vitest 97,
  installer test, `go test -race`, `go vet`). New packages and tests:
  `internal/access` (code alphabet and normalization, pairing and
  one-time use, code expiry, code replacement, 5 attempts, an invalid
  name does not consume an attempt, 0600 file without the token, renewal
  once a day, a device in daily use does not expire, an unused one
  expires even after reopening, saving the last use, removal, quarantine
  of a corrupted file, an unreadable file stops startup, concurrency
  under `-race`); `internal/fsutil` (permissions of a new and an existing
  file, quarantine with a collision); `internal/server/access_test.go`
  (401 with `pairing_required` on 15 protected routes without a cookie
  and with a forged one, health and SPA public, bootstrap code in the log
  only with no devices and only once per validity, the token is neither
  in the log nor in the body, cookie attributes for 4 `Secure` modes,
  wrong code 400 without a cookie and without the code in the log, empty
  name 422, pairing is subject to NFR-12, device management and sign-out,
  SSE stream ends after the device is removed, a router without access
  stays open); `cmd/marionette` (variables and their invalid values, code
  in the log only with no devices, `off` with a warning, an unreadable
  file stops startup, integration run: 401 → pairing with the code from
  the log → 200 → shutdown saves the device, token nowhere in the log or
  file; order of shutdown steps).
  `make e2e` with `MARIONETTE_AUTH=off`: failed once right after
  `make verify` with no captured output, the next four runs 8/8.
- **Deviations from the plan**: (1) The endpoint for another device's
  code is `POST /api/pairing/code` instead of `/api/devices/pairing-code`
  — Go's `ServeMux` rejects the combination of `DELETE /api/devices/{id}`
  with a fixed path under the same prefix within method fallbacks (405).
  (2) Besides startup, the bootstrap code is also written on
  `GET /api/session` if none is valid; otherwise the code from startup
  would expire after 10 minutes and the operator would have to restart
  the service. (3) An unreadable `devices.json` stops startup (fail
  closed) instead of a read-only mode as with the configuration. (4) E2E
  runs with `MARIONETTE_AUTH=off` until block 0044.
- **Documentation updated**: yes — `docs/architecture/overview.md`
  ("Access from paired devices" section, security note), README
  (variables, "Pairing devices", recovery, curl),
  `deploy/marionette.default`, requirements (section 3.6, NFR-01, NFR-13,
  section 13), ADR-0011 Accepted, roadmap.
