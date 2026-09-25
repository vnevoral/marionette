# Implementační blok: REST API kontrakt a HTTP transport

- **Fáze**: 5 — REST API
- **Vazba na požadavky**: FR-40, FR-41, NFR-01
- **Vazba na ADR**: ADR-0004, ADR-0005, ADR-0006
- **Stav**: Návrh
- **Závislosti**: Bloky 0001–0009; schválený vývojový workflow

## Cíl bloku

Backend poskytne stabilní REST/JSON transport pro health check, CRUD karet a
read-only čtení běhů a posledního statusu. Čtení statusu je synchronní pouze
jako rychlé načtení uložené projekce; nikdy nespouští status akci. Samostatné
enqueue endpointy pro primární i ruční status akci vracejí potvrzení zařazení
bez čekání na dokončení.

## Rozsah

- **Uvnitř**:
  - `GET /api/health` jako liveness endpoint;
  - `GET/POST /api/cards`;
  - `GET/PUT/DELETE /api/cards/{id}`;
  - `GET /api/cards/{id}/runs`;
  - `GET /api/cards/{id}/status`;
  - `GET /api/cards/{id}/status/history`;
  - `POST /api/cards/{id}/actions/status/check` jako asynchronní enqueue;
  - dependency injection pro store a read-only aplikační služby;
  - jednotný JSON error response a mapování HTTP stavů;
  - bounded JSON body, odmítnutí neplatného JSON a neočekávaných trailing dat.
- **Mimo rozsah**:
  - implementace background execution manageru (0011);
  - samotné provedení primární/status akce (0011);
  - scheduler lifecycle a reconcile (0012);
  - autentizace, stránkování, async joby a settings endpoint;
  - změny Vue UI.

## Schválení

- **Schválil**: čeká
- **Datum schválení**: čeká
- **Poznámky k rozhodnutí**: API používá existující JSON typy domény bez nové DTO vrstvy.

## Návrh řešení

Rozšířit `internal/server` o handler s injektovaným `config.Store` a rozhraními
pro budoucí akční služby. Router bude zachovávat SPA fallback. Úspěšné
odpovědi budou používat existující typy `ActionCard`, `Run`, `StatusSnapshot`
a `StatusChange`.

Doporučené mapování chyb: malformed/validation `400`, `ErrNotFound` `404`,
duplicate create `409`, chybějící status akce při enqueue `422`, neočekávaná
chyba `500`.
Všechny JSON odpovědi nastaví `Content-Type: application/json`.

## Testovací plán

- `httptest` pro health a všechny CRUD endpointy;
- validní i nevalidní JSON, chybějící pole a trailing JSON;
- `404`, `409`, `422` a `500` mapování;
- čtení primární historie, posledního statusu a transition historie;
- nepovolené HTTP metody a SPA fallback.

## Kritérium hotovosti

Viz [Definition of Done](../../devops/definition-of-done.md). Specificky:
HTTP kontrakt je pokrytý `httptest` testy a router nepoužívá globální stav.

## Uzavření

- **Stav po implementaci**: čeká
- **Ověření**: čeká
- **Dokumentace aktualizována**: čeká
