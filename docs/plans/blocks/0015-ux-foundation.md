# Implementační blok: UX-01 — Design tokeny a AppShell

- **Fáze**: 9 — UX redesign a sdílený design systém
- **Vazba na požadavky**: FR-24, FR-25, FR-26, FR-28, FR-29, NFR-08, NFR-09
- **Vazba na ADR**: ADR-0003, ADR-0007
- **Stav**: Hotovo
- **Závislosti**: Fáze 6, ADR-0007, [UX specifikace](../../ux/ui-ux-specification.md)

## Cíl bloku

Aplikace dostane jednotný vizuální základ: semantic design tokeny, PrimeVue
theme mapping, typografii, globální focus/spacing pravidla a sdílený AppShell
s navigací Overview / Manage cards. Stávající obsah obrazovek zůstane funkčně
beze změny; jejich postupné převedení je v UX-02 a UX-03.

## Rozsah

- **Uvnitř**:
  - centrální CSS tokeny pro canvas, surface, text, border, accent a stavy;
  - společné spacing, radius, focus a body typography pravidla;
  - PrimeVue Aura semantic overrides bez změny major verze;
  - `AppShell` s názvem aplikace, aktivní navigací a responzivním layoutem;
  - navigace na `/` a `/manage` dostupná klávesnicí;
  - globální anglické UI labels pro shell.
- **Mimo rozsah**:
  - redesign obsahu dashboardových karet;
  - redesign management formuláře;
  - detail karty, běhy, status timeline a nové API;
  - změna PrimeVue, routeru nebo přidání runtime závislosti.

## Schválení

- **Schválil**: projektový vlastník
- **Datum schválení**: 2026-09-25
- **Poznámky k rozhodnutí**: První implementační řez UX fáze; angličtina je jediný jazyk UI a palette je quiet botanical green-first.

## Návrh řešení

Přidat `web/src/styles/tokens.css` a načíst jej v `main.ts`. AppShell bude
sdílená komponenta obalující `RouterView`; aktivní route se zvýrazní přes
`RouterLink`. PrimeVue konfigurace dostane semantic color/content surface
mapping odpovídající tokenům. Globální CSS nesmí používat lokální hex hodnoty
pro semantic stavové barvy.

## Testovací plán

- `npm run format`, `npm run lint`, `npm run build`.
- Manuálně ověřit navigaci klávesnicí, aktivní route a viewport 320 px / desktop.
- Ověřit, že původní `/` a `/manage` se načtou uvnitř shellu bez změny API chování.

## Kritérium hotovosti

Viz [Definition of Done](../../devops/definition-of-done.md). Specificky:
obě existující obrazovky používají stejný shell, žádná navigace nezmizí na
mobilu a tokeny jsou definované na jednom místě.

## Uzavření

- **Stav po implementaci**: Hotovo
- **Ověření**: `npm run format`, `npm run lint`, `npm run build` — úspěšné; 320px browser smoke test bez horizontálního overflow
- **Dokumentace aktualizována**: ano; roadmapa a UX specifikace
