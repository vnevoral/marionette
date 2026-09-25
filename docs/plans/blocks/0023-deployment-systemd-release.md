# Implementační blok: Fáze 7 — systemd nasazení a release artefakt

- **Fáze**: 7 — Balíčkování a nasazení
- **Vazba na požadavky**: FR-01..FR-04, NFR-02
- **Vazba na ADR**: ADR-0002
- **Stav**: Probíhá
- **Závislosti**: Bloky 0001–0022, existující `make build-arm64`

## Cílové prostředí

- produkční cíl: obecný Linux se systemd, včetně Raspberry Pi ARM64
  (`linux/arm64`);
- referenční validační prostředí: Ubuntu 24.x na Raspberry Pi ARM64 a
  Ubuntu 24.x `linux/amd64`;
- cílový host nesmí vyžadovat Node.js, Go ani jiný runtime.

## Cíl bloku

Po dokončení bude možné sestavit linux/arm64 release, nainstalovat Marionette
na podporovaný Linux se systemd jedním instalačním skriptem a provozovat ji
jako službu po bootu. Referenční ověření proběhne na Raspberry Pi s Ubuntu
24.x; cílový host nebude potřebovat Node.js, Go ani jiný runtime.

## Rozsah

- přidat verzovaný systemd unit `deploy/marionette.service`;
- přidat instalační skript, který vytvoří servisního uživatele, datový adresář,
  nainstaluje binárku a unit a provede `daemon-reload` + `enable --now`;
- podporovat konfiguraci přes `/etc/default/marionette`, zejména
  `MARIONETTE_CONFIG` a `MARIONETTE_ADDR`;
- nastavit bezpečná oprávnění datového adresáře a konfiguračního souboru;
- doplnit release Make target pro Raspberry Pi `linux/arm64` a ověřit, že
  artefakt je statický/bez runtime závislosti cílového UI;
- aktualizovat README a `docs/devops/ci-cd.md` o instalační a rollback postup.

Mimo rozsah:

- TLS terminace, reverzní proxy a autentizace;
- změna HTTP API nebo konfigurace aplikace;
- automatické nahrávání artefaktů do GitHub Releases;
- Debian/RPM balíček a podpora jiných init systémů než systemd.

## Schválení

- **Schválil**: projektový vlastník
- **Datum schválení**: 2026-09-25
- **Poznámky k rozhodnutí**: Deployment má podporovat obecný Linux se systemd;
  Ubuntu 24.x na Raspberry Pi ARM64 slouží jako referenční validační prostředí.

## Návrh řešení

- `deploy/marionette.service` poběží jako dedikovaný uživatel `marionette`,
  bude mít `WorkingDirectory=/var/lib/marionette`, načte `/etc/default/marionette`
  a nastaví `Restart=on-failure`.
- Instalační skript přijme cestu k binárce nebo použije artefakt
  `bin/marionette-linux-arm64`, ověří platformu/elf formát, vytvoří
  `/usr/local/bin/marionette` a zachová existující konfiguraci.
- Výchozí data budou v `/var/lib/marionette/marionette.json`; konfigurace bude
  čitelná pouze uživatelem služby. Skript nebude přepisovat existující data bez
  explicitní volby.
- Makefile dostane release target, který sestaví UI, cross-compile binárku a
  vytvoří distribuovatelný archiv obsahující binárku, unit, default config a
  instalační skript.

## Testovací plán

- ověřit `make build` a `make build-arm64`;
- ověřit `file`/`readelf` release artefaktu a absenci dynamické runtime závislosti;
- shell test instalačního skriptu v dočasném rootu nebo containeru bez zásahu do
  hostního `/etc` a `/var`;
- na Ubuntu 24.x ověřit `systemd-analyze verify`, start po instalaci,
  `/api/health`, restart po ukončení procesu a zachování konfigurace;
- návrh konfigurace služby musí porovnat `/etc/default`, systemd drop-in nebo
  jiný vhodný mechanismus a zvolit variantu s nejlepším přenosem na obecný
  systemd Linux;
- na ARM64 validačním hostu ověřit spuštění výsledného artefaktu a dostupnost
  embedded SPA bez Node.js;
- ověřit rollback na předchozí binárku bez smazání `marionette.json`.

## Kritérium hotovosti

Viz [Definition of Done](../devops/definition-of-done.md) +:

- `deploy/marionette.service` projde `systemd-analyze verify`;
- instalační skript je idempotentní a nepřepisuje existující konfiguraci;
- `make release-arm64` vytvoří zdokumentovaný linux/arm64 artefakt pro
  Raspberry Pi i obecný ARM64 Linux;
- služba po rebootu nebo restartu systemd automaticky naběhne a lokální health
  endpoint odpoví;
- dokumentace obsahuje install, update a rollback postup.

## Uzavření

- **Stav po implementaci**: Probíhá
- **Ověření**: `make release-arm64`, `go build ./...`, `go vet ./...`,
  `go test ./... -count=1`, `bash -n deploy/install.sh` a `git diff --check`
  prošly. Archiv obsahuje binárku, unit, default konfiguraci, JSON šablonu a
  instalační skript; binárka je ELF64 AArch64 a statická. Staged instalace
  ověřila idempotenci a zachování existující konfigurace.
- **Zbývá ověřit**: `systemd-analyze verify`, start/stop/restart, health endpoint
  a rollback na referenčním Ubuntu 24.x hostu. Devcontainer nástroj
  `systemd-analyze` neposkytuje.
- **Dokumentace aktualizována**: README, `docs/devops/ci-cd.md`, requirements,
  testing strategy a roadmapa.
