---
description: "Use when writing or modifying Go backend code in internal/ or cmd/ — routing, domain packages, config store, execution engine."
applyTo: "cmd/**/*.go,internal/**/*.go"
---

# Go backend

- `gofumpt` formatting (`gofumpt -modpath marionette -w`), clean `go vet`
  and `golangci-lint` with no findings (`.golangci.yml`); exported symbols
  have a godoc comment.
- Errors are returned (`error`); `panic` is not used except for
  initializing the embedded FS in `internal/webui`.
- External commands (execution engine, phase 3) are always run via
  `exec.CommandContext(ctx, name, args...)` with an explicit argument
  list — never build a shell command as a string from user input.
- Every external command run has a timeout (`context.WithTimeout`).
- Domain logic (config store, action evaluation) is separated from the
  HTTP layer (`internal/server`) and testable without the network/real
  process execution.
- After every change: `go build ./...`, `go vet ./...`, `go test ./...`;
  before closing, `make verify`.
- New functionality must have a corresponding FR/NFR in
  [docs/requirements/requirements.md](../../docs/requirements/requirements.md)
  and, where applicable, an ADR in
  [docs/architecture/decisions/](../../docs/architecture/decisions).
