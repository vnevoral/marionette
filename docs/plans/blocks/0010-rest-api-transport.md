# Implementation block: REST API contract and HTTP transport

- **Phase**: 5 — REST API
- **Requirements**: FR-40, FR-41, NFR-01
- **ADRs**: ADR-0004, ADR-0005, ADR-0006
- **Status**: Done
- **Dependencies**: Blocks 0001–0009; approved development workflow

## Goal

The backend provides a stable REST/JSON transport for the health check,
card CRUD and read-only reads of runs and the latest status. The status
read is synchronous only as a fast load of the stored projection; it never
runs the status action. Separate enqueue endpoints for the primary and the
manual status action return an enqueue confirmation without waiting for
completion.

## Scope

- **In scope**:
  - `GET /api/health` as a liveness endpoint;
  - `GET/POST /api/cards`;
  - `GET/PUT/DELETE /api/cards/{id}`;
  - `GET /api/cards/{id}/runs`;
  - `GET /api/cards/{id}/status`;
  - `GET /api/cards/{id}/status/history`;
  - dependency injection for the store and read-only application services;
  - a uniform JSON error response and HTTP status mapping;
  - bounded JSON body, rejection of invalid JSON and unexpected trailing data.
- **Out of scope**:
  - action enqueue endpoints and the background execution manager (0011);
  - scheduler lifecycle and reconcile (0012);
  - authentication, pagination, async jobs and a settings endpoint;
  - Vue UI changes.

## Approval

- **Approved by**: pending
- **Approval date**: pending
- **Decision notes**: The API uses the existing domain JSON types without a new DTO layer.

## Proposed solution

Extend `internal/server` with a handler with an injected `config.Store` and
interfaces for future action services. The router will keep the SPA
fallback. Successful responses will use the existing types `ActionCard`,
`Run`, `StatusSnapshot` and `StatusChange`.

Recommended error mapping: malformed/validation `400`, `ErrNotFound` `404`,
duplicate create `409`, unexpected error `500`. Errors of the action
enqueue endpoints are part of block 0011.
All JSON responses set `Content-Type: application/json`.

## Test plan

- `httptest` for health and all CRUD endpoints;
- valid and invalid JSON, missing fields and trailing JSON;
- `404`, `409` and `500` mapping;
- reading the primary history, the latest status and the transition history;
- disallowed HTTP methods and the SPA fallback.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md). Specifically:
CRUD, read-only history/status and the strict JSON contract are covered by
`httptest` tests and the router does not use global state.

## Closure

- **Status after implementation**: Done
- **Verification**: `go test ./internal/server -count=1`
- **Documentation updated**: yes; action enqueue remains in block 0011
