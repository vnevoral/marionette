# ADR-0008: SSE stream for live status changes

- **Status**: Accepted
- **Date**: 2026-09-25

## Context

The status scheduler updates the projection of card statuses on the server,
but the browser learns about a change only on a manual refresh, navigation or
periodic REST polling. Marionette needs a one-way transport for immediate
delivery of status changes without introducing a bidirectional protocol or
an external message broker.

The requirement is recorded in FR-42 and NFR-11. The status projection
already distinguishes the last check from an actual status transition
(ADR-0006), so repeated checks of the same status should not create
unnecessary client events.

## Decision

Marionette will use Server-Sent Events for live delivery of status changes
from the server to connected browser clients.

- A public endpoint will provide `text/event-stream` and the event
  `status.changed`.
- The event will carry an `id`, the card identifier and a JSON
  `StatusSnapshot`.
- The event is published only after the card's interpreted status changes; a
  repeated check without a status change is published only as an internal
  projection update.
- The server will send a heartbeat and, when a subscriber's HTTP request
  context ends, safely unsubscribe it without blocking the scheduler, the
  REST API or other clients.
- The client will use the native `EventSource`, automatic reconnect and the
  REST read-only API both for the initial load and as a fallback when SSE is
  unavailable.
- SSE does not replace the internal scheduler, the status service or the
  REST contract and does not require an external broker.

## Considered alternatives

- **Polling from the browser only** — rejected as the main solution; it
  increases the number of requests, adds delay and the client does not know
  when the change actually happened. It remains as a fallback.
- **WebSocket** — rejected; the application only needs a server → browser
  flow and has no requirement for bidirectional communication.
- **External message broker** — rejected; it contradicts the goal of a single
  binary process without runtime/database dependencies.

## Consequences

- The dashboard can show a status change practically immediately without a
  manual refresh.
- Lifecycle and memory management of connected SSE clients is added; the
  broadcaster must have bounded buffers and must not wait for a slow
  subscriber.
- The REST API remains necessary for the first load, reconnect
  synchronization and fallback.
- The SSE endpoint must be part of the HTTP tests and the browser smoke test,
  including disconnects, reconnects and parallel clients.
- In the MVP no event log is persisted and the client cannot request
  historical events; on reconnect it loads the current projection via REST.

## Addendum 2026-09-26 (block 0034)

The broadcaster lives in a separate package `internal/events`
(`events.Broker` with `Publish`, `Subscribe`, `Close`, `Done`);
`internal/server` contains only the SSE writing via the `EventSource`
interface. The wiring `Store.OnStatusChange → Broker.Publish` is done by the
composition root in `cmd/marionette`, not by the router.

## Addendum 2026-09-26 (block 0027)

The broadcaster (`StatusEventBroker`) has `Close()`, which on a graceful
application shutdown disconnects all subscribers and ends every running
`/api/events` handler immediately; it is called from
`http.Server.RegisterOnShutdown`, so `Shutdown` does not wait for long-lived
SSE connections. Events published after `Close()` are discarded. The client
reconnects after a service restart thanks to the native `EventSource`
reconnect.

## Addendum 2026-09-27: `run.recorded` event (block 0058)

> Addendum status: **Accepted** (2026-09-27, by approval of block 0058).

Besides `status.changed`, the `/api/events` stream also carries the event
`run.recorded` (FR-42a). It is sent after a finished run of the **primary
action** is written to the history (`Store.AppendRun`), with `cardId` and
the run in the same shape as an item of `GET /api/cards/{id}/runs`. Running
an action stays asynchronous (`202`); the event only announces that the run
has finished and is recorded.

The same rules as for `status.changed` apply: the broker does not block the
producer, a slow subscriber loses the oldest events, there is no event log
or replay. The client therefore treats the event as a speed-up, not as the
only source. After reconnecting it loads the runs via REST and while waiting
for a run it has fallback requests and a time budget (block 0050). A run
that is not written (an action canceled on service shutdown before it got a
slot to run, or a card deleted during the run) does not create an event; a
run interrupted while running is recorded as `canceled`.
Events for enqueueing and starting an action are not introduced.

## Addendum 2026-09-27: subscription before `: connected` (block 0061)

> Addendum status: **Accepted** (2026-09-27, by approval of block 0061).

The `/api/events` handler subscribes to the broker **before** it writes and
flushes the `: connected` comment. A client that has received the comment
therefore gets every event published after it; the REST reload the client
performs when the stream opens cannot miss a change that happens meanwhile.
Before this, a change published between the comment and the subscription
was lost for that client. Replay of events published before the connection
(`Last-Event-ID`) is still not provided.
