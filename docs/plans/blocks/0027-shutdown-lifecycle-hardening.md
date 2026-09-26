# Implementační blok: Řízené ukončení — pořadí kroků, SSE a fronta akcí

- **Fáze**: 8 — Zpevnění
- **Vazba na požadavky**: FR-35, FR-18, NFR-03, NFR-04, NFR-11
- **Vazba na ADR**: ADR-0004, ADR-0008
- **Stav**: Schváleno
- **Závislosti**: Blok 0024 (kontext skrz Runner), blok 0012 (lifecycle), blok 0022 (SSE)

## Cíl bloku

Po dokončení proběhne graceful shutdown deterministicky a rychle: historie
se uloží dříve, než by systemd mohl proces zabít, SSE klienti se odpojí
okamžitě a fronta akcí zahodí nespuštěné joby místo čekání na jejich doběh.

## Rozsah

- **Uvnitř**:
  - `StatusEventBroker.Close()` uzavře všechny odběratele a `events`
    handler skončí bez čekání na `srv.Close()`;
  - `http.Server` dostane `IdleTimeout` a `BaseContext` odvozený z
    aplikačního kontextu;
  - nové pořadí v `main`: stop příjmu HTTP → uložení config + historie
    (první průchod) → `BackgroundActions.Close()` (zahodí čekající joby,
    dokončí běžící s krátkým limitem) → `scheduler.Stop()` → finální
    uložení historie (druhý průchod, jen pokud se od prvního změnila);
  - `BackgroundActions.Close()` je idempotentní (`sync.Once`), loguje počet
    zahozených jobů;
  - odstranění `requestTracker` nebo zdokumentování důvodu jeho existence
    (rozhodne implementace podle toho, zda `srv.Shutdown` pokrývá potřebu);
  - chování fronty podle upřesněného FR-18: plná fronta → 503 s hlavičkou
    `Retry-After` (hodnota = odhad z délky fronty, min. 1 s); požadavek na
    kartu + druh akce, které už ve frontě čekají, se nezařazuje znovu a
    vrací 202 idempotentně (deduplikace jen pro čekající, ne pro běžící);
  - konfigurovatelný celkový limit shutdownu `MARIONETTE_SHUTDOWN_TIMEOUT`
    (výchozí 20 s, menší než systemd `TimeoutStopSec` 90 s) — dokumentovat
    v `deploy/marionette.default`.
- **Mimo rozsah**:
  - změna formátu uložené historie;
  - přerušení již běžícího procesu akce jiným způsobem než zrušením kontextu
    (řeší 0024);
  - persistence historie za běhu (ADR-0004 to záměrně nedělá).

## Schválení

- **Schválil**: projektový vlastník
- **Datum schválení**: 2026-09-26
- **Poznámky k rozhodnutí**: Schváleno jako oprava a narovnání stavu.
  Nález revize 2026-09-26 (M-2, M-5, M-8, L-9, L-10); FR-18 upřesněno
  téhož dne (sekce 13 requirements).
  Dnes každý shutdown s otevřeným dashboardem trvá plných 5 s a historie se
  ukládá až po doběhnutí fronty, což při dlouhých akcích porušuje FR-35.

## Návrh řešení

- `internal/server/status_events.go`: `done chan struct{}` v brokeru;
  `Close()` ho zavře pod zámkem a odpojí subscribery; `events` handler
  `select` na `r.Context().Done()`, `broker.done`, heartbeat a eventy.
- `cmd/marionette/main.go`: extrahovat `run(ctx, env, logger) error`;
  sekvence shutdownu jako samostatná funkce `shutdown(ctx, deps)` s
  jednotlivými kroky a logem doby trvání každého kroku.
- `internal/server/actions.go` (nebo nový balíček dle 0034):
  `Close()` zavře frontu, vyprázdní nezpracované joby, zruší kontext
  běžících a počká na `WaitGroup` nejvýše `shutdownTimeout`.
- `config.Store.SaveFileWithHistory` volaná dvakrát je levná (malý JSON),
  druhé volání jen pokud `store` hlásí změnu od posledního uložení
  (jednoduchý `dirty` příznak pod `persistMu`).

## Testovací plán

- Jednotkové testy:
  - `TestBrokerCloseEndsEventsHandler`: připojený `httptest` klient na
    `/api/events`, `Close()` → handler vrátí do 100 ms;
  - `TestBackgroundActionsCloseDropsQueued`: 10 jobů, 1 worker blokující na
    ctx, `Close()` → vrátí se do limitu, zahozené joby spočítané;
  - `TestBackgroundActionsCloseIdempotent`: dvojí `Close()` bez paniky;
  - `TestEnqueueDeduplicatesWaitingJob`: druhý enqueue stejné karty a druhu
    vrátí 202 bez nového jobu; po spuštění jobu je nový enqueue opět
    zařazen;
  - handler test: plná fronta → 503 a `Retry-After` ≥ 1;
  - `TestShutdownSavesHistoryBeforeQueueDrain`: fake store zaznamená pořadí
    volání `SaveFileWithHistory` vs `Close()`.
- Integrační smoke test `run()`: start, POST akce s dlouhým timeoutem,
  SIGTERM → proces skončí do 3 s a soubor obsahuje historii.
- `go test -race ./...`, `go vet ./...`.
- Manuální ověření na referenčním hostu: `systemctl restart marionette` s
  otevřeným dashboardem trvá < 3 s; journal ukazuje kroky shutdownu.

## Kritérium hotovosti

Viz [Definition of Done](../../devops/definition-of-done.md) +:

- shutdown s připojeným SSE klientem nečeká na 5 s timeout;
- historie je na disku dříve, než se čeká na frontu a scheduler;
- `MARIONETTE_SHUTDOWN_TIMEOUT` je zdokumentován v README a FR-34 doplněn o
  tuto proměnnou.

## Uzavření

- **Stav po implementaci**: čeká
- **Ověření**: čeká
- **Dokumentace aktualizována**: čeká
