# Implementační blok: Standardní a zrychlený polling scheduler

- **Fáze**: 4 — Status/health-check engine
- **Vazba na požadavky**: FR-15, FR-15a, FR-17, FR-18
- **Vazba na ADR**: ADR-0006
- **Stav**: Návrh

## Cíl bloku

Po dokončení scheduler pravidelně spouští status kontroly karet se zapnutým
standardním pollingem a po primární akci dočasně používá fast interval. Po
uplynutí fast window se vrátí ke standardnímu intervalu.

## Rozsah

- **Uvnitř**:
  - start/stop lifecycle scheduleru přes `context.Context`;
  - per-card standardní interval (`PollingIntervalSeconds`);
  - aktivace fast window po programovém `NotifyPrimaryAction(cardID)`;
  - fast interval a návrat po `FastPollingWindowSeconds`;
  - ignorování fast nastavení, pokud je standardní polling vypnutý;
  - napojení na `StatusCheckService` a sdílený execution runner;
  - bezpečné přidání/odebrání karet při restartu scheduleru.
- **Mimo rozsah**:
  - samotné vyhodnocení status výsledku (0008);
  - API endpointy a UI;
  - perzistentní fronta polling úloh nebo prioritizace;
  - ukládání historie každého ticku.

## Návrh řešení

Nový scheduler v `internal/status` bude mít jeden lifecycle context a timer
pro každou aktivní kartu. `NotifyPrimaryAction` nastaví konec fast window pro
konkrétní kartu; při dalším plánování se použije fast interval. Po skončení
okna se timer vrátí na standardní interval. Kontroly půjdou přes stejný
runner jako ruční spuštění, takže platí globální limit souběžnosti.

## Testovací plán

- karta s pollingem spouští kontroly v nastaveném standardním intervalu;
- karta s pollingem 0 nespouští žádné kontroly;
- fast window po primární akci používá fast interval a po vypršení standardní;
- fast konfigurace bez standardního pollingu nemá efekt;
- stop ukončí timery a nevytvoří další kontroly;
- více karet má nezávislé intervaly a sdílí runner limit;
- fake clock/check service, bez čekání na reálné desítky sekund.

## Kritérium hotovosti

Viz Definition of Done. Specificky: `go test -race ./...` prochází a testy
ověří standardní interval, fast window i návrat na standardní polling.
