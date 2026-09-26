# Implementační blok: Párování zařízení — backend

- **Fáze**: 8 — Zpevnění (položka „auth/access control“)
- **Vazba na požadavky**: FR-50..FR-56, NFR-01, NFR-12, NFR-13
- **Vazba na ADR**: ADR-0011
- **Stav**: Hotovo
- **Závislosti**: Bloky 0025, 0026, 0038; blok 0044 (frontend) navazuje

## Cíl bloku

Po dokončení backend vyžaduje device token na všech API endpointech kromě
health a párování, umí vydat token za jednorázový kód, spravuje seznam
zařízení a při startu bez zařízení zapíše párovací kód do logu. SPA v tomto
bloku ještě nemá obrazovku párování (blok 0044); do té doby jde přístup
vypnout `MARIONETTE_AUTH=off`.

## Rozsah

- **Uvnitř**:
  - balíček `internal/access`: `Devices` (načtení/uložení `devices.json`,
    přidání, odebrání, ověření tokenu, `lastSeen`, expirace po uplynutí platnosti),
    `Pairing` (jediný platný kód, 10 min, 5 pokusů), generování tokenů
    a kódů z `crypto/rand`, SHA-256 hash, `subtle.ConstantTimeCompare`;
  - sdílený atomický zápis souboru (`internal/fsutil.WriteFileAtomic`)
    vyčleněný z `config` a použitý oběma balíčky;
  - middleware `requireDevice` v `internal/server`: cookie
    `marionette_device` → zařízení v kontextu požadavku; bez něj `401`
    s obálkou `{"error": "...", "code": "pairing_required"}`; obnova cookie
    nejvýš jednou denně;
  - endpointy:
    - `GET /api/session` → `200 {device}` nebo `401` s
      `"bootstrap": true`, když není spárované žádné zařízení;
    - `POST /api/pairing` `{code, name}` → `201 {device}` + `Set-Cookie`;
      neplatný/prošlý kód → `400` s jednou obecnou zprávou;
    - `POST /api/pairing/code` → `201 {code, expiresAt}` (viz Odchylky);
    - `GET /api/devices` → seznam s příznakem `current`;
    - `DELETE /api/devices/{id}` → `204`; u vlastního zařízení smaže cookie;
  - SSE: při každém heartbeatu ověří, že zařízení stále existuje, jinak
    stream ukončí;
  - `cmd/marionette`: `MARIONETTE_AUTH` (`on` výchozí / `off` s varováním),
    `MARIONETTE_COOKIE_SECURE` (`auto` výchozí = podle TLS, `always`,
    `never`), `MARIONETTE_DEVICES` (výchozí `devices.json` vedle
    `MARIONETTE_CONFIG`), `MARIONETTE_DEVICE_EXPIRY_DAYS` (výchozí 60,
    1–400); bootstrap kód do logu při startu bez zařízení;
    uložení `lastSeen` při shutdownu;
  - dokumentace: overview (sekce Přístup), README (párování, obnova, curl,
    proměnné), `deploy/marionette.default`, requirements (NFR-01 finální
    znění, sekce 13), ADR-0011 → Přijato.
- **Mimo rozsah**: UI (blok 0044), bearer/API tokeny pro skripty, passkeys,
  mTLS, víceuživatelské role.

## Schválení

- **Schválil**: projektový vlastník
- **Datum schválení**: 2026-09-26
- **Poznámky k rozhodnutí**: Požadavek vlastníka 2026-09-26: vnitřní síť +
  VPN, jednorázové ověření zařízení, žádné opakované heslo. Schváleno se
  změnou: platnost 60 dní místo 180 a konfigurovatelná.

## Návrh řešení

- `devices.json`: `{"devices":[{"id","name","tokenHash","pairedAt",
"lastSeenAt"}]}`, práva `0600`; poškozený soubor se karanténuje stejně
  jako konfigurace (`.corrupt-<čas>`) a služba startuje bez zařízení
  → bootstrap kód.
- `lastSeenAt` se v paměti aktualizuje každým požadavkem, na disk se
  zapisuje jen při obnově cookie (≤ 1× denně na zařízení) a při shutdownu —
  kvůli opotřebení SD karty (stejný princip jako ADR-0004).
- Pořadí middleware: `requireSameOrigin` (NFR-12) → `requireDevice` → mux.
  Veřejné: `GET /api/health`, `GET /api/session`, `POST /api/pairing`,
  vše mimo `/api/`.
- Chybné pokusy o párování se logují (bez kódu) na úrovni `warn`.

## Testovací plán

- `internal/access`: generování (délka, abeceda), expirace kódu, jednorázové
  použití, 5 pokusů, nový kód ruší starý, ověření tokenu, hash na disku
  (token v souboru není), práva 0600, expirace po uplynutí platnosti, karanténa
  poškozeného souboru, souběžný přístup (`-race`).
- `internal/server`: 401 na každé chráněné routě (tabulka rout jako
  u NFR-12), veřejné routy bez tokenu, párování nastaví cookie se
  správnými atributy (`Secure` podle režimu), obnova cookie, odebrání
  vlastního zařízení smaže cookie, SSE skončí po odebrání zařízení.
- `cmd/marionette`: bootstrap kód v logu jen bez zařízení; `MARIONETTE_AUTH=off`
  zaloguje varování a nechá API otevřené; neplatné hodnoty proměnných
  → chyba startu.
- `make verify`; `make e2e` s `MARIONETTE_AUTH=off` (do bloku 0044).

## Kritérium hotovosti

Viz [Definition of Done](../../devops/definition-of-done.md) +:

- `grep` logu a `devices.json` po testech neobsahuje žádný token;
- ADR-0011 je `Přijato`, NFR-01 má finální znění.

## Uzavření

- **Stav po implementaci**: Hotovo (2026-09-26)
- **Ověření**: `make verify` prošel (`golangci-lint`, Vitest 97, installer
  test, `go test -race`, `go vet`). Nové balíčky a testy:
  `internal/access` (abeceda a normalizace kódu, párování a jednorázovost,
  expirace kódu, nahrazení kódu, 5 pokusů, neplatné jméno nespotřebuje
  pokus, soubor 0600 bez tokenu, obnova 1× denně, zařízení v denním
  používání nevyprší, nepoužité vyprší i po znovuotevření, uložení
  posledního použití, odebrání, karanténa poškozeného souboru, nečitelný
  soubor zastaví start, souběh pod `-race`); `internal/fsutil` (práva nového
  a existujícího souboru, karanténa s kolizí); `internal/server/access_test.go`
  (401 s `pairing_required` na 15 chráněných routách bez cookie i
  s podvrženou, health a SPA veřejné, bootstrap kód v logu jen bez zařízení
  a jen jednou za platnost, token není v logu ani v těle, atributy cookie
  pro 4 režimy `Secure`, chybný kód 400 bez cookie a bez kódu v logu,
  prázdné jméno 422, párování podléhá NFR-12, správa zařízení a
  odhlášení, SSE stream skončí po odebrání zařízení, router bez přístupu
  zůstává otevřený); `cmd/marionette` (proměnné a jejich chybné hodnoty,
  kód v logu jen bez zařízení, `off` s varováním, nečitelný soubor zastaví
  start, integrační běh: 401 → párování kódem z logu → 200 → shutdown uloží
  zařízení, token nikde v logu ani souboru; pořadí kroků shutdownu).
  `make e2e` s `MARIONETTE_AUTH=off`: jednou selhal hned po `make verify`
  bez zachyceného výstupu, další čtyři běhy 8/8.
- **Odchylky od návrhu**: (1) Endpoint pro kód dalšího zařízení je
  `POST /api/pairing/code` místo `/api/devices/pairing-code` — `ServeMux`
  v Go odmítá kombinaci `DELETE /api/devices/{id}` s pevnou cestou pod
  stejným prefixem v rámci metodových fallbacků (405). (2) Bootstrap kód se
  kromě startu zapíše i při `GET /api/session`, pokud žádný neplatí;
  jinak by kód ze startu po 10 minutách vypršel a operátor by musel
  restartovat službu. (3) Nečitelný `devices.json` zastaví start (fail
  closed) místo režimu jen pro čtení jako u konfigurace. (4) E2E běží do
  bloku 0044 s `MARIONETTE_AUTH=off`.
- **Dokumentace aktualizována**: ano — `docs/architecture/overview.md`
  (sekce „Přístup ze spárovaných zařízení“, bezpečnostní poznámka), README
  (proměnné, „Pairing devices“, obnova, curl), `deploy/marionette.default`,
  requirements (sekce 3.6, NFR-01, NFR-13, sekce 13), ADR-0011 Přijato,
  roadmapa.
