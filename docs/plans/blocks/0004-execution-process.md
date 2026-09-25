# Implementační blok: Spuštění procesu a vynucený timeout

- **Fáze**: 3 — Execution engine
- **Vazba na požadavky**: FR-11, FR-19, NFR-01, NFR-04
- **Vazba na ADR**: ADR-0005
- **Stav**: Návrh

## Cíl bloku

Po dokončení existuje základ execution engine v `internal/exec`, který spustí
jednu validní `config.Action` bez shellu, nastaví pracovní adresář a prostředí
a po timeoutu proces vynuceně ukončí. Blok ještě neřeší globální frontu ani
polling.

## Rozsah

- **Uvnitř**:
  - veřejný executor API pro jedno spuštění akce;
  - `os/exec.CommandContext` se strukturovanými argumenty;
  - `Action.Dir` a `Action.Env`;
  - `context.WithTimeout` podle `Action.TimeoutSec`;
  - rozlišení dokončení, chyby procesu a timeoutu;
  - zachování start time a duration v interním výsledku.
- **Mimo rozsah**:
  - výstupní limit a regex/exit-code vyhodnocení (0005);
  - globální semaphore/fronta (0006);
  - zápis `Run` do config store, HTTP API a polling.

## Návrh řešení

Nový balíček `internal/exec` s package name `execengine`. Executor bude mít
malou testovatelnou abstrakci nad tvorbou procesu. Výsledkem bude struktura,
kterou další blok převede na `config.Run`; timeout bude jednoznačně označen.

## Testovací plán

- validní příkaz s argumentem a pracovním adresářem;
- předání proměnné prostředí;
- neexistující příkaz skončí jako chyba procesu bez pádu serveru;
- dlouhý příkaz po timeoutu skončí a nevisí dál;
- duration a timeout jsou deterministicky rozlišitelné;
- unit testy používají fake factory, integrační test timeoutu použije malý
  reálný příkaz.

## Kritérium hotovosti

Viz Definition of Done. Specificky: `go test -race ./...` a ověření, že po
timeoutu nezůstává spuštěný child process.
