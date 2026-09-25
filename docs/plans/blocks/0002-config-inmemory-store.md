# Implementační blok: In-memory config store (CRUD karet + historie běhů)

- **Fáze**: 2 — Doménový model + config store (in-memory + JSON perzistence)
- **Vazba na požadavky**: FR-10, FR-17, FR-18, FR-31, FR-32, NFR-05, NFR-07
- **Vazba na ADR**: ADR-0004
- **Stav**: Návrh

## Cíl bloku

Po dokončení existuje thread-safe in-memory `Store` v `internal/config`
poskytující CRUD nad `ActionCard` a `Settings` a ukládající historii běhů
(`Run`) per akce jako ring buffer omezený `Settings.HistorySize`. Vše čistě
v paměti — bez čtení/zápisu na disk (to řeší blok 0003).

## Rozsah

- **Uvnitř**:
  - `Store` struct s `sync.RWMutex`, drží `Settings` a mapu
    `map[string]ActionCard` (klíč = `ActionCard.ID`).
  - `NewStore(settings Settings) *Store` — vytvoří prázdný store (bez karet).
  - `ListCards() []ActionCard`, `GetCard(id string) (ActionCard, bool)`.
  - `CreateCard(card ActionCard) (ActionCard, error)` — vygeneruje `ID`
    (pokud prázdné), zvaliduje (`card.Validate()` z bloku 0001), odmítne
    duplicitní ID.
  - `UpdateCard(id string, card ActionCard) (ActionCard, error)`,
    `DeleteCard(id string) error` — chyba `ErrNotFound`, pokud karta
    neexistuje.
  - `GetSettings() Settings`, `UpdateSettings(s Settings) error` — validuje;
    pokud se `HistorySize` zmenší, existující historie se ořízne na nový
    limit (zahodí nejstarší záznamy).
  - `AppendRun(cardID string, run Run) error` — přidá běh do ring bufferu
    dané karty+`run.ActionKind` (samostatná historie pro primární a status
    akci), udrží max `Settings.HistorySize` posledních záznamů.
  - `GetRuns(cardID string, actionKind string) ([]Run, error)` — vrátí kopii
    historie (od nejstaršího po nejnovější nebo obráceně — zvolit a
    zdokumentovat, doporučeno nejnovější první).
  - Veškeré návratové hodnoty jsou kopie (žádné sdílené mutable struktury
    mezi voláními), aby volající nemohl obejít mutex.
- **Mimo rozsah**: JSON perzistence a atomický zápis (blok 0003), spouštění
  příkazů/vyhodnocení `OutputRule` (fáze 3 — tento blok jen ukládá hotové
  `Run` záznamy, nevytváří je), HTTP API (fáze 5).

## Návrh řešení

Nový soubor `internal/config/store.go`. Interní stav:

```go
type Store struct {
    mu       sync.RWMutex
    settings Settings
    cards    map[string]ActionCard
    history  map[string]map[string][]Run // cardID -> actionKind -> ring buffer
}
```

Ring buffer lze implementovat jednoduše jako slice s ořezáváním na
`Settings.HistorySize` při každém `AppendRun` (na desítky karet a nízké N
není potřeba kruhový index).

## Testovací plán

- CRUD: create/get/update/delete happy path + `ErrNotFound` na
  update/delete neexistující karty, duplicitní ID na create.
- `AppendRun`/`GetRuns`: naplnění přes limit `HistorySize` ořízne nejstarší
  záznamy; historie primární a status akce se navzájem neovlivňuje.
- `UpdateSettings` se zmenšujícím `HistorySize` ořízne existující historie
  všech karet.
- Souběžný přístup: test s `go test -race` spouštějící CRUD a `AppendRun`
  ze více goroutin současně bez race podmínek.
- Vrácené slice/struct z `ListCards`/`GetRuns` nejsou ovlivněny pozdější
  mutací store (kopie, ne reference na interní stav).

## Kritérium hotovosti

Viz [Definition of Done](../../devops/definition-of-done.md). Specificky:
`go test -race ./...` čistě prochází pro `internal/config`.
