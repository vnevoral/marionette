# Požadavky na Marionette (SRS) — v0.4

> Stav: **zpřesněno** (fáze 1, 2026-09-25; doplněno o dva polling intervaly a
> vynucenou terminaci akcí, 2026-09-25; doplněno o perzistenci historie běhů
> při řízeném ukončení, 2026-09-25; řízení status akcí interním schedulerem,
> 2026-09-25). Otevřené otázky z v0.1 byly rozhodnuty
> s vlastníkem projektu, viz [Rozhodnutí fáze 1](#7-rozhodnutí-fáze-1),
> [Rozhodnutí — polling a terminace](#9-rozhodnutí-polling-a-terminace-2026-09-25)
> a [Rozhodnutí — perzistence historie při vypnutí](#11-rozhodnutí-perzistence-historie-při-vypnutí-2026-09-25).
> Každá položka má ID pro zpětné odkazování z ADR a implementačních bloků.

## 1. Účel a rozsah

Marionette je samostatně nasaditelná aplikace, která běží jako služba na
cílovém hostu (typicky Raspberry Pi s Ubuntu, linux/arm64, případně obecný
Linux) a poskytuje webové rozhraní pro definici, spouštění a sledování stavu
uživatelsky nakonfigurovaných akcí nad tímto hostem (příp. nad jeho síťovým
okolím).

Nasazení nesmí vyžadovat instalaci runtime prostředí (Node.js, JVM, Python, ...)
na cílovém hostu — celá aplikace (backend + UI) je distribuována jako jeden
spustitelný soubor.

## 2. Aktéři

- **Operátor** — uživatel přistupující do webového rozhraní, spouští akce a
  sleduje stav karet.
- **Administrátor** — uživatel konfigurující akční karty a akce (může být
  stejná osoba jako operátor v MVP).
- **Host** — stroj, na kterém Marionette běží a na kterém se reálně provádí
  konfigurované příkazy.

## 3. Funkční požadavky

### 3.1 Nasazení a provoz

- **FR-01**: Aplikace se distribuuje jako jeden binární soubor obsahující
  zabudované webové UI (bez nutnosti instalace Node.js/npm na cíli).
- **FR-02**: Aplikace běží jako systémová služba (systemd unit na Ubuntu /
  Raspberry Pi OS), se startem při bootu a automatickým restartem při pádu.
- **FR-03**: Aplikace naslouchá na konfigurovatelném HTTP portu a slouží jak
  API, tak statický obsah UI ze stejného procesu.
- **FR-04**: Podporované cílové platformy: linux/amd64 (vývoj/testy) a
  linux/arm64 (Raspberry Pi).

### 3.2 Akční karty a akce

- **FR-10**: Uživatel může vytvořit, upravit a smazat akční kartu. Karta má
  název, popis, ikonu a je viditelná na dashboardu.
- **FR-11**: Každá karta má právě jednu **primární akci** — definici příkazu
  spouštěného na hostu (příkaz, argumenty, pracovní adresář, proměnné
  prostředí, timeout). Timeout je závazná **maximální doba čekání na
  dokončení akce** — po jejím uplynutí execution engine proces vynuceně
  ukončí (terminate/kill), běh se zaznamená jako neúspěšný/timeout a nesmí
  zůstat viset (viz FR-19, NFR-04).
- **FR-12**: Každá karta může mít volitelně **status akci** (health check) —
  akci stejného typu jako primární, jejíž výsledek určuje aktuální stav karty.
- **FR-13**: Spuštění akce zachytí exit kód, stdout/stderr a čas běhu.
  Zachycený kombinovaný výstup (stdout+stderr) je omezen na **4 KB**; nad tuto
  velikost se výstup ořízne a označí jako zkrácený (aplikace kvůli tomu
  neselže).
- **FR-14**: Vyhodnocení výsledku akce je dané exit kódem (`0` = úspěch, jinak
  neúspěch) a volitelně dalším pravidlem porovnávajícím zachycený výstup proti
  regulárnímu výrazu (např. „musí obsahovat“ / „nesmí obsahovat“). Pravidlo na
  výstup je nepovinné rozšíření nad rámec exit kódu.
- **FR-15** _(aktualizováno 2026-09-25)_: Status akce se může vyvolat ručně
  přes API, ale stejně jako primární akce pouze asynchronně: API potvrdí její
  zařazení ke spuštění a nečeká na její dokončení. Status akci také volitelně
  automaticky spouštějí interní background procesy, především scheduler, v
  nastaveném **standardním polling intervalu**, per karta. API čtení statusu je
  oddělená synchronní read-only operace, která vždy vrací poslední známou
  status projekci a status akci nespouští. Výchozí standardní interval je
  **60 s**; polling lze pro danou kartu zcela vypnout. Standardní polling je
  předpoklad pro FR-15a (zrychlený polling) — bez zapnutého standardního
  pollingu se zrychlený polling neaktivuje.
- **FR-15a**: Bezprostředně po vyvolání **primární akce** karty (ruční nebo
  budoucí naplánované spuštění) se status akce dočasně přepne na
  **zrychlený polling interval**, výchozí **10 s**, po dobu výchozích
  **120 s** (obě hodnoty konfigurovatelné globálně/per karta). Po uplynutí
  této doby se karta vrátí na standardní polling interval (FR-15). Pokud
  karta nemá standardní polling zapnutý, zrychlený polling se neaktivuje
  (viz FR-15).
- **FR-16**: Příklad referenčního use-case: primární akce = Wake-on-LAN paket
  na MAC adresu; status akce = `ping` na IP/hostname cílového PC.
- **FR-19**: Vypršení timeoutu akce (FR-11) vede k vynucené terminaci
  spuštěného procesu ze strany execution enginu (nikdy k jeho ponechání běžet
  na pozadí) — ochrana proti nekontrolovanému hromadění nedokončených
  procesů na slabém hardwaru (NFR-04, NFR-07).
- **FR-17**: Pro primární akci se uchovává historie posledních **N běhů**
  (výchozí N = 20, konfigurovatelné globálně). Status akce má odděleně
  poslední výsledek kontroly pro aktuální stav a historii pouze skutečných
  přechodů stavů. Každý přechod obsahuje nový stav, začátek a konec nebo dobu
  trvání; opakované kontroly se stejným stavem nový historický záznam
  nevytvářejí. UI zobrazuje aktuální stav, dobu jeho trvání a historii změn.
- **FR-18**: Počet akcí spuštěných současně v rámci celé aplikace (ruční
  spuštění i polling dohromady) je omezen konfigurovatelným limitem, výchozí
  **4**; akce nad limit čekají ve frontě, žádná se neztrácí.

### 3.3 Dashboard a UI

- **FR-20**: Hlavní obrazovka (Dashboard) zobrazuje všechny akční karty jako
  mřížku/seznam s aktuálním stavem (barevně odlišené: neznámý/OK/chyba/běží).
- **FR-21**: Z karty lze jedním klikem spustit primární akci i status akci a
  sledovat průběžný/poslední výsledek.
- **FR-22**: Existuje samostatná obrazovka pro správu (konfiguraci) karet a
  akcí (CRUD).
- **FR-23**: UI je postaveno na Vue 3 + PrimeVue (viz
  [ADR-0003](../architecture/decisions/0003-vue-primevue-frontend.md)).

### 3.4 Konfigurace a perzistence

- **FR-30**: Veškerá konfigurace (karty, akce) je uložena v jediném
  konfiguračním souboru (JSON) na disku hosta.
- **FR-31**: Za běhu je konfigurace držena v paměti (in-memory store); zápis
  do souboru se provádí při každé změně (create/update/delete) a při startu se
  soubor načte.
- **FR-32**: Očekávaný objem dat: jednotky až nízké desítky akčních karet —
  neřeší se škálování na velké objemy ani multi-tenant.
- **FR-33**: Poškozený/chybějící konfigurační soubor při startu nesmí shodit
  aplikaci — spustí se s prázdnou konfigurací a chyba se zaloguje.
- **FR-34**: Cesta ke konfiguračnímu souboru je nastavitelná proměnnou
  prostředí `MARIONETTE_CONFIG` (výchozí `./marionette.json`), analogicky k
  již existující `MARIONETTE_ADDR` (výchozí `:8080`) pro HTTP adresu/port.
- **FR-35**: Při **řízeném ukončení aplikace** (přijetí SIGINT/SIGTERM a
  doběhnutí graceful shutdown) se historie primárních běhů a historie změn
  statusu (FR-17) uloží na
  disk společně s konfigurací. Při startu aplikace se historie načte
  společně s konfigurací, pokud je k dispozici a platná — díky tomu se při
  řízeném vypnutí/restartu (např. update binárky, restart systemd služby)
  historie běhů neztrácí. Při neřízeném ukončení (pád procesu, výpadek
  napájení, `SIGKILL`) se historie od posledního uložení ztratí — to je
  akceptované riziko (viz [ADR-0004](../architecture/decisions/0004-action-card-domain-model.md)
  a NFR-03, důvod proč se historie nepersistuje při každém běhu).
  Poškozená/chybějící uložená historie při startu se chová jako FR-33 —
  nesmí shodit aplikaci, jen se historie nenačte (začne prázdná).

### 3.5 API

- **FR-40**: Backend poskytuje REST/JSON API pro CRUD nad kartami/akcemi a pro
  spouštění akcí a čtení historie/posledního výsledku.
- **FR-41**: `GET /api/health` (již existuje) slouží jako liveness endpoint
  procesu samotného (odlišné od status akcí uživatelských karet).

## 4. Nefunkční požadavky

- **NFR-01 Bezpečnost**: MVP neimplementuje autentizaci/autorizaci k UI/API —
  vychází se z předpokladu nasazení v důvěryhodné síti (domácí/lab síť za
  firewallem, bez přímé expozice do internetu). Pokud je potřeba přístup zvenčí,
  je to odpovědnost provozovatele (VPN/reverse proxy s vlastní autentizací), ne
  aplikace samotné. Přesto: (a) akce se vždy spouští přes `exec.Command(name,
args...)` se strukturovanými argumenty, nikdy skládáním shell příkazu ze
  stringu/uživatelského vstupu (žádný shell interpolation), (b) rozhraní config
  store/API se navrhuje tak, aby šlo auth vrstvu doplnit později bez zásadní
  přestavby (viz roadmapa fáze 8).
- **NFR-02 Provozní jednoduchost**: Instalace na nový host = zkopírovat jeden
  binární soubor (+ volitelně systemd unit) a spustit. Žádné externí závislosti
  (DB server, runtime).
- **NFR-03 Výkon**: Aplikace je určena pro nízkovýkonný hardware (Raspberry
  Pi) — nízká paměťová a CPU náročnost, žádné zbytečné pozadí běžící procesy.
- **NFR-04 Spolehlivost**: Pád spuštěné akce (např. neexistující binárka)
  nesmí ovlivnit běh serveru ani ostatních karet.
- **NFR-05 Testovatelnost**: Doménová logika (vyhodnocení akcí, config store)
  je pokryta jednotkovými testy nezávisle na skutečném spouštění procesů
  (abstrakce nad exec).
- **NFR-06 Udržovatelnost**: Bez zbytečných závislostí/frameworků nad rámec
  potřeby (viz [ADR log](../architecture/decisions)).
- **NFR-07 Ochrana slabého HW při souběhu**: Počet současně běžících akcí je
  omezen konfigurovatelným limitem (výchozí 4, viz FR-18), aby polling více
  karet najednou nezahltil Raspberry Pi.

## 5. Omezení a předpoklady

- Cílové OS: Ubuntu Server / Raspberry Pi OS (systemd). Windows není cílová
  produkční platforma.
- Jednouživatelské / malotýmové nasazení v důvěryhodné síti (domácí/lab síť) —
  víceuživatelská RBAC není v MVP požadována, ale NFR-01 musí umožnit budoucí
  rozšíření.
- Data (konfigurace) nejsou citlivá v míře vyžadující šifrování na disku v
  MVP; toto je předpoklad k ověření s administrátorem.

## 6. Otevřené otázky

Žádné blokující otevřené otázky pro fáze 2–5 (viz níže rozhodnutí fáze 1 a
rozhodnutí o pollingu/terminaci). Budoucí otázky (auth pro víceuživatelský
provoz, škálování historie běhů nad rámec N záznamů) se řeší až s konkrétní
potřebou, samostatným requirementem.

## 7. Rozhodnutí fáze 1 (2026-09-25)

Následující otevřené otázky z v0.1 byly rozhodnuty s vlastníkem projektu a
promítnuty do FR/NFR výše:

| Otázka                         | Rozhodnutí                                                                                       |
| ------------------------------ | ------------------------------------------------------------------------------------------------ |
| Autentizace/autorizace         | Žádná v MVP; předpoklad důvěryhodné sítě (NFR-01).                                               |
| Vyhodnocení status akce        | Exit kód + volitelné pravidlo na výstup (regex) (FR-14).                                         |
| Historie běhů                  | Posledních N běhů, výchozí N = 20 (FR-17).                                                       |
| Automatický polling            | Volitelný, per karta, výchozí interval 60 s (FR-15).                                             |
| Validace příkazů               | Bez whitelistingu; jen strukturované argumenty, žádný shell string (NFR-01).                     |
| Limit výstupu akce             | 4 KB kombinovaně stdout+stderr, zbytek se ořízne (FR-13).                                        |
| Cesta ke config souboru / port | `MARIONETTE_CONFIG` (default `./marionette.json`) / `MARIONETTE_ADDR` (default `:8080`) (FR-34). |
| Počet akcí na kartu            | 1 primární + 1 nepovinná status akce, víc akcí na kartu není v MVP (viz FR-11, FR-12).           |
| Souběžnost spouštění akcí      | Konfigurovatelný limit, výchozí 4 (FR-18, NFR-07).                                               |

## 9. Rozhodnutí — polling a terminace (2026-09-25)

| Otázka                             | Rozhodnutí                                                                                                                                       |
| ---------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------ |
| Standardní polling interval        | Výchozí **60 s** (dříve 30 s), per karta (FR-15).                                                                                                |
| Zrychlený polling po primární akci | Výchozí interval **10 s** po dobu **120 s** od vyvolání primární akce, pak návrat na standardní (FR-15a).                                        |
| Zrychlený polling bez standardního | Neaktivuje se — zrychlený polling je jen dočasné zesílení standardního, vyžaduje ho mít zapnutý (FR-15a).                                        |
| Vynucená terminace po timeoutu     | Existující `TimeoutSec` (FR-11) je závazná maximální doba čekání; po vypršení execution engine proces zabije (FR-19), žádný nový parametr navíc. |

## 11. Rozhodnutí — perzistence historie při vypnutí (2026-09-25)

| Otázka                            | Rozhodnutí                                                                                                                                                                                                                                                                                                                              |
| --------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Ztráta historie běhů při restartu | Historie se navíc uloží na disk při **řízeném** ukončení aplikace (graceful shutdown) a načte zpět při startu spolu s konfigurací (FR-35). Za běhu (mezi jednotlivými běhy akcí) se nadále nepersistuje kvůli opotřebení SD karty (ADR-0004). Při neřízeném pádu/výpadku se historie od posledního uložení ztrácí — akceptované riziko. |

## 12. Sledovatelnost

Každý implementační blok v [roadmapě](../plans/roadmap.md) musí odkazovat na
alespoň jedno FR/NFR z tohoto dokumentu. Nové požadavky se přidávají promptem
`/new-requirement` a dostávají další volné ID v rámci příslušné sekce.
