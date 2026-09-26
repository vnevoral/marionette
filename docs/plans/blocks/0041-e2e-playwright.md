# Implementační blok: End-to-end testy (Playwright) proti skutečné binárce

- **Fáze**: 8 — Zpevnění (dodatek po uzavření fáze)
- **Vazba na požadavky**: FR-20..29, FR-40..42, NFR-08, NFR-10, NFR-12
- **Vazba na ADR**: ADR-0003, ADR-0007
- **Stav**: Hotovo
- **Závislosti**: Bloky 0026, 0030, 0039, 0040

## Cíl bloku

Po dokončení existuje automatická sada, která spustí skutečnou binárku
s embedded SPA a ovládá ji ze skutečného prohlížeče: vytvoření, běh, kontrola
a smazání karty, chyby validace, dialogy, cross-site ochrana a layout ve
320 px. Jednotkové testy tyto vrstvy (embed, SSE, hlavičky prohlížeče,
reálné CSS) nepokrývají.

## Rozsah

- **Uvnitř**:
  - `@playwright/test` (devDependency `web/`), `web/playwright.config.ts`,
    `web/e2e/start-server.sh` (binárka, prázdná konfigurace v dočasném
    adresáři, `127.0.0.1:18080`), `web/e2e/helpers.ts`, specy
    `card-lifecycle.e2e.ts` a `security-and-layout.e2e.ts`;
  - `make e2e`, `npm run test:e2e`, CI job `e2e` s artefaktem reportu,
    Chromium v `post-create.sh` devcontaineru;
  - ESLint, `vue-tsc -b` (`tsconfig.node.json`) a Prettier pokrývají
    `e2e/` a konfiguraci; výstupy v `.gitignore`;
  - dokumentace: testing strategy, CI/CD, DoD, README, roadmapa.
- **Mimo rozsah**: vizuální regresní snímky, další prohlížeče (Firefox,
  WebKit), testy na ARM64 hostu, zařazení do `make verify`.

## Schválení

- **Schválil**: projektový vlastník
- **Datum schválení**: 2026-09-26
- **Poznámky k rozhodnutí**: Položka „volitelný E2E blok“ z revize; pokyn
  „pojďme na další kroky“ s odložením ověření na Pi. Chromium only, mimo
  `make verify` kvůli závislosti na prohlížeči; DoD vyžaduje `make e2e` u
  bloků měnících UI tok nebo HTTP kontrakt.

## Návrh řešení

- Jeden worker, specy si stav připraví přes API (`createCard`,
  `deleteAllCards` jako non-browser klient bez `Origin`).
- Lokátory podle rolí a přístupných jmen; `actionEditor(page, title)` a
  `currentStatusBadge(page)` pro sekce bez vlastního jména.
- Cross-site: `request` s `Origin`/`Sec-Fetch-Site`/`text/plain` a stránka
  na `localhost` (jiný origin než `127.0.0.1`), která pošle `no-cors` POST;
  ověřuje se, že nevznikl žádný běh.
- 320 px: `scrollWidth - clientWidth ≤ 0` na `/`, detailu, editaci
  a novém formuláři.

## Testovací plán

- `make e2e`: 7 testů; `--repeat-each=3` bez flaky výsledků.
- `make verify` (lint a typy pokrývají `e2e/`).

## Kritérium hotovosti

Viz [Definition of Done](../../devops/definition-of-done.md) +:

- CI job `e2e` je definovaný a `make e2e` projde lokálně.

## Uzavření

- **Stav po implementaci**: Hotovo (2026-09-26)
- **Ověření**: `make e2e` — 7 passed (~13 s); `npx playwright test
--repeat-each=3` — 21 passed; `make verify` prošel. Pokryté scénáře UX
  spec §10: 1 (prázdný přehled → vytvoření), 2 částečně (320 px bez
  horizontálního scrollu; tablet/desktop a vzhled zůstávají na ruční
  kontrole), 3 (Queued/Running → výsledek, žádný falešný úspěch), 5
  (422 zachová hodnoty formuláře), 6 (neuložené změny), 7 (mazání
  s potvrzením). NFR-12 ověřeno s hlavičkami skutečného prohlížeče.
- **Nález během implementace**: `GET /api/cards/{id}/runs` vracel pro
  kartu bez běhů `null` místo `[]` (`Store.GetRuns` kopíroval přes `append`
  do nil slice). Frontend to maskoval `?? []`, jiný klient by spadl.
  Opraveno v `store.go`, přidán `TestRouterServesEmptyHistoriesAsArrays`
  (runs i status history → `[]`).
- **Odchylky od návrhu**: žádné.
- **Poznámka k prostředí**: běžící devcontainer má Node 20, zatímco
  `devcontainer.json`, `.nvmrc` a `engines` předepisují 22 — kontejner je
  sestavený před změnou a potřebuje „Rebuild Container“; Playwright
  i ostatní nástroje na Node 20 fungují.
- **Dokumentace aktualizována**: ano — `docs/devops/testing-strategy.md`
  (sekce E2E), `docs/devops/ci-cd.md` (job `e2e`), DoD (`make e2e` u UI
  a HTTP změn), README, roadmapa.
