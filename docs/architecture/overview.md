# Architektura — přehled

## Kontextový diagram

```mermaid
flowchart LR
    Operator["Operátor / Administrátor\n(prohlížeč)"] -->|HTTP/HTTPS| Marionette
    subgraph Host["Host (Raspberry Pi / Ubuntu)"]
        Marionette["Marionette\n(jeden Go proces)"]
        ConfigFile[("marionette.json\n(konfigurace karet)")]
        Marionette -->|čte/zapisuje| ConfigFile
        Marionette -->|spouští příkazy| OS["OS příkazy / procesy hosta\n(wakeonlan, ping, ...)"]
    end
```

## Komponenty (dnešní stav + plánované)

| Komponenta             | Umístění                      | Odpovědnost                                                                              | Stav                                     |
| ---------------------- | ----------------------------- | ---------------------------------------------------------------------------------------- | ---------------------------------------- |
| HTTP server / routing  | `internal/server`             | API routy, SPA fallback                                                                  | Implementováno (health endpoint)         |
| Embed vestavěného UI   | `internal/webui`              | `go:embed` `web/dist`                                                                    | Implementováno                           |
| SPA (Vue 3 + PrimeVue) | `web/src`                     | Dashboard, správa karet                                                                  | Implementován skelet (health check demo) |
| Config store           | `internal/config` (plán)      | In-memory karty + Settings (JSON perzistence); in-memory historie běhů (nepersistuje se) | Plánováno — ADR-0004 (přijato), fáze 2   |
| Execution engine       | `internal/exec` (plán)        | Bezpečné spouštění akcí na hostu, capture výstupu                                        | Plánováno — fáze 3                       |
| Status/health engine   | `internal/status` (plán)      | Vyhodnocení stavu karty, volitelný polling                                               | Plánováno — fáze 4                       |
| REST API domény        | `internal/server` (rozšíření) | CRUD karet/akcí, spouštění, čtení stavu                                                  | Plánováno — fáze 5                       |

## Vztah k dokumentaci

- Požadavky, které komponenty naplňují: [../requirements/requirements.md](../requirements/requirements.md)
- Zdůvodnění klíčových rozhodnutí: [decisions/](decisions)
- Kdy a v jakém pořadí se komponenty staví: [../plans/roadmap.md](../plans/roadmap.md)

## Bezpečnostní poznámka

Execution engine spouští procesy na hostu na základě uživatelské konfigurace.
Návrh musí od začátku počítat s: absencí shell interpolace uživatelského
vstupu (spouštět přes `exec.Command(name, args...)`, ne přes shell string),
timeoutem pro každý běh, a omezením přístupu k UI/API (viz NFR-01 a otevřené
otázky v requirements.md). Toto se doladí samostatným ADR před implementací
fáze 3.
