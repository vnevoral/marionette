# Implementační blok: Globální limit souběžných akcí

- **Fáze**: 3 — Execution engine
- **Vazba na požadavky**: FR-18, NFR-07
- **Vazba na ADR**: ADR-0005
- **Stav**: Hotovo

## Cíl bloku

Po dokončení sdílí všechny execution runs jeden globální semaphore podle
`Settings.MaxConcurrentActions`. Akce nad limitem čekají ve frontě a po
uvolnění slotu pokračují; žádná se tiše nezahodí.

## Rozsah

- **Uvnitř**:
  - concurrency-limited runner nad executor API z bloků 0004–0005;
  - slot před spuštěním procesu a uvolnění ve všech cestách dokončení/chyby;
  - konfigurace limitu z `config.Settings`;
  - bezpečná změna limitu vytvořením nové runner instance pro nové nastavení;
  - testy pořadí dokončení, čekání a uvolnění slotu po timeoutu.
- **Mimo rozsah**:
  - polling a fast-polling window (fáze 4);
  - prioritizace nebo zrušení čekajících akcí;
  - HTTP API a zápis výsledků do historie.

## Návrh řešení

Použít buffered channel jako semaphore a `defer` pro vrácení slotu. Veřejné
API zachová možnost synchronního spuštění; volání nad limitem blokuje místo
tichého dropu. Limit bude kontrolován při vytvoření runneru a odmítne
neplatné nastavení.

## Testovací plán

- při limitu 1 běží současně nejvýše jedna akce;
- druhá akce čeká a následně se spustí;
- timeout i chyba procesu vždy uvolní slot;
- více souběžných callerů neztratí žádný běh;
- test s `go test -race` a fake executorem bez reálných procesů.

## Kritérium hotovosti

Viz Definition of Done. Specificky: `go test -race ./...` prochází a test
prokazuje, že globální limit platí pro všechny runnery sdílející jednu
konfiguraci.
