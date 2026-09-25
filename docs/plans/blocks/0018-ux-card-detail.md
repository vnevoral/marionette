# Implementační blok: UX-04 — Detail karty, běhy a status timeline

- **Fáze**: 9 — UX redesign a sdílený design systém
- **Vazba na požadavky**: FR-17, FR-20, FR-21, FR-24, FR-25, FR-26, FR-27, FR-29, NFR-08..10
- **Vazba na ADR**: ADR-0003, ADR-0006, ADR-0007
- **Stav**: Hotovo
- **Závislosti**: Bloky 0015–0017, read-only API runs/status/history

## Cíl bloku

Detail karty poskytne diagnostický pohled na poslední známý status, poslední
běhy primární akce a historii skutečných status transitions. Uživatel se z
Overview dostane na detail jedním jasným odkazem.

## Rozsah

- **Uvnitř**:
  - route `/cards/:id` a odkaz `View details` z Overview;
  - header s názvem, ikonou, stavem a poslední kontrolou;
  - summary aktuálního stavu a délky posledního status intervalu;
  - actions `Run action` a `Check status`;
  - recent primary runs s outcome, exit code, časem, délkou a sbalitelným outputem;
  - status transition timeline s novým stavem, začátkem, koncem a délkou;
  - loading, empty, not-found a error states;
  - responsive layout a keyboard-accessible controls.
- **Mimo rozsah**:
  - změny backend API nebo job IDs;
  - editace konfigurace přímo v detailu;
  - live websocket updates;
  - status run tabulka oddělená od posledního status snapshotu.

## Schválení

- **Schválil**: projektový vlastník
- **Datum schválení**: 2026-09-25
- **Poznámky k rozhodnutí**: Detail používá read-only API a zachovává oddělení provozu a konfigurace.

## Návrh řešení

Rozšířit `web/src/api.ts` o `getCard`, `getRuns` a `getStatusHistory`. Přidat
`CardDetailView.vue` s lokálním načtením card/status/runs/history přes
`Promise.allSettled`, aby selhání jedné read-only sekce neshodilo ostatní.
Duration hodnoty z Go JSON jsou nanosekundy a UI je převede na čitelnou délku.
Output bude ve `<details>` prvku, aby dlouhý výstup nerozbil skenování.

## Testovací plán

- `npm run format`, `npm run lint`, `npm run build`.
- Browser smoke test detailu s empty history a s runs/status history.
- Ověřit `/cards/missing` jako not-found/error state.
- Ověřit keyboard focus na back, action buttons a output disclosure.
- Ověřit 320px layout bez horizontálního overflow.

## Kritérium hotovosti

Detail používá skutečné read-only endpointy, zobrazuje transition history jako
časovou osu, neukazuje technické raw labels jako hlavní text a žádná sekce
neschová celý detail kvůli izolované chybě.

## Uzavření

- **Stav po implementaci**: Hotovo
- **Ověření**: `npm run format`, `npm run lint -- --quiet`, `npm run build` — úspěšné; browser detail smoke test a 320px responsive test — úspěšné
- **Dokumentace aktualizována**: ano; roadmapa a UX specifikace
