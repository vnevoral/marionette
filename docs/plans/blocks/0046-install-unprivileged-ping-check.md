# Implementační blok: Kontrola neprivilegovaného pingu při instalaci

- **Fáze**: 7 — Balíčkování a nasazení
- **Vazba na požadavky**: FR-05, FR-16, FR-02
- **Vazba na ADR**: ADR-0002 (nové ADR není potřeba: nemění se architektura
  ani unita, jen instalátor a dokumentace)
- **Stav**: Hotovo
- **Závislosti**: Blok 0023 (instalátor, unita), blok 0045 (vzor
  `print_pairing_hint`)

## Cíl bloku

Při validaci na Raspberry Pi (Ubuntu, 2026-09-26) karta s referenčním
scénářem WoL + ping trvale hlásila **Problem**, i když cílové PC běželo.
`sudo -u marionette ping …` prošel, ale ve službě `ping` skončil
`socket: Operation not permitted` (exit 2): unita má `NoNewPrivileges=true`,
takže se file capability `cap_net_raw` na `/usr/bin/ping` ignoruje, a host
měl `net.ipv4.ping_group_range = 1 0` (neprivilegovaný ICMP zakázán).
Nápravou je `net.ipv4.ping_group_range = 0 2147483647`.

Po dokončení bloku instalátor tento stav pozná a vypíše návod na nápravu;
README popíše příčinu a způsob ověření v prostředí služby.

## Rozsah

- **Uvnitř**:
  - `deploy/install.sh`: funkce `print_ping_hint` volaná po úspěšné (i
    staged) instalaci vedle `print_pairing_hint`. Přečte
    `/proc/sys/net/ipv4/ping_group_range` (cesta přepsatelná proměnnou
    prostředí kvůli testu); pokud rozsah neobsahuje primární skupinu
    servisního uživatele (ve staged režimu, kdy skupina ještě neexistuje,
    se posuzuje, zda je rozsah prázdný, tj. `min > max`), vypíše vysvětlení
    a příkazy:
    `echo 'net.ipv4.ping_group_range = 0 2147483647' | sudo tee /etc/sysctl.d/99-marionette-ping.conf`
    a `sudo sysctl --system`. Nečitelný soubor (jiné jádro, container) se
    tiše přeskočí;
  - `deploy/install_test.sh`: návod se vypíše pro rozsah `1 0`, nevypíše
    pro `0 2147483647` ani pro chybějící soubor;
  - README: sekce „Status akce ping hlásí Problem“ — příčina, ověření
    příkazem
    `systemd-run … -p User=marionette -p NoNewPrivileges=true /usr/bin/ping …`,
    náprava, a proč se nedoporučuje
    `AmbientCapabilities=CAP_NET_RAW` (raw sockety by dostaly všechny akce);
  - blok 0023: odkaz na tento nález v sekci ověření na ARM64 hostu.
- **Mimo rozsah**:
  - automatická změna `sysctl` instalátorem (FR-05: systémová nastavení
    hostu se nemění bez vědomí operátora; viz otevřená otázka níže);
  - změna unity (`AmbientCapabilities`, `CapabilityBoundingSet`);
  - obecná diagnostika dalších příkazů akcí (`wakeonlan`, `nc` …).

## Schválení

- **Schválil**: projektový vlastník
- **Datum schválení**: 2026-09-26
- **Poznámky k rozhodnutí**: Schváleno vlastníkem („vše schvaluji“). Otevřená otázka rozhodnuta ve prospěch návrhu: instalátor jen vypíše návod, sysctl sám nemění.

## Návrh řešení

Viz rozsah. Kontrola je čistě informativní a nikdy nezpůsobí neúspěch
instalace. Hodnoty se čtou `read` z jednoho řádku (`min max`), bez volání
`sysctl(8)`, aby test nepotřeboval root ani skutečné jádro.

## Testovací plán

- `deploy/install_test.sh` (součást `make test` a CI) se třemi variantami
  podvrženého `ping_group_range`.
- Manuálně na referenčním Raspberry Pi: s `1 0` instalátor návod vypíše; po
  nápravě a opakované instalaci už ne a status akce karty WoL + ping hlásí
  **Healthy** u zapnutého PC.
- `make verify`.

## Kritérium hotovosti

Viz [Definition of Done](../../devops/definition-of-done.md) +:

- scénář z cíle bloku je popsaný v README tak, aby ho operátor vyřešil bez
  znalosti systemd hardeningu.

## Uzavření

- **Stav po implementaci**: Hotovo (2026-09-26)
- **Ověření**: `bash -n deploy/install.sh` a `bash deploy/install_test.sh`
  prošly (varianty `1 0` → návod, `0 2147483647` → bez návodu, chybějící
  soubor → bez návodu, existující uživatel `root` s rozsahem
  `1 2147483647` → návod). `shellcheck` v prostředí není k dispozici.
  `make verify` prošel (včetně `install_test`). Manuální ověření na referenčním
  Raspberry Pi zbývá provést při příští instalaci.
- **Odchylky od návrhu**:
  - proměnná prostředí pro test se jmenuje `PING_GROUP_RANGE_FILE` (v
    souladu s `DATA_DIR`, `UNIT_FILE` …);
  - posuzuje se primární skupina existujícího servisního uživatele
    (`id -g`), jinak GID skupiny `SERVICE_GROUP`; pokud neexistuje ani
    jedno, rozhoduje jen prázdný rozsah (`min > max`). Test navíc pokrývá
    větev s existujícím uživatelem;
  - sekce v README je anglicky („Ping status check reports Problem“), protože
    celé README je v angličtině;
  - odkaz z bloku 0023 na tento nález existuje už z plánování bloku
    (sekce „Nález na Raspberry Pi“).
- **Dokumentace aktualizována**: ano — README (instalace + sekce
  „Ping status check reports Problem“).
