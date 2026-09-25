---
description: "Use when writing or modifying the Vue 3 + PrimeVue frontend in web/src — components, views, router, API calls."
applyTo: "web/src/**"
---

# Vue + PrimeVue frontend

- `<script setup lang="ts">` s Composition API; žádné Options API komponenty.
- UI prvky přednostně z PrimeVue (`Card`, `Button`, `Tag`, `DataTable`,
  `Dialog`, formulářové vstupy) místo vlastního HTML/CSS od nuly — viz
  [ADR-0003](../../docs/architecture/decisions/0003-vue-primevue-frontend.md).
- Odsazení tabulátorem, dvojité uvozovky (viz `.prettierrc.json`); po úpravě
  spustit `npm run format` a `npm run lint` z `web/`.
- Nepřidávat novou major verzi PrimeVue/Vue/Vite bez nového ADR, který
  zhodnotí breaking changes.
- API volání směřují na `/api/...` (v dev proxováno Vite na backend `:8080`,
  v produkci stejný proces/port).
- Po každé změně: `npm run lint` a `npm run build` ve `web/`.
