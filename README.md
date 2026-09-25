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
```

## Production build

```bash
make build         # builds UI, embeds it, builds a binary for the host platform
make build-arm64   # cross-compiles for Raspberry Pi (Ubuntu, linux/arm64)
```

The resulting binary in `bin/` serves the UI and API from a single process —
no separate web server or Node runtime is needed on the Raspberry Pi.

## Project process & documentation

Development is spec-driven: requirements, architecture decisions (ADRs) and
the implementation roadmap live in [docs/](docs/README.md) and are the source
of truth. AI coding agents working in this repo should start at
[AGENTS.md](AGENTS.md).

- [docs/requirements/requirements.md](docs/requirements/requirements.md)
- [docs/architecture/overview.md](docs/architecture/overview.md) and [decisions](docs/architecture/decisions)
- [docs/plans/roadmap.md](docs/plans/roadmap.md)
- [docs/devops/testing-strategy.md](docs/devops/testing-strategy.md), [ci-cd.md](docs/devops/ci-cd.md)
