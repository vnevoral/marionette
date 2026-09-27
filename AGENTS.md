# Marionette — instructions for AI agents

This file is the main entry point for any AI coding agent (Copilot,
Claude Code, Cursor, ...) working in this repository. Always read it
first before starting work.

## About the project

Marionette is a self-contained deployable application (a single binary)
running as a service on a Raspberry Pi (Ubuntu, linux/arm64) or on Linux in
general, without having to install any runtime (Node.js, JVM, ...) on the
target machine. The application provides a web interface (dashboard) for
defining and controlling so-called **action cards** — each card runs a
configured command/action on the host and can verify the resulting state
using an associated status/health-check action (e.g. Wake-on-LAN + ping).
The configuration of cards and actions is persisted to a single
configuration file (JSON) and lives in an in-memory structure at runtime.

A detailed and binding description of the requirements and decisions is in [docs/](docs/README.md):

- [docs/requirements/requirements.md](docs/requirements/requirements.md) — what is to be built (SRS)
- [docs/architecture/overview.md](docs/architecture/overview.md) — how it is built
- [docs/architecture/decisions/](docs/architecture/decisions) — ADR log (why)
- [docs/plans/roadmap.md](docs/plans/roadmap.md) — phases and implementation blocks
- [docs/devops/testing-strategy.md](docs/devops/testing-strategy.md) and [docs/devops/ci-cd.md](docs/devops/ci-cd.md)

If anything in the code disagrees with the documents above, the documents
are the source of truth — first propose updating them (ADR / requirements),
only then change the code.

## Architecture at a glance

- `cmd/marionette` — entrypoint: service composition, `slog`, lifecycle and
  graceful shutdown
- `internal/server` — pure HTTP layer: routes, request validation,
  cross-site protection, SSE writing, SPA fallback (no goroutines outside
  handlers)
- `internal/config` — domain model of cards, in-memory store, JSON persistence
- `internal/execengine` — safe execution of actions (process group, timeout,
  minimal environment) and the concurrency limit
- `internal/status` — card status evaluation and the polling scheduler
- `internal/actions` — background action queue (bounded, deduplication, drain)
- `internal/events` — status event broker for SSE
- `internal/webui` — `go:embed` of the built `web/dist` into the binary
- `web` — Vue 3 + PrimeVue 4 (Aura theme) SPA, built via Vite into
  `internal/webui/dist`

The result of `make build` is a single binary with no external runtime
dependencies (no Node.js on the target, no separate web server).

## Build & test commands

```bash
make ui-install     # once: npm install for web/
make ui-dev          # Vite dev server :5173 (proxy /api -> :8080)
make backend-dev     # Go backend with hot-reload (air) on :8080
make build           # build UI + embed + Go binary for the current platform
make build-arm64     # cross-compile for Raspberry Pi (linux/arm64)
make test            # go test -race ./... + npm test (Vitest)
make lint            # golangci-lint + eslint + vue-tsc + prettier --check
make verify          # the single validation command: build UI + lint + test + go build/vet
```

For the web alone (`cd web`): `npm run lint`, `npm run lint:types`,
`npm run format` / `npm run format:check`, `npm run build`
(`vue-tsc -b && vite build`).

The local dev configuration `./marionette.json` is ignored by git and is
created from `deploy/dev-fixture.json` by `make dev-config` (called
automatically from `make backend-dev` / `backend-run`).

**Before closing any change, run `make verify`** — it is the only place
where the set of validation commands is defined (CI and the Definition of
Done refer to it). During fast iteration you can use the partial
`make test`, `make lint`, or directly `go test ./...` and `npm run lint`.

## Conventions

- Language: everything committed to the repository is in English —
  requirements, ADRs, roadmap, implementation blocks, UX specification,
  README, prompts and instructions, code comments and commit messages. The
  conversation with the project owner may be in another language; artifacts
  written into the repository are not.
- Go: `gofumpt` formatting (a stricter superset of `gofmt`, enforced by
  `make lint`), packages without unnecessary abstractions, errors are
  returned, `panic` is not used outside the `internal/webui` initialization
  of the embedded FS.
- Vue/TS: `<script setup lang="ts">`, PrimeVue components instead of custom
  UI elements, tab indentation (see `.prettierrc.json`), ESLint flat config.
- The application configuration (action cards) is a single JSON file — do
  not introduce an external database without a new ADR that justifies it
  (see [ADR-0004](docs/architecture/decisions/0004-action-card-domain-model.md)).
- Actions run on the host (shell commands) are security-sensitive — never
  add the ability to run arbitrary user input without validation/escaping;
  see the security requirements in [docs/requirements/requirements.md](docs/requirements/requirements.md).

## Development process (spec-driven, AI-agent driven)

The binding process is described in
[docs/devops/development-workflow.md](docs/devops/development-workflow.md).
The flow is: **request → requirement → ADR if needed → roadmap →
implementation block → approval → implementation + tests → Definition of
Done → closure and documentation update**.

For the individual steps use the prompts in `.github/prompts/`:

- `/new-requirement` — write/edit a requirement in requirements.md
- `/new-adr` — propose a new architectural decision
- `/plan-block` — elaborate a roadmap phase into a concrete implementation block
- `/implement-block` — implement one approved block incl. tests

Do not implement functionality that is not covered by a requirement, any
accepted ADR, and an approved implementation block in the roadmap. The
detailed rules for statuses, approval, deviations and the Definition of
Done are in the workflow document mentioned above.
