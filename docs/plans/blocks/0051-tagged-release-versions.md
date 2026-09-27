# Implementační blok: Pojmenované verze releasu (git tagy)

- **Fáze**: 7 — Balíčkování a nasazení (dodatek po uzavření fáze)
- **Vazba na požadavky**: FR-01, FR-41 (verze v `/api/health`), NFR-02
- **Vazba na ADR**: ADR-0002; nové ADR není potřeba
- **Stav**: Hotovo
- **Závislosti**: Blok 0023 (release archiv, `VERSION` z `git describe`)

## Cíl bloku

Dnes je verzí releasu hash commitu (`25a0094`), protože repozitář nemá
žádný tag; z `/api/health` ani z názvu archivu nejde poznat, o jakou verzi
jde a jestli je novější. Po dokončení bloku má každý release verzi
`vMAJOR.MINOR.PATCH` z git tagu, archiv ji nese v názvu a release se
nedá omylem sestavit z necommitnutých změn.

## Rozsah

- **Uvnitř**:
  - `Makefile`:
    - `release-arm64` odmítne sestavit release, pokud `VERSION` končí na
      `-dirty` (necommitnuté změny) nebo neodpovídá tagu
      `vMAJOR.MINOR.PATCH` přesně na `HEAD`; vypíše, jak tag vytvořit.
      Vývojový build (`make build`, `build-arm64`) zůstává bez omezení;
    - archiv `bin/marionette-<verze>-linux-arm64.tar.gz` (např.
      `marionette-v1.0.0-linux-arm64.tar.gz`); obsah archivu a jméno
      binárky uvnitř se nemění, takže instalační postup zůstává stejný;
    - vedle archivu soubor `.sha256` pro kontrolu po přenosu;
  - pravidla verzování v `docs/devops/ci-cd.md` (sekce Release proces):
    PATCH = opravy, MINOR = nová funkčnost se zpětně kompatibilní
    konfigurací a API, MAJOR = nekompatibilní změna konfigurace, API nebo
    instalace; tag vytváří a pushuje vlastník projektu; postup
    `git tag -a vX.Y.Z -m "…" && make release-arm64`;
  - README (instalace): název archivu s verzí, kontrola `sha256sum -c`,
    ověření verze přes `/api/health`;
  - automatický test pravidla releasu (skript volaný z `make test`,
    podobně jako `deploy/install_test.sh`): odmítnutí `-dirty`, odmítnutí
    verze bez tagu, přijetí `v1.2.3`.
- **Mimo rozsah**:
  - automatické publikování do GitHub Releases nebo sestavení releasu v CI;
  - zobrazení verze v UI (dnes je v `/api/health`);
  - změna verze v `web/package.json` (balíček je soukromý a verzi
    aplikace neurčuje);
  - vytvoření a push prvního tagu — provede vlastník projektu.

## Schválení

- **Schválil**: projektový vlastník
- **Datum schválení**: 2026-09-27
- **Poznámky k rozhodnutí**: Schváleno vlastníkem („souhlas s návrhy“). První verze `v1.0.0` podle návrhu; tag vytvoří a pushne vlastník.

## Návrh řešení

- Kontrola verze v Makefile přes `git describe --tags --exact-match
  --match 'v[0-9]*.[0-9]*.[0-9]*'` a `git status --porcelain`; logika je
  v malém shell skriptu `deploy/release_version.sh` (vrací verzi nebo
  chybu s nápovědou), aby šla otestovat bez Makefile.
- Bez tagu `release-arm64` skončí chybou; `VERSION=… make release-arm64`
  (ruční přepsání) se nepodporuje, aby verze vždy odpovídala tagu.

## Testovací plán

- Skriptový test v dočasném git repozitáři: čistý strom s tagem
  `v1.2.3` → verze `v1.2.3`; necommitnutá změna → chyba; commit za
  tagem → chyba; tag jiného tvaru (`test`) → chyba.
- Ruční: `make release-arm64` bez tagu selže s nápovědou; po tagu vytvoří
  archiv s verzí v názvu a `.sha256`; `/api/health` na Pi vrátí tag.
- `make verify`.

## Kritérium hotovosti

Viz [Definition of Done](../../devops/definition-of-done.md) +:

- release archiv nese verzi z tagu v názvu i v `/api/health` a nejde
  sestavit z necommitnutých změn.

## Uzavření

- **Stav po implementaci**: Hotovo (2026-09-27)
- **Ověření**:
  - `bash -n deploy/release_version.sh`, `bash -n deploy/release_version_test.sh` — OK;
  - `bash deploy/release_version_test.sh` — `release_version_test: ok`
    (dočasný repozitář: commit bez tagu → chyba; čistý strom s anotovaným
    tagem `v1.2.3` → `v1.2.3`; změněný sledovaný i nový nesledovaný soubor →
    chyba; commit za tagem → chyba; tagy `test`, `v1.2`, `v1.2.3-rc1` →
    chyba; lehký tag `v1.3.0` → `v1.3.0`; adresář mimo repozitář → chyba;
    chybové hlášky obsahují nápovědu `git tag -a vX.Y.Z`);
  - `bash deploy/install_test.sh` — `install_test: ok`;
  - `make -n release-arm64` v pracovním stromu s necommitnutými změnami
    vypíše recept, samotný `make release-arm64` skončí chybou s nápovědou
    (ověřeno v dočasném klonu); v dočasném klonu s tagem `v1.0.0` vypíše
    `make -n release-arm64` i s `VERSION=hack` a `RELEASE_VERSION=bad`
    build s `-X main.version=v1.0.0`, archiv
    `bin/marionette-v1.0.0-linux-arm64.tar.gz` a `.sha256`;
  - závěrečné ověření: `make verify` prošel; plný `make release-arm64`
    v dočasné kopii repozitáře (commit + testovací tag `v1.0.0`, skutečný
    repozitář beze změny) vytvořil `marionette-v1.0.0-linux-arm64.tar.gz`
    a `.sha256`, `sha256sum -c` → OK, obsah archivu beze změny, binárka
    nese `v1.0.0`, strom po buildu zůstal čistý. Ověření `/api/health` na
    Pi provede vlastník po vytvoření skutečného tagu `v1.0.0`.
- **Odchylky od návrhu**:
  - verzi určuje `git tag --points-at HEAD` filtrovaný regulárním výrazem
    `^v[0-9]+\.[0-9]+\.[0-9]+$` místo `git describe --exact-match --match`
    (glob by propustil např. `v1.2.3-rc1`); při více tazích na `HEAD` se
    bere nejvyšší verze;
  - `release-arm64` už nezávisí na `build-arm64` jako na prerekvizitě:
    nejdřív ověří verzi a pak volá `$(MAKE) build-arm64 VERSION=<tag>`,
    aby se kontrola provedla před buildem a verze v binárce odpovídala tagu;
  - test je zapojen jako nový cíl `make release-test` (součást `make test`)
    a jako krok jobu `backend` v `.github/workflows/ci.yml`, protože CI
    `make test` nevolá.
- **Dokumentace aktualizována**: `docs/devops/ci-cd.md` (CI job backend,
  pravidla verzování a postup releasu), `README.md` (instalace: archiv
  s verzí, `sha256sum -c`, ověření verze přes `/api/health`).
