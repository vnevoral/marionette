# Implementační blok: Frontend testovací infrastruktura (Vitest)

- **Fáze**: 8 — Zpevnění
- **Vazba na požadavky**: NFR-05, NFR-06, FR-27
- **Vazba na ADR**: ADR-0003
- **Stav**: Hotovo
- **Závislosti**: `docs/devops/testing-strategy.md` (slib Vitest ve fázi 6), blok 0013

## Cíl bloku

Po dokončení má `web/` spustitelné jednotkové testy (`npm test`), CI je
vyžaduje a existuje první sada testů pro API vrstvu a čistou logiku view,
takže bloky 0029 a 0032 mohou psát regresní testy.

## Rozsah

- **Uvnitř**:
  - devDependencies `vitest`, `@vue/test-utils`, `happy-dom` (nebo
    `jsdom`), `@vitest/coverage-v8`;
  - `web/vitest.config.ts` sdílející alias `@` s Vite; `environment:
    happy-dom`; `include: src/**/*.spec.ts`;
  - scripty `test` (`vitest run`), `test:watch`, `test:coverage`;
  - `tsconfig.app.json` include `src/**/*.spec.ts` a `vitest/globals` typy;
  - první testy: `api.ts` (`request`: 204, 2xx JSON, 4xx `{error}`, 5xx bez
    JSON, síťová chyba, ztráta `Accept` hlavičky při spreadu), `StatusBadge`
    (render label + tón), `CardEditView` čisté funkce `validate`/`actionFrom`
    (vyexportované do `src/views/cardEditModel.ts`);
  - CI job `web` doplní `npm test`; Makefile `test` spouští i `npm test`
    (nebo nový target `ui-test` volaný z `test`);
  - `testing-strategy.md` aktualizuje sekci Frontend na skutečný stav.
- **Mimo rozsah**:
  - E2E/browser testy (Playwright) — zvážit samostatným blokem po fázi 8;
  - visual regression;
  - refaktor view kvůli testovatelnosti nad rámec extrakce čistých funkcí.

## Schválení

- **Schválil**: projektový vlastník
- **Datum schválení**: 2026-09-26
- **Poznámky k rozhodnutí**: Schváleno jako oprava chyb. Nález revize 2026-09-26 (H3). Testovací
  strategie slibuje Vitest od fáze 6; fáze 6 i 9 jsou hotové bez testů.

## Návrh řešení

- `web/vitest.config.ts` přes `mergeConfig(viteConfig, defineConfig({ test:
  { environment: "happy-dom", globals: true, coverage: { provider: "v8",
  include: ["src/**"] } } }))`.
- `web/src/api.ts`: `request` testovatelný přes `globalThis.fetch` mock
  (`vi.stubGlobal`).
- `web/src/views/cardEditModel.ts`: čisté funkce dnes uvnitř
  `CardEditView.vue` (`validate`, `actionFrom`, `fingerprint`).
- CI: krok `npm test -- --run` mezi lint a build; cache beze změny.

## Testovací plán

- Testy z rozsahu výše prochází lokálně a v CI.
- Ověřit, že `npm run build` nezahrne `*.spec.ts` do bundlu (Vite include
  jen z `main.ts`; tsconfig include spec souborů nesmí rozbít `vue-tsc -b`).
- `npm run lint` s `.spec.ts` soubory (globals `describe/it/expect`
  v ESLint config).

## Kritérium hotovosti

Viz [Definition of Done](../../devops/definition-of-done.md) +:

- `npm test` běží v CI a v `make test`;
- pokrytí `api.ts` ≥ 80 %;
- `testing-strategy.md` odpovídá skutečnosti.

## Uzavření

- **Stav po implementaci**: Hotovo (2026-09-26)
- **Ověření**: `make verify` prošel (`make test` nyní volá `ui-test` =
  `npm test` a poté `go test -race`). Vitest 5 + Vue Test Utils 2.5 +
  happy-dom 20 + coverage-v8; 24 testů ve třech souborech: `src/api.spec.ts`
  (2xx JSON, 204, 4xx obálka, 5xx bez JSON, chybějící `error`, síťová chyba,
  zachování `Accept` při vlastních hlavičkách, `Content-Type` u mutujících
  volání bez těla, odstranění `currentStatus` z payloadu, URL s enkódovaným
  ID, `connectStatusEvents` s falešným `EventSource` vč. neplatných
  událostí), `src/components/StatusBadge.spec.ts` (label, ikona, tón,
  výchozí tón), `src/views/cardEditModel.spec.ts` (`actionFrom`,
  `validate`, `fingerprint`, `copyAction`, `environmentRows`). Pokrytí
  `api.ts` ≥ 80 % (viz `npm run test:coverage`), `cardEditModel.ts` 100 %.
  `npm run build` spec soubory do bundlu nezahrnuje (Vite vychází z
  `main.ts`), `vue-tsc -b` je typově kontroluje; `npm run lint` a
  `format:check` procházejí. **Nález:** test odhalil skutečnou chybu v
  `request` — `...options` za sloučenými hlavičkami zahazoval `Accept` u
  každého volání s vlastním `Content-Type`; opraveno.
- **Odchylky od návrhu**: (1) testy importují `describe/it/expect/vi`
  explicitně z `vitest` místo `globals: true` + `vitest/globals` typů —
  žádné globální symboly v ESLint ani tsconfig; (2) `EnvironmentRow`,
  `emptyAction`, `emptyCard`, `copyAction` a nový `environmentRows` jsou
  vyexportované společně s `validate`/`actionFrom`/`fingerprint`;
  `validate` vrací `{ fieldErrors, firstError }` místo mutace reaktivního
  objektu; (3) instalace vyžadovala `npx npm@11` kvůli chybě `edgesOut`
  v npm 10.8 při řešení peer závislostí (lockfile v3 zůstává kompatibilní
  s `npm ci`); (4) přibyl `web/.prettierignore` a `web/coverage/` v
  `.gitignore`, aby výstup pokrytí nerozbil `format:check`.
- **Dokumentace aktualizována**: ano — `docs/devops/testing-strategy.md`
  (sekce Frontend odpovídá skutečnosti), AGENTS.md (`make test`), CI
  (`npm test` mezi `format:check` a `build`), Makefile (`ui-test`),
  roadmapa.
