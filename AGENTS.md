# Marionette — instrukce pro AI agenty

Tento soubor je hlavní vstupní bod pro jakéhokoli AI coding agenta (Copilot,
Claude Code, Cursor, ...) pracujícího v tomto repozitáři. Čti ho vždy jako
první před zahájením práce.

## O projektu

Marionette je samostatně nasaditelná aplikace (jeden binární soubor) běžící
jako služba na Raspberry Pi (Ubuntu, linux/arm64) nebo obecně na Linuxu, bez
nutnosti instalovat jakýkoli runtime (Node.js, JVM, ...) na cílovém stroji.
Aplikace poskytuje webové rozhraní (dashboard) pro definici a ovládání tzv.
**akčních karet** (action cards) — každá karta spouští na hostu konfigurovaný
příkaz/akci a umí ověřit výsledný stav pomocí přidružené status/health-check
akce (např. Wake-on-LAN + ping). Konfigurace karet a akcí se persistuje do
jednoho konfiguračního souboru (JSON) a za běhu žije v in-memory struktuře.

Podrobný a závazný popis požadavků a rozhodnutí najdeš v [docs/](docs/README.md):

- [docs/requirements/requirements.md](docs/requirements/requirements.md) — co se má postavit (SRS)
- [docs/architecture/overview.md](docs/architecture/overview.md) — jak je to postavené
- [docs/architecture/decisions/](docs/architecture/decisions) — ADR log (proč)
- [docs/plans/roadmap.md](docs/plans/roadmap.md) — fáze a implementační bloky
- [docs/devops/testing-strategy.md](docs/devops/testing-strategy.md) a [docs/devops/ci-cd.md](docs/devops/ci-cd.md)

Pokud cokoliv v kódu nesouhlasí s dokumenty výše, dokumenty jsou zdroj pravdy —
nejprve navrhni jejich aktualizaci (ADR / requirements), teprve poté měň kód.

## Architektura v kostce

- `cmd/marionette` — entrypoint: kompozice služeb, `slog`, lifecycle a
  graceful shutdown
- `internal/server` — čistě HTTP vrstva: routy, validace požadavků,
  cross-site ochrana, SSE zápis, SPA fallback (žádné goroutiny mimo handlery)
- `internal/config` — doménový model karet, in-memory store, JSON persistence
- `internal/execengine` — bezpečné spouštění akcí (procesní skupina, timeout,
  minimální prostředí) a limit souběžnosti
- `internal/status` — vyhodnocení statusu karty a polling scheduler
- `internal/actions` — fronta akcí na pozadí (bounded, deduplikace, drain)
- `internal/events` — broker statusových událostí pro SSE
- `internal/webui` — `go:embed` vestavěného `web/dist` do binárky
- `web` — Vue 3 + PrimeVue 4 (Aura theme) SPA, buildí se přes Vite do
  `internal/webui/dist`

Výsledkem `make build` je jeden binární soubor bez externích závislostí za
běhu (žádný Node.js na cíli, žádný samostatný webserver).

## Build & test příkazy

```bash
make ui-install     # jednou: npm install pro web/
make ui-dev          # Vite dev server :5173 (proxy /api -> :8080)
make backend-dev     # Go backend s hot-reload (air) na :8080
make build           # build UI + embed + Go binárka pro aktuální platformu
make build-arm64     # cross-compile pro Raspberry Pi (linux/arm64)
make test            # go test -race ./... + npm test (Vitest)
make lint            # golangci-lint + eslint + vue-tsc + prettier --check
make verify          # jediný validační příkaz: build UI + lint + test + go build/vet
```

Pro web samostatně (`cd web`): `npm run lint`, `npm run lint:types`,
`npm run format` / `npm run format:check`, `npm run build`
(`vue-tsc -b && vite build`).

Lokální dev konfigurace `./marionette.json` je ignorovaná gitem a vzniká
z `deploy/dev-fixture.json` příkazem `make dev-config` (volá se automaticky
z `make backend-dev` / `backend-run`).

**Před uzavřením jakékoli změny spusť `make verify`** — je to jediné místo,
kde je definována sada validačních příkazů (CI a Definition of Done na něj
odkazují). Při rychlé iteraci lze použít dílčí `make test`, `make lint`,
nebo přímo `go test ./...` a `npm run lint`.

## Konvence

- Go: formát `gofumpt` (přísnější nadmnožina `gofmt`, vynucuje `make lint`),
  balíčky bez zbytečných abstrakcí, chyby se vracejí,
  nepoužívá se `panic` mimo `internal/webui` inicializaci vestavěného FS.
- Vue/TS: `<script setup lang="ts">`, PrimeVue komponenty místo vlastních UI
  prvků, odsazení tabulátorem (viz `.prettierrc.json`), ESLint flat config.
- Konfigurace aplikace (akční karty) je jeden JSON soubor — nezaváděj externí
  databázi bez nové ADR, která to zdůvodní (viz [ADR-0004](docs/architecture/decisions/0004-action-card-domain-model.md)).
- Akce spouštěné na hostu (shell příkazy) jsou bezpečnostně citlivé — nikdy
  nepřidávej możnost spouštět libovolný uživatelský vstup bez validace/escapingu;
  viz bezpečnostní požadavky v [docs/requirements/requirements.md](docs/requirements/requirements.md).

## Proces vývoje (spec-driven, AI-agent řízený)

Závazný proces je popsán v
[docs/devops/development-workflow.md](docs/devops/development-workflow.md).
Postup je: **požadavek → requirement → ADR podle potřeby → roadmapa →
implementační blok → schválení → implementace + testy → Definition of Done →
uzavření a aktualizace dokumentace**.

Pro jednotlivé kroky použij prompty v `.github/prompts/`:

- `/new-requirement` — zápis/úprava požadavku do requirements.md
- `/new-adr` — návrh nového architektonického rozhodnutí
- `/plan-block` — rozpracování fáze z roadmapy do konkrétního implementačního bloku
- `/implement-block` — implementace jednoho schváleného bloku vč. testů

Neimplementuj funkčnost, která není pokrytá requirementem, případným přijatým
ADR a schváleným implementačním blokem v roadmapě. Detailní pravidla stavů,
schvalování, odchylek a Definition of Done jsou v uvedeném workflow dokumentu.
