# ADR-0007: Jednotný UI design systém a UX model

- **Stav**: Přijato
- **Datum**: 2026-09-25

## Kontext

První MVP obrazovky používají lokální CSS, odlišné layouty a nekonzistentní
názvosloví. Dashboard a správa karet proto nepůsobí jako jedna aplikace a
uživatel obtížně rozlišuje konfiguraci, požadavek na akci a poslední známý
stav. Požadavky FR-24 až FR-29 a NFR-08 až NFR-10 vyžadují společný základ
před dalším rozšiřováním UI.

## Rozhodnutí

Marionette dostane sdílený UI shell a malý projektový design systém nad Vue 3
a PrimeVue 4. PrimeVue zůstává zdrojem základních komponent; projektová vrstva
definuje CSS tokeny a vzory pro layout, typografii, semantic status colors,
akční stavy, formuláře, tabulky/historie, chyby a potvrzení destruktivních akcí.

Každý hlavní tok používá stejný model zpětné vazby: `idle`, `loading`,
`accepted`, `running`, `success`, `error` a `stale/unknown`, přičemž poslední
známý status zařízení je vizuálně oddělen od stavu HTTP požadavku. Texty,
labels a stavové názvy se centralizují v jednom slovníku. Výchozím jazykem UI
je angličtina (`en-US`); lokalizace do češtiny není součástí této fáze.

## Zvažované alternativy

- Vlastní sada komponent mimo PrimeVue — zamítnuto; zvyšuje údržbu a obchází
  přijaté ADR-0003.
- Pouhé lokální úpravy jednotlivých view — zamítnuto; neřeší konzistenci a
  vede k dalším odchylkám.
- Přechod na jinou UI knihovnu nebo Tailwind — odloženo; není potřeba měnit
  technologickou základnu, dokud PrimeVue 4 pokryje komponentové potřeby.

## Důsledky

- Layout, tokeny a stavové vzory budou sdílené mezi dashboardem, správou a
  budoucím detailem karty.
- Vizuální změny se budou ověřovat na desktopu i mobilní šířce a přes hlavní
  workflow, což zvýší čas implementace, ale sníží regresní UX.
- Nevzniká nová runtime závislost ani změna PrimeVue major verze.
- Výchozí jazyk UI je angličtina; případná další lokalizace je budoucí rozšíření.
