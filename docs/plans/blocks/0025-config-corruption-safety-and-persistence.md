# Implementační blok: Ochrana konfigurace při poškozeném souboru a konzistence persistence

- **Fáze**: 8 — Zpevnění
- **Vazba na požadavky**: FR-30, FR-31, FR-33, FR-35, NFR-04
- **Vazba na ADR**: ADR-0004
- **Stav**: Schváleno
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

- **Stav po implementaci**: čeká
- **Ověření**: čeká
- **Dokumentace aktualizována**: čeká
