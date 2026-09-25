# Implementační blok: Stavová projekce a historie přechodů

- **Fáze**: 4 — Status/health-check engine
- **Vazba na požadavky**: FR-12, FR-17, FR-20, FR-35
- **Vazba na ADR**: ADR-0006
- **Stav**: Hotovo

## Cíl bloku

Po dokončení config vrstva umí držet poslední status kontrolu odděleně od
historie změn stavu. Opakovaná kontrola stejného stavu nezvětší historii;
při změně se předchozí stav uzavře a lze spočítat jeho trvání.

## Rozsah

- **Uvnitř**:
  - `StatusState` (`unknown`, `ok`, `fail`) a `StatusChange` s novým stavem,
    `StartedAt`, uzavřením/trváním;
  - poslední status projekce včetně posledního `Run` a času kontroly;
  - thread-safe Store API pro načtení projekce a atomické zpracování nové
    kontroly;
  - samostatný limit historie přechodů a ořezání nejstarších změn;
  - hluboké kopie návratových hodnot a JSON tagy;
  - aktualizace persistence snapshotu pro poslední status a přechody.
- **Mimo rozsah**:
  - spuštění status příkazu (0008);
  - časovače a polling (0009);
  - HTTP API, dashboard a migrace starého `history.status` formátu.

## Návrh řešení

Rozšířit `internal/config` o `StatusState`, `StatusChange` a projekci poslední
kontroly. Store bude při nové kontrole porovnávat předchozí stav; stejný stav
pouze nahradí poslední kontrolu, zatímco změna uzavře předchozí interval a
vloží nový otevřený interval. Historie bude vracet kopie v pořadí od
nejnovější změny.

## Testovací plán

- první kontrola vytvoří počáteční změnu z `unknown`;
- opakované `ok`/`fail` kontroly nevytvoří další změnu;
- `ok -> fail -> ok` uzavře intervaly správnou dobou trvání;
- ořezání historie zachová posledních N přechodů;
- souběžné kontroly jedné karty nezpůsobí race ani nekonzistentní interval;
- JSON round-trip poslední projekce a historie změn.

## Kritérium hotovosti

Viz Definition of Done. Specificky: `go test -race ./...` prochází a testy
prokazují, že počet status kontrol neovlivňuje počet historických změn.
