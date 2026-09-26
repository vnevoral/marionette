# CI/CD

## CI (GitHub Actions, `.github/workflows/ci.yml`)

Na každý push a pull request:

1. **backend** — `go build ./...`, `go vet ./...`, `go test ./...` na
   linux/amd64.
2. **web** — `npm ci`, `npm run lint`, `npm run build` (zahrnuje type-check
   přes `vue-tsc`) ve `web/`.
3. (volitelně, jakmile bude `golangci-lint` v devcontaineru/CI) `make lint`.

CI běh musí projít čistě (bez chyb; varování z lintů se řeší, ne potlačují)
před mergem do hlavní větve.

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
- PrimeFlex je připnutý na `4.0.0` jako zamrzlá layout vrstva pro MVP (viz
  [ADR-0009](../architecture/decisions/0009-primeflex-frozen-layout-layer.md));
  Dependabot ho neaktualizuje a migrace na Tailwind se zváží až po fázi 8.
