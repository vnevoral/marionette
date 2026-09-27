# Implementation block: SSE subscription before the connected comment, reliable timing tests

- **Phase**: after Phase 7 (maintenance of the public repository)
- **Requirements**: FR-42, FR-42a, NFR-11 (live updates), NFR-05
  (testability)
- **ADRs**: ADR-0008 (SSE status event stream)
- **Status**: Proposed
- **Dependencies**: 0022 (SSE), 0027 (shutdown), 0058 (`run.recorded`)

## Goal

An event published right after a client receives `: connected` is never
lost, and the two Go tests that fail intermittently in CI pass reliably, so
pull requests from contributors are not blocked by random failures.

## Scope

In scope:

- **SSE subscription order (bug).** `internal/server/status_events.go` writes
  and flushes `: connected` and only then subscribes to the event broker. A
  status change or recorded run published in between is lost for that
  client. The client reloads status and runs over REST when the stream
  opens, but that request can also complete before the subscription exists,
  so the gap is real, if short. This is the cause of the intermittent
  failure of `TestNewRouterDoesNotMutateStoreAndStreamsBrokerEvents`
  (30 s timeout in CI, Dependabot PR #4): the test publishes right after
  reading `: connected`.
- **Shutdown timing test.**
  `TestRunShutsDownQuicklyWithOpenSSEClientAndSavesHistory` requires the
  shutdown to finish in under 3 s with a 2 s budget; on a loaded machine it
  took 3.45 s (block 0059). The test is meant to prove that shutdown neither
  waits for the open SSE stream nor for the 30 s action.

Out of scope:

- Replaying missed events (`Last-Event-ID`), which ADR-0008 does not require.
- Other timing tests that have not failed.

## Approval

- **Approved by**: pending
- **Approval date**: pending
- **Decision notes**: —

## Proposed solution

1. In the SSE handler, call `eventSource.Subscribe()` (and defer
   `unsubscribe`) **before** writing and flushing `: connected`. Once a
   client has seen `: connected`, every later event reaches it. Document
   this guarantee in the handler comment and in the ADR-0008 addendum
   ("the connected comment is sent after the subscription exists").
2. Shutdown test: keep the 2 s budget, but compare the elapsed time with a
   limit derived from it that is still far below the 30 s action, e.g.
   `ShutdownTimeout + 3 s`, with a comment explaining what the limit
   proves. Keep the assertions on the closed stream and the persisted
   canceled run.

## Test plan

- Unit test in `internal/server`: with a fake `EventSource`, the handler
  subscribes before the first byte of the body is flushed (the fake records
  the order of `Subscribe` and the first write). It fails on the current
  order.
- `go test -race -count=200 -run TestNewRouterDoesNotMutateStoreAndStreamsBrokerEvents ./internal/server`
  passes; the same under CPU load (e.g. `stress`-like background load or
  `-cpu 1,2,4`).
- `go test -race -count=20 -run TestRunShutsDownQuickly ./cmd/marionette`
  passes under the same load.
- `make verify` and `make e2e` pass.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md), plus:

- Dependabot PR #4 CI passes after a rebase on the fix.

## Closure

- **Status after implementation**: —
- **Verification**: —
- **Documentation updated**: —
