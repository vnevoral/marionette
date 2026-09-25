# Implementační blok: SSE — živé změny statusů

- **Fáze**: 10 — Realtime status delivery
- **Vazba na požadavky**: FR-20, FR-27, FR-40, FR-42, NFR-03, NFR-06, NFR-11
- **Vazba na ADR**: ADR-0008 (SSE status event stream)
- **Stav**: Hotovo
- **Závislosti**: FR-42/NFR-11, existující status projection a scheduler (bloky 0007–0009), REST API (bloky 0010–0012), Dashboard (blok 0013)

## Cíl bloku

Po dokončení backend publikuje změny status projekce přes Server-Sent Events a
Dashboard je zobrazuje bez čekání na další ruční akci nebo periodický refresh.
REST zůstává zdrojem počátečního načtení a fallbacku při nedostupném SSE.

## Rozsah

- **Uvnitř**:
  - SSE endpoint pro klientské odběratele statusových událostí;
  - event `status.changed` s ID karty a aktuální `StatusSnapshot`;
  - publikace pouze při skutečné změně statusu, ne při opakované kontrole stejného stavu;
  - heartbeat, korektní ukončení připojení a omezená správa odběratelů;
  - reconnect klienta a návrat k REST status API při výpadku streamu;
  - napojení Dashboardu na `EventSource` a aktualizace příslušné karty;
  - jednotkové, HTTP/integration a browser smoke testy.
- **Mimo rozsah**:
  - WebSocket nebo obousměrná komunikace;
  - nahrazení interního status scheduleru SSE vrstvou;
  - doručování historie událostí nebo durable event broker;
  - autentizace/autorizace připojení;
  - změna formátu uložené konfigurace nebo status historie.

## Schválení

- **Schválil**: projektový vlastník
- **Datum schválení**: 2026-09-25
- **Poznámky k rozhodnutí**: SSE je požadovaný jednosměrný transport server → browser; veřejný kontrakt je definovaný v ADR-0008.

## Návrh řešení

Přidat do serverové vrstvy broadcaster statusových událostí s bezpečným
přihlášením/odhlášením klientů. Status service po úspěšné aktualizaci projekce
publikuje pouze přechod stavu; broadcaster zapíše každému odběrateli SSE event
s `event: status.changed`, `id` a JSON payloadem. Heartbeat bude posílán v
pravidelném intervalu a pomalý/odpojený klient nesmí blokovat scheduler ani
ostatní klienty.

Frontend po načtení Dashboardu otevře `EventSource`, při události aktualizuje
status konkrétní karty a při `error` použije omezený REST fallback/backoff.
Stávající periodický polling zůstane jako bezpečnostní fallback, dokud browser
SSE připojení není potvrzené.

## Testovací plán

- jednotkový test broadcasteru: publish, subscribe, unsubscribe a odpojení;
- test, že opakovaný stejný status nevytvoří SSE event;
- test, že pomalý odběratel neblokuje publikaci ani HTTP API;
- HTTP test SSE hlaviček, heartbeat a korektního ukončení request contextu;
- test Dashboardu: event aktualizuje správnou kartu, reconnect a REST fallback;
- `go test -race ./...`, `go vet ./...`, `npm run lint`, `npm run build`;
- manuální browser smoke test změny statusu mezi dvěma otevřenými klienty.

## Kritérium hotovosti

SSE připojení je omezené na životnost klientského requestu, neblokuje status
scheduler ani REST API, změna statusu se v Dashboardu projeví bez ručního
refresh a při výpadku SSE zůstane dostupný REST fallback. Všechny kontroly
z Definition of Done projdou.

## Uzavření

- **Stav po implementaci**: Hotovo
- **Ověření**: `go build ./...`, `go vet ./...`, `go test -race ./...`,
  `npm run lint -- --quiet`, `npm run build`, `git diff --check` — vše úspěšné.
- **Dokumentace aktualizována**: ano; ADR-0008, roadmapa, FR-42/NFR-11.
