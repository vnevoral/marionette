# Implementační blok: Ochrana konfigurace při poškozeném souboru a konzistence persistence

- **Fáze**: 8 — Zpevnění
- **Vazba na požadavky**: FR-30, FR-31, FR-33, FR-35, NFR-04
- **Vazba na ADR**: ADR-0004
- **Stav**: Hotovo
- **Závislosti**: Bloky 0002–0003 (store a persistence), 0010 (REST API), 0012 (kompozice)

## Cíl bloku

Po dokončení aplikace nikdy nepřepíše existující konfigurační soubor, který se
nepodařilo načíst, a stav v paměti zůstane konzistentní s odpovědí API i při
selhání zápisu na disk.

## Rozsah

- **Uvnitř**:
  - `config.LoadFile` rozlišuje sentinel chybami „soubor neexistuje“ od
    „soubor je nečitelný/poškozený“;
  - při poškozeném souboru `main` soubor přejmenuje na
    `<cesta>.corrupt-<RFC3339 čas>` před prvním zápisem, zaloguje varování a
    až poté nastartuje s prázdným store (FR-33 zůstává splněno);
  - rollback in-memory mutace ve `Store` (`Create/Update/Delete/UpdateStatus`),
    pokud `OnChange` vrátí chybu; API pak vrací 500 a stav v paměti odpovídá
    disku;
  - sentinel chyby `config.ErrNotFound`, `config.ErrAlreadyExists`,
    `config.ErrValidation`; handler mapuje přes `errors.Is`, ne porovnáváním
    textu;
  - atomický zápis: pevný název temp souboru `<cesta>.tmp`, zachování práv
    původního souboru (`Stat` + `Chmod`), `fsync` adresáře po `rename`;
  - `loadStatus` validuje `StatusSnapshot.State` stejně jako `UpdateStatus`.
- **Mimo rozsah**:
  - migrace formátu konfigurace, verzování schématu;
  - read-only režim API při poškozeném configu (zvoleno přejmenování, ne
    blokace);
  - šifrování nebo zálohy konfigurace nad rámec jednoho `.corrupt-*` souboru.

## Schválení

- **Schválil**: projektový vlastník
- **Datum schválení**: 2026-09-26
- **Poznámky k rozhodnutí**: Schváleno jako oprava chyb. Nález revize 2026-09-26 (H-3, M-3, M-4, L-2,
  L-3). Reprodukováno: `CreateTemp` + `Rename` mění práva `0644` na `0600`.

## Návrh řešení

- `internal/config/persistence.go`: `var ErrConfigMissing`, `ErrConfigCorrupt`;
  `LoadFile` obaluje chyby `fmt.Errorf("…: %w", ErrConfigCorrupt)`. Nová
  funkce `QuarantineFile(path string, now time.Time) (string, error)`.
- `cmd/marionette/main.go`: `switch { case errors.Is(err, ErrConfigMissing):
  info log; case errors.Is(err, ErrConfigCorrupt): QuarantineFile + warn }`.
- `internal/config/store.go`: mutace pod `mu` uloží předchozí hodnotu, po
  neúspěšném `OnChange` ji vrátí (pod stejným zámkem nebo přes
  `persistMu` sekvenci — zachovat dnešní oddělení zámků). `ErrAlreadyExists`
  vrací `fmt.Errorf("card %q: %w", id, ErrAlreadyExists)`.
- `internal/server/server.go`: `errors.Is(err, config.ErrAlreadyExists)` →
  409, `config.ErrNotFound` → 404, `config.ErrValidation` → 422.
- Atomický zápis: `writeFileAtomic(path, data)` s `path+".tmp"`, právy z
  `os.Stat(path)` (fallback `0600`), `dir.Sync()` po rename.

## Testovací plán

- Jednotkové testy (`internal/config`):
  - poškozený JSON → `errors.Is(err, ErrConfigCorrupt)`; neexistující →
    `ErrConfigMissing`; EACCES → `ErrConfigCorrupt` (nečitelný);
  - `QuarantineFile` přejmenuje a vrátí novou cestu; opakované volání ve
    stejné sekundě nekoliduje (sufix nebo `CreateTemp` vzor);
  - `OnChange` vrátí chybu → `Get` po `Create` vrátí `ErrNotFound`, po
    `Update` původní hodnotu, po `Delete` kartu stále vrací;
  - zápis zachová práva `0644` existujícího souboru; nový soubor je `0600`;
  - `loadStatus` s neznámým stavem soubor odmítne jako poškozený.
- Jednotkové testy (`internal/server`): 500 při selhání persistence a následný
  `GET` kartu nevrací; 409 přes sentinel; existující test
  `server_test.go` fixující „mutace zůstane“ přepsat na nové chování.
- Smoke test `main`: refaktor na `run(ctx, env) error` a test se
  zkomoleným souborem v temp adresáři ověří vznik `.corrupt-*` a zachování
  původního obsahu.
- `go test -race ./...`, `go vet ./...`.

## Kritérium hotovosti

Viz [Definition of Done](../../devops/definition-of-done.md) +:

- žádná cesta v kódu nepřepíše soubor, který `LoadFile` nedokázal načíst;
- `strings.Contains(err.Error(), …)` se v `internal/server` nevyskytuje;
- README a `docs/architecture/overview.md` popisují chování `.corrupt-*`.

## Uzavření

- **Stav po implementaci**: Hotovo (2026-09-26)
- **Ověření**: `make verify` prošel (golangci-lint, eslint, vue-tsc, prettier,
  `go test -race -count=1 ./...`, build, vet). Nové testy (všechny běžely,
  žádný přeskočen — testy oprávnění jsou podmíněné `os.Geteuid() != 0`):
  `TestLoadFileUnreadableReturnsUnreadableError`,
  `TestQuarantineFilePreservesContentAndAvoidsCollisions`,
  `TestSaveFilePreservesExistingPermissions` (0644 zachováno, nový soubor
  0600, žádný `.tmp` nezůstává), `TestLoadFileIgnoresStatusWithUnknownState`,
  `TestStoreRollsBackMutationsWhenPersistenceFails` (create/update/delete
  vč. obnovy historie a statusu, settings), `TestStoreErrorsAreClassifiable`,
  `TestRouterMapsValidationErrorsToUnprocessableEntity`, upravený
  `TestRouterMapsPersistenceFailureToInternalServerError` (karta po 500
  neexistuje, GET vrací 404), `TestOpenStoreMissingFileStartsEmptyAndPersists`,
  `TestOpenStoreQuarantinesCorruptFile` (původní obsah zachován i po dalším
  uložení), `TestOpenStoreUnreadableFileRejectsChanges`.
- **Odchylky od návrhu**: (1) nečitelný soubor (EACCES) se nekarantenuje —
  jeho obsah může být v pořádku, proto se rozlišuje `ErrConfigUnreadable` a
  aplikace startuje jen pro čtení (mutace vrací 500 a vrací se zpět), zatímco
  `ErrConfigCorrupt` vede ke karanténě; (2) `loadStatus` neplatný stav
  ignoruje s varováním místo odmítnutí souboru, protože jde o runtime stav
  a FR-35 vyžaduje, aby poškozený runtime stav aplikaci ani konfiguraci
  neshodil; (3) `main` byl refaktorován jen na `openStore(path, now)`,
  plný `run(ctx, env)` je naplánován v bloku 0027 spolu se shutdownem;
  (4) chybějící soubor nevrací sentinel `ErrConfigMissing` — zůstává
  dosavadní kontrakt „prázdný store, nil“ (FR-33); (5) `UpdateStatus`
  rollback nepotřebuje, protože status se nepersistuje při změně (ADR-0004);
  (6) validační chyby se mapují na 422 už v tomto bloku (dříve 400), v
  souladu s návrhem bloku 0028.
- **Dokumentace aktualizována**: ano — `docs/architecture/overview.md`
  (sekce „Ochrana konfiguračního souboru“), README (chování `.corrupt-*`
  a režimu jen pro čtení), godoc na `LoadFile`, `QuarantineFile`,
  `PersistenceError`, `writePersistedFile`, `writeStoreError`, roadmapa.
