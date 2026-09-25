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

1. `make build-arm64` vyprodukuje `bin/marionette-linux-arm64`.
2. Artefakt + `systemd` unit soubor (`deploy/marionette.service`, vznikne ve
   fázi 7) se nahrají na cílový Linux host; referenční ověření probíhá na
   Raspberry Pi ARM64 s Ubuntu 24.x.
3. Instalace = zkopírovat binárku (např. do `/opt/marionette/`), nainstalovat
   systemd unit, `systemctl enable --now marionette`.
4. Žádný krok nevyžaduje instalaci Go, Node.js ani jiného runtime na cíli.

## Verzování závislostí

- Go: `go.mod` — udržovat na aktuální stabilní verzi Go (viz README pro
  minimální verzi); zvyšovat obezřetně a testovat build na arm64.
- npm (`web/`): držet PrimeVue na stabilní major větvi (`v4-stable` dist-tag),
  viz [ADR-0003](../architecture/decisions/0003-vue-primevue-frontend.md).
  Před přijetím nové major verze jakékoli klíčové závislosti (Vite, Vue,
  PrimeVue, TypeScript) ověřit changelog a napsat/aktualizovat ADR, pokud
  přináší breaking changes.
