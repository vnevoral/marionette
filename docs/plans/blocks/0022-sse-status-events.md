# Implementation block: SSE — live status changes

- **Phase**: 10 — Real-time status delivery over SSE
- **Requirements**: FR-20, FR-27, FR-40, FR-42, NFR-03, NFR-06, NFR-11
- **ADRs**: ADR-0008 (SSE status event stream)
- **Status**: Done
- **Dependencies**: FR-42/NFR-11, the existing status projection and scheduler (blocks 0007–0009), REST API (blocks 0010–0012), Dashboard (block 0013)

## Goal

When done, the backend publishes changes of the status projection via
Server-Sent Events and the Dashboard displays them without waiting for
another manual action or a periodic refresh. REST remains the source of
the initial load and the fallback when SSE is unavailable.

## Scope

- **In scope**:
  - an SSE endpoint for client subscribers of status events;
  - a `status.changed` event with the card ID and the current `StatusSnapshot`;
  - publishing only on an actual status change, not on a repeated check of the same state;
  - heartbeat, correct connection termination and bounded subscriber management;
  - client reconnect and falling back to the REST status API when the stream fails;
  - hooking the Dashboard up to `EventSource` and updating the relevant card;
  - unit, HTTP/integration and browser smoke tests.
- **Out of scope**:
  - WebSocket or bidirectional communication;
  - replacing the internal status scheduler with the SSE layer;
  - delivering event history or a durable event broker;
  - authentication/authorization of connections;
  - changing the format of the stored configuration or status history.

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-09-25
- **Decision notes**: SSE is the required one-way transport server → browser; the public contract is defined in ADR-0008.

## Proposed solution

Add a status event broadcaster to the server layer with safe client
subscribe/unsubscribe. After a successful projection update, the status
service publishes only a state transition; the broadcaster writes to each
subscriber an SSE event with `event: status.changed`, `id` and a JSON
payload. A heartbeat will be sent at a regular interval and a slow or
disconnected client must not block the scheduler or other clients.

After the Dashboard loads, the frontend opens an `EventSource`, on an event
it updates the status of the specific card and on `error` it uses a bounded
REST fallback/backoff. The existing periodic polling stays as a safety
fallback until the browser SSE connection is confirmed.

## Test plan

- broadcaster unit test: publish, subscribe, unsubscribe and disconnect;
- test that a repeated identical status does not create an SSE event;
- test that a slow subscriber does not block publishing or the HTTP API;
- HTTP test of SSE headers, heartbeat and correct termination of the request context;
- Dashboard test: an event updates the correct card, reconnect and REST fallback;
- `go test -race ./...`, `go vet ./...`, `npm run lint`, `npm run build`;
- manual browser smoke test of a status change between two open clients.

## Done criteria

The SSE connection is bounded by the lifetime of the client request, it
does not block the status scheduler or the REST API, a status change shows
up in the Dashboard without a manual refresh and when SSE fails the REST
fallback remains available. All checks from the Definition of Done pass.

## Closure

- **Status after implementation**: Done
- **Verification**: `go build ./...`, `go vet ./...`, `go test -race ./...`,
  `npm run lint -- --quiet`, `npm run build`, `git diff --check` — all successful.
- **Documentation updated**: yes; ADR-0008, roadmap, FR-42/NFR-11.
