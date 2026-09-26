# Implementační blok: Návod na párování po instalaci

- **Fáze**: 8 — Zpevnění (dodatek po uzavření fáze)
- **Vazba na požadavky**: FR-53, FR-01..04
- **Vazba na ADR**: ADR-0011
- **Stav**: Hotovo
- **Závislosti**: Bloky 0023, 0043, 0044

## Cíl bloku

Po instalaci je UI zamčené, dokud operátor nespáruje první zařízení kódem
z logu služby. `install.sh` o tom dosud mlčel a končil jen hláškou
„Marionette installed and started“. Po dokončení instalátor na hostu bez
spárovaných zařízení vypíše adresu UI a příkaz, kterým se první kód najde.

## Rozsah

- **Uvnitř**:
  - `deploy/install.sh`: `print_pairing_hint` po úspěšné (i staged)
    instalaci. Návod se vypíše, jen když `MARIONETTE_AUTH` není `off`
    a soubor zařízení neexistuje. Cesta se odvodí z `MARIONETTE_DEVICES`,
    jinak z adresáře `MARIONETTE_CONFIG`, port z `MARIONETTE_ADDR`.
    Hodnoty se z `/etc/default/marionette` čtou `sed`em, soubor se
    nespouští (`source`);
  - `deploy/install_test.sh`: návod při první instalaci, port podle
    `MARIONETTE_ADDR`, žádný návod s existujícím `devices.json` ani
    s `MARIONETTE_AUTH=off`;
  - README (sekce instalace).
- **Mimo rozsah**:
  - vypsání samotného kódu instalátorem (služba se teprve rozbíhá a kód
    v logu ještě nemusí být; příkaz z návodu funguje vždy);
  - zjištění skutečného jména nebo IP hostu (za VPN by bylo často
    zavádějící, návod proto uvádí `<host>`).

## Schválení

- **Schválil**: projektový vlastník
- **Datum schválení**: 2026-09-26
- **Poznámky k rozhodnutí**: Odpověď „ano“ na návrh č. 1 („Návod na
  párování po instalaci“) po uzavření bloků 0043–0044.

## Návrh řešení

Viz rozsah. Návod se vypisuje i ve staged režimu (`DESTDIR`), aby ho
pokryl automatický test bez root a systemd.

## Testovací plán

- `deploy/install_test.sh` (součást `make test` a CI).
- `make verify`.

## Kritérium hotovosti

Viz [Definition of Done](../../devops/definition-of-done.md).

## Uzavření

- **Stav po implementaci**: Hotovo (2026-09-26)
- **Ověření**: `bash deploy/install_test.sh` prošel, `make verify` prošel.
  `make e2e` se nespouštěl: blok nemění UI flow ani HTTP kontrakt.
- **Odchylky od návrhu**: žádné.
- **Dokumentace aktualizována**: ano — README (instalace), roadmapa.
