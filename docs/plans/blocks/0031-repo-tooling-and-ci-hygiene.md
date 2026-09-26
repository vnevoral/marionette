# Implementační blok: Hygiena repozitáře, lint a CI

- **Fáze**: 8 — Zpevnění
- **Vazba na požadavky**: NFR-05, NFR-06
- **Vazba na ADR**: ADR-0002, ADR-0003
- **Stav**: Schváleno
- **Závislosti**: žádné (lze implementovat jako první)

## Cíl bloku

Po dokončení jsou validační příkazy jednotné napříč Makefile, CI, DoD a
prompty, lint skutečně selhává při nálezech, generované soubory nejsou
verzované a `go test ./...` netestuje `node_modules`.

## Rozsah

- **Uvnitř**:
  - `web/go.mod` s `module marionette-web-ignore` (bez závislostí), aby
    `./...` nesahalo do `web/node_modules`; ověřit `go list ./...`;
  - `tsconfig.node.json`: `noEmit: true`, `tsBuildInfoFile` do
    `node_modules/.tmp`; `git rm --cached web/vite.config.js
    web/vite.config.d.ts web/tsconfig.node.tsbuildinfo`; `.gitignore`
    doplnit `web/*.tsbuildinfo`, `web/vite.config.js`, `web/vite.config.d.ts`;
  - ESLint: `eslint-config-prettier` jako poslední položka, odstranit
    `vue/html-indent`, `lint` script `eslint . --max-warnings 0`, nový
    `format:check` (`prettier --check .`) a `lint:types`
    (`vue-tsc --noEmit -p tsconfig.app.json`); opravit `tone` prop v
    `StatusBadge.vue` (required + default);
  - `.golangci.yml` s výchozí sadou + `gofumpt`, `errcheck`, `unused`,
    `staticcheck`, `govet`, `revive` (exported doc); opravit tři dnešní
    nálezy (nepoužitý `actionQueueDependencies`, nekontrolované
    `scheduler.Stop()` v testech, gofumpt formát `actions.go`);
  - Makefile `test` = `go test -race ./...` + `npm test` (po 0030),
    `lint` = `golangci-lint run ./...` + `npm run lint` + `format:check`;
    `verify` = `build + lint + test`;
  - CI: `make lint` v obou jobech, `go test -race`, `npm audit --omit=dev
    --audit-level=high`; přidat `.github/dependabot.yml` (gomod, npm,
    github-actions, týdně);
  - sjednocení Node verze: `web/.nvmrc` = 22, `engines.node >=22`,
    devcontainer feature `node: 22`;
  - `.editorconfig` (tab, LF, trailing whitespace) — devcontainer už
    doporučuje extension;
  - `CLAUDE.md` v kořeni s jediným řádkem odkazujícím na `AGENTS.md` (Claude
    Code mimo VS Code integraci AGENTS.md nenačítá automaticky);
  - AGENTS.md a `go-backend.instructions.md`: `gofumpt` místo `gofmt`
    (devcontainer ho už používá), odkaz na `make verify` jako jediný
    validační příkaz; DoD, workflow a prompt `implement-block` odkazují na
    `make verify` místo vlastních seznamů příkazů;
  - `marionette.json` v kořeni přesunout do `deploy/dev-fixture.json`
    (bez sekcí `status`/`history`), Makefile/air/launch.json ho kopírují do
    ignorovaného `./marionette.json` při prvním spuštění;
  - `LICENSE` (MIT, rozhodnuto 2026-09-26) a `CODEOWNERS`.
- **Mimo rozsah**:
  - release workflow do GitHub Releases (blok 0023 / samostatný blok);
  - změny aplikačního kódu nad rámec oprav lint nálezů.

## Schválení

- **Schválil**: projektový vlastník
- **Datum schválení**: 2026-09-26
- **Poznámky k rozhodnutí**: Schváleno jako narovnání stavu repozitáře;
  licence MIT. Nález revize 2026-09-26 (L-1, H4, M5, sekce „Struktura a
  nastavení agentů“). Doporučeno implementovat jako první blok fáze 8.

## Návrh řešení

Změny jsou konfigurační; jediné zásahy do kódu jsou tři lint opravy a
`tone?` v `StatusBadge`. `make verify` se stane jediným místem definice
validace, ostatní dokumenty na něj odkazují (workflow zakazuje paralelní
metodiku, proto se seznamy příkazů centralizují).

## Testovací plán

- `go list ./...` nevrací žádný balíček pod `web/`;
- `git status` po `npm run build` a `make backend-dev` je čistý;
- `npm run lint` selže na jednom uměle přidaném warningu; po `npm run
  format` je počet warningů 0;
- `golangci-lint run ./...` prochází bez nálezů;
- CI běh na PR je zelený; Dependabot otevře první PR.

## Kritérium hotovosti

Viz [Definition of Done](../../devops/definition-of-done.md) +:

- `make verify` je zdokumentován v README, AGENTS.md, DoD a promptech;
- žádný generovaný soubor není v `git ls-files`;
- Node verze je shodná v devcontaineru, CI a `.nvmrc`.

## Uzavření

- **Stav po implementaci**: čeká
- **Ověření**: čeká
- **Dokumentace aktualizována**: čeká
