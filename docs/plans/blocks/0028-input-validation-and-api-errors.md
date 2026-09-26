# Implementační blok: Validace vstupů a konzistence chybových odpovědí API

- **Fáze**: 8 — Zpevnění
- **Vazba na požadavky**: FR-10, FR-11, FR-40, NFR-01, NFR-03, NFR-04
- **Vazba na ADR**: ADR-0004
- **Stav**: Schváleno
- **Závislosti**: Blok 0001 (typy a validace), 0010 (REST API), 0025 (sentinel chyby)

## Cíl bloku

Po dokončení API odmítne kartu, jejíž ID nejde použít v URL, jejíž pole
překračují rozumné limity nebo jejíž timeout by trvale obsadil slot, a
všechny chybové odpovědi na `/api/*` mají jednotnou JSON obálku včetně 405,
413 a 415.

## Rozsah

- **Uvnitř**:
  - ID karty: `^[A-Za-z0-9_-]{1,64}$` (server-generovaná ID vyhovují);
    ostatní hodnoty se odmítají 422;
  - limity délek: `Name` ≤ 120, `Description` ≤ 2000, `Command` ≤ 512, každý
    `Args` prvek ≤ 1024 a max 64 prvků, `Dir` ≤ 1024, `Env` max 64 položek,
    klíč `^[A-Za-z_][A-Za-z0-9_]*$` ≤ 128, hodnota ≤ 4096;
  - `TimeoutSec` 1..3600 (horní mez konstanta `MaxTimeoutSec`);
  - `Icon` omezit na známé hodnoty ze slovníku ikon (viz `CARD_ICON_OPTIONS`
    ve frontendu) nebo prefix `pi pi-` s délkou ≤ 64;
  - validační chyby vrací strukturovaně: `{"error": "...", "fields":
    {"primary.timeoutSec": "must be between 1 and 3600"}}`;
  - dekodér: `http.MaxBytesError` → 413; chyby JSON dekodéru vrací obecné
    „invalid JSON body“ + pozice bez interních názvů Go typů;
  - vlastní 405 handler s JSON obálkou a hlavičkou `Allow`;
  - `Cache-Control` pro SPA: `index.html` `no-cache`, `assets/*`
    `public, max-age=31536000, immutable`; adresářové cesty nevrací listing;
  - `GET /api/health` vrací `{"status":"ok","version":"…","uptimeSec":n}`;
    verze se vkládá přes `-ldflags -X` v Makefile.
- **Mimo rozsah**:
  - whitelisting příkazů (rozhodnutí fáze 1: bez whitelistu);
  - změna chování fronty (blok 0027);
  - autentizace.

## Schválení

- **Schválil**: projektový vlastník
- **Datum schválení**: 2026-09-26
- **Poznámky k rozhodnutí**: Schváleno včetně navržených limitů. Nález
  revize 2026-09-26 (M-7, L-7, L-8, L-11 část health).

## Návrh řešení

- `internal/config/types.go`: konstanty limitů, `ValidationError` typ s
  polem `Fields map[string]string`, `Validate()` sbírá všechny chyby místo
  první; `ErrValidation` sentinel (z 0025).
- `internal/server/server.go`: `writeValidationError`, `methodNotAllowed`
  handler zaregistrovaný pro známé cesty s nepodporovanou metodou (Go 1.22
  mux: registrovat `/api/cards/{id}` bez metody jako fallback), `decodeJSON`
  s `errors.As(err, &maxBytesErr)`.
- `internal/server/spa.go` (rozdělení `server.go`): `serveSPA` s
  `fs.Stat` + `IsDir` kontrolou a cache hlavičkami.
- `cmd/marionette/main.go`: `var version = "dev"`; Makefile
  `-ldflags "-X main.version=$(git describe --tags --always)"`.
- Frontend: zobrazení `fields` u formuláře řeší blok 0032; zde jen kontrakt.

## Testovací plán

- Tabulkové testy validace: každý limit má případ „na hranici“ a „přes“;
  ID s `/`, mezerou, diakritikou, prázdné; env klíč s `=`; timeout 0, 3601.
- Handler testy: 405 s `Allow`, 413 pro tělo > 1 MiB, 415 (po 0026),
  422 s `fields`, chybová zpráva neobsahuje `Go struct field`.
- SPA: `/assets` (adresář) → `index.html`, `/assets/<hash>.js` → immutable.
- Health: JSON obsahuje `version` a `uptimeSec` ≥ 0.
- `go test -race ./...`, `go vet ./...`.

## Kritérium hotovosti

Viz [Definition of Done](../../devops/definition-of-done.md) +:

- žádná odpověď na `/api/*` není `text/plain`;
- limity jsou vypsané v `docs/architecture/overview.md` (API kontrakt) a
  ADR-0004 má doplněk „Limity hodnot“;
- `deploy/marionette.example.json` limity splňuje.

## Uzavření

- **Stav po implementaci**: čeká
- **Ověření**: čeká
- **Dokumentace aktualizována**: čeká
