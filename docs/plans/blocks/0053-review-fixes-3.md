# Implementační blok: Opravy z code review 3

- **Fáze**: 8 — Zpevnění (dodatek po uzavření fáze)
- **Vazba na požadavky**: FR-22a, FR-21, FR-17
- **Vazba na ADR**: ADR-0012 (doplnění 2026-09-27)
- **Stav**: Hotovo
- **Závislosti**: Bloky 0048, 0050

## Cíl bloku

Opravit dva nálezy z `/code-review` commitů `25a0094..ccd8191`:

1. **Víceřádkový argument v editoru příkazu** (`commandLine.ts`,
   `ActionEditor.vue`). Review hlásilo, že parser odmítne konec řádku i
   uvnitř uvozovek. Ověření ukázalo, že parser konec řádku v uvozovkách
   přijme a `parse(format(…))` ho zachová; skutečná chyba je v poli:
   `<input type="text">` konec řádku z hodnoty tiše odstraní
   (`sh -c 'echo a⏎echo b'` se zobrazí jako `sh -c 'echo aecho b'`) a první
   úprava řádku by skript změnila bez varování.
2. **Čekání na nový běh bez výchozího bodu** (`CardDetailView.vue`,
   `useCardActivity.ts`). Když se seznam běhů při kliknutí na **Run action**
   ještě načítal nebo jeho čtení selhalo, výchozím bodem bylo „žádný běh“,
   takže první úspěšné čtení vzalo starý běh za nový a čekání skončilo;
   nový běh se objevil až po obnovení stránky.

## Rozsah

- **Uvnitř**: oprava obou nálezů, testy, doplnění ADR-0012 a UX
  specifikace §7.2.
- **Mimo rozsah**: změna gramatiky (žádné nové escape sekvence), změna API.

## Schválení

- **Schválil**: projektový vlastník
- **Datum schválení**: 2026-09-27
- **Poznámky k rozhodnutí**: Pokyn „fix all findings“ po `/code-review`.

## Návrh řešení

1. `ActionEditor` zobrazí příkazový řádek s koncem řádku v PrimeVue
   `Textarea` (auto-resize) místo `InputText`; jakmile se pole jednou
   přepne, zůstane víceřádkové (žádná výměna prvku a ztráta fokusu při
   smazání konce řádku). Konec řádku mimo uvozovky zůstává chybou.
2. `useCardActivity` sleduje, zda seznam běhů odpovídá serveru (poslední
   čtení pro kartu uspělo), a `runBaseline()` vrací buď nejnovější známý
   `startedAt`, nebo `null`. S `null` čekání žádný běh za nový nepovažuje
   a seznam obnovuje po celý rozpočet (`timeoutSec + 30 s`), takže nový běh
   se objeví vždy.

## Testovací plán

- Vitest: `ActionEditor` (víceřádkový argument v `textarea`, náhled,
  pole zůstane `textarea`), `commandLine` (konec řádku v uvozovkách,
  zpětné složení s `\n` a `\r\n`), `cardEditModel` (uložení beze změny
  zachová víceřádkový skript), `CardDetailView` (neúspěšné a probíhající
  první čtení běhů: starý běh se za nový nepovažuje, nový se objeví,
  čtení skončí po rozpočtu).
- `make verify`, `make e2e`.

## Kritérium hotovosti

Viz [Definition of Done](../../devops/definition-of-done.md).

## Uzavření

- **Stav po implementaci**: Hotovo (2026-09-27)
- **Ověření**: `make verify` prošel (179 testů Vitest, `install_test`,
  `release_version_test`, lint, typy, `go test -race`); `make e2e` prošel
  (17 scénářů). Nové testy bez oprav selhávají (starý běh by ukončil
  čekání; `input` by zobrazil řádek bez konce řádku).
- **Odchylky od návrhu**: nález 1 opraven jinak, než review navrhovalo
  (parser byl správně, chyba byla v poli).
- **Dokumentace aktualizována**: ano — ADR-0012 (doplnění), UX
  specifikace §7.2, roadmapa.
