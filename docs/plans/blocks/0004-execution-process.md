# Implementation block: Process launch and enforced timeout

- **Phase**: 3 — Execution engine
- **Requirements**: FR-11, FR-19, NFR-01, NFR-04
- **ADRs**: ADR-0005
- **Status**: Done

## Goal

When done, the foundation of the execution engine exists in
`internal/exec`; it runs a single valid `config.Action` without a shell,
sets the working directory and environment, and forcibly terminates the
process after the timeout. The block does not yet handle the global queue
or polling.

## Scope

- **In scope**:
  - a public executor API for a single action run;
  - `os/exec.CommandContext` with structured arguments;
  - `Action.Dir` and `Action.Env`;
  - `context.WithTimeout` based on `Action.TimeoutSec`;
  - distinguishing completion, process error and timeout;
  - keeping the start time and duration in the internal result.
- **Out of scope**:
  - output limit and regex/exit-code evaluation (0005);
  - global semaphore/queue (0006);
  - writing `Run` to the config store, HTTP API and polling.

## Proposed solution

A new package `internal/exec` with the package name `execengine`. The
executor will have a small testable abstraction over process creation.
The result will be a structure that the next block converts to
`config.Run`; a timeout will be marked unambiguously.

## Test plan

- a valid command with an argument and a working directory;
- passing an environment variable;
- a nonexistent command ends as a process error without crashing the
  server;
- a long command ends after the timeout and does not keep hanging;
- duration and timeout are deterministically distinguishable;
- unit tests use a fake factory, the timeout integration test uses a
  small real command.

## Done criteria

See Definition of Done. Specifically: `go test -race ./...` and
verification that no child process is left running after the timeout.
