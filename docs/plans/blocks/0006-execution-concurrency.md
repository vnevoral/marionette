# Implementation block: Global limit of concurrent actions

- **Phase**: 3 — Execution engine
- **Requirements**: FR-18, NFR-07
- **ADRs**: ADR-0005
- **Status**: Done

## Goal

Once complete, all execution runs share one global semaphore according to
`Settings.MaxConcurrentActions`. Actions over the limit wait in a queue and
continue once a slot is released; none is silently dropped.

## Scope

- **In scope**:
  - a concurrency-limited runner on top of the executor API from blocks
    0004–0005;
  - a slot acquired before the process starts and released on all
    completion/error paths;
  - limit configuration from `config.Settings`;
  - safe change of the limit by creating a new runner instance for the new
    setting;
  - tests of completion order, waiting and slot release after a timeout.
- **Out of scope**:
  - polling and the fast-polling window (phase 4);
  - prioritization or cancellation of waiting actions;
  - HTTP API and writing results to the history.

## Proposed solution

Use a buffered channel as the semaphore and `defer` to return the slot. The
public API keeps the option of synchronous execution; a call over the limit
blocks instead of being silently dropped. The limit will be checked when the
runner is created, rejecting an invalid setting.

## Test plan

- with a limit of 1 at most one action runs at a time;
- the second action waits and then runs;
- both a timeout and a process error always release the slot;
- multiple concurrent callers do not lose any run;
- test with `go test -race` and a fake executor without real processes.

## Done criteria

See Definition of Done. Specifically: `go test -race ./...` passes and a
test proves that the global limit applies to all runners sharing one
configuration.
