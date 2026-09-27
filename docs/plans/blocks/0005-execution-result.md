# Implementation block: Output capture and result evaluation

- **Phase**: 3 — Execution engine
- **Requirements**: FR-13, FR-14, FR-17
- **ADRs**: ADR-0005
- **Status**: Done

## Goal

When done, the execution engine builds a complete `config.Run` from a
single execution: it captures combined stdout/stderr up to 4 KB, marks
truncation and evaluates the exit code or the `OutputRule`.

## Scope

- **In scope**:
  - a shared stdout+stderr limit of 4096 bytes;
  - `Run.Output` and `Run.Truncated`;
  - `exit_code`, `match` and `not_match` rules;
  - the result `ok`, `fail` or `timeout`;
  - capturing the exit code even for a failed process;
  - a testable pure function for evaluating the result.
- **Out of scope**:
  - process execution and timeout (0004);
  - global concurrency (0006);
  - storing history in the store, polling and the HTTP API.

## Proposed solution

Extend `internal/exec` with a limited writer and a result evaluator. The
evaluator will not recompile the regex on every run; the configuration is
validated on save and load. The output is evaluated after truncation, in
line with ADR-0004.

## Test plan

- exit code 0/non-zero;
- valid `match` and `not_match`, including empty and non-matching output;
- exactly 4096 bytes and output over the limit;
- combined stdout/stderr with the limit;
- a timeout always has `RunOutcomeTimeout`, even if the process returned a different exit code;
- an invalid rule is rejected before execution.

## Done criteria

See Definition of Done. Specifically: the evaluator unit tests cover the
main and edge scenarios and `go test -race ./...` passes.
