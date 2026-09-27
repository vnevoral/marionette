# Implementation block: Protecting mutating endpoints against cross-site requests

- **Phase**: 8 — Hardening
- **Requirements**: NFR-01, NFR-12, FR-40
- **ADRs**: none new; extends the security part of ADR-0005 (action execution)
- **Status**: Done
- **Dependencies**: NFR-12 (accepted 2026-09-26); block 0010 (REST API); block 0022 (SSE)

## Goal

Once complete, a foreign web page opened in the operator's browser cannot
create, change or run an action card; the API rejects mutating requests
without the correct `Content-Type` or with a foreign origin. Authentication
remains outside the MVP (NFR-01).

## Scope

- **In scope**:
  - middleware for `POST/PUT/DELETE` on `/api/*`:
    1. a request with a body must have `Content-Type: application/json`
       (otherwise 415);
    2. `Sec-Fetch-Site: cross-site` → 403;
    3. if an `Origin` header is present, its host must match `r.Host`
       (otherwise 403); a missing `Origin` on a request without
       `Sec-Fetch-Site` is allowed for non-browser clients (curl);
  - optional host allowlist `MARIONETTE_ALLOWED_HOSTS` (comma-separated);
    empty = no `Host` check (default, preserves today's behavior);
  - the SPA sends `Content-Type: application/json` on mutating calls even
    for an empty body (enqueue endpoints), so that they pass rule 1;
  - JSON error envelope for 403/415 identical to other errors;
  - documentation in `docs/architecture/overview.md` (security section).
- **Out of scope**:
  - authentication, session, tokens, RBAC (roadmap phase 8, separate block);
  - CORS headers allowing other origins (intentionally none);
  - TLS, reverse proxy, rate limiting.

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-09-26
- **Decision notes**: Finding of the 2026-09-26 review (H-2). `POST
  /api/cards/{id}/actions/primary` is a "simple request" without a preflight;
  `POST /api/cards` can be sent via an `enctype=text/plain` form. NFR-12 was
  accepted on 2026-09-26 together with the approval of the block.

## Proposed solution

- `internal/server/origin.go`: `func requireSameOrigin(allowedHosts []string,
  next http.Handler) http.Handler`. Applied in `NewRouterWithDependencies`
  to the `/api/` subrouter only for mutating methods; `GET`, `HEAD`, SSE and
  the SPA fallback remain unchanged.
- Origin comparison: `url.Parse(origin).Host` vs `r.Host` (including the
  port); `Host` is compared case-insensitively.
- `cmd/marionette/main.go`: reading `MARIONETTE_ALLOWED_HOSTS`, passing it to
  the router; `deploy/marionette.default` gets a commented-out example.
- `web/src/api.ts`: `enqueuePrimary/enqueueStatus` send
  `Content-Type: application/json` (body `null` or an empty object depending
  on the server decoder — the server does not read the body on enqueue, but
  requires the header).

## Test plan

- Unit tests (`internal/server`, `httptest`):
  - `POST /api/cards` without `Content-Type` → 415; with `text/plain` → 415;
  - `Sec-Fetch-Site: cross-site` → 403; `same-origin` → passes;
  - `Origin: http://evil.example` against `Host: pi.local:8080` → 403;
    matching origin → passes; without `Origin` and without `Sec-Fetch-Site`
    → passes;
  - `MARIONETTE_ALLOWED_HOSTS=pi.local:8080` and `Host: 192.168.1.5:8080` → 403;
  - `GET /api/cards` and `GET /api/events` with a foreign `Origin` →
    unchanged (200);
  - table test of the origin comparison function (port, letter case, IPv6).
- Frontend: existing calls pass (`npm run build` + manual smoke test of the
  dashboard: run/check/create/edit/delete).
- Manual verification: a local HTML page on a different port with a form
  `POST` to `/api/cards/{id}/actions/primary` gets 403 and the action does
  not run.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md) plus:

- NFR-12 links to this block and `overview.md` to NFR-12;
- all mutating endpoints are covered by the middleware (the test iterates
  over the registered routes);
- `overview.md` has a section "Protection against cross-site requests".

## Closure

- **Status after implementation**: Done (2026-09-26)
- **Verification**: `make verify` passed (golangci-lint, eslint, vue-tsc,
  prettier, `go test -race -count=1 ./...`, build, vet). New tests in
  `internal/server/origin_test.go`: `TestMutatingRoutesRejectCrossSiteRequests`
  (a table of all five mutating routes × `Sec-Fetch-Site: cross-site`,
  foreign `Origin`, `Origin: null`, form and `text/plain` content, body
  without `Content-Type`), `TestMutatingRoutesAcceptSameOriginAndNonBrowserClients`
  (same-origin, `charset` parameter, `Sec-Fetch-Site: none`, bare `curl`),
  `TestReadOnlyRoutesIgnoreForeignOrigin` (GET, SSE, SPA fallback),
  `TestAllowedHostsRestrictMutatingRequests`, `TestSameOriginComparison`
  (port, default port, letter case, IPv6, `null`, foreign scheme);
  `TestLoadEnvironmentDefaultsAndShutdownTimeout` extended with
  `MARIONETTE_ALLOWED_HOSTS`. Adjusted
  `TestRouterRejectsInvalidJSONAndDuplicateCard` (a body without
  `Content-Type` is now correctly 415). Verified on the real binary via
  `curl`: POST with `Origin: http://evil.example` → 403, with form content →
  415, `Sec-Fetch-Site: cross-site` → 403, bare `curl -X POST` → passed the
  middleware; with `MARIONETTE_ALLOWED_HOSTS=pi.local:8080` a foreign `Host`
  returns 403 and `Host: pi.local:8080` 202. The manual smoke test of the
  dashboard (run/check/create/edit/delete) remains to be done on the
  reference host.
- **Deviations from the plan**: (1) the middleware wraps the whole mux and
  decides by itself based on the method and the `/api/` prefix, because the
  Go 1.22 `ServeMux` has no subrouter; (2) rule 1 requires
  `application/json` also when the `Content-Type` header is present on an
  empty body (e.g. `curl -d ''` sends the form type) — stricter than the
  plan, a bare `curl -X POST` without the header still passes; (3)
  `Origin: null` and an `Origin` with a scheme other than `http`/`https` are
  rejected; (4) a missing port in `Origin` or `Host` is filled in with the
  scheme's default port, so `http://pi.local` matches `pi.local:80`; (5) the
  SPA sends `Content-Type: application/json` also on `DELETE`, not only on
  enqueue calls; (6) the host allowlist applies only to mutating routes (in
  line with the scope), not to reads.
- **Documentation updated**: yes — `docs/architecture/overview.md`
  (section "Protection against cross-site requests", the security note links
  to NFR-12), README (variables table, paragraph on the protection),
  `deploy/marionette.default`, godoc of `requireSameOrigin`, roadmap. NFR-12
  already links to block 0026.
