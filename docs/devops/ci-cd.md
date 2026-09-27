# CI/CD

## CI (GitHub Actions, `.github/workflows/ci.yml`)

Na každý push a pull request:

1. **backend** — `golangci-lint` (verze připnutá shodně s devcontainerem,
   konfigurace `.golangci.yml`), `go build ./...`, `go vet ./...`,
   `go test -race -count=1 ./...` na linux/amd64, `bash -n` a staged test
   instalačního skriptu (`deploy/install_test.sh`) a test pravidla verze
   releasu (`deploy/release_version_test.sh`).
2. **web** — `npm ci`, `npm audit --omit=dev --audit-level=high`,
   `npm run lint` (`--max-warnings 0`), `npm run format:check`,
   `npm run build` (zahrnuje type-check přes `vue-tsc`) ve `web/`. Verze Node
   je určena souborem `web/.nvmrc` (shodná s devcontainerem a `engines`).
3. **e2e** — `npm ci`, `npx playwright install --with-deps chromium`,
   `make e2e` (build binárky a Playwright testy proti ní, blok 0041); při
   selhání se nahraje report a trace jako artefakt `playwright-report`.

Stejnou sadu (kromě jobu `e2e`, lokálně `make e2e`) spouští lokálně
`make verify`; CI ji pouze zrcadlí. CI běh
musí projít čistě (lint warningy selhávají build, nepotlačují se) před
mergem do hlavní větve. Dependabot (`.github/dependabot.yml`) otevírá týdenní
PR pro Go moduly, npm a GitHub Actions; major verze klíčových UI závislostí
jsou vyloučené (ADR-0003, ADR-0010).

## Release proces (ověřeno na Raspberry Pi 2026-09-26, fáze 7)

### Verzování releasů (blok 0051)

Každý release má verzi `vMAJOR.MINOR.PATCH` z git tagu (první release je
`v1.0.0`):

- **PATCH** — opravy chyb bez změny chování konfigurace, API a instalace;
- **MINOR** — nová funkčnost, která je zpětně kompatibilní (včetně nových
  volitelných položek konfigurace a nových endpointů API);
- **MAJOR** — nekompatibilní změna konfigurace, API nebo instalačního
  postupu.

Tag vytváří a pushuje vlastník projektu. `make release-arm64` přebírá verzi
výhradně z tagu tvaru `vMAJOR.MINOR.PATCH` přesně na `HEAD`
(`deploy/release_version.sh`) a odmítne sestavit release, pokud `HEAD`
takový tag nemá nebo pracovní strom obsahuje necommitnuté změny (včetně
nesledovaných souborů). Ruční přepsání `VERSION=… make release-arm64` se
ignoruje. Vývojové buildy (`make build`, `make build-arm64`) omezení nemají
a verzi berou z `git describe`.

### Postup

1. Na čistém stromu vytvořit tag: `git tag -a vX.Y.Z -m "…"`.
2. `make release-arm64` vyprodukuje archiv
   `bin/marionette-vX.Y.Z-linux-arm64.tar.gz` s binárkou
   (`marionette-linux-arm64`) a instalačními soubory a vedle něj
   `bin/marionette-vX.Y.Z-linux-arm64.tar.gz.sha256`.
3. `git push origin vX.Y.Z`.
4. Archiv i `.sha256` se nahrají na cílový Linux host; referenční ověření
   probíhá na Raspberry Pi ARM64 s Ubuntu 24.x. V adresáři s oběma soubory
   se integrita ověří `sha256sum -c marionette-vX.Y.Z-linux-arm64.tar.gz.sha256`.
5. Po rozbalení se spustí `sudo ./install.sh ./marionette-linux-arm64`.
   Skript nainstaluje unit, zachová existující konfiguraci a provede
   `systemctl enable` + start/restart služby.
6. Nasazená verze se ověří přes `curl -s http://localhost:8080/api/health`
   (pole `version` musí odpovídat tagu).
7. Aktualizace používá stejný skript s novou binárkou; data
   (`marionette.json`, `devices.json`) i `/etc/default/marionette` zůstávají,
   přepíše se binárka a unit. Rollback spustí instalátor předchozího
   releasu, který je proto potřeba si ponechat. Postup se zálohou, kontrolou
   a rollbackem přes MAJOR verzi je v README, sekce „Upgrading and rolling
   back“.
8. Žádný krok nevyžaduje instalaci Go, Node.js ani jiného runtime na cíli.

## Verzování závislostí

- Go: `go.mod` — udržovat na aktuální stabilní verzi Go (viz README pro
  minimální verzi); zvyšovat obezřetně a testovat build na arm64.
- npm (`web/`): držet PrimeVue na stabilní major větvi (`v4-stable` dist-tag),
  viz [ADR-0003](../architecture/decisions/0003-vue-primevue-frontend.md).
  Před přijetím nové major verze jakékoli klíčové závislosti (Vite, Vue,
  PrimeVue, TypeScript) ověřit changelog a napsat/aktualizovat ADR, pokud
  přináší breaking changes.
- Layout utility jsou vlastní `web/src/styles/layout.css` (14 tříd, viz
  [ADR-0010](../architecture/decisions/0010-own-layout-utilities-replace-primeflex.md));
  PrimeFlex byl odstraněn v bloku 0035. PrimeVue zůstává na v4 (MIT), verze 5
  má komerční licenci a vyžaduje nové ADR.
