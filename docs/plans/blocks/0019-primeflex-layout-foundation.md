# Implementační blok: PrimeFlex layout foundation

- **Fáze**: 9 — UX redesign a sdílený design systém
- **Vazba na požadavky**: FR-25, FR-29, NFR-08, NFR-09
- **Vazba na ADR**: ADR-0003, ADR-0007
- **Stav**: Hotovo
- **Závislosti**: Bloky 0015–0018, PrimeVue 4

## Cíl bloku

Projekt používá PrimeFlex jako standardní utility vrstvu pro layout, grid,
flex, spacing a responsive breakpointy. Vlastní CSS zůstává pouze pro produktové
kompozice a tokeny, ne pro opakované obecné margin/padding pravidla.

## Rozsah

- **Uvnitř**:
  - instalace `primeflex@4`;
  - globální načtení `primeflex/primeflex.css`;
  - dokumentace hranice mezi PrimeFlex layout utilities a vlastními tokeny;
  - zachování PrimeVue theme a green-first semantic tokenů.
- **Mimo rozsah**:
  - kompletní přepis všech existujících view na utility classes;
  - změna PrimeVue major verze;
  - změna barevné palety nebo API.

## Schválení

- **Schválil**: projektový vlastník
- **Datum schválení**: 2026-09-25
- **Poznámky k rozhodnutí**: PrimeFlex je závazná layout utility vrstva pro další UX bloky.

## Návrh řešení

PrimeFlex se načte globálně v `main.ts`. Nové a refaktorované view používají
utility classes pro grid/flex/spacing/responsive layout. CSS tokeny zůstávají
pro semantic colors, typography, radius, focus a produktové komponenty.

## Testovací plán

- `npm install` dokončí bez dependency konfliktu.
- `npm run format`, `npm run lint -- --quiet`, `npm run build`.
- Browser smoke test ověří, že PrimeFlex CSS je dostupné a aplikace se načte.

## Kritérium hotovosti

PrimeFlex je dostupný v bundlu, globálně načtený a dokumentovaný; žádná nová
layoutová změna nepřidává další utility framework.

## Uzavření

- **Stav po implementaci**: Hotovo
- **Ověření**: `npm install primeflex`, `npm run format`, `npm run lint -- --quiet`, `npm run build` — úspěšné
- **Dokumentace aktualizována**: ano; roadmapa a ADR-0003
