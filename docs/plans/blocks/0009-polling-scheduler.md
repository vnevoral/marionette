# Implementation block: Standard and fast polling scheduler

- **Phase**: 4 — Status/health-check engine
- **Requirements**: FR-15, FR-15a, FR-17, FR-18
- **ADRs**: ADR-0006
- **Status**: Done

## Goal

After this block, the scheduler regularly runs status checks for cards
with standard polling enabled and temporarily uses the fast interval after
a primary action. Once the fast window has elapsed, it returns to the
standard interval.

## Scope

- **In scope**:
  - start/stop lifecycle of the scheduler via `context.Context`;
  - per-card standard interval (`PollingIntervalSeconds`);
  - activating the fast window after a programmatic `NotifyPrimaryAction(cardID)`;
  - fast interval and return after `FastPollingWindowSeconds`;
  - ignoring the fast settings when standard polling is disabled;
  - wiring to `StatusCheckService` and the shared execution runner;
  - safely adding/removing cards when the scheduler restarts.
- **Out of scope**:
  - the evaluation of the status result itself (0008);
  - API endpoints and UI;
  - a persistent queue of polling jobs or prioritization;
  - storing history for every tick.

## Proposed solution

A new scheduler in `internal/status` will have one lifecycle context and a
timer for each active card. `NotifyPrimaryAction` sets the end of the fast
window for the given card; the next scheduling uses the fast interval.
After the window ends, the timer returns to the standard interval. Checks
go through the same runner as manual runs, so the global concurrency
limit applies.

## Test plan

- a card with polling runs checks at the configured standard interval;
- a card with polling 0 runs no checks;
- the fast window after a primary action uses the fast interval, and the
  standard one after it expires;
- a fast configuration without standard polling has no effect;
- stop ends the timers and creates no further checks;
- multiple cards have independent intervals and share the runner limit;
- fake clock/check service, without waiting for real tens of seconds.

## Done criteria

See Definition of Done. Specifically: `go test -race ./...` passes and the
tests verify the standard interval, the fast window and the return to
standard polling.
