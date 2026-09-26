# Implementační blok: Sdílený slovník stavů, komponenty a theme preset

- **Fáze**: 8 — Zpevnění
- **Vazba na požadavky**: FR-25, FR-26, NFR-06, NFR-08
- **Vazba na ADR**: ADR-0003, ADR-0007, ADR-0009
- **Stav**: Schváleno
- **Závislosti**: Bloky 0029, 0032 (aby refaktor nekolidoval s opravami), 0030 (testy)

## Cíl bloku

Po dokončení existuje jeden modul se slovníkem stavů a textů, sdílené typy
a komponenty ze spec §8.4, a PrimeVue theme je přizpůsobené přes
`definePreset` místo `!important` přepisů.

## Rozsah

- **Uvnitř**:
  - `src/ui/vocabulary.ts`: labely, `statusPresentation(state)` →
    `{label, icon, tone}`, texty prázdných/chybových stavů; `StatusTone`
    definován jednou v `src/types.ts` spolu s `EnvironmentRow` a dalšími
    sdílenými typy;
  - komponenty `PageHeader`, `EmptyState`, `RequestState`, `ActionCard`,
    `RunTable`, `StatusTimeline` (spec §8.4); view se zmenší pod ~250 řádků;
  - globální třídy `.eyebrow`, `.page`, `.page-lede`, `.loading-state` v
    `tokens.css`, odstranění duplicit ze scoped CSS;
  - `definePreset(Aura, …)` v `main.ts` pro primary/formField/surface
    tokeny, odstranění všech `!important` v `tokens.css` a `:deep` přepisů
    v `CardEditView.vue`; tokeny srovnat se spec §8.2 (nebo spec upravit
    podle skutečnosti, rozhodne vlastník);
  - odlišit tón `Queued` (amber) a `Running` (blue) dle spec §4; doplnit
    „Last checked …/Not checked yet“ a „state duration“ (FR-17) tam, kde
    chybí; line-clamp popisu na kartě;
  - `env.d.ts` shim odstranit, pokud `vue-tsc` funguje bez něj.
- **Mimo rozsah**:
  - migrace PrimeFlex → Tailwind (zamítnuto pro MVP, viz ADR-0009);
  - Pinia nebo jiný store (velikost aplikace to nevyžaduje);
  - nové funkce.

## Schválení

- **Schválil**: projektový vlastník
- **Datum schválení**: 2026-09-26
- **Poznámky k rozhodnutí**: Schváleno; PrimeFlex zůstává dle ADR-0009 a
  jeho verze se v tomto bloku připne na `4.0.0`. Nález revize 2026-09-26
  (M4, M6, L4, L6).
  ADR-0007 slovník explicitně vyžaduje; dnes je `StatusTone` definován 3×.

## Návrh řešení

- `src/ui/vocabulary.ts` exportuje `const STATUS = { ok: {label:"Healthy",
  icon:"pi pi-check-circle", tone:"success"}, … } satisfies Record<StatusState,
  Presentation>`; `StatusBadge` i `ActionCard` z něj čtou.
- `src/theme/preset.ts`: `definePreset(Aura, { semantic: { primary: {…},
  colorScheme: { light: { surface: {…}, formField: {…} } } } })`.
- Refaktor po jedné view: `HomeView` → `ActionCard` + `EmptyState`;
  `CardDetailView` → `RunTable` + `StatusTimeline`; `CardEditView` →
  `PageHeader` + `RequestState`.

## Testovací plán

- Vitest: `statusPresentation` pokrývá všechny `StatusState` (typový test
  `satisfies`); `ActionCard` render pro každý stav; `StatusTimeline`
  výpočet doby trvání.
- `grep -r "!important" web/src` vrací 0 řádků; `grep -rn "type StatusTone"`
  vrací 1 řádek.
- Manuální screenshot smoke test všech tří obrazovek v 320 px a desktop
  (NFR-10), kontrola kontrastu tokenů (axe).
- `npm run lint`, `npm run build`, `npm test`.

## Kritérium hotovosti

Viz [Definition of Done](../../devops/definition-of-done.md) +:

- žádný `!important` v `web/src`;
- každá view < 300 řádků;
- UX spec §8.2/§8.4 odpovídá implementaci.

## Uzavření

- **Stav po implementaci**: čeká
- **Ověření**: čeká
- **Dokumentace aktualizována**: čeká
