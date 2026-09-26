# CI/CD

## CI (GitHub Actions, `.github/workflows/ci.yml`)

Na každý push a pull request:

1. **backend** — `golangci-lint` (verze připnutá shodně s devcontainerem,
   konfigurace `.golangci.yml`), `go build ./...`, `go vet ./...`,
   `go test -race -count=1 ./...` na linux/amd64, `bash -n` a staged test
   instalačního skriptu (`deploy/install_test.sh`).
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

## Release proces (cílový stav, viz roadmapa fáze 7)

1. `make release-arm64` vyprodukuje archiv
   `bin/marionette-linux-arm64.tar.gz` s binárkou a instalačními soubory.
2. Archiv se nahraje na cílový Linux host; referenční ověření probíhá na
   Raspberry Pi ARM64 s Ubuntu 24.x.
3. Po rozbalení se spustí `sudo ./install.sh ./marionette-linux-arm64`.
   Skript nainstaluje unit, zachová existující konfiguraci a provede
   `systemctl enable` + start/restart služby.
4. Aktualizace používá stejný skript s novou binárkou. Rollback používá
   předchozí binárku a nemaže `/var/lib/marionette/marionette.json`.
5. Žádný krok nevyžaduje instalaci Go, Node.js ani jiného runtime na cíli.

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
