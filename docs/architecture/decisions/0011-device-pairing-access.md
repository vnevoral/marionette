# ADR-0011: Přístup ze spárovaných zařízení (device token)

- **Stav**: Přijato
- **Datum**: 2026-09-26

## Kontext

NFR-01 dosud vylučovalo autentizaci z MVP s odkazem na důvěryhodnou síť.
Vlastník projektu 2026-09-26 upřesnil nasazení a požadavek:

- služba běží ve vnitřní síti, zvenčí je dostupná přes VPN;
- přístup má být omezený na „bezpečná“ zařízení vlastníka;
- ověření má proběhnout **jednou** při propojení klienta, potom už se klient
  neověřuje — žádné opakované zadávání hesla.

Hrozba, kterou řešíme: jiné zařízení ve vnitřní síti nebo ve VPN (host,
kompromitované IoT zařízení, spolupracovník) otevře UI nebo zavolá API
a spustí akci na hostu. Neřešíme kompromitaci samotného hostu ani
spárovaného zařízení.

Omezení prostředí: přístup přes VPN typicky po plain HTTP na IP adresu
nebo interní jméno, tedy bez TLS a bez „secure context“ v prohlížeči.

## Rozhodnutí

Přístup k API (kromě health a párování) vyžaduje **device token** v cookie.
Token zařízení získá jednorázovým **párovacím kódem**:

1. První kód vygeneruje služba při startu, pokud nemá žádné spárované
   zařízení, a zapíše ho do logu (`journalctl -u marionette`). Kdo čte log
   služby na hostu, je oprávněný operátor.
2. Další kódy generuje UI na už spárovaném zařízení (Devices → Pair a new
   device).
3. Kód: 8 znaků Crockford Base32, platnost 10 minut, jednorázový, po
   5 chybách zneplatněn, jen v paměti.
4. Token: 256 bitů z `crypto/rand`, cookie `marionette_device`
   (`HttpOnly`, `SameSite=Strict`, `Path=/`, `Secure` při TLS nebo podle
   `MARIONETTE_COOKIE_SECURE`). Na disku jen SHA-256 hash v `devices.json`
   (0600) vedle konfigurace. Zařízení nepoužité po dobu platnosti vyprší;
   platnost je `MARIONETTE_DEVICE_EXPIRY_DAYS` (výchozí 60, rozsah 1–400,
   horní mez je limit prohlížečů pro cookie) a stejnou dobu má `Max-Age`
   cookie. Používání obnovuje cookie i čas posledního použití nejvýš jednou
   denně.
5. Seznam zařízení a jejich odebrání v UI; obnova ztraceného přístupu
   smazáním `devices.json` a restartem.
6. `MARIONETTE_AUTH=off` vypne ověřování pro vývoj; služba to při startu
   zaloguje jako varování. Výchozí stav je zapnuto.

NFR-12 (cross-site ochrana) zůstává beze změny; `SameSite=Strict` ji
doplňuje, nenahrazuje.

Schváleno vlastníkem 2026-09-26: první kód v logu služby, platnost zkrácena
ze 180 na 60 dní a konfigurovatelná, bez tokenů pro skripty.

## Zvažované alternativy

- **Heslo / HTTP Basic auth** — zamítnuto: vlastník výslovně nechce
  opakované zadávání; prohlížeč heslo sice pamatuje, ale Basic auth nemá
  odhlášení ani správu zařízení a heslo se posílá s každým požadavkem.
- **Passkeys (WebAuthn)** — nejsilnější vazba na zařízení, ale vyžaduje
  secure context (HTTPS nebo `localhost`), který přístup přes VPN po HTTP
  nemá, a potvrzení (biometrie/PIN) při každém přihlášení. Zamítnuto pro
  tuto fázi; lze doplnit později jako druhý způsob párování, až bude TLS.
- **Klientské TLS certifikáty (mTLS)** — skutečná vazba na zařízení, ale
  vyžaduje TLS v aplikaci, vlastní CA a ruční instalaci certifikátu do
  každého zařízení; pro jednoho operátora nepřiměřené.
- **Allowlist IP adres** — VPN adresy se mohou měnit a neodliší dvě
  zařízení za stejnou adresou; zamítnuto jako hlavní mechanismus.
- **Spoléhat jen na VPN (stav dosud)** — neřeší hrozbu z vnitřní sítě,
  kterou vlastník chce pokrýt.
- **Bearer token v hlavičce pro skripty** — mimo rozsah; žádný
  neprohlížečový klient dnes neexistuje. Lze přidat později („API tokeny“)
  nad stejným úložištěm zařízení.

## Důsledky

- Každé zařízení se páruje jednou; pak přístup nevyžaduje nic dalšího, dokud
  zařízení nesmaže cookies, nepoužije jiný prohlížeč nebo ho operátor
  neodebere.
- Cookie je vázaná na prohlížeč, ne na hardware: kdo zkopíruje cookie ze
  spárovaného zařízení, získá přístup. Proti tomu chrání `HttpOnly` (JS ji
  nepřečte), VPN a možnost zařízení odebrat. Silnější vazba = passkeys/mTLS
  (viz alternativy).
- Bez TLS putuje token po síti nešifrovaně; v nasazení za VPN je tunel
  šifrovaný. Při přímém přístupu po LAN bez VPN je token odposlouchatelný —
  README doporučí TLS proxy a `MARIONETTE_COOKIE_SECURE=always`.
- Nové API: párovací endpointy a správa zařízení; všechny ostatní API
  odpovědi mohou být `401`. SPA potřebuje obrazovku párování a stránku
  Devices; E2E sada se páruje v rámci přípravy.
- `curl` bez tokenu přestane fungovat (kromě health); dokumentace
  i instalační postup se upraví.
- Nový soubor stavu `devices.json`; zálohy a obnova se musí zmínit
  v dokumentaci.
