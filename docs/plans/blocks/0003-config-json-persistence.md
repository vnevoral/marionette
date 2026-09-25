# Implementační blok: JSON perzistence konfigurace a načtení při startu

- **Fáze**: 2 — Doménový model + config store (in-memory + JSON perzistence)
- **Vazba na požadavky**: FR-30, FR-31, FR-33, FR-34
- **Vazba na ADR**: ADR-0004
- **Stav**: Návrh

## Cíl bloku

Po dokončení se konfigurace (`Settings` + seznam `ActionCard`, **bez**
historie běhů) persistuje do jednoho JSON souboru při každé mutační operaci
a načítá se z něj při startu aplikace. Cesta k souboru je konfigurovatelná
přes `MARIONETTE_CONFIG`. `cmd/marionette` při startu store vytvoří a načte.

## Rozsah

- **Uvnitř**:
  - Formát souboru: `{"settings": Settings, "cards": []ActionCard}`.
  - `LoadFile(path string) (*Store, error)` — pokud soubor neexistuje,
    vrátí prázdný `Store` s výchozím `Settings` (`DefaultHistorySize`,
    `DefaultMaxConcurrentActions`) a `nil` chybou (FR-33: chybějící soubor
    není chyba). Pokud soubor existuje, ale obsahuje neplatný JSON nebo
    nevaliduje se (`ActionCard.Validate()`), vrátí prázdný `Store` a chybu
    (volající — `cmd/marionette` — chybu zaloguje, ale pokračuje se startem,
    dle FR-33).
  - `(*Store) SaveFile(path string) error` — zápis do dočasného souboru ve
    stejném adresáři (`*.tmp`) a `os.Rename` na cílovou cestu (atomicita).
  - `Store` z bloku 0002 rozšířen o volitelný "persist hook": po každé
    úspěšné mutaci (`CreateCard`, `UpdateCard`, `DeleteCard`,
    `UpdateSettings`) store zavolá nakonfigurovanou funkci pro uložení
    (např. `Store.OnChange func(*Store) error`, nastavenou po `LoadFile`
    voláním `SetPersistPath` nebo obdobně) — `AppendRun` tento hook
    **nevolá** (historie se nepersistuje, viz ADR-0004).
  - `cmd/marionette/main.go`: čtení `MARIONETTE_CONFIG` (default
    `./marionette.json`), volání `config.LoadFile`, log chyby při
    poškozeném souboru, propojení store s persist hookem na stejnou cestu.
- **Mimo rozsah**: HTTP endpointy pro CRUD (fáze 5) — store je zatím jen
  interně vytvořený v `main.go`, bez napojení na `internal/server`
  (přidání proměnné/pole pro pozdější use lze, ale bez routování).

## Návrh řešení

Nový soubor `internal/config/persistence.go` s `LoadFile`/`SaveFile`.
Atomický zápis: `os.CreateTemp(dir, "marionette-*.json")`, zapsat, `Sync`,
`Close`, `os.Rename(tmp.Name(), path)`.

## Testovací plán

- `LoadFile` na neexistující cestě → prázdný store, žádná chyba.
- `LoadFile` na poškozeném JSON → chyba + prázdný store (ne panic).
- Round-trip: `SaveFile` → `LoadFile` vrátí stejná data (karty i settings).
- Zápis do souboru, ke kterému chybí oprávnění / neexistující adresář →
  čitelná chyba, aplikace/nevolá `panic`.
- `MARIONETTE_CONFIG` override: test na `main.go` úrovni není nutný
  (jednoduché čtení env), stačí unit test na `LoadFile`/`SaveFile` s
  explicitní cestou v `t.TempDir()`.

## Kritérium hotovosti

Viz [Definition of Done](../../devops/definition-of-done.md). Specificky:
manuální ověření, že `make backend-run` s prázdným/chybějícím
`marionette.json` naběhne bez pádu a že provedená (budoucí, zatím bez API)
mutace by se zapsala atomicky — do doby, než existuje API (fáze 5), lze
ověřit jen testy, ne end-to-end přes UI.
