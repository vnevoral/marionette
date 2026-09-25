# Implementační blok: JSON perzistence konfigurace, historie a načtení při startu

- **Fáze**: 2 — Doménový model + config store (in-memory + JSON perzistence)
- **Vazba na požadavky**: FR-30, FR-31, FR-33, FR-34, FR-35
- **Vazba na ADR**: ADR-0004
- **Stav**: Hotovo

## Cíl bloku

Po dokončení se konfigurace (`Settings` + seznam `ActionCard`) persistuje do
jednoho JSON souboru při každé mutační operaci a načítá se z něj při startu
aplikace. Cesta k souboru je konfigurovatelná přes `MARIONETTE_CONFIG`.
Navíc: při **řízeném ukončení** aplikace (SIGINT/SIGTERM) se do téhož
souboru jednorázově uloží i aktuální historie běhů (FR-35) a při příštím
startu se načte spolu s konfigurací. `cmd/marionette` store při startu
vytvoří/načte a při shutdownu uloží.

## Rozsah

- **Uvnitř**:
  - Formát souboru: `{"settings": Settings, "cards": []ActionCard,
"history": {cardID: {"primary": []Run, "status": []StatusChange}}}` — klíč
    `history` je volitelný při čtení (chybí-li, historie je prázdná).
  - `LoadFile(path string) (*Store, error)` — pokud soubor neexistuje,
    vrátí prázdný `Store` s výchozím `Settings` (`DefaultHistorySize`,
    `DefaultMaxConcurrentActions`) a `nil` chybou (FR-33: chybějící soubor
    není chyba). Pokud soubor existuje, ale obsahuje neplatný JSON nebo
    nevaliduje se (`ActionCard.Validate()`), vrátí prázdný `Store` a chybu
    (volající — `cmd/marionette` — chybu zaloguje, ale pokračuje se startem,
    dle FR-33). Pokud jsou validní `settings`/`cards`, ale `history` chybí
    nebo je poškozená, načtení pokračuje s prázdnou historií (loguje se jen
    varování, ne chyba, viz FR-35).
  - `(*Store) SaveFile(path string) error` — uloží **jen** `settings` +
    `cards` (volá se po každé mutační operaci, viz persist hook níže).
  - `(*Store) SaveFileWithHistory(path string) error` — uloží `settings` +
    `cards` + `history`; volá se výhradně z graceful-shutdown cesty v
    `cmd/marionette`, ne po běžných mutacích ani po `AppendRun`.
  - Obě metody zapisují atomicky: do dočasného souboru ve stejném adresáři
    (`*.tmp`) a `os.Rename` na cílovou cestu.
  - `Store` z bloku 0002 rozšířen o volitelný "persist hook": po každé
    úspěšné mutaci (`CreateCard`, `UpdateCard`, `DeleteCard`,
    `UpdateSettings`) store zavolá nakonfigurovanou funkci pro uložení přes
    `SaveFile` (např. `Store.OnChange func(*Store) error`, nastavenou po
    `LoadFile`) — `AppendRun` tento hook **nevolá** (historie se
    nepersistuje průběžně, jen při shutdownu, viz ADR-0004).
  - `cmd/marionette/main.go`: čtení `MARIONETTE_CONFIG` (default
    `./marionette.json`), volání `config.LoadFile`, log chyby při
    poškozeném souboru/historii, propojení store s persist hookem na
    stejnou cestu. V existující obsluze `SIGINT`/`SIGTERM` (po zastavení
    HTTP serveru, `srv.Shutdown`) se navíc zavolá `store.SaveFileWithHistory`
    před ukončením procesu.
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
- Round-trip `SaveFile` → `LoadFile`: karty a settings shodné, historie
  prázdná (protože `SaveFile` historii neukládá).
  - Round-trip `SaveFileWithHistory` → `LoadFile`: karty, settings, primární
    historie běhů i historie přechodů statusu shodné.
- `LoadFile` na souboru s validními `settings`/`cards`, ale poškozeným
  `history` → načte se konfigurace, historie zůstane prázdná, zaloguje se
  varování (ne fatální chyba).
- Zápis do souboru, ke kterému chybí oprávnění / neexistující adresář →
  čitelná chyba, aplikace nevolá `panic`.
- `MARIONETTE_CONFIG` override: test na `main.go` úrovni není nutný
  (jednoduché čtení env), stačí unit test na `LoadFile`/`SaveFile`/
  `SaveFileWithHistory` s explicitní cestou v `t.TempDir()`.

## Kritérium hotovosti

Viz [Definition of Done](../../devops/definition-of-done.md). Specificky:
manuální ověření, že `make backend-run` s prázdným/chybějícím
`marionette.json` naběhne bez pádu, že `Ctrl+C` (SIGINT) uloží soubor vč.
historie a že opětovný start historii načte zpět — do doby, než existuje
API (fáze 5), lze historii do store dostat jen testem/dočasným voláním
`AppendRun`, ne end-to-end přes UI.
