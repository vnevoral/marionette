# Implementační blok: Opravy z code review 2 — backend (WaitDelay, lenientní load, atomický zápis, store)

- **Fáze**: 8 — Zpevnění (dodatek po uzavření fáze)
- **Vazba na požadavky**: FR-11, FR-13, FR-30..33, FR-35, NFR-04, NFR-05
- **Vazba na ADR**: ADR-0004, ADR-0005
- **Stav**: Hotovo
- **Závislosti**: Bloky 0003, 0025, 0028, 0036

## Cíl bloku

Po dokončení se úspěšná akce, jejíž potomek drží rouru s výstupem, zapíše jako
`ok`; konfigurace zapsaná před zpřísněním limitů se načte místo karantény;
souběžný zápis konfigurace nemůže poškodit soubor; chyba fsync adresáře po
úspěšném `rename` nevede k rollbacku v paměti; tři mutace store sdílí jeden
protokol rollbacku.

## Rozsah

- **Uvnitř** (nálezy code review 2026-09-26, druhé kolo):
  1. `internal/execengine/executor.go`: `exec.ErrWaitDelay` (proces skončil
     0, jen osiřelý potomek držel rouru) se vyhodnotí jako exit 0 podle
     pravidla výstupu; `ProcessErr` zůstává pro diagnostiku.
  2. `internal/config/types.go` + `persistence.go`: dvouúrovňová validace.
     `ValidateEssential()` (Action i ActionCard) kontroluje, co engine
     potřebuje: neprázdné ID a příkaz, `timeoutSec ≥ 1`, platné pravidlo
     výstupu, nezáporný polling. `Validate()` = essential + limity (délky,
     znaky ID, ikona, `MaxTimeoutSec`, vztah intervalů) a zůstává na API
     hranici (`CreateCard`/`UpdateCard`). `LoadFile` karantenuje soubor jen
     při selhání essential validace; karta porušující jen limity se načte
     s varováním (`card violates current limits`) a opraví se při první
     editaci. Executor validuje `ValidateEssential()`.
  3. `writePersistedFile`: unikátní temp soubor (`os.CreateTemp(dir,
     "<název>.*.tmp")`) při zachování práv (chmod na režim cílového
     souboru); `SaveFileWithHistory` bere `persistMu`, takže se serializuje
     s `OnChange` mutací.
  4. Chyba `syncDirectory` po úspěšném `rename` se vrací jako
     `ErrDirectorySync`; `SaveFileWithHistory` přesto označí snapshot jako
     uložený; `openStore` ji v `OnChange` zaloguje jako varování a
     nerollbackuje (soubor už obsahuje změnu).
  5. `internal/config/store.go`: společný `store.mutate(operation, apply)`
     pro `CreateCard`/`UpdateCard`/`DeleteCard` — lock, apply, `changes++`,
     `notifyChange`, při chybě undo + `changes--` + `PersistenceError`.
- **Mimo rozsah**:
  - migrace formátu konfigurace; změny API; frontend (blok 0039).

## Schválení

- **Schválil**: projektový vlastník
- **Datum schválení**: 2026-09-26
- **Poznámky k rozhodnutí**: Schváleno („oprav všechny nálezy“). Bod 3 mění
  rozhodnutí bloku 0025 (pevný `<cesta>.tmp`): důvod pevného názvu — ztráta
  práv při `CreateTemp` — je pokryt explicitním `chmod`, který kód už má;
  unikátní název vrací ochranu před dvěma zapisovateli.

## Návrh řešení

- `executor.go`: `if errors.Is(processErr, exec.ErrWaitDelay) { … evaluate
  as exit 0 }` před stávajícím `switch`.
- `types.go`: `func (action Action) validate(strict bool) fieldErrors`,
  `Validate()` = `validate(true)`, `ValidateEssential()` = `validate(false)`;
  totéž pro `ActionCard`.
- `persistence.go`: `ErrDirectorySync`; `LoadFile` loguje
  `card violates current limits` s `fields`.
- `main.go` `openStore`: `OnChange` wrapper — `ErrDirectorySync` → `Warn`,
  return nil.

## Testovací plán

- `TestExecutorTreatsWaitDelayAsSuccess` (fake proces vrací
  `exec.ErrWaitDelay`, výstup se vyhodnotí pravidlem; s `match` pravidlem a
  nesedícím výstupem → `fail`).
- `TestLoadFileKeepsCardOverCurrentLimits` (ID s tečkou, `timeoutSec`
  7200, ikona `pi pi-Home` → načteno, varování v logu), stávající
  `TestLoadFileInvalidCardReturnsErrorAndEmptyStore` (prázdný příkaz →
  stále `ErrConfigCorrupt`).
- `TestValidateEssentialIgnoresLimits`.
- `TestWritePersistedFileConcurrentWritersKeepFileValid` (N goroutin
  `SaveFile`/`SaveFileWithHistory`, výsledný soubor je platný JSON).
- `TestSaveFilePreservesExistingPermissions` zůstává zelený (chmod).
- `TestStoreKeepsMutationWhenOnlyDirectorySyncFails` (OnChange vrací
  `ErrDirectorySync` → karta zůstává, žádný `PersistenceError`) — přes
  wrapper v `openStore`: test v `cmd/marionette`.
- `make verify`.

## Kritérium hotovosti

Viz [Definition of Done](../../devops/definition-of-done.md) +:

- `grep -n 'changes--' internal/config/store.go` najde právě jeden výskyt
  (v `mutate`);
- `docs/architecture/overview.md` popisuje lenientní load.

## Uzavření

- **Stav po implementaci**: Hotovo (2026-09-26)
- **Ověření**: `make verify` prošel (lint, `vue-tsc`, Prettier, Vitest,
  `vite build`, `go test -race`, `go vet`). Nové testy:
  `TestExecutorTreatsWaitDelayAsSuccess` (výstup se dál hodnotí pravidlem),
  `TestExecutorRunsActionOverCurrentLimits` (timeout nad `MaxTimeoutSec`
  běží, nulový ne), `TestLoadFileKeepsCardOverCurrentLimits` (ID s tečkou,
  ikona `pi pi-Home`, timeout 3601 → načteno s varováním, `UpdateCard`
  stále odmítne), `TestValidateEssentialIgnoresLimits` (8 essential
  porušení), `TestWritePersistedFileConcurrentWritersKeepFileValid` (8 × 2
  souběžných uložení, soubor načitatelný, žádné `*.tmp`),
  `TestPersistOnChangeSavesAndReportsWriteFailures`. Stávající
  `TestLoadFileInvalidCardReturnsErrorAndEmptyStore` (prázdný příkaz →
  `ErrConfigCorrupt`) a `TestSaveFilePreservesExistingPermissions` zůstávají
  zelené. `grep -c 'changes--' store.go` = 1.
- **Odchylky od návrhu**: (1) Větev `ErrDirectorySync` v `persistOnChange`
  není pokrytá testem — selhání `fsync` adresáře po úspěšném `rename` nelze
  v testu vyvolat bez injektování souborového systému; pokryto je, že běžná
  chyba zápisu prochází a není klasifikována jako `ErrDirectorySync`. (2)
  Rozhodnutí bloku 0025 o pevném názvu `<cesta>.tmp` je tímto blokem
  nahrazeno (důvod — ztráta práv — řeší explicitní `chmod`); blok 0025 se
  zpětně nemění.
- **Dokumentace aktualizována**: ano — `docs/architecture/overview.md`
  (sekce „Ochrana konfiguračního souboru“: essential vs. plná validace,
  unikátní temp soubor, `persistMu`, `ErrDirectorySync`), roadmapa.
