# Marionette

Go backend with an embedded Vue 3 + PrimeVue single-page application. The
production target is a Raspberry Pi running Ubuntu (linux/arm64); development
happens in this repo's dev container on any host.

## Layout

- `cmd/marionette` — application entrypoint (composition, logging, lifecycle)
- `internal/server` — HTTP layer (API routes, SSE, SPA fallback)
- `internal/config` — card domain model, in-memory store, JSON persistence
- `internal/execengine` — action execution (process groups, timeouts, minimal environment)
- `internal/status` — status checks and polling scheduler
- `internal/actions` — background action queue
- `internal/events` — status event broker feeding SSE
- `internal/webui` — embeds `web/dist` (the built SPA) into the Go binary via `go:embed`
- `web` — Vue 3 + PrimeVue SPA source (built with Vite)

## Development

Open the folder in the dev container (VS Code: "Reopen in Container"), then:

```bash
make ui-install   # once, installs npm dependencies
make ui-dev       # Vite dev server on :5173, proxies /api to :8080
make backend-dev  # Go backend with hot reload (air) on :8080
make verify       # build UI + lint + tests; run before every commit
make e2e          # browser tests against the built binary (Playwright, Chromium)
```

The backend reads `./marionette.json`, which is git-ignored and created from
`deploy/dev-fixture.json` on the first `make backend-dev` (or `make
dev-config`). Node 22 is required for the UI (`web/.nvmrc`).

Access control is on in development too: the first start prints a pairing
code in the backend output; enter it on the pairing screen once (the paired
device is kept in `./devices.json`). Set `MARIONETTE_AUTH=off` to skip
pairing while developing.

## Production build

```bash
make build         # builds UI, embeds it, builds a binary for the host platform
make build-arm64   # cross-compiles for Raspberry Pi (Ubuntu, linux/arm64)
```

The resulting binary in `bin/` serves the UI and API from a single process —
no separate web server or Node runtime is needed on the Raspberry Pi. The
build stamps the version from `git describe` into the binary (override with
`make build VERSION=…`); `GET /api/health` reports it together with the
uptime in seconds.

## Installation on Linux with systemd

Create the ARM64 release archive on a build host:

```bash
make release-arm64
```

Copy `bin/marionette-linux-arm64.tar.gz` to the target Linux host, extract it,
and run the installer as root:

```bash
tar -xzf marionette-linux-arm64.tar.gz
sudo ./install.sh ./marionette-linux-arm64
```

The installer creates the `marionette` service account, preserves an existing
`/var/lib/marionette/marionette.json`, installs the unit, and starts the
service. On a host without paired devices it ends with the command that shows
the first pairing code (see [Pairing devices](#pairing-devices)). Runtime settings are read from `/etc/default/marionette`:

| Variable                      | Default             | Meaning                                                                                                   |
| ----------------------------- | ------------------- | --------------------------------------------------------------------------------------------------------- |
| `MARIONETTE_CONFIG`           | `./marionette.json` | Path of the configuration file (cards, settings, saved history).                                          |
| `MARIONETTE_ADDR`             | `:8080`             | HTTP listen address.                                                                                      |
| `MARIONETTE_SHUTDOWN_TIMEOUT` | `20s`               | Total budget for a graceful stop (Go duration). Keep it below the unit's `TimeoutStopSec` (90 s default). |
| `MARIONETTE_ALLOWED_HOSTS`    | empty               | Comma-separated hosts (optionally `host:port`; 80/443 equal no port) accepted for mutating API requests; empty accepts any host. |
| `MARIONETTE_LOG_FORMAT`       | `text`              | `text` (journald friendly) or `json` structured logs.                                                     |
| `MARIONETTE_LOG_LEVEL`        | `info`              | Minimum log level: `debug`, `info`, `warn` or `error`.                                                    |
| `MARIONETTE_AUTH`             | `on`                | `on` requires paired devices; `off` leaves the API open (development only, logged as a warning).          |
| `MARIONETTE_DEVICES`          | next to the config  | Path of the paired devices file (`devices.json` in the directory of `MARIONETTE_CONFIG`).                 |
| `MARIONETTE_DEVICE_EXPIRY_DAYS` | `60`              | A device unused for this many days must pair again (1–400). Devices in use are renewed automatically.     |
| `MARIONETTE_COOKIE_SECURE`    | `auto`              | `Secure` attribute of the device cookie: `auto` (when the connection uses TLS), `always` (behind a TLS proxy) or `never`. |

Global settings (`historySize`, `maxConcurrentActions`) live in the
`settings` object of the configuration file. There is no API for them: edit
the file and restart the service for a change to take effect.

Actions do not inherit the service environment. Each process gets only
`PATH`, `HOME`, `LANG` and `TZ` from the service plus the variables defined
on the action itself, so secrets in `/etc/default/marionette` never reach
user-configured commands.

On `SIGTERM`/`SIGINT` Marionette stops accepting requests, closes the live
status streams, saves the configuration and run history, discards queued
actions, lets running actions finish within the remaining budget (minus a
3 s reserve for terminating them), stops the status scheduler and saves the
history again if it changed. A second signal terminates the process at once.

Mutating API requests (creating, editing, deleting or running cards) are
protected against cross-site requests from other websites open in the
operator's browser: they must use `Content-Type: application/json`, and a
browser-supplied `Origin` or `Sec-Fetch-Site: cross-site` that does not match
the server is rejected with 403. Hosts are compared without regard to the
default ports 80 and 443, so the check also works behind a TLS-terminating
reverse proxy.

### Pairing devices

Only paired devices (browsers) can use Marionette. A device pairs once with a
one-time code and is never asked again; there is no password.

1. **First device.** After installation, open Marionette in a browser. It
   shows the pairing screen. Read the code from the service log on the host
   and enter it with a name for the device:

   ```bash
   journalctl -u marionette | grep "pairing code"
   ```

   The code is valid for 10 minutes; opening the pairing screen again writes
   a fresh one while no device is paired.
2. **More devices.** On a paired device open **Devices** → **Pair a new
   device** and enter the shown code (or open the link) on the new device.
3. **Removing a device** on the Devices page ends its access at once.
   A device that is not used for `MARIONETTE_DEVICE_EXPIRY_DAYS` (60 by
   default) expires; a device in use is renewed automatically.
4. **Lost every device?** On the host:

   ```bash
   sudo rm /var/lib/marionette/devices.json
   sudo systemctl restart marionette
   ```

   A new pairing code appears in the log.

The device token lives in an `HttpOnly` cookie and is stored on the host only
as a hash in `devices.json`; back that file up together with
`marionette.json`. Over plain HTTP the token travels unencrypted: use the
service inside a VPN, or put a TLS reverse proxy in front and set
`MARIONETTE_COOKIE_SECURE=always`. The API requires a paired browser; `curl`
without the cookie only reaches `/api/health`.

If the configuration file cannot be parsed at startup, Marionette moves it to
`marionette.json.corrupt-<timestamp>` (logged as a warning), starts with an
empty configuration and never overwrites the original. Fix the quarantined
file and move it back to restore the cards. If the file exists but is not
readable, the service starts read-only and rejects configuration changes
until the permissions are fixed and the service is restarted.

For an update, run the installer again with the new binary. It keeps the
existing configuration and restarts the service. To roll back, run the
installer with the previous binary; configuration data is not removed.

## License

MIT, see [LICENSE](LICENSE).

## Project process & documentation

Development is spec-driven: requirements, architecture decisions (ADRs) and
the implementation roadmap live in [docs/](docs/README.md) and are the source
of truth. AI coding agents working in this repo should start at
[AGENTS.md](AGENTS.md).

- [docs/requirements/requirements.md](docs/requirements/requirements.md)
- [docs/architecture/overview.md](docs/architecture/overview.md) and [decisions](docs/architecture/decisions)
- [docs/plans/roadmap.md](docs/plans/roadmap.md)
- [docs/devops/testing-strategy.md](docs/devops/testing-strategy.md), [ci-cd.md](docs/devops/ci-cd.md)
