# Implementační blok: UX-05 — Vizuální konzistence, accessibility a responsive audit

- **Fáze**: 9 — UX redesign a sdílený design systém
- **Vazba na požadavky**: FR-24..FR-29, NFR-08..NFR-10
- **Vazba na ADR**: ADR-0007
- **Stav**: Hotovo
- **Závislosti**: Bloky 0015–0020

## Cíl bloku

Dokončit fázi UX tak, aby Overview, Manage cards a Card detail používaly jeden
vizuální a interakční systém. Manage cards nesmí působit jako samostatný tmavý
nebo technický produkt; všechny obrazovky musí sdílet stejné tokeny, typografii,
povrchy, spacing, stavové barvy a responsivní chování.

## Rozsah

- sjednotit page header, content width, vertikální rytmus a hlavní akce;
- sjednotit povrchy panelů/karet, border, radius, shadow a form controls;
- odstranit lokální barvy, fonty a layout pravidla, která obcházejí tokeny;
- převést Manage cards na stejný green-first visual language jako Overview;
- sjednotit semantic status/request feedback a kontrast stavů;
- doplnit konzistentní keyboard focus, labels, live regions a error feedback;
- ověřit responsive chování na 320 px, mobilu, tabletu a desktopu;
- doplnit vizuální smoke/regression kontrolu hlavních workflow;
- aktualizovat UX specifikaci a roadmapu po uzavření bloku.

## Mimo rozsah

- nový API kontrakt nebo změna doménového modelu;
- autentizace a autorizace;
- změna PrimeVue major verze nebo přidání dalšího UI frameworku;
- změna produktové palety mimo sjednocení existujících tokenů.

## Schválení

- **Schválil**: projektový vlastník
- **Datum schválení**: 2026-09-25
- **Poznámky k rozhodnutí**: Fáze 9 pokračuje sjednocením celé aplikace; nejde pouze o redesign dashboardu.

## Testovací plán

- `npm run format`, `npm run lint -- --quiet`, `npm run build`;
- browser smoke test Overview, Manage a Detail na 320 px, 390 px, tabletu a desktopu;
- ověřit žádný horizontální overflow a konzistentní computed font/background/border tokeny;
- ověřit keyboard tab order, focus visibility a accessible names;
- ověřit empty, loading, error, save, delete a action feedback states.

## Kritérium hotovosti

Všechny tři hlavní obrazovky používají stejnou page/header/panel/form language,
Manage cards vizuálně patří do stejné aplikace jako Overview, žádný běžný stav
není komunikován pouze barvou a responsive/accessibility smoke testy procházejí.

## Evidence dokončení

- `npm run format`, `npm run lint -- --quiet` a `npm run build` ve `web/`
  prošly.
- Browser smoke test formuláře ověřil field-level validation, `aria-invalid`,
  unsaved-changes confirmation při navigaci a funkční tab order.
- Responsive kontrola na šířkách 320, 390, 768 a 1440 px neodhalila
  horizontální overflow ani kolizi obsahu.
- Ověřeny byly také loading, error, save feedback a běžné empty-state workflow
  hlavních obrazovek.
