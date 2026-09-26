# ADR-0003: Vue 3 + PrimeVue pro frontend

- **Stav**: Přijato
- **Datum**: 2026-09-25

## Kontext

Potřebujeme UI knihovnu pro dashboard s kartami, formuláři pro konfiguraci
akcí a stavovými indikátory (FR-20 až FR-23), bez nutnosti stavět vlastní
sadu komponent od nuly, a s rozumnou velikostí výsledného bundlu pro nasazení
na slabší hardware.

## Rozhodnutí

Frontend je Vue 3 (Composition API, `<script setup>`) s komponentovou
knihovnou **PrimeVue v4** (téma Aura z `@primevue/themes`), layout utility
vrstvou **PrimeFlex v4** a ikonami `primeicons`. Routing řeší `vue-router`.
Build nástroj je Vite.

PrimeVue major verze se drží na stabilní `v4` větvi (npm dist-tag
`v4-stable`) — vyšší major verze se nepřijímají automaticky, ale až po
ověření changelogu/migrace a aktualizaci tohoto ADR.

## Zvažované alternativy

- Vlastní minimalistické komponenty bez knihovny — zamítnuto, zbytečné
  náklady na vývoj a údržbu pro admin/dashboard UI.
- Vuetify / Element Plus — nezamítnuto kategoricky, ale PrimeVue nabízí širokou
  sadu komponent (karty, datové tabulky, formulářové prvky, badge/tag pro
  stavy) a dobrou podporu Vue 3 + TS; zvoleno jako výchozí volba projektu.
- Tailwind + headless UI — zamítnuto pro MVP kvůli vyšším nákladům na sestavení
  vlastního design systému; lze zvážit později jako doplněk k PrimeVue
  (unstyled mode), pokud vznikne potřeba.

## Důsledky

- Veškeré nové UI komponenty přednostně skládat z PrimeVue prvků
  (`Card`, `Button`, `Tag`/`Badge`, `DataTable`, `Dialog`, formulářové vstupy)
  místo psaní vlastního HTML/CSS od nuly.
- Obecný layout, grid, flex, spacing a responsive breakpointy řešit přes
  PrimeFlex; vlastní CSS používat pro produktové kompozice a design tokeny.
- Upgrade na PrimeVue v5 (nebo jinou budoucí major verzi) vyžaduje nové ADR,
  které zhodnotí breaking changes a licenční/balíčkové změny (v5 mění
  strukturu závislostí na `@primeuix/*` balíčky).

> Doplněno 2026-09-26: rozhodnutí o ponechání PrimeFlex jako zamrzlé layout
> vrstvy pro MVP je v [ADR-0009](0009-primeflex-frozen-layout-layer.md).

> Doplněno 2026-09-26 (blok 0035): PrimeVue 5.0.x přešlo z MIT na komerční
> PrimeUI License s licenčním klíčem; projekt zůstává na větvi v4 (MIT,
> dist-tag `v4-stable`). PrimeFlex byl nahrazen vlastní utility vrstvou
> ([ADR-0010](0010-own-layout-utilities-replace-primeflex.md)); věta
> o PrimeFlexu v Rozhodnutí a Důsledcích výše je tím překonaná.
