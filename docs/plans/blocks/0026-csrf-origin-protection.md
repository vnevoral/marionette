# Implementační blok: Ochrana mutujících endpointů před cross-site požadavky

- **Fáze**: 8 — Zpevnění
- **Vazba na požadavky**: NFR-01, NFR-12, FR-40
- **Vazba na ADR**: žádné nové; rozšiřuje bezpečnostní část ADR-0005 (spouštění akcí)
- **Stav**: Schváleno
- **Závislosti**: NFR-12 (přijato 2026-09-26); blok 0010 (REST API); blok 0022 (SSE)

## Cíl bloku

Po dokončení nelze z cizí webové stránky otevřené v prohlížeči operátora
vytvořit, změnit ani spustit akční kartu; API odmítne mutující požadavky
bez správného `Content-Type` nebo s cizím původem. Autentizace zůstává mimo
MVP (NFR-01).

## Rozsah

- **Uvnitř**:
  - middleware pro `POST/PUT/DELETE` na `/api/*`:
    1. požadavek s tělem musí mít `Content-Type: application/json`
       (jinak 415);
    2. `Sec-Fetch-Site: cross-site` → 403;
    3. je-li přítomna hlavička `Origin`, její host se musí shodovat s
       `r.Host` (jinak 403); chybějící `Origin` u požadavku bez
       `Sec-Fetch-Site` je povolen kvůli non-browser klientům (curl);
  - volitelný allowlist hostů `MARIONETTE_ALLOWED_HOSTS` (čárkou oddělený);
    prázdný = bez kontroly `Host` (výchozí, zachovává dnešní chování);
  - SPA posílá u mutujících volání `Content-Type: application/json` i pro
    prázdné tělo (enqueue endpointy), aby procházela pravidlem 1;
  - JSON chybová obálka pro 403/415 shodná s ostatními chybami;
  - dokumentace v `docs/architecture/overview.md` (sekce bezpečnost).
- **Mimo rozsah**:
  - autentizace, session, tokeny, RBAC (roadmapa fáze 8, samostatný blok);
  - CORS hlavičky pro povolení jiných originů (záměrně žádné);
  - TLS, reverzní proxy, rate limiting.

## Schválení

- **Schválil**: projektový vlastník
- **Datum schválení**: 2026-09-26
- **Poznámky k rozhodnutí**: Nález revize 2026-09-26 (H-2). `POST
  /api/cards/{id}/actions/primary` je „simple request“ bez preflightu; `POST
  /api/cards` lze poslat přes `enctype=text/plain` formulář. NFR-12 byl přijat
  2026-09-26 společně se schválením bloku.

## Návrh řešení

- `internal/server/origin.go`: `func requireSameOrigin(allowedHosts []string,
  next http.Handler) http.Handler`. Aplikuje se v `NewRouterWithDependencies`
  na `/api/` subrouter pouze pro mutující metody; `GET`, `HEAD`, SSE a SPA
  fallback zůstávají beze změny.
- Porovnání originu: `url.Parse(origin).Host` vs `r.Host` (včetně portu);
  `Host` se porovnává case-insensitive.
- `cmd/marionette/main.go`: čtení `MARIONETTE_ALLOWED_HOSTS`, předání do
  routeru; `deploy/marionette.default` dostane zakomentovaný příklad.
- `web/src/api.ts`: `enqueuePrimary/enqueueStatus` posílají
  `Content-Type: application/json` (tělo `null` nebo prázdný objekt dle
  serverového dekodéru — server u enqueue tělo nečte, hlavičku ale vyžaduje).

## Testovací plán

- Jednotkové testy (`internal/server`, `httptest`):
  - `POST /api/cards` bez `Content-Type` → 415; s `text/plain` → 415;
  - `Sec-Fetch-Site: cross-site` → 403; `same-origin` → průchod;
  - `Origin: http://evil.example` proti `Host: pi.local:8080` → 403;
    shodný origin → průchod; bez `Origin` a bez `Sec-Fetch-Site` → průchod;
  - `MARIONETTE_ALLOWED_HOSTS=pi.local:8080` a `Host: 192.168.1.5:8080` → 403;
  - `GET /api/cards` a `GET /api/events` s cizím `Origin` → beze změny (200);
  - tabulkový test funkce porovnání originů (port, velikost písmen, IPv6).
- Frontend: existující volání procházejí (`npm run build` + manuální smoke
  test dashboardu: run/check/create/edit/delete).
- Manuální ověření: lokální HTML stránka na jiném portu s formulářem
  `POST` na `/api/cards/{id}/actions/primary` dostane 403 a akce se
  nespustí.

## Kritérium hotovosti

Viz [Definition of Done](../../devops/definition-of-done.md) +:

- NFR-12 odkazuje na tento blok a `overview.md` na NFR-12;
- všechny mutující endpointy jsou pokryté middlewarem (test iteruje přes
  registrované routy);
- `overview.md` má sekci „Ochrana před cross-site požadavky“.

## Uzavření

- **Stav po implementaci**: čeká
- **Ověření**: čeká
- **Dokumentace aktualizována**: čeká
