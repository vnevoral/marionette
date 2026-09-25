# ADR-0004: Doménový model akčních karet a JSON konfigurace

- **Stav**: Přijato (historie statusu zpřesněna v ADR-0006)
- **Datum**: 2026-09-25 (aktualizováno po zpřesnění požadavků na v0.2, v0.3 a v0.4)

## Kontext

Potřebujeme doménový model pro akční karty (FR-10 až FR-19) a způsob jejich
perzistence (FR-30 až FR-35), který je dost jednoduchý na jednotky až nízké
desítky karet a nevyžaduje provoz databázového serveru na Raspberry Pi. Po
zpřesnění požadavků (viz [requirements.md §7](../../requirements/requirements.md#7-rozhodnutí-fáze-1),
[§9](../../requirements/requirements.md#9-rozhodnutí-polling-a-terminace-2026-09-25)
a [§11](../../requirements/requirements.md#11-rozhodnutí-perzistence-historie-při-vypnutí-2026-09-25))
je potřeba doménový model doplnit o: pravidlo vyhodnocení na výstup (FR-14),
historii běhů (FR-17) vč. její perzistence při řízeném ukončení (FR-35),
globální limit souběžnosti (FR-18/NFR-07), dva polling intervaly per karta —
standardní a dočasně zrychlený po primární akci (FR-15, FR-15a) — a
explicitní vynucenou terminaci akce po timeoutu (FR-19).

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
    karta `StatusAction`): - `PollingIntervalSeconds` (0/nil = standardní polling vypnutý; výchozí
    doporučená hodnota `60`) — pravidelný interval, dokud není aktivní
    zrychlené okno. - `FastPollingIntervalSeconds` (výchozí `10`) — interval použitý po
    dobu zrychleného okna. - `FastPollingWindowSeconds` (výchozí `120`) — jak dlouho po vyvolání
    **primární akce** karty platí `FastPollingIntervalSeconds`, než se
    polling vrátí na `PollingIntervalSeconds`.
    Zrychlené okno se aktivuje **pouze** pokud má karta `PollingIntervalSeconds
    > 0`(standardní polling zapnutý) — bez něj`FastPollingIntervalSeconds`/
`FastPollingWindowSeconds` nemají efekt (FR-15a).
  - `Run` — jedno spuštění primární nebo status akce: exit kód, zachycený
    výstup (max 4 KB, s příznakem `Truncated`), čas startu/konce, odvozený
    výsledek (`ok`/`fail`/`timeout`) dle `OutputRule` (`timeout`, pokud engine
    akci vynuceně ukončil dle `TimeoutSec`). Status běhy slouží jako poslední
    kontrola, ne jako dlouhodobá historie každého pollingu.
  - `StatusChange` — skutečný přechod stavu status akce, s novým stavem,
    začátkem a koncem nebo dobou trvání; opakované kontroly beze změny se
    neukládají do historie (ADR-0006).
- Config store (`internal/config`, přesný název upřesní implementační blok) je
  in-memory struktura držící **dvě oddělené věci**:
  1. **Konfigurace** (persistovaná do JSON): seznam `ActionCard` a globální
     `Settings{ HistorySize int (default 20), MaxConcurrentActions int
(default 4) }`. Perzistuje se synchronně při každé mutační operaci
     (create/update/delete karty nebo změna `Settings`) — zápis do
     dočasného souboru a atomické přejmenování (`rename`), cesta dle
     `MARIONETTE_CONFIG` (default `./marionette.json`, viz FR-34).
  2. **Historie primárních běhů** (`Run`, ring buffer velikosti
     `Settings.HistorySize`) a **historie změn statusu** (`StatusChange`,
     samostatný limit posledních změn): během běhu se drží **jen v paměti** a
     průběžně se na disk nezapisují. Poslední status kontrola se drží jako
     aktuální projekce mimo historii. Důvodem je, že polling po 10–60 s
     vytváří mnoho shodných kontrol bez informační hodnoty.
     **Výjimka (FR-35)**: při **řízeném ukončení** aplikace (graceful
     shutdown po SIGINT/SIGTERM) se aktuální obsah historie jednorázově
     uloží do stejného konfiguračního souboru (nový top-level klíč
     `history`, viz níže) a při příštím startu se načte spolu s
     konfigurací. Když aplikace skončí neřízeně (pád, `SIGKILL`, výpadek
     napájení), poslední uložená historie zůstane stará/chybějící —
     akceptované riziko, není cuklá pojistka pro každý jednotlivý běh.
     JSON struktura souboru: `{"settings": Settings, "cards": []ActionCard,
"history": {cardID: {"primary": []Run, "status": []StatusChange}}}` — klíč
     `history` je volitelný (starší/ručně vytvořené soubory bez něj se
     načítají s prázdnou historií).
- Spouštění akcí je oddělené od config store (samostatný „execution engine“
  balíček, fáze 3) — config store nezná detaily `os/exec`, jen drží definice
  a přijímá zápis výsledků (`Run`) do historie dané akce.
- Souběžnost: execution engine drží globální semafor (buffered channel)
  velikosti `Settings.MaxConcurrentActions`; ruční spuštění i naplánovaný
  polling žádají o slot ze stejného semaforu — akce nad limit čekají ve
  frontě (FR-18), nezahazují se.
- Polling: samostatný scheduler (fáze 4) udržuje pro každou kartu s aktivním
  `PollingIntervalSeconds` časovač, který přes execution engine (a jeho
  semafor) spouští `StatusAction`, aktualizuje poslední status projekci a při
  skutečné změně stavu zapíše `StatusChange` do historie. Po každém
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
- Persistovat historii běhů při **každém** `AppendRun` — zamítnuto (viz
  výše, opotřebení SD karty při pollingu každých 10–60 s). Místo toho se
  historie ukládá jen jednorázově při řízeném vypnutí (FR-35) — kompromis
  mezi trvanlivostí přes běžný restart/update a počtem zápisů na disk.
- Periodické průběžné zapisování historie (např. jednou za minutu) —
  zamítnuto pro MVP jako zbytečná komplexita; lze zvážit později samostatným
  ADR, pokud se ukáže, že řízené vypnutí není dostatečné (např. časté
  neřízené pády v provozu).

## Důsledky

- Žádné externí závislosti na databázi ani ORM.
- Po **řízeném** restartu/vypnutí (systemd `stop`/`restart`, update binárky)
  je historie běhů zachována — načte se z `history` klíče konfiguračního
  souboru. Po **neřízeném** ukončení (pád, výpadek napájení, `SIGKILL`) je
  historie jen tak stará, jak poslední úspěšné řízené vypnutí — mezitím
  proběhlé běhy (vč. posledního zobrazeného stavu karty) se ztratí a UI musí
  umět zobrazit stav „neznámý“, dokud neproběhne nový health-check
  (v ideálním případě po startu rovnou vyvolat jeden status-check pro karty
  s pollingem — upřesní implementační blok fáze 4).
- `cmd/marionette` musí zachytávat `SIGINT`/`SIGTERM`, provést graceful
  shutdown HTTP serveru a až poté uložit store (config + history) — v tomto
  pořadí, aby neprobíhal zápis souběžně s ještě běžícími handlery.
- `Settings` (velikost historie, limit souběžnosti) jsou součástí
  persistované konfigurace a měnitelné přes stejné API/UI jako karty.
- Toto ADR je vstupem pro implementační blok(y) fáze 2 v roadmapě.
