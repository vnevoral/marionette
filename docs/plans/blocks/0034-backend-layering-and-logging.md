# Implementační blok: Vrstvení backendu, strukturované logování a drobné čistky

- **Fáze**: 8 — Zpevnění
- **Vazba na požadavky**: NFR-03, NFR-05, NFR-06
- **Vazba na ADR**: ADR-0005, ADR-0008
- **Stav**: Hotovo
- **Závislosti**: Bloky 0024, 0025, 0027 (aby se refaktor dělal nad opraveným kódem)

## Cíl bloku

Po dokončení je `internal/server` čistě HTTP vrstva, aplikační služby
(fronta akcí, broker událostí) mají vlastní balíčky, logování používá
`log/slog` s úrovněmi a kontextem karty a scheduler má sjednocenou logiku
výběru karet.

## Rozsah

- **Uvnitř**:
  - přesun `BackgroundActions` do `internal/actions`, `StatusEventBroker`
    do `internal/events`; `NewRouterWithDependencies` přijímá rozhraní
    místo mutace `Store.OnStatusChange` (kompozice v `main`);
  - odstranit variadický `NewRouter(stores ...*config.Store)`;
  - `log/slog` s `TextHandler` (journald čitelný) nebo `JSONHandler` podle
    `MARIONETTE_LOG_FORMAT`; úroveň přes `MARIONETTE_LOG_LEVEL`; atributy
    `card`, `action`, `outcome`, `duration`;
  - scheduler: společná `desiredCards()` pro `Start` i `Reconcile`,
    přejmenovat pole `context` → `ctx`, po zrušení rodičovského kontextu
    `running=false`;
  - `cloneStatusSnapshot` odstranit nebo zdokumentovat;
  - godoc pro všechny exportované chyby a typy (`revive` z 0031 to
    vynutí);
  - přejmenování adresáře `internal/exec` → `internal/execengine`, aby
    název adresáře odpovídal balíčku (aktualizace ADR-0005 odkazu);
  - `actionEnvironment`: minimální základní prostředí (`PATH`, `HOME`,
    `LANG`, `TZ`) + `action.Env` místo dědění celého `os.Environ()`;
    dokumentovat v ADR-0005 (NFR-01 c už přijato; dopad na uživatele:
    proměnné služby nejsou akcím dostupné);
  - `UpdateSettings` se z veřejného API store odstraní včetně testů
    (rozhodnuto 2026-09-26, YAGNI); README uvede, že limit souběžnosti se
    mění v konfiguračním souboru a projeví se po restartu.
- **Mimo rozsah**:
  - změny chování API kromě `settings` rozhodnutí;
  - metriky/Prometheus endpoint (samostatný requirement, pokud bude).

## Schválení

- **Schválil**: projektový vlastník
- **Datum schválení**: 2026-09-26
- **Poznámky k rozhodnutí**: Schváleno; minimální prostředí akcí je
  zakotveno v NFR-01 (c), `settings` endpoint se nezavádí. Nález revize
  2026-09-26 (M-6, L-4, L-5, L-6, L-11, L-12, L-13, L-14).

## Návrh řešení

- `internal/actions/queue.go`: `type Queue` s `Enqueue(ctx, job) error`,
  `Close(ctx) error`; rozhraní `Runner` definované na straně konzumenta.
- `internal/events/broker.go`: `Broker` s `Publish`, `Subscribe`, `Close`;
  `internal/server/status_events.go` zůstane jen SSE zápis.
- `cmd/marionette/main.go`: `store.OnStatusChange = broker.Publish`,
  `server.NewRouter(server.Dependencies{Store, Queue, Broker, Logger})`.
- `slog`: logger předáván explicitně přes `Dependencies`, žádný globální
  stav; testy používají `slog.New(slog.NewTextHandler(io.Discard, nil))`.

## Testovací plán

- Existující testy projdou po přesunu balíčků beze změny chování (pouze
  importy);
- test, že `NewRouter` nemutuje předaný store;
- test `desiredCards()` pro karty bez status akce a bez pollingu;
- test minimálního prostředí akce: `env` příkaz vrací jen očekávané klíče;
- `go test -race ./...`, `golangci-lint run ./...`, `go vet ./...`.

## Kritérium hotovosti

Viz [Definition of Done](../../devops/definition-of-done.md) +:

- `internal/server` neimportuje `sync.WaitGroup` ani nedrží goroutiny mimo
  HTTP handlery;
- `log.Printf` se v `cmd`/`internal` nevyskytuje;
- `docs/architecture/overview.md` odpovídá novým balíčkům.

## Uzavření

- **Stav po implementaci**: Hotovo (2026-09-26)
- **Ověření**: `make verify` prošel (golangci-lint, eslint, vue-tsc, prettier,
  `go test -race -count=1 ./...`, build, vet). Stávající testy fronty a
  brokeru přesunuty beze změny chování do `internal/actions/queue_test.go`
  a `internal/events/broker_test.go` (jen importy a názvy). Nové testy:
  `TestNewRouterDoesNotMutateStoreAndStreamsBrokerEvents` (router
  `Store.OnStatusChange` nenastaví; po napojení v kompozici SSE událost
  dorazí), `TestSchedulerDesiredCardsRequireStatusActionAndPolling`,
  `TestSchedulerStopsWhenParentContextIsCanceled` (po zrušení rodiče
  `Reconcile`/`Stop` vrací `ErrSchedulerStopped`, `Start` lze zopakovat),
  `TestActionEnvironmentIsMinimalAndOverridable`,
  `TestExecutorRunsActionWithMinimalEnvironment` (skutečný `env`: jen
  `PATH`/`HOME`/`LANG`/`TZ` + `Action.Env`, proměnná služby neunikne),
  `TestBrokerCloseIsIdempotentAndDropsSubscribers`, rozšířený
  `TestLoadEnvironmentDefaultsAndShutdownTimeout` (`MARIONETTE_LOG_FORMAT`,
  `MARIONETTE_LOG_LEVEL`, JSON výstup s atributy). Kritéria: `internal/server`
  neimportuje `sync` a nespouští goroutiny (ověřeno grepem), `log.Printf`
  se v `cmd`/`internal` nevyskytuje, `git grep internal/exec"` prázdný.
- **Odchylky od návrhu**: (1) fronta si ponechala `EnqueuePrimary`/
  `EnqueueStatus` místo generického `Enqueue(ctx, job)` — deduplikace
  potřebuje kartu a druh a stávající testy tak zůstaly beze změny;
  (2) `config.LoadFile(path, logger)` přijímá logger explicitně (nil =
  zahodit) místo návratu varování jako dat; (3) scheduler po zrušení
  rodičovského kontextu přechází do `running=false` přes sledovací
  goroutinu (`watchParent`), `Stop()` pak vrací `ErrSchedulerStopped`, což
  shutdown už toleruje; (4) `cloneStatusSnapshot` a nepoužívaný
  `cloneHistory` odstraněny (`StatusSnapshot` je hodnotový typ bez
  referencí, zdokumentováno u `OnStatusChange`); (5) `Dependencies.Logger`
  je volitelný (nil = zahodit), aby testy handlerů nemusely logger
  předávat; (6) `slog.DiscardHandler` není v Go 1.23, používá se
  `TextHandler` nad `io.Discard`; (7) `main` při chybě loguje přes `slog`
  a volá `os.Exit(1)` místo `log.Fatalf`; (8) `MARIONETTE_LOG_LEVEL` se
  parsuje přes `slog.Level.UnmarshalText`, takže přijímá i tvary
  `INFO+2`.
- **Dokumentace aktualizována**: ano — AGENTS.md a README (rozložení
  balíčků, proměnné logování, změna `settings` jen v souboru + restart,
  minimální prostředí akcí), `docs/architecture/overview.md` (tabulka
  komponent, sekce „Kompozice a logování“), ADR-0005 (přejmenování
  adresáře, minimální prostředí), ADR-0008 (balíček `events`), requirements
  FR-34 (proměnné logování), `deploy/marionette.default`, roadmapa.
