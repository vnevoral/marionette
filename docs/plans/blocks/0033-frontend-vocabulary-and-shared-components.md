# Implementační blok: Sdílený slovník stavů, komponenty a theme preset

- **Fáze**: 8 — Zpevnění
- **Vazba na požadavky**: FR-25, FR-26, NFR-06, NFR-08
- **Vazba na ADR**: ADR-0003, ADR-0007, ADR-0009
- **Stav**: Hotovo
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

- **Stav po implementaci**: Hotovo (2026-09-26)
- **Ověření**: `make verify` prošel (lint, `vue-tsc`, Prettier, Vitest 77
  testů ve 14 souborech, `vite build`, `go test -race`, `go vet`). Nové testy:
  `ui/vocabulary.spec.ts` (každý `StatusState` a `Run.outcome` má label,
  ikonu i tón; raw hodnoty `ok/fail/accepted/running/timeout` se nikdy
  nezobrazí; Queued = amber, Running = blue), `ui/format.spec.ts`
  (nanosekundy, hrubé rozpětí, „Last checked/Not checked yet“,
  `transitionDuration` pro ukončený i aktuální přechod), `components/ActionCard.spec.ts`
  (render pro ok/fail/unknown, bez status akce, queued/running s blokovanými
  tlačítky, výsledek, emity), `components/StatusTimeline.spec.ts`
  (doba trvání aktuálního přechodu z `now`, uložená doba u ukončených,
  loading/empty/error). Kritéria: `grep -rn "!important" web/src` = 0,
  `grep -rn "type StatusTone" web/src` = 1, view: `HomeView` 243,
  `CardEditView` 293, `CardDetailView` 294 řádků. `env.d.ts` odstraněn —
  `vue-tsc -b` i `vite build` fungují bez shim. PrimeFlex připnut na
  `4.0.0` v `package.json` i lockfile. Manuální screenshot smoke test
  (320 px / desktop) a kontrola kontrastu axe zůstávají na referenční host.
- **Odchylky od návrhu**: (1) Tóny `StatusTone` zůstávají
  `healthy|problem|unknown|info|warning` (spec §4 „Queued amber / Running
  blue“ mapováno na `warning`/`info`), ne `success` jako v návrhu — názvy
  odpovídají existujícím CSS třídám `status-*` a testům z 0030. (2) Kromě
  komponent ze spec §8.4 (`PageHeader`, `EmptyState`, `RequestState`,
  `InlineError`, `ActionCard`, `ActionControls`, `RunTable`,
  `StatusTimeline`, `FormSection`, `ConnectionStatus` z 0032) přibyly
  `DetailPanel`, `DetailErrorState`, `StatusSummary`, `CardIdentityFields`,
  `PollingFields` a `SaveBar`, aby se všechny view vešly pod 300 řádků;
  `UnsavedChangesDialog` je realizován jako composable
  `useUnsavedChangesGuard` nad sdíleným `ConfirmDialog` (žádná druhá
  dialogová komponenta). Logika detailu je v composables
  `useCardActivity` (runs + history s generací) a `useTransientMessage`.
  (3) Slovník je rozdělen na `src/ui/vocabulary.ts` (labely, prezentace,
  texty) a `src/ui/format.ts` (formátování času a dob trvání); sdílené typy
  v `src/types.ts`, `cardEditModel.ts` je re-exportuje. (4) Tokeny v UX
  spec §8.2 byly upraveny podle skutečnosti (paleta doladěná v blocích
  0019/0020 zůstává), spec nově uvádí, že PrimeVue barvy jdou
  z `src/theme/preset.ts` přes `definePreset(Aura, …)`; `darkModeSelector`
  je vypnutý, aplikace má jedno světlé schéma. (5) FR-17 „state duration“:
  detail ukazuje „In this state for …“ z aktuálního přechodu historie,
  `StatusTimeline` značí aktuální přechod „(current)“ a obnovuje uplynulý čas
  každou minutu. (6) `.primary-action-button` a `.primary-action-link`
  zůstávají globální třídy (bez `!important`); měkký zelený „primary“ styl
  není v Aura tokenech vyjádřitelný bez varianty tlačítka.
- **Dokumentace aktualizována**: ano — UX spec §8.2 (tokeny podle
  skutečnosti, preset) a §8.4 (stav komponent), `docs/architecture/overview.md`
  (řádek SPA), `docs/devops/testing-strategy.md` (nové testy), roadmapa
  (fáze 8 hotová).
