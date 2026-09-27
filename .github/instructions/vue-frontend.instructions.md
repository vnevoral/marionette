---
description: "Use when writing or modifying the Vue 3 + PrimeVue frontend in web/src — components, views, router, API calls."
applyTo: "web/src/**"
---

# Vue + PrimeVue frontend

- `<script setup lang="ts">` with the Composition API; no Options API
  components.
- Prefer UI elements from PrimeVue (`Card`, `Button`, `Tag`, `DataTable`,
  `Dialog`, form inputs) over custom HTML/CSS from scratch — see
  [ADR-0003](../../docs/architecture/decisions/0003-vue-primevue-frontend.md).
- Tab indentation, double quotes (see `.prettierrc.json`); after a change
  run `npm run format` and `npm run lint` from `web/`.
- Do not add a new major version of PrimeVue/Vue/Vite without a new ADR
  that evaluates the breaking changes.
- API calls go to `/api/...` (in dev proxied by Vite to the backend
  `:8080`, in production the same process/port).
- After every change: `npm run lint` and `npm run build` in `web/` (lint
  runs with `--max-warnings 0`, formatting is enforced by Prettier via
  `eslint-config-prettier`); before closing, `make verify` from the
  repository root.
