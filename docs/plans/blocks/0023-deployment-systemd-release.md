# Implementační blok: Fáze 7 — systemd nasazení a release artefakt

- **Fáze**: 7 — Balíčkování a nasazení
- **Vazba na požadavky**: FR-01..FR-04, NFR-02
- **Vazba na ADR**: ADR-0002
- **Stav**: Hotovo
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

Viz [Definition of Done](../../devops/definition-of-done.md) +:

- `deploy/marionette.service` projde `systemd-analyze verify`;
- instalační skript je idempotentní a nepřepisuje existující konfiguraci;
- `make release-arm64` vytvoří zdokumentovaný linux/arm64 artefakt pro
  Raspberry Pi i obecný ARM64 Linux;
- služba po rebootu nebo restartu systemd automaticky naběhne a lokální health
  endpoint odpoví;
- dokumentace obsahuje install, update a rollback postup.

## Uzavření

- **Stav po implementaci**: Hotovo (2026-09-26)
- **Ověření**: `make release-arm64`, `go build ./...`, `go vet ./...`,
  `go test ./... -count=1`, `bash -n deploy/install.sh` a `git diff --check`
  prošly. Archiv obsahuje binárku, unit, default konfiguraci, JSON šablonu a
  instalační skript; binárka je ELF64 AArch64 a statická. Staged instalace
  ověřila idempotenci a zachování existující konfigurace.
- **Ověření v devcontaineru (x86_64, 2026-09-26)**: `sudo ./deploy/install.sh
  bin/marionette` (skutečná, ne staged instalace) odhalilo chybu — binárka
  se instalovala do `/usr/local/marionette`, zatímco unita spouští
  `/usr/local/bin/marionette`; opraveno (`binary_target=$prefix/bin/marionette`).
  Po opravě: `systemd-analyze verify /etc/systemd/system/marionette.service`
  (systemd 252 doinstalován jako balík, bez běžícího PID 1) prošel bez
  nálezu. Služba spuštěna s prostředím unity (`runuser -u marionette`,
  `EnvironmentFile`, `WorkingDirectory`, `umask 077`): `GET /api/health`
  → `{"status":"ok","version":"e68c28d"}`, SPA `lang="en"`, `POST /api/cards`
  s `Origin: https://evil.example` → 403, `Content-Type: text/plain` → 415,
  scénář TLS proxy (`Origin: https://pi.local`, `Host: pi.local`) → 201
  (blok 0036), primární akce → 202 a záznam v `runs`, `marionette.json`
  `0640 marionette:marionette`. SIGTERM: kroky „stop HTTP server“, „save
  history“, „close action queue“, „stop status scheduler“ v logu, historie
  v souboru; po opětovném startu jsou karta i běh načtené. Třetí běh
  instalátoru (update/rollback stejným postupem) zachoval `marionette.json`
  i `/etc/default/marionette`. Instalátor v containeru hlásí „installed and
  started“, protože `systemctl` je zde shim vracející 0 — na hostu se
  systemd to neplatí.
- **Automatický test instalátoru (2026-09-26)**: přidán
  `deploy/install_test.sh` (`make deploy-test`, součást `make test`, CI job
  `backend`): staged instalace do dočasného `DESTDIR`, kontrola cest proti
  unitě (`ExecStart`, `EnvironmentFile`, `WorkingDirectory`/`ReadWritePaths`,
  `MARIONETTE_CONFIG`), práv 755/750/640, zachování konfigurace a defaults
  při opakovaném běhu a odmítnutí ne-ELF souboru. Ověřeno, že test s
  původní chybou cesty binárky selže. Test odhalil druhou chybu: kontrola
  formátu binárky závisela na `file(1)`, který minimální image (včetně
  devcontaineru) nemá, a bez něj se tiše přeskočila; nahrazeno kontrolou
  ELF hlavičky (`7f 45 4c 46`) bez závislostí. Kontrola architektury proti
  hostu zůstává mimo (chybný artefakt se projeví při startu služby).
- **Ověřeno na referenčním Ubuntu 24.x ARM64 hostu (Raspberry Pi,
  2026-09-26)**: vlastník projektu nainstaloval release archiv z commitu
  `25a0094` (`make release-arm64`, SHA-256
  `8bea40a8…6a623b`) a podle kontrolního seznamu ověřil instalaci,
  `/api/health` s verzí, `systemctl restart`, `Restart=on-failure` po
  `kill -9`, náběh po rebootu a kartu WoL + ping ve stavu **Healthy**;
  vše funguje. Podrobné výstupy příkazů nebyly zaznamenány.
- **Původně zbývalo ověřit na hostu**: skutečné
  `systemctl enable/start/stop/restart`, `Restart=on-failure` po zabití
  procesu, náběh po rebootu, běh ARM64 artefaktu a chování hardening
  direktiv (`ProtectSystem=strict`, `ReadWritePaths`) s reálným systemd.
  Poznámka: `systemd-analyze security` hodnotí unitu 8.6 „EXPOSED“; další
  zpřísnění (`ProtectKernelTunables`, `RestrictAddressFamilies`,
  `SystemCallFilter`…) je mimo rozsah bloku a vyžaduje test na hostu.
- **Nález na Raspberry Pi (Ubuntu, 2026-09-26)**: referenční scénář
  WoL + ping (FR-16) — `wakeonlan` ze služby funguje, status akce `ping`
  ale vždy končí `socket: Operation not permitted` (exit 2). Příčina:
  `NoNewPrivileges=true` ruší file capability `cap_net_raw` a host měl
  `net.ipv4.ping_group_range = 1 0`. Ověřeno `systemd-run -p
  User=marionette -p NoNewPrivileges=true … /usr/bin/ping`; náprava
  `ping_group_range = 0 2147483647`. Unita se nemění; detekce v instalátoru
  a dokumentace jsou v novém bloku
  [0046](0046-install-unprivileged-ping-check.md) (FR-05), zobrazení výstupu
  status kontroly v UI v bloku
  [0047](0047-status-check-output-in-detail.md) (FR-21a).
- **Dokumentace aktualizována**: README, `docs/devops/ci-cd.md`, requirements,
  testing strategy a roadmapa.
