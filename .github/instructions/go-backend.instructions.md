---
description: "Use when writing or modifying Go backend code in internal/ or cmd/ — routing, domain packages, config store, execution engine."
applyTo: "cmd/**/*.go,internal/**/*.go"
---

# Go backend

- Formát `gofumpt` (`gofumpt -modpath marionette -w`), `go vet` čistota a
  `golangci-lint` bez nálezů (`.golangci.yml`); exportované symboly mají
  godoc komentář.
- Chyby se vracejí (`error`), nepoužívá se `panic` mimo inicializaci
  vestavěného FS v `internal/webui`.
- Spouštění externích příkazů (execution engine, fáze 3) vždy přes
  `exec.CommandContext(ctx, name, args...)` s explicitním seznamem argumentů —
  nikdy neskládat shell příkaz jako string z uživatelského vstupu.
- Každé spuštění externího příkazu má timeout (`context.WithTimeout`).
- Doménová logika (config store, vyhodnocení akcí) je oddělená od HTTP vrstvy
  (`internal/server`) a testovatelná bez sítě/reálného spouštění procesů.
- Po každé změně: `go build ./...`, `go vet ./...`, `go test ./...`; před
  uzavřením `make verify`.
- Nová funkčnost musí mít odpovídající FR/NFR v
  [docs/requirements/requirements.md](../../docs/requirements/requirements.md)
  a případně ADR v
  [docs/architecture/decisions/](../../docs/architecture/decisions).
