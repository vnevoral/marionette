---
description: "Use when writing or modifying the Vue 3 frontend in web/src — components, views, router, API calls."
applyTo: "web/src/**"
---

# Vue frontend

- `<script setup lang="ts">` with the Composition API; no Options API
  components.
- Build screens from the components in `web/src/ui/` (`MButton`, `MIcon`,
  `MTag`, `MMessage`, `MCard`, form controls, `useToast`/`useConfirm`) and
  the patterns in `web/src/components/`; a new base component goes into
  `web/src/ui/`, on a Reka UI primitive when it needs focus management or
  ARIA, styled with the tokens from `tokens.css`. Icons are Lucide names
  registered in `ui/icons.ts` — see
  [ADR-0014](../../docs/architecture/decisions/0014-replace-primevue-with-reka-ui.md).
- Tab indentation, double quotes (see `web/.prettierrc.json`); after a change
  run `npm run format` and `npm run lint` from `web/`.
- Do not add a new major version of Vue/Vite/Reka UI/Lucide without a new ADR
  that evaluates the breaking changes.
- API calls go to `/api/...` (in dev proxied by Vite to the backend
  `:8080`, in production the same process/port).
- After every change: `npm run lint` and `npm run build` in `web/` (lint
  runs with `--max-warnings 0`, formatting is enforced by Prettier via
  `eslint-config-prettier`); before closing, `make verify` from the
  repository root.
