# Implementační blok: PrimeFlex view migration

- **Fáze**: 9 — UX redesign a sdílený design systém
- **Vazba na požadavky**: FR-25, FR-29, NFR-08, NFR-09
- **Vazba na ADR**: ADR-0003, ADR-0007
- **Stav**: Hotovo
- **Závislosti**: Blok 0019, UX-02, UX-03, UX-04

## Cíl bloku

Overview, Manage cards a Card detail používají PrimeFlex pro obecný layout,
grid, flex, spacing a responsive breakpointy. Vlastní view CSS přestane
opakovat obecné layout rules.

## Rozsah

- **Uvnitř**:
  - page headers, action groups a panel spacing přes PrimeFlex;
  - Overview card grid přes responsive PrimeFlex columns;
  - Manage list/editor layout přes responsive columns;
  - Card detail summary/panels a action groups přes PrimeFlex;
  - zachování green-first tokenů a vizuálního vzhledu.
- **Mimo rozsah**:
  - změna API nebo komponentové hierarchie;
  - nový layout framework;
  - odstranění CSS, které řeší skutečně produktový vzhled komponent.

## Schválení

- **Schválil**: projektový vlastník
- **Datum schválení**: 2026-09-25
- **Poznámky k rozhodnutí**: PrimeFlex je standardní layout utility vrstva.

## Návrh řešení

Použít `grid`, `col-*`, `flex`, `align-items-*`, `justify-content-*`, `gap-*`,
`p-*`, `m-*` a responsive varianty. Lokální CSS ponechat pro barvy, border,
shadow, typography hierarchy, card rows a semantic feedback.

## Testovací plán

- `npm run format`, `npm run lint -- --quiet`, `npm run build`.
- Browser smoke test Overview, Manage a Detail na desktopu a 320 px.
- Ověřit absence horizontálního overflow a zachování keyboard focus.

## Kritérium hotovosti

Obecné layout CSS pro grid/flex/spacing není duplikované mezi view; responsive
chování je definované PrimeFlex classes a všechny tři obrazovky vizuálně drží
stejný shell.

## Uzavření

- **Stav po implementaci**: Hotovo
- **Ověření**: `npm run format`, `npm run lint -- --quiet`, `npm run build` — úspěšné; browser smoke test Overview/Manage/Detail desktop + 320px bez overflow
- **Dokumentace aktualizována**: ano; roadmapa
