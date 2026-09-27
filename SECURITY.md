# Security policy

Marionette runs commands on the host it is installed on, so security
reports are taken seriously.

## Supported versions

Only the latest release on
[GitHub Releases](https://github.com/vnevoral/marionette/releases) receives
security fixes. Fixes are published as a new PATCH or MINOR release; upgrade
as described in the README section
[Upgrading and rolling back](README.md#upgrading-and-rolling-back).

## Reporting a vulnerability

Please do **not** open a public issue for a security problem.

Report it privately through GitHub:
[Report a vulnerability](https://github.com/vnevoral/marionette/security/advisories/new)
(the **Security** tab of the repository → **Report a vulnerability**).

Include what you can of:

- the affected version (`GET /api/health` or the UI footer);
- the steps to reproduce, or a proof of concept;
- the impact you expect (for example command execution, access without a
  paired device, cross-site requests);
- relevant configuration, with secrets removed.

You should receive an answer within 7 days. Once the problem is confirmed,
a fix is prepared and released, and the advisory is published with credit
to the reporter unless you prefer otherwise.

## Security model

When assessing a problem, keep in mind what Marionette is designed for:

- It is meant for a **trusted local network or a VPN**. Over plain HTTP the
  device token travels unencrypted; for other networks put a TLS reverse
  proxy in front and set `MARIONETTE_COOKIE_SECURE=always`.
- **Only paired devices** can use the API (except the health, session and
  pairing endpoints). A device pairs with a one-time code shown in the
  service log; tokens are stored on the host only as hashes.
- **Mutating requests are protected against cross-site requests** (JSON
  content type, `Origin` and `Sec-Fetch-Site` checks, optional host
  allowlist).
- **Actions are configured commands, not free input.** They run without a
  shell, with a timeout, in their own process group, with a minimal
  environment and as the unprivileged `marionette` service account under a
  hardened systemd unit.
- **Anyone with a paired device can define commands** that run as the
  service account. That is intended behavior; limit what the account may do
  on the host accordingly.

Details are in the requirements (NFR-01, NFR-12, NFR-13) and in
[ADR-0005](docs/architecture/decisions/0005-execution-engine.md) and
[ADR-0011](docs/architecture/decisions/0011-device-pairing-access.md).
