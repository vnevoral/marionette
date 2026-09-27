# Požadavky na Marionette (SRS) — v0.11

> Stav: **zpřesněno** (fáze 1 a UX specifikace, 2026-09-25; doplněno o SSE stream
> pro živé změny statusů, 2026-09-25; doplněno o dva polling intervaly a
> vynucenou terminaci akcí, 2026-09-25; doplněno o perzistenci historie běhů
> při řízeném ukončení, 2026-09-25; řízení status akcí interním schedulerem,
> 2026-09-25; doplněno o UX požadavky FR-24 až FR-29 a NFR-08 až NFR-10,
> 2026-09-25; revize projektu a kódu 2026-09-26: NFR-12, upřesnění FR-18 a
> NFR-01, viz [Rozhodnutí — revize 2026-09-26](#13-rozhodnutí-revize-projektu-2026-09-26);
> doplněno FR-05, FR-21a a FR-22a po validaci na Raspberry Pi, 2026-09-26;
> FR-57 přejmenování zařízení, 2026-09-27; FR-10a barva karty, 2026-09-27;
> FR-06, FR-41a a FR-42a, 2026-09-27).
> Otevřené otázky z v0.1 byly rozhodnuty
> s vlastníkem projektu, viz [Rozhodnutí fáze 1](#7-rozhodnutí-fáze-1),
> [Rozhodnutí — polling a terminace](#9-rozhodnutí-polling-a-terminace-2026-09-25)
> a [Rozhodnutí — perzistence historie při vypnutí](#11-rozhodnutí-perzistence-historie-při-vypnutí-2026-09-25).
> Každá položka má ID pro zpětné odkazování z ADR a implementačních bloků.

## 1. Účel a rozsah

Marionette je samostatně nasaditelná aplikace, která běží jako služba na
cílovém hostu (typicky Raspberry Pi ARM64 s Ubuntu 24.x, případně obecný
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
- **FR-02**: Aplikace běží jako systémová služba na podporovaném Linuxu
  (systemd unit; referenčně ověřeno na Ubuntu 24.x), se startem při bootu a
  automatickým restartem při pádu.
- **FR-03**: Aplikace naslouchá na konfigurovatelném HTTP portu a slouží jak
  API, tak statický obsah UI ze stejného procesu.
- **FR-04**: Podporované cílové platformy: linux/amd64 (vývoj/testy) a
  linux/arm64 (Raspberry Pi).
- **FR-05** _(doplněno 2026-09-26)_: Referenční status akce `ping` (FR-16)
  musí jít spustit pod servisním uživatelem v hardenované unitě
  (`NoNewPrivileges=true` znemožní `ping` použít file capability
  `cap_net_raw`, takže funguje jen neprivilegovaný ICMP podle
  `net.ipv4.ping_group_range`). Instalátor zjistí, zda host neprivilegovaný
  ICMP povoluje; pokud ne, vypíše srozumitelný návod na nápravu. Systémová
  nastavení hostu sám nemění. Návod je i v dokumentaci instalace.
- **FR-06** _(doplněno 2026-09-27)_: Release archiv pro linux/arm64 (verze
  z git tagu `vMAJOR.MINOR.PATCH`, soubor `.sha256`) sestaví a zveřejní CI
  automaticky po pushnutí tagu jako GitHub Release. Tag dál vytváří jen
  vlastník projektu. Release se nezveřejní, pokud neprojdou testy.

### 3.2 Akční karty a akce

- **FR-10**: Uživatel může vytvořit, upravit a smazat akční kartu. Karta má
  název, popis, ikonu a je viditelná na dashboardu.
- **FR-10a** _(doplněno 2026-09-27)_: Karta má volitelnou **barvu** pro
  vizuální rozlišení na dashboardu, zobrazenou jako barevný proužek na okraji
  karty. Barva se vybírá v editaci karty z pevné palety přibližně 8 možností
  včetně „bez barvy“ (výchozí, karta bez proužku) a černé; každá možnost má
  v editoru textový název. Barva nenese žádný stav: stav karty dál ukazuje
  jen status badge (FR-20, FR-25) a paleta se nepřekrývá se stavovými
  barvami. Karta bez barvy (i z konfigurace starší verze) se chová jako dnes.
  Barva je vidět i v detailu karty.
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
- **FR-18** _(aktualizováno 2026-09-26)_: Počet akcí spuštěných současně v
  rámci celé aplikace (ruční spuštění i polling dohromady) je omezen
  konfigurovatelným limitem, výchozí **4**; akce nad limit čekají ve frontě.
  Fronta je omezená (kapacita `4 × limit`); požadavek na spuštění při plné
  frontě API odmítne stavem 503 s hlavičkou `Retry-After`, takže klient je
  vždy informován a žádná přijatá (202) akce se neztrácí. Požadavek na
  akci, která už pro stejnou kartu a stejný druh (primární/status) ve
  frontě čeká, se nezařazuje znovu a API vrací 202 idempotentně. Původní
  znění „žádná se neztrácí“ bez omezení fronty nahrazeno tímto upřesněním.

### 3.3 Dashboard a UI

- **FR-20**: Hlavní obrazovka (Dashboard) zobrazuje všechny akční karty jako
  mřížku/seznam s aktuálním stavem (barevně odlišené: neznámý/OK/chyba/běží).
- **FR-21**: Z karty lze jedním klikem spustit primární akci i status akci a
  sledovat průběžný/poslední výsledek.
- **FR-21a** _(doplněno 2026-09-26)_: Detail karty zobrazuje u poslední
  status kontroly kromě výsledku také exit kód, dobu běhu a zachycený výstup
  (FR-13, na vyžádání rozbalitelný, s označením zkrácení), aby operátor
  zjistil příčinu stavu **Problem** bez přístupu k API nebo k hostu.
  Dashboard zůstává skenovatelný; výstup na kartě nezobrazuje.
- **FR-22**: Existuje samostatná obrazovka pro správu (konfiguraci) karet a
  akcí (CRUD).
- **FR-22a** _(doplněno 2026-09-26)_: V editoru karty se primární i status
  akce zadává jako **jeden příkazový řádek** (např.
  `/usr/bin/ping -c 1 -W 2 192.168.1.10`), ne jako samostatný příkaz a
  opakovatelné řádky argumentů. UI řádek rozloží na příkaz a argumenty
  (uvozovky a `\` pro argumenty s mezerou, bez expanze proměnných a globů)
  a pod polem zobrazí výsledek rozkladu; existující akce se zobrazí zpět
  jako jeden řádek. Shellové znaky (`|`, `&`, `;`, `<`, `>`, `(`, `)`,
  `` ` ``, `$`) mimo uvozovky editor odmítne s vysvětlením. Uložený tvar
  akce, API a spouštění se nemění (FR-11, NFR-01 a); viz
  [ADR-0012](../architecture/decisions/0012-single-line-command-editor.md).
- **FR-23**: UI je postaveno na Vue 3 + PrimeVue (viz
  [ADR-0003](../architecture/decisions/0003-vue-primevue-frontend.md)).

- **FR-24**: UI má jednotnou informační architekturu pro operátora a
  administrátora. Dashboard slouží pro rychlé sledování a spuštění akcí,
  správa karet pro konfiguraci a detail karty pro historii a diagnostiku.
  Navigace mezi těmito kontexty je dostupná z každé hlavní obrazovky.
- **FR-25**: UI používá jednotný design systém s pojmenovanými tokeny pro
  barvy, typografii, spacing, rozměry ovládacích prvků, povrchy a stavy.
  Stavové barvy mají stejný význam v tagu, ikoně, textu i případném grafu;
  stav nesmí být komunikován pouze barvou.
- **FR-26**: Názvy stavů, akcí, tlačítek, chyb, potvrzení a prázdných stavů
  používají jednotný slovník. Rozhraní používá jeden zvolený jazyk napříč
  všemi obrazovkami; míchání jazyků a technických interních názvů v běžném
  uživatelském textu není přípustné.
- **FR-27**: Každá asynchronní operace rozlišuje stav požadavku (čeká,
  přijato, probíhá, dokončeno, selhalo) a poslední známý stav zařízení.
  Uživatel dostane lokální zpětnou vazbu, serverovou chybu a možnost opakovat
  načtení bez ztráty neuložených změn.
- **FR-28**: Hlavní workflow jsou použitelné klávesnicí a na dotykové
  obrazovce. Interaktivní prvky mají viditelný focus, popisný název a
  dostatečný kontrast; ikona bez textu má tooltip nebo jiný dostupný popis.
- **FR-29**: Dashboard, správa i detail jsou použitelné na šířkách od 320 px
  po desktop bez horizontálního scrollu, překrytí nebo změny významu ovládacích
  prvků. Hustota informací se přizpůsobí kontextu: dashboard je skenovatelný,
  formulář čitelný a historie porovnatelná.

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
- **FR-34** _(doplněno 2026-09-26)_: Cesta ke konfiguračnímu souboru je
  nastavitelná proměnnou prostředí `MARIONETTE_CONFIG` (výchozí
  `./marionette.json`), analogicky k již existující `MARIONETTE_ADDR`
  (výchozí `:8080`) pro HTTP adresu/port. Celkový limit řízeného ukončení
  (FR-35) je nastavitelný proměnnou `MARIONETTE_SHUTDOWN_TIMEOUT` (formát Go
  `time.Duration`, výchozí `20s`; musí být kratší než `TimeoutStopSec`
  systemd jednotky, výchozí 90 s). Logování řídí `MARIONETTE_LOG_FORMAT`
  (`text`/`json`, výchozí `text`) a `MARIONETTE_LOG_LEVEL` (`debug`, `info`,
  `warn`, `error`; výchozí `info`) — blok 0034.
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
- **FR-41a** _(doplněno 2026-09-27)_: UI zobrazuje verzi běžící aplikace
  (z `GET /api/health`) na každé obrazovce, aby operátor po upgradu ověřil
  verzi bez příkazové řádky.
- **FR-42**: Backend poskytuje Server-Sent Events stream pro živé změny stavů
  karet. Připojený klient se přihlásí k endpointu pro události a při každé
  změně status projekce obdrží událost obsahující identifikátor karty a novou
  `StatusSnapshot`; opakované kontroly beze změny stavu událost nevytvářejí.
  Stream posílá pravidelný heartbeat, podporuje opětovné připojení klienta a
  klient při dočasné nedostupnosti streamu použije REST read-only API jako
  fallback. REST API zůstává zdrojem pro počáteční načtení obrazovky a
  synchronní načtení aktuální projekce.
- **FR-42a** _(doplněno 2026-09-27)_: Stejný stream posílá událost po
  **zapsání dokončeného běhu primární akce** do historie (FR-17), s
  identifikátorem karty a zapsaným během. Spuštění akce zůstává asynchronní
  (FR-15, `202`); událost klientovi oznámí, že akce doběhla, aniž by musel
  historii opakovaně načítat. Běh, který se nezapíše (akce zrušená při
  vypnutí služby dřív, než začala běžet, nebo karta mezitím smazaná),
  událost nevytvoří. Klient proto dál počítá s REST
  načtením po znovupřipojení a s časovým limitem čekání.

### 3.6 Přístup a párování zařízení _(přijato 2026-09-26, ADR-0011)_

Přístup do UI a API mají jen **spárovaná zařízení** (prohlížeče). Zařízení
se ověří jednou, jednorázovým kódem, a pak už se neověřuje; žádné heslo se
nezadává. Služba běží ve vnitřní síti, zvenčí přes VPN.

- **FR-50 Povinné spárování**: Všechny endpointy pod `/api/` kromě
  `GET /api/health` a párovacích endpointů vyžadují platný device token;
  bez něj odpoví `401` s JSON obálkou. Statické soubory SPA zůstávají
  veřejné (neobsahují data); SPA na `401` zobrazí obrazovku párování.
  SSE stream (`/api/events`) vyžaduje token stejně jako REST.
- **FR-51 Párovací kód**: Kód má 8 znaků z abecedy bez zaměnitelných znaků
  (Crockford Base32, ~40 bitů), platí 10 minut, je jednorázový a po
  5 chybných pokusech se zneplatní. Existuje nejvýš jeden platný kód; nový
  kód předchozí zneplatní. Kódy se drží jen v paměti.
- **FR-52 Device token**: Po zadání platného kódu a názvu zařízení (např.
  „Pracovní notebook“) server vydá náhodný token (256 bitů) v cookie
  `HttpOnly`, `SameSite=Strict`, `Path=/`. Server ukládá jen hash tokenu.
  Zařízení, které se nepoužilo déle než **platnost zařízení**, vyprší;
  platnost je parametr `MARIONETTE_DEVICE_EXPIRY_DAYS` (výchozí 60 dní,
  rozsah 1–400) a stejnou dobu má i cookie. Používané zařízení nevyprší:
  server při používání obnoví cookie i čas posledního použití (nejvýš
  jednou denně).
- **FR-53 První zařízení**: Pokud není spárované žádné zařízení, služba při
  startu vygeneruje párovací kód a zapíše ho do logu
  (`journalctl -u marionette`); obrazovka párování operátora na log odkáže.
  Jakmile existuje spárované zařízení, kód se do logu nezapisuje.
- **FR-54 Další zařízení**: Spárované zařízení může v UI (**Devices** →
  **Pair a new device**) vygenerovat kód pro další zařízení; zobrazí se
  kód a odkaz, který ho předvyplní.
- **FR-55 Správa zařízení**: UI zobrazí seznam spárovaných zařízení (název,
  kdy spárováno, kdy naposledy použito, označení „this device“) a umožní
  zařízení odebrat. Odebrání platí okamžitě pro REST; otevřený SSE stream
  odebraného zařízení skončí nejpozději při dalším heartbeatu (15 s).
  Odebrání vlastního zařízení je odhlášení.
- **FR-56 Obnova přístupu**: Při ztrátě všech zařízení operátor na hostu
  smaže soubor se zařízeními a restartuje službu; při startu se pak
  vygeneruje nový kód podle FR-53. Postup je v README.
- **FR-57 Přejmenování zařízení** _(doplněno 2026-09-27)_: Název zařízení
  se zadává při párování (FR-52) a spárované zařízení ho může později
  změnit na stránce **Devices**, a to u kteréhokoli spárovaného zařízení
  včetně vlastního (stejně jako odebrání, FR-55). Pro nový název platí
  stejná pravidla jako při párování (povinný, po oříznutí mezer nejvýš
  64 znaků); neplatný název server odmítne `422` s chybou u pole. Změna se
  okamžitě uloží do souboru zařízení a neovlivní token, platnost ani čas
  posledního použití. Názvy nemusí být unikátní.

## 4. Nefunkční požadavky

- **NFR-01 Bezpečnost** _(změněno 2026-09-26: přístup jen ze spárovaných
  zařízení, sekce 3.6, ADR-0011; věta o chybějící autentizaci níže tím
  přestává platit, ostatní body a–c platí dál)_: MVP neimplementuje autentizaci/autorizaci k UI/API —
  vychází se z předpokladu nasazení v důvěryhodné síti (domácí/lab síť za
  firewallem, bez přímé expozice do internetu). Pokud je potřeba přístup zvenčí,
  je to odpovědnost provozovatele (VPN/reverse proxy s vlastní autentizací), ne
  aplikace samotné. Přesto: (a) akce se vždy spouští přes `exec.Command(name,
args...)` se strukturovanými argumenty, nikdy skládáním shell příkazu ze
  stringu/uživatelského vstupu (žádný shell interpolation), (b) rozhraní config
  store/API se navrhuje tak, aby šlo auth vrstvu doplnit později bez zásadní
  přestavby (viz roadmapa fáze 8), (c) _(doplněno 2026-09-26)_ spouštěná
  akce nedědí celé prostředí procesu služby; dostane minimální základ
  (`PATH`, `HOME`, `LANG`, `TZ`) a proměnné definované v akci (`Env`), aby se
  případné citlivé proměnné služby nedostaly do uživatelských příkazů.
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
- **NFR-08 Vizuální konzistence**: Nové i upravené obrazovky používají sdílený
  shell, tokeny a komponentové vzory; lokální ad-hoc barvy, typografie a
  spacing se nepřidávají bez zdůvodnění v design dokumentaci.
- **NFR-09 Přístupnost**: UI splňuje základní pravidla WCAG 2.2 AA pro kontrast,
  focus, názvy ovládacích prvků, klávesové ovládání a změny stavu; kritické
  workflow jsou ověřeny alespoň klávesnicí a na mobilní šířce.
- **NFR-10 UX ověřitelnost**: Každý UX blok má popsané stavy loading, empty,
  error, success a destructive action a před uzavřením projde screenshot nebo
  manuální smoke test hlavních workflow.
- **NFR-11 Realtime aktualizace**: SSE stream pro změny statusů nesmí blokovat
  REST API ani běh status scheduleru. Odpojení klienta nesmí vytvářet
  neomezenou práci ani růst paměti na serveru; připojení má být možné bezpečně
  ukončit při zavření stránky nebo aplikace.

- **NFR-12 Ochrana před cross-site požadavky** _(přijato 2026-09-26)_:
  Mutující API endpointy (vytvoření/změna/smazání karty,
  spuštění akce) odmítnou požadavek, který pochází z jiného původu než
  vlastní UI: požadavek s tělem musí mít `Content-Type: application/json`,
  požadavek označený prohlížečem jako `Sec-Fetch-Site: cross-site` nebo s
  hlavičkou `Origin` neshodnou s hostem serveru je odmítnut (403/415).
  Volitelně lze omezit přijímané hodnoty `Host` proměnnou
  `MARIONETTE_ALLOWED_HOSTS`. Porovnání hostů nezávisí na schématu ani na
  výchozích portech 80/443, takže ochrana funguje i za TLS-terminující
  reverse proxy (blok 0036). Read-only endpointy a SSE stream zůstávají bez
  omezení. Důvod: NFR-01 předpokládá důvěryhodnou síť, ne důvěryhodný
  prohlížeč — cizí webová stránka otevřená operátorem by jinak mohla spustit
  libovolnou nakonfigurovanou akci na hostu. Neřeší autentizaci (ta zůstává
  mimo MVP). Implementace: blok 0026.
- **NFR-13 Ochrana párování a tokenů** _(přijato 2026-09-26, ADR-0011)_:
  Tokeny a kódy se generují z `crypto/rand`, porovnávají v konstantním čase
  a na disk se ukládá jen SHA-256 hash tokenu v souboru s právy `0600`
  odděleném od konfigurace karet. Token se nikdy nezapisuje do logu ani
  nevrací v těle odpovědi. Cookie má atribut `Secure`, pokud spojení
  používá TLS nebo je to vynucené konfigurací (za TLS proxy). Autentizace
  nenahrazuje NFR-12 — cross-site ochrana zůstává.

## 5. Omezení a předpoklady

- Cílové prostředí: Linux se systemd; Ubuntu 24.x je referenční prostředí pro
  deployment validaci na Raspberry Pi ARM64 i linux/amd64. Windows není cílová
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

Otázky otevřené revizí 2026-09-26 byly rozhodnuty vlastníkem téhož dne,
viz [sekce 13](#13-rozhodnutí-revize-projektu-2026-09-26).

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
| Cesta ke config souboru / port | `MARIONETTE_CONFIG` (default `./marionette.json`) / `MARIONETTE_ADDR` (default `:8080`) / `MARIONETTE_SHUTDOWN_TIMEOUT` (default `20s`) (FR-34). |
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

## 13. Rozhodnutí — revize projektu (2026-09-26)

Revize struktury, procesu a kódu (2026-09-26) otevřela následující otázky.
Vlastník projektu potvrdil navržená řešení; jde převážně o opravy a narovnání
stavu, ne o novou funkčnost. Implementace je v blocích 0024–0034 (roadmapa,
fáze 8).

| Otázka                         | Rozhodnutí                                                                                                                                                  |
| ------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Omezená fronta akcí vs. FR-18  | Fronta je omezená, plná fronta vrací 503 + `Retry-After`; stejná karta + druh akce ve frontě → idempotentní 202 (FR-18 upřesněno, blok 0027).               |
| Prostředí spouštěných akcí     | Minimální základ `PATH`, `HOME`, `LANG`, `TZ` + `Env` akce; nedědí se `os.Environ()` služby (NFR-01 c, blok 0034).                                          |
| Endpoint pro globální nastavení | `Store.UpdateSettings` se z veřejného API store odstraní (YAGNI); limit souběžnosti se mění v souboru a projeví se po restartu (blok 0034).                 |
| Budoucnost PrimeFlex           | Po fázi 8 odstraněn a nahrazen vlastní utility vrstvou se stejnými třídami; Tailwind zamítnut, PrimeVue zůstává na v4 (MIT) kvůli licenci v5 ([ADR-0010](../architecture/decisions/0010-own-layout-utilities-replace-primeflex.md), blok 0035). |
| Cross-site ochrana API         | NFR-12 přijat; implementace v bloku 0026.                                                                                                                   |
| Licence repozitáře             | MIT (blok 0031 přidá `LICENSE`).                                                                                                                            |
| Autentizace (NFR-01)           | Přístup jen ze spárovaných zařízení: jednorázový kód (první v logu služby), pak device token v cookie bez dalšího ověřování; platnost 60 dní nepoužívání, konfigurovatelná; bez tokenů pro skripty (FR-50..56, NFR-13, [ADR-0011](../architecture/decisions/0011-device-pairing-access.md), bloky 0043–0044). |

## 12. Sledovatelnost

Každý implementační blok v [roadmapě](../plans/roadmap.md) musí odkazovat na
alespoň jedno FR/NFR z tohoto dokumentu. Nové požadavky se přidávají promptem
`/new-requirement` a dostávají další volné ID v rámci příslušné sekce.
