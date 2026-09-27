# Contributing to Marionette

Thanks for your interest in Marionette. Bug reports, ideas and pull
requests are welcome. This page explains how the project works so that a
contribution can be accepted without surprises.

## Before you start

- **Bugs and ideas** go to
  [GitHub Issues](https://github.com/vnevoral/marionette/issues); use the
  bug report or feature request form.
- **Security problems** must not be reported in a public issue; follow
  [SECURITY.md](SECURITY.md).
- **Larger changes** (new features, changes to the API, the configuration
  file or the installation) should start with an issue. Marionette is
  developed spec-first, so a new feature needs an agreed requirement before
  code is written.

## How changes are planned

The project follows a spec-driven workflow described in
[docs/devops/development-workflow.md](docs/devops/development-workflow.md):

request → requirement → ADR (if the architecture changes) → roadmap →
implementation block → approval → implementation and tests → Definition of
Done → closure.

In practice:

- Requirements live in
  [docs/requirements/requirements.md](docs/requirements/requirements.md),
  architecture decisions in
  [docs/architecture/decisions/](docs/architecture/decisions) and
  implementation blocks in [docs/plans/blocks/](docs/plans/blocks).
- A small bug fix or a documentation correction can go straight to a pull
  request.
- A new feature or a behavior change needs a requirement and an
  implementation block approved by the maintainer. You may propose them in
  the same pull request, or in the issue first.
- If the code and the documents disagree, the documents win: update them
  first, then the code.

## Development setup

The easiest way is the dev container (VS Code: "Reopen in Container"). It
provides Go, Node 22, golangci-lint and the Playwright browser. Without it,
install Go 1.27 or newer and Node 22 (`web/.nvmrc`).

```bash
make ui-install   # once, installs npm dependencies
make ui-dev       # Vite dev server on :5173, proxies /api to :8080
make backend-dev  # Go backend with hot reload on :8080
```

See the [README](README.md#development) for the local configuration and
device pairing during development.

## Checks

Run before every pull request:

```bash
make verify   # build UI + lint + unit tests + go build/vet (the same as CI)
make e2e      # browser tests against the built binary, for UI flow changes
```

CI runs the same checks on every pull request; lint warnings fail the
build. New behavior comes with tests (Go tests, Vitest, or Playwright for
end-to-end flows); see
[docs/devops/testing-strategy.md](docs/devops/testing-strategy.md) and the
[Definition of Done](docs/devops/definition-of-done.md).

## Conventions

- **English** for everything committed: code, comments, documentation and
  commit messages.
- **Go:** `gofumpt` formatting, errors are returned, no `panic` outside the
  embedded UI initialization, no new dependencies without a reason.
- **Vue/TypeScript:** `<script setup lang="ts">`, PrimeVue components
  instead of custom UI elements, tab indentation, Prettier and ESLint.
- **Security:** actions run commands on the host. Never pass user input
  through a shell or run it without validation; see the security
  requirements (NFR-01, NFR-12, NFR-13) in the requirements document.
- **Storage:** the configuration stays a single JSON file; a database needs
  an ADR.
- **Commit messages** follow the `type: summary` style used in the history
  (`feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `chore:`).

AI coding agents working in this repository start at
[AGENTS.md](AGENTS.md).

## Pull requests

- Keep a pull request focused on one change.
- Fill in the pull request template: what changed, why, which requirement
  or block it covers and how it was verified.
- Update the documentation that describes the changed behavior (README,
  requirements, block status, roadmap).
- Releases are tagged and published by the maintainer only.

## License

By contributing you agree that your contribution is licensed under the
[MIT License](LICENSE) of the project.
