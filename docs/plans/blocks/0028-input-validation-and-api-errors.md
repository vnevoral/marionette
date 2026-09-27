# Implementation block: Input validation and consistent API error responses

- **Phase**: 8 — Hardening
- **Requirements**: FR-10, FR-11, FR-40, NFR-01, NFR-03, NFR-04
- **ADRs**: ADR-0004
- **Status**: Done
- **Dependencies**: Block 0001 (types and validation), 0010 (REST API), 0025 (sentinel errors)

## Goal

After completion, the API rejects a card whose ID cannot be used in a URL,
whose fields exceed reasonable limits or whose timeout would permanently
occupy a slot, and all error responses on `/api/*` have a uniform JSON
envelope, including 405, 413 and 415.

## Scope

- **In scope**:
  - card ID: `^[A-Za-z0-9_-]{1,64}$` (server-generated IDs comply);
    other values are rejected with 422;
  - length limits: `Name` ≤ 120, `Description` ≤ 2000, `Command` ≤ 512,
    each `Args` element ≤ 1024 and max 64 elements, `Dir` ≤ 1024, `Env`
    max 64 entries, key `^[A-Za-z_][A-Za-z0-9_]*$` ≤ 128, value ≤ 4096;
  - `TimeoutSec` 1..3600 (upper bound constant `MaxTimeoutSec`);
  - restrict `Icon` to known values from the icon vocabulary (see
    `CARD_ICON_OPTIONS` in the frontend) or the prefix `pi pi-` with length
    ≤ 64;
  - validation errors are returned in a structured form: `{"error": "...",
    "fields": {"primary.timeoutSec": "must be between 1 and 3600"}}`;
  - decoder: `http.MaxBytesError` → 413; JSON decoder errors return a
    generic "invalid JSON body" + position without internal Go type names;
  - a custom 405 handler with the JSON envelope and an `Allow` header;
  - `Cache-Control` for the SPA: `index.html` `no-cache`, `assets/*`
    `public, max-age=31536000, immutable`; directory paths do not return a
    listing;
  - `GET /api/health` returns `{"status":"ok","version":"…","uptimeSec":n}`;
    the version is embedded via `-ldflags -X` in the Makefile.
- **Out of scope**:
  - command whitelisting (phase 1 decision: no whitelist);
  - changing queue behavior (block 0027);
  - authentication.

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-09-26
- **Decision notes**: Approved including the proposed limits. Finding of
  the 2026-09-26 review (M-7, L-7, L-8, L-11 health part).

## Proposed solution

- `internal/config/types.go`: limit constants, a `ValidationError` type
  with a `Fields map[string]string` field, `Validate()` collects all
  errors instead of the first one; `ErrValidation` sentinel (from 0025).
- `internal/server/server.go`: `writeValidationError`, a
  `methodNotAllowed` handler registered for known paths with an
  unsupported method (Go 1.22 mux: register `/api/cards/{id}` without a
  method as a fallback), `decodeJSON` with `errors.As(err, &maxBytesErr)`.
- `internal/server/spa.go` (split of `server.go`): `serveSPA` with an
  `fs.Stat` + `IsDir` check and cache headers.
- `cmd/marionette/main.go`: `var version = "dev"`; Makefile
  `-ldflags "-X main.version=$(git describe --tags --always)"`.
- Frontend: displaying `fields` in the form is handled by block 0032; here
  only the contract.

## Test plan

- Table-driven validation tests: every limit has an "at the boundary" and
  an "over" case; ID with `/`, a space, diacritics, empty; env key with
  `=`; timeout 0, 3601.
- Handler tests: 405 with `Allow`, 413 for a body > 1 MiB, 415 (after
  0026), 422 with `fields`, the error message does not contain
  `Go struct field`.
- SPA: `/assets` (directory) → `index.html`, `/assets/<hash>.js` →
  immutable.
- Health: the JSON contains `version` and `uptimeSec` ≥ 0.
- `go test -race ./...`, `go vet ./...`.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md) plus:

- no response on `/api/*` is `text/plain`;
- the limits are listed in `docs/architecture/overview.md` (API contract)
  and ADR-0004 has a "Value limits" addendum;
- `deploy/marionette.example.json` satisfies the limits.

## Closure

- **Status after implementation**: Done (2026-09-26)
- **Verification**: `make verify` passed (golangci-lint, eslint, vue-tsc,
  prettier, `go test -race -count=1 ./...`, build, vet). New tests:
  `TestValidateCollectsAllFieldsAndMatchesErrValidation`,
  `TestValidateLimits` (36 cases: every limit "at the boundary" and
  "over", ID with `/`, a space, diacritics, empty, the shape of a
  generated ID, env key with `=` and a leading digit, timeout
  0/3600/3601, icons),
  `TestRouterAnswersMethodNotAllowedWithAllowAndJSON` (7 paths, `HEAD`
  on a `GET` route passes), `TestRouterRejectsOversizedBodyWithJSON413`,
  `TestRouterDecodeErrorsHideGoTypes` (7 shapes of invalid body, message
  without "Go struct" and package names),
  `TestRouterReturnsValidationFields`,
  `TestHealthReportsVersionAndUptime`, `TestSPACacheHeadersAndDirectories`
  (`/assets/<hash>` immutable, `/assets` and `/assets/` return
  `index.html` without a listing). Existing validation (substring) and
  server tests passed unchanged. `deploy/marionette.example.json` and
  `dev-fixture.json` satisfy the limits (verified by a script: IDs,
  lengths, `pi pi-*` icons, 5 s timeouts).
- **Deviations from the plan**: (1) the icon is not checked against the
  `CARD_ICON_OPTIONS` vocabulary (that would duplicate the frontend list
  in Go), only against the shape `pi pi-<lowercase letters, digits, ->`
  ≤ 64 — the fixture uses `pi pi-exclamation-triangle`, which is not in
  the vocabulary; (2) the 405 fallback is generated automatically from the
  route table (`routeTable`), not manually per path; `Allow` on `GET`
  routes also includes `HEAD`; (3) `ValidationError` has an
  `Is(ErrValidation)` method, so the store does not wrap the validation
  error in `fmt.Errorf("%w: %w")` (the message would be duplicated); (4)
  `Settings.Validate` and `OutputRule.Validate` also return
  `ValidationError` for uniform field merging; (5) `http.FileServer`
  redirects `/index.html` to `/` (301) — kept, `Cache-Control: no-cache`
  is set on `/`; (6) the SSE error "streaming is not supported" was
  converted from `http.Error` (text/plain) to the JSON envelope because of
  the criterion "no response on `/api/*` is `text/plain`"; (7)
  `backend-run` and `build*` use the same `LDFLAGS`, the version can be
  overridden with `make build VERSION=…`.
- **Documentation updated**: yes — `docs/architecture/overview.md`
  (section "API contract — error responses and value limits"), ADR-0004
  ("Addendum — Value limits"), README (version in health), godoc
  (`ValidationError`, `Validate`, `routeTable`, `decodeJSON`,
  `spaHandler`), roadmap. Displaying `fields` in the form is handled by
  block 0032.
