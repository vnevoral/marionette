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
knihovnou **PrimeVue v4** (téma Aura z `@primevue/themes`) a ikonami
`primeicons`. Routing řeší `vue-router`. Build nástroj je Vite.

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
- Upgrade na PrimeVue v5 (nebo jinou budoucí major verzi) vyžaduje nové ADR,
  které zhodnotí breaking changes a licenční/balíčkové změny (v5 mění
  strukturu závislostí na `@primeuix/*` balíčky).
