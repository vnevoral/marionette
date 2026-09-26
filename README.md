# Marionette

Go backend with an embedded Vue 3 + PrimeVue single-page application. The
production target is a Raspberry Pi running Ubuntu (linux/arm64); development
happens in this repo's dev container on any host.

## Layout

- `cmd/marionette` — application entrypoint
- `internal/server` — HTTP routing (API + SPA fallback)
- `internal/webui` — embeds `web/dist` (the built SPA) into the Go binary via `go:embed`
- `web` — Vue 3 + PrimeVue SPA source (built with Vite)

## Development

Open the folder in the dev container (VS Code: "Reopen in Container"), then:

```bash
make ui-install   # once, installs npm dependencies
make ui-dev       # Vite dev server on :5173, proxies /api to :8080
make backend-dev  # Go backend with hot reload (air) on :8080
make verify       # build UI + lint + tests; run before every commit
```

The backend reads `./marionette.json`, which is git-ignored and created from
`deploy/dev-fixture.json` on the first `make backend-dev` (or `make
dev-config`). Node 22 is required for the UI (`web/.nvmrc`).

## Production build

```bash
make build         # builds UI, embeds it, builds a binary for the host platform
make build-arm64   # cross-compiles for Raspberry Pi (Ubuntu, linux/arm64)
```

The resulting binary in `bin/` serves the UI and API from a single process —
no separate web server or Node runtime is needed on the Raspberry Pi.

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
service. Runtime settings are read from `/etc/default/marionette`.

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
