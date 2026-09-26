# Implementační blok: Viditelné selhání čtení statusu (UX spec §4)

- **Fáze**: 8 — Zpevnění (dodatek po uzavření fáze)
- **Vazba na požadavky**: FR-21, FR-25, FR-26, NFR-08
- **Vazba na ADR**: ADR-0007
- **Stav**: Hotovo
- **Závislosti**: Bloky 0033, 0037, 0039

## Cíl bloku

Po dokončení dashboard i detail karty ukážou, že se status nepodařilo obnovit:
badge přejde na **Unknown** a u karty je text „Status could not be
refreshed“, poslední známý čas kontroly zůstává vidět. Tím kód odpovídá UX
spec §4 („A status read failure keeps the card visible and marks its data as
unknown“), kterou dnes nesplňuje: `useCardStatus.failed` se nastavuje, ale
nikde nezobrazuje, a dashboard stav selhání po bloku 0037 nesleduje vůbec.

## Rozsah

- **Uvnitř**:
  - `vocabulary.ts`: `FEEDBACK.statusUnavailable`;
  - `ActionCard.vue`: prop `statusUnavailable`; badge **Unknown** (pokud
    neběží požadavek), pod časem kontroly řádek s textem;
  - `StatusSummary.vue`: prop `unavailable`; poznámka nad seznamem;
  - `HomeView.vue`: `statusUnavailable` per karta — nastaví se při selhání
    `listCards` v polling režimu (všem kartám se status akcí) a při chybě
    REST čtení během čekání na akci; smaže se každým úspěšně přijatým
    snapshotem pro kartu a novým načtením dashboardu;
  - `CardDetailView.vue`: badge a `StatusSummary` čtou `cardStatus.failed`.
- **Mimo rozsah**: změna chování pollingu, Toast, globální banner.

## Schválení

- **Schválil**: projektový vlastník
- **Datum schválení**: 2026-09-26
- **Poznámky k rozhodnutí**: Narovnání kódu vůči UX spec §4 (dokumentace je
  zdroj pravdy, AGENTS.md); odchylka zaznamenaná v Uzavření bloku 0037.
  Pokyn „pokračuj“ po druhém code review.

## Návrh řešení

- `ActionCard.badge`: `pending` → REQUEST, `statusUnavailable` →
  `STATUS.unknown`, jinak stav. Řádek `.card-unavailable` bez live regionu
  (UX spec §9: bez zahlcení asistivních technologií).
- `HomeView.applySnapshot` nejdřív smaže příznak (čtení uspělo), pak
  aplikuje `supersedes`.

## Testovací plán

- `ActionCard.spec.ts`: `statusUnavailable` → badge Unknown a text; během
  běžícího požadavku má přednost Running.
- `HomeView.spec.ts`: selhání `listCards` v polling režimu označí kartu se
  status akcí (ne kartu bez ní), úspěšný další refresh příznak smaže.
- `CardDetailView.spec.ts`: selhání `getStatus` při polling ticku → Unknown
  a text; další úspěšné čtení → původní stav.
- `make verify`.

## Kritérium hotovosti

Viz [Definition of Done](../../devops/definition-of-done.md).

## Uzavření

- **Stav po implementaci**: Hotovo (2026-09-26)
- **Ověření**: `make verify` prošel; Vitest 93 → 96 testů. Nové:
  `ActionCard.spec.ts` (Unknown + text, poslední čas kontroly zůstává,
  Running má přednost, karta bez status akce text nezobrazí),
  `HomeView.spec.ts` (selhání `listCards` v polling režimu označí Printer,
  ne Lamp; další úspěšný refresh vrátí Healthy), `CardDetailView.spec.ts`
  (selhání `getStatus` při polling ticku → Unknown a poznámka v souhrnu,
  další úspěšné čtení → Healthy). Kontrast textu `--color-warning-strong`
  (#94652d) na `--color-surface` 4,82:1, na bílé 5,05:1 (WCAG 2.2 AA).
- **Odchylky od návrhu**: žádné.
- **Dokumentace aktualizována**: ano — UX spec §4 (podoba označení a kdy
  zmizí), roadmapa.
