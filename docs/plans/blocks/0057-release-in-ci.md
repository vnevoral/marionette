# Implementační blok: Release z CI po pushnutí tagu

- **Fáze**: 7 — Balíčkování a nasazení (dodatek po uzavření fáze)
- **Vazba na požadavky**: FR-06, FR-01, FR-04, NFR-02
- **Vazba na ADR**: ADR-0002; nové ADR není potřeba
- **Stav**: Hotovo
- **Závislosti**: Blok 0051 (`deploy/release_version.sh`,
  `make release-arm64`)

## Cíl bloku

Release dnes sestavuje vlastník ručně (`make release-arm64`) a archiv
přenáší na Pi ze svého počítače. Výsledek tak závisí na lokálním prostředí
a nikde není zveřejněný. Po dokončení bloku stačí pushnout tag `vX.Y.Z`:
CI spustí testy, sestaví archiv stejným příkazem a zveřejní ho jako GitHub
Release. Na Pi jde archiv stáhnout přímo odkazem.

## Rozsah

- **Uvnitř**:
  - nový workflow `.github/workflows/release.yml`, spouštěný při push tagu
    `v*.*.*`:
    - checkout s tagy (`fetch-depth: 0`), Go a Node podle `go.mod`
      a `web/.nvmrc`;
    - `make verify` a `make e2e`; při selhání se release nezveřejní;
    - `make release-arm64`; `release_version.sh` sám odmítne tag jiného
      tvaru nebo nečistý strom, takže pravidla zůstávají na jednom místě;
    - GitHub Release pro tag s archivem a `.sha256` (`gh release create`
      s `GITHUB_TOKEN`, oprávnění `contents: write` jen pro tento
      workflow). Poznámky k releasu se vygenerují z commitů
      (`--generate-notes`);
  - `docs/devops/ci-cd.md` (Release proces): tag → push → workflow; ruční
    `make release-arm64` zůstává jako záložní postup;
  - README (instalace a upgrade): stažení archivu z GitHub Releases
    (`curl -LO …/releases/download/vX.Y.Z/…`) včetně `.sha256`.
- **Mimo rozsah**:
  - release pro linux/amd64 (cíl je Pi; přidat, až bude potřeba);
  - podepisování archivů (cosign, GPG);
  - automatické vytváření tagů nebo verzí z commitů;
  - předběžné verze (`-rc`), které `release_version.sh` záměrně odmítá.

## Schválení

- **Schválil**: projektový vlastník
- **Datum schválení**: 2026-09-27
- **Poznámky k rozhodnutí**: Schváleno vlastníkem („souhlas se vším“). Tag a jeho push dál provádí vlastník.

## Návrh řešení

Samostatný workflow místo úpravy `ci.yml`: release má jiný spouštěč,
potřebuje právo zápisu a nemá běžet u pull requestů. Používá se `gh` CLI,
který je na runnerech GitHubu předinstalovaný, místo akce třetí strany.
Tak nepřibyde další závislost s přístupem k tokenu.

## Testovací plán

- `actionlint`, pokud je k dispozici, jinak kontrola syntaxe YAML.
- Ověření na skutečném repozitáři: vlastník pushne tag (např. `v1.1.0`),
  workflow projde a Release obsahuje archiv a `.sha256`;
  `sha256sum -c` po stažení na Pi → OK; `/api/health` vrátí tag.
- Negativní případy v forku nebo testovacím repozitáři: tag
  `v1.2.3-rc1` workflow spustí (filtr `v*.*.*` je glob), ale
  `release_version.sh` ho odmítne a Release nevznikne; tag na commitu
  s padajícím testem Release také nevytvoří.
- `make verify`.

## Kritérium hotovosti

Viz [Definition of Done](../../devops/definition-of-done.md) +:

- po pushnutí tagu `vX.Y.Z` vznikne bez ručního kroku GitHub Release
  s archivem a `.sha256` a archiv jde podle README nainstalovat na Pi.

## Uzavření

- **Stav po implementaci**: Hotovo v repozitáři (2026-09-27); ověření na
  skutečném tagu provede vlastník (první tag po tomto bloku, např.
  `v1.1.0`).
- **Ověření**: `actionlint` v1.7.7 (`go run`) bez nálezů pro `ci.yml`
  i `release.yml`; `make verify` prošel. Build release archivu ověřil už
  blok 0051 (`make release-arm64` v dočasném klonu s tagem); release job
  spouští stejný cíl na čistém checkoutu, kde `npm ci` strom nezmění
  (`node_modules` je ignorovaný). Negativní scénáře (tag `v1.2.3-rc1`,
  padající test) nejdou ověřit bez pushnutí tagu. Ověří se, až je vlastník
  bude potřebovat, v forku.
- **Odchylky od návrhu**: místo samostatných kroků `make verify` a
  `make e2e` volá release workflow celý `ci.yml` (nový spouštěč
  `workflow_call`), takže release prochází přesně stejnými joby jako
  push do `main` (backend, web, e2e) a pravidla CI jsou na jednom místě.
- **Dokumentace aktualizována**: `docs/devops/ci-cd.md` (CI, postup
  releasu z CI a záložní ruční postup), README (stažení z GitHub Releases,
  upgrade).
