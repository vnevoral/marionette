# Implementační blok: UX-02 — Overview a action feedback

- **Fáze**: 9 — UX redesign a sdílený design systém
- **Vazba na požadavky**: FR-20, FR-21, FR-24, FR-25, FR-26, FR-27, FR-28, FR-29, NFR-08..10
- **Vazba na ADR**: ADR-0003, ADR-0007
- **Stav**: Hotovo
- **Závislosti**: Blok 0015, [UX specifikace](../../ux/ui-ux-specification.md)

## Cíl bloku

Overview bude používat sdílené tokeny a jasnou hierarchii karty. Uživatel
rozliší poslední známý stav zařízení od aktuálního stavu požadavku a uvidí
lokální feedback při enqueue, polling refreshi, chybě i prázdném seznamu.

## Rozsah

- **Uvnitř**:
  - redesign `/` podle specifikace Overview;
  - stabilní page header s počtem karet, refresh a New card;
  - semantic status label + ikona + barva bez závislosti pouze na barvě;
  - request states `Queued`, `Running`, `Updated`, `Error`;
  - izolované status read errors na konkrétní kartě;
  - empty, loading a page-level error states;
  - responsive grid a full-width mobile controls;
  - odstranění lokálních barev a fontových override mimo tokeny.
- **Mimo rozsah**:
  - detail karty a runs/status timeline;
  - nový backend nebo job ID API;
  - redesign management formuláře;
  - lokalizace mimo angličtinu.

## Schválení

- **Schválil**: projektový vlastník
- **Datum schválení**: 2026-09-25
- **Poznámky k rozhodnutí**: Overview je první produktová obrazovka převedená na design systém.

## Návrh řešení

Upravit `HomeView.vue` na semantic tokeny a sdílené komponentové vzory.
Request feedback zůstane lokální klientská projekce, zatímco device state bude
vždy vycházet z `StatusSnapshot`. Status read failures se uloží podle ID karty
bez zahození načteného seznamu. AppShell bude poskytovat jediný landmark
`main`; view nebude vytvářet vnořený shell.

## Testovací plán

- `npm run format`, `npm run lint`, `npm run build`.
- Browser smoke test na 320 px a desktopu bez horizontálního overflow.
- Manuálně ověřit empty, loading, status read error a accepted/running feedback.
- Ověřit klávesový focus na refresh, New card a akčních tlačítkách.

## Kritérium hotovosti

Overview používá pouze sdílené tokeny, žádný stav není pouze barva, status
chyba jedné karty neskrývá ostatní karty a hlavní akce mají anglické,
jednoznačné labels.

## Uzavření

- **Stav po implementaci**: Hotovo
- **Ověření**: `npm run format`, `npm run lint`, `npm run build` — úspěšné; 320px browser smoke test bez overflow a s jediným `main` landmarkem
- **Dokumentace aktualizována**: ano; roadmapa a UX specifikace
