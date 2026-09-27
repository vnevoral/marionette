# Implementation block: Application version in the UI

- **Phase**: 8 — Hardening (addendum after the phase was closed), UI and operations area
- **Requirements**: FR-41a, FR-41, FR-26
- **ADRs**: no new ADR is needed (reads an existing endpoint)
- **Status**: Done
- **Dependencies**: Block 0051 (version from the git tag in `/api/health`)

## Goal

After an upgrade (README, "Upgrading and rolling back") the operator
currently verifies the version with `curl … /api/health` on the host.
After this block, they see it directly in the UI on every screen, even
from a phone.

## Scope

- **In scope**:
  - `web/src/api.ts`: `getHealth()` for `GET /api/health` (type
    `{ status, version, uptime… }` per `healthResponse`);
  - `AppShell.vue`: an unobtrusive version line in the layout footer
    ("Marionette v1.1.0"), in a muted color and small font. The version
    is loaded once after the SPA starts. A load failure hides the version
    and affects nothing else. A development build shows `dev`, as
    returned by the server;
  - the version is reloaded after SSE reconnects (`Live`), because that
    is typically the moment after the service restarts during an upgrade.
    The page thus shows the new version after an upgrade without a
    reload;
  - the pairing screen (not paired) does not show the version:
    `/api/health` is public, but the pairing screen should stay minimal;
  - UX specification §2.1 (app shell);
  - tests: Vitest (`AppShell` shows the version, nothing on error,
    reloads after reconnect), E2E (the footer shows the server version).
- **Out of scope**:
  - a notice that a newer version is available (the UI has no way to
    find out and the service has no outbound access);
  - a notice that the page runs an older UI build than the server
    (after an upgrade the SPA is loaded from the new binary only after a
    page reload; address only if it becomes a problem in practice).

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-09-27
- **Decision notes**: Approved by the owner ("agree with everything").

## Proposed solution

Footer instead of the top bar: the version is not operational information
for everyday work, and at 320 px the top bar only has room for navigation
and the `Live` state. The footer is at the end of every screen in the
shared `AppShell`, so one place in the code is enough.

## Test plan

- `cd web && npx vitest run`.
- `make e2e`.
- `make verify`.
- Manually on the Pi after an upgrade: the footer shows the new version
  after reconnecting.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md) plus:

- the version in the footer matches `/api/health` and changes without a
  page reload after the service restarts with a new version.

## Closure

- **Status after implementation**: Done (2026-09-27)
- **Verification**: `make verify` and `make e2e` (20 scenarios) passed. New
  tests: Vitest `AppVersion.spec.ts` (shows the version; nothing on
  error; reloads the version after the stream reconnects, not after the
  first connect), E2E "the footer shows the version the server reports
  (FR-41a)" (footer = `version` from `/api/health`); the existing 320 px
  no-horizontal-scroll test also passed with the footer.
- **Deviations from the plan**:
  - the version is shown by a separate `AppVersion.vue` component in the
    `AppShell` footer; the shell layout is a column flex, so on short
    pages the footer sits at the bottom of the window;
  - the version reload watches the stream disconnected → connected
    transition (not `onRefresh`, which is also called on every poll
    during an outage).
- **Documentation updated**: UX specification §2.1, `docs/devops/ci-cd.md`
  (verifying the version in the UI).
