# Implementační blok: Výstup poslední status kontroly v detailu karty

- **Fáze**: 8 — Zpevnění (dodatek po uzavření fáze)
- **Vazba na požadavky**: FR-21a, FR-13, FR-17, FR-24
- **Vazba na ADR**: ADR-0007 (UX a design systém); nové ADR není potřeba
- **Stav**: Návrh
- **Závislosti**: Bloky 0018 (detail karty), 0033 (sdílené komponenty),
  0040 (viditelné selhání čtení statusu)

## Cíl bloku

Při validaci na Raspberry Pi (2026-09-26) status akce `ping` selhávala
s `socket: Operation not permitted` (viz blok 0046), ale UI ukázalo jen
výsledek **Fail**. Výstup kontroly backend ukládá (`StatusSnapshot.lastCheck`
s `exitCode`, `duration`, `output`, `truncated`) a vrací ho
v `GET /api/cards/{id}/status` i v událostech SSE, UI ho ale nezobrazuje;
operátor musel číst JSON z API. Po dokončení bloku detail karty ukáže exit
kód, dobu běhu a rozbalitelný výstup poslední kontroly.

## Rozsah

- **Uvnitř**:
  - `web/src/components/StatusSummary.vue`: pod „Last outcome“ řádky
    „Exit code“ a „Duration“ a rozbalitelný výstup (`<details>` „View
    output“, `…` při `truncated`) ve stejné podobě jako u běhů
    v `RunTable.vue`; bez výstupu se `<details>` nevykreslí;
  - společná prezentace výstupu pro `RunTable` a `StatusSummary`
    (malá komponenta `RunOutput.vue`), aby se styl `pre` neduplikoval;
  - UX specifikace §6: Summary obsahuje i exit kód, dobu a výstup poslední
    kontroly;
  - testy komponent (Vitest): výstup se zobrazí jen na vyžádání, zkrácení
    je označené, chybějící `lastCheck` nebo prázdný výstup nic navíc
    nevykreslí, stav `unavailable` (blok 0040) se nemění;
  - E2E (Playwright): scénář s nepovedenou status kontrolou ověří, že
    v detailu jde výstup rozbalit.
- **Mimo rozsah**:
  - výstup nebo exit kód na kartě dashboardu (FR-21a: dashboard zůstává
    skenovatelný; operátor přejde do detailu přes **View details**);
  - historie výstupů starších status kontrol (status historie podle
    ADR-0006 drží jen přechody stavů, ne jednotlivé běhy);
  - změny backendu, API kontraktu nebo perzistence (data už existují).

## Schválení

- **Schválil**: čeká
- **Datum schválení**: čeká
- **Poznámky k rozhodnutí**: —

## Návrh řešení

Viz rozsah. `StatusSummary` už dostává celý `StatusSnapshot`, takže stačí
číst `status.lastCheck` (`exitCode`, `duration`, `output`, `truncated`).
Popisky jdou přes `src/ui/vocabulary.ts` (FR-26); `<details>/<summary>` je
dostupné klávesnicí (FR-28), `pre` se zalamuje a na 320 px nezpůsobí
horizontální scroll stránky (FR-29).

## Testovací plán

- Jednotkové testy `StatusSummary` a `RunTable` (po vyčlenění
  `RunOutput`).
- `make e2e` — nový nebo rozšířený scénář detailu karty.
- `make verify`.

## Kritérium hotovosti

Viz [Definition of Done](../../devops/definition-of-done.md) +:

- operátor v detailu karty s neúspěšnou status kontrolou uvidí výstup
  příkazu (např. `socket: Operation not permitted`) bez přístupu k API.

## Uzavření

- **Stav po implementaci**: —
- **Ověření**: —
- **Dokumentace aktualizována**: —
