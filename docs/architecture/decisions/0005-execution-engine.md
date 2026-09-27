# ADR-0005: Safe action execution and concurrency limit

- **Status**: Accepted
- **Date**: 2026-09-25

## Context

Phase 3 must run the commands defined in `config.Action` on the host
without shell interpolation, with a mandatory timeout, output limiting and
a global limit on concurrent processes. The result is then stored as a
`config.Run`.

## Decision

- The execution engine will live in the `internal/exec` package (the Go
  package name will be `execengine` so it is not confused with the
  standard `os/exec`).
- The command is run via `os/exec.CommandContext(ctx, action.Command,
action.Args...)`; `Dir` and `Env` are set directly on `exec.Cmd`. No
  shell and no command-string assembly will be used.
- Each run creates a `context.WithTimeout` from `Action.TimeoutSec`.
  When the timeout expires, the process is terminated via `CommandContext`
  and the result is `RunOutcomeTimeout`; the process must not keep running
  in the background.
- Stdout and stderr are captured into one shared limited writer of
  4096 bytes. Exceeding the limit sets `Run.Truncated` but does not by
  itself cause a run error.
- Exit code, output, start time and duration are part of the result.
  A failure to start or run the process is converted to `RunOutcomeFail`;
  an API error is reserved for invalid input or an internal engine error.
- The global concurrency limit is handled by a buffered-channel
  semaphore. Every manual and future polling run must acquire a slot and
  release it when finished; actions over the limit wait, they are not
  dropped.
- The engine will depend on a narrow interface/factory for creating the
  process, so that unit tests do not have to run real commands. A small
  number of integration tests will verify a real `echo` and a timeout.

## Consequences

- User input is never interpreted as a shell program.
- Timeout and output limit are enforced in one place for both primary and
  status actions.
- The engine will not deal with card configuration, polling or the HTTP
  API; those belong to the config store, phase 4 and phase 5.

> Added 2026-09-26 (block 0024): termination after a timeout kills the
> action's whole **process group** (`Setpgid` + `SIGKILL` on `-pgid`), not
> just the direct child, and `Cmd.WaitDelay` (2 s) bounds the wait for the
> output pipe held by any surviving processes to close. `Executor.Execute`
> and `Runner.Run` accept the caller's context; canceling it (shutdown,
> card reconfiguration) terminates the process with the result
> `RunOutcomeCanceled`, which differs from `RunOutcomeTimeout` in that it
> was not caused by the action's timeout. A status check canceled by the
> caller does not change the card's last known status.

> Added 2026-09-26 (block 0034): the package directory was renamed to
> `internal/execengine` to match the package name. The spawned process
> **does not inherit the service environment**: it gets only `PATH`,
> `HOME`, `LANG` and `TZ` (if set) and the variables from `Action.Env`,
> which take precedence (NFR-01 c). Impact on users: variables defined in
> `/etc/default/marionette` or in the systemd unit are not available to
> actions; whatever an action needs must be in the card's `env`.
