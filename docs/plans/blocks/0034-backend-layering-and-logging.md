# Implementační blok: Vrstvení backendu, strukturované logování a drobné čistky

- **Fáze**: 8 — Zpevnění
- **Vazba na požadavky**: NFR-03, NFR-05, NFR-06
- **Vazba na ADR**: ADR-0005, ADR-0008
- **Stav**: Schváleno
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

- **Stav po implementaci**: čeká
- **Ověření**: čeká
- **Dokumentace aktualizována**: čeká
