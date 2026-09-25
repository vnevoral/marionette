# Implementační blok: Status check service

- **Fáze**: 4 — Status/health-check engine
- **Vazba na požadavky**: FR-12, FR-14, FR-17, FR-19
- **Vazba na ADR**: ADR-0005, ADR-0006
- **Stav**: Hotovo

## Cíl bloku

Po dokončení lze programově provést jednu status akci karty přes execution
runner. Výsledek se uloží jako poslední kontrola a do historie se promítne
jen tehdy, když znamená změnu stavu.

## Rozsah

- **Uvnitř**:
  - service přijímající kartu nebo její status akci a card ID;
  - volání `execengine.Runner`;
  - mapování výsledku `ok`/`fail`/`timeout` na `StatusState`;
  - atomické předání výsledku do status projekce/store;
  - ruční/programové `CheckNow` API bez HTTP vrstvy.
- **Mimo rozsah**:
  - scheduler a polling intervaly (0009);
  - CRUD karet a HTTP endpointy (fáze 5);
  - uložení každé opakované kontroly do status historie.

## Návrh řešení

Nový balíček `internal/status` dostane store a `execengine.Runner` přes
konstruktor. `CheckNow(cardID)` načte kartu, odmítne kartu bez status akce,
spustí status action a předá `config.Run` jako poslední kontrolu. Store
rozhodne, zda vznikne `StatusChange`.

## Testovací plán

- status akce s výsledkem `ok`, `fail` a `timeout`;
- chybějící status akce a neexistující karta;
- opakované stejné výsledky aktualizují poslední kontrolu, ale ne historii;
- změna výsledku vytvoří jeden přechod s odpovídajícím časem;
- execution error neukončí service ani scheduler;
- fake runner/store a `go test -race`.

## Kritérium hotovosti

Viz Definition of Done. Specificky: `CheckNow` je testovatelný bez reálné sítě
nebo procesu a nikdy nezapisuje duplicitní status změnu při stejném výsledku.
