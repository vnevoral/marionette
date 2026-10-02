# Implementation block: Flaky shutdown in the paired-device integration test

- **Phase**: 8 — Hardening (addendum after the phase was closed), tests
- **Requirements**: NFR-04, NFR-05
- **ADRs**: no new ADR needed
- **Status**: Done
- **Dependencies**: Block 0043 (device pairing), block 0061 (reliable timing tests)

## Goal

The CI run for `08c94b6` on `main` (2026-09-29) failed in
`TestRunRequiresAPairedDevice` with `run() error = stop HTTP server:
context deadline exceeded`. Locally the test failed about once in 100 runs
with `-race`. After this block the test is deterministic; the service
itself does not change.

## Cause

Logging `http.Server.ConnState` in the failing runs showed a second client
connection that stays in `StateNew`: the shared `http.DefaultTransport`
occasionally dials a spare connection that never sends a request.
`http.Server.Shutdown` treats a `StateNew` connection as idle only after
5 s, which is longer than the test's 2 s `ShutdownTimeout`, so the stop
step ran out of time. In production the shutdown budget is 20 s, so such
a connection only delays shutdown by up to 5 s.

## Scope

- **In scope**: the test uses its own `http.Transport` for all requests
  and closes its idle connections before cancelling the context.
- **Out of scope**: changing the server shutdown sequence or timeouts;
  `TestRunShutsDownQuicklyWithOpenSSEClientAndSavesHistory`, which does
  not show the problem (100 runs with `-race` passed).

## Approval

- **Approved by**: project owner
- **Approval date**: 2026-10-02
- **Decision notes**: Requested by the owner: check the last CI run.

## Test plan

- `go test -race -count=300 -run TestRunRequiresAPairedDevice ./cmd/marionette/`.
- `make verify`.

## Closure

- **Status after implementation**: Done (2026-10-02)
- **Verification**: before the fix 2 of 200 runs failed with the CI
  error; after the fix 300 of 300 runs passed; `make verify` passed.
- **Deviations from the plan**: none.
- **Documentation updated**: roadmap.
