# ADR-0004: Doménový model akčních karet a JSON konfigurace

- **Stav**: Přijato
- **Datum**: 2026-09-25 (aktualizováno po zpřesnění požadavků na v0.2 a v0.3)

## Kontext

Potřebujeme doménový model pro akční karty (FR-10 až FR-19) a způsob jejich
perzistence (FR-30 až FR-34), který je dost jednoduchý na jednotky až nízké
desítky karet a nevyžaduje provoz databázového serveru na Raspberry Pi. Po
zpřesnění požadavků (viz [requirements.md §7](../../requirements/requirements.md#7-rozhodnutí-fáze-1)
a [§9](../../requirements/requirements.md#9-rozhodnutí-polling-a-terminace-2026-09-25))
je potřeba doménový model doplnit o: pravidlo vyhodnocení na výstup (FR-14),
historii běhů (FR-17), globální limit souběžnosti (FR-18/NFR-07), dva polling
intervaly per karta — standardní a dočasně zrychlený po primární akci
(FR-15, FR-15a) — a explicitní vynucenou terminaci akce po timeoutu (FR-19).

## Rozhodnutí

- Doménové entity:
  - `Action` — příkaz, argumenty (`[]string`, nikdy shell string), pracovní
    adresář, env, `TimeoutSec` a `OutputRule` pro vyhodnocení výsledku.
    `TimeoutSec` je závazná maximální doba běhu — execution engine (fáze 3)
    po jejím uplynutí proces vynuceně ukončí (`SIGKILL`/`cmd.Process.Kill()`
    přes `context.WithTimeout`), nikdy ho neponechá běžet na pozadí (FR-19).
  - `OutputRule` — `Type: "exit_code" | "match" | "not_match"` (výchozí jen
    `exit_code`), u `match`/`not_match` navíc `Pattern` (regulární výraz
    aplikovaný na zachycený, případně ořízlý výstup).
  - `ActionCard` — id, název, popis, ikona, `PrimaryAction`, volitelná
    `StatusAction`, a tři volitelné parametry pollingu (relevantní jen má-li
    karta `StatusAction`):
    - `PollingIntervalSeconds` (0/nil = standardní polling vypnutý; výchozí
      doporučená hodnota `60`) — pravidelný interval, dokud není aktivní
      zrychlené okno.
    - `FastPollingIntervalSeconds` (výchozí `10`) — interval použitý po
      dobu zrychleného okna.
    - `FastPollingWindowSeconds` (výchozí `120`) — jak dlouho po vyvolání
      **primární akce** karty platí `FastPollingIntervalSeconds`, než se
      polling vrátí na `PollingIntervalSeconds`.
      Zrychlené okno se aktivuje **pouze** pokud má karta `PollingIntervalSeconds
      > 0`(standardní polling zapnutý) — bez něj`FastPollingIntervalSeconds`/
`FastPollingWindowSeconds` nemají efekt (FR-15a).
  - `Run` — jedno spuštění akce: exit kód, zachycený výstup (max 4 KB, s
    příznakem `Truncated`), čas startu/konce, odvozený výsledek
    (`ok`/`fail`/`timeout`) dle `OutputRule` (`timeout`, pokud engine akci
    vynuceně ukončil dle `TimeoutSec`).
- Config store (`internal/config`, přesný název upřesní implementační blok) je
  in-memory struktura držící **dvě oddělené věci**:
  1. **Konfigurace** (persistovaná do JSON): seznam `ActionCard` a globální
     `Settings{ HistorySize int (default 20), MaxConcurrentActions int
(default 4) }`. Perzistuje se synchronně při každé mutační operaci
     (create/update/delete karty nebo změna `Settings`) — zápis do
     dočasného souboru a atomické přejmenování (`rename`), cesta dle
     `MARIONETTE_CONFIG` (default `./marionette.json`, viz FR-34).
  2. **Historie běhů** (`Run`, per akce, ring buffer velikosti
     `Settings.HistorySize`): drží se **jen v paměti**, nepersistuje se do
     JSON souboru. Důvod: jde o provozní/log data měnící se při každém běhu
     (i při automatickém pollingu po 30 s) — persistovat by znamenalo časté
     zápisy na SD kartu Raspberry Pi (opotřebení, NFR-03) bez odpovídající
     hodnoty (historie běhů není potřeba přežít restart). Po restartu se
     historie vynuluje, poslední konfigurace karet zůstává.
- Spouštění akcí je oddělené od config store (samostatný „execution engine“
  balíček, fáze 3) — config store nezná detaily `os/exec`, jen drží definice
  a přijímá zápis výsledků (`Run`) do historie dané akce.
- Souběžnost: execution engine drží globální semafor (buffered channel)
  velikosti `Settings.MaxConcurrentActions`; ruční spuštění i naplánovaný
  polling žádají o slot ze stejného semaforu — akce nad limit čekají ve
  frontě (FR-18), nezahazují se.
- Polling: samostatný scheduler (fáze 4) udržuje pro každou kartu s aktivním
  `PollingIntervalSeconds` časovač, který přes execution engine (a jeho
  semafor) spouští `StatusAction` a zapisuje `Run` do historie. Po každém
  spuštění **primární** akce scheduler přepne časovač dané karty na
  `FastPollingIntervalSeconds` a naplánuje návrat na `PollingIntervalSeconds`
  po `FastPollingWindowSeconds` (implementační detail — jednoduchý časový
  příznak „do kdy platí zrychlené okno“ u karty ve scheduleru, ne v
  persistované konfiguraci).

## Zvažované alternativy

- SQLite soubor — zamítnuto pro MVP, zbytečná komplexita (schema/migrace) pro
  desítky záznamů karet; historie běhů navíc řešena jako in-memory, takže
  potřeba perzistentního úložiště pro časové řady odpadá.
- Ukládání každé karty do vlastního souboru — zamítnuto, ztěžuje atomicitu a
  přehlednost (FR-30 vyžaduje jeden soubor).
- Persistovat i historii běhů do JSON — zamítnuto (viz výše, opotřebení
  SD karty); pokud se v budoucnu ukáže potřeba historii přežívající restart,
  řešit samostatným ADR (např. append-only log soubor place mimo hlavní
  config JSON).

## Důsledky

- Žádné externí závislosti na databázi ani ORM.
- Po restartu aplikace je historie běhů (a tedy i poslední zobrazený stav
  karty, dokud neproběhne nový health-check) prázdná — UI musí umět zobrazit
  stav „neznámý“ a v ideálním případě po startu rovnou vyvolat jeden
  status-check pro karty s pollingem (upřesní implementační blok fáze 4).
- `Settings` (velikost historie, limit souběžnosti) jsou součástí
  persistované konfigurace a měnitelné přes stejné API/UI jako karty.
- Toto ADR je vstupem pro implementační blok(y) fáze 2 v roadmapě.
