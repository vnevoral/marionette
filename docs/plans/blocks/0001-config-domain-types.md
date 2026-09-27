# Implementation block: Domain types and configuration validation

- **Phase**: 2 — Domain model + config store (in-memory + JSON persistence)
- **Requirements**: FR-10, FR-11, FR-12, FR-13, FR-14, FR-15, FR-15a, FR-17, FR-18, FR-19
- **ADRs**: ADR-0004
- **Status**: Done

## Goal

When done, there is an `internal/config` package with domain types
(`Action`, `OutputRule`, `ActionCard`, `Run`, `Settings`) and their
validation, without any in-memory management (store) or disk I/O — that is
handled by blocks 0002 and 0003.

## Scope

- **In scope**:
  - Types `Action`, `OutputRule`, `ActionCard`, `Run`, `Settings` in
    `internal/config`, including JSON tags for future (de)serialization.
  - `OutputRule.Type` as `"exit_code" | "match" | "not_match"` (default
    `"exit_code"` for an empty value); for `match`/`not_match` a mandatory
    `Pattern` valid as a `regexp`.
  - A validation method (e.g. `(ActionCard) Validate() error` and similarly
    for `Action`, `Settings`) covering: mandatory fields (id, name, primary
    action, command), positive timeout, compilability of the regex in
    `OutputRule`, `Settings.HistorySize > 0`,
    `Settings.MaxConcurrentActions > 0`, non-negative
    `PollingIntervalSeconds`/`FastPollingIntervalSeconds`/
    `FastPollingWindowSeconds` and `FastPollingIntervalSeconds <
PollingIntervalSeconds` if standard polling is enabled.
  - Constant default values (`DefaultHistorySize = 20`,
    `DefaultMaxConcurrentActions = 4`, `DefaultPollingIntervalSeconds = 60`,
    `DefaultFastPollingIntervalSeconds = 10`,
    `DefaultFastPollingWindowSeconds = 120`) as recommended values for
    newly created cards (set by the caller, e.g. the API layer in Phase 5),
    not enforced inside `Validate()`.
- **Out of scope** (handled by other blocks): in-memory store/CRUD (0002),
  JSON load/save and atomic write (0003), command execution (Phase 3), HTTP
  API (Phase 5).

## Proposed solution

```go
package config

type OutputRuleType string

const (
    OutputRuleExitCode OutputRuleType = "exit_code"
    OutputRuleMatch    OutputRuleType = "match"
    OutputRuleNotMatch OutputRuleType = "not_match"
)

type OutputRule struct {
    Type    OutputRuleType `json:"type"`
    Pattern string         `json:"pattern,omitempty"`
}

type Action struct {
    Command    string            `json:"command"`
    Args       []string          `json:"args,omitempty"`
    Dir        string            `json:"dir,omitempty"`
    Env        map[string]string `json:"env,omitempty"`
    TimeoutSec int               `json:"timeoutSec"`
    Rule       OutputRule        `json:"rule"`
}

type ActionCard struct {
    ID                          string  `json:"id"`
    Name                        string  `json:"name"`
    Description                 string  `json:"description,omitempty"`
    Icon                        string  `json:"icon,omitempty"`
    Primary                     Action  `json:"primary"`
    Status                      *Action `json:"status,omitempty"`
    PollingIntervalSeconds      int     `json:"pollingIntervalSeconds,omitempty"`
    FastPollingIntervalSeconds  int     `json:"fastPollingIntervalSeconds,omitempty"`
    FastPollingWindowSeconds    int     `json:"fastPollingWindowSeconds,omitempty"`
}

type RunOutcome string

const (
    RunOutcomeOK      RunOutcome = "ok"
    RunOutcomeFail    RunOutcome = "fail"
    RunOutcomeTimeout RunOutcome = "timeout"
)

type Run struct {
    ActionKind string        `json:"actionKind"` // "primary" | "status"
    StartedAt  time.Time     `json:"startedAt"`
    Duration   time.Duration `json:"duration"`
    ExitCode   int           `json:"exitCode"`
    Output     string        `json:"output"`
    Truncated  bool          `json:"truncated"`
    Outcome    RunOutcome    `json:"outcome"`
}

type Settings struct {
    HistorySize           int `json:"historySize"`
    MaxConcurrentActions  int `json:"maxConcurrentActions"`
}
```

The implementation may slightly adjust the exact names/layout of the fields,
but it must stay compatible with the JSON structure described in ADR-0004
(`settings` + `cards` at the top level, see block 0003).
`FastPollingIntervalSeconds`/`FastPollingWindowSeconds` only take effect
when `PollingIntervalSeconds > 0` (FR-15a) — validation does not enforce
this as an error, it only documents the behavior for the scheduler
(Phase 4). `RunOutcome` has three states (`ok`/`fail`/`timeout`) instead of
a boolean `Success`, so that the history can distinguish enforced
termination (FR-19) from an ordinary failure.

## Test plan

- `ActionCard` validation: missing name/primary action/command → error.
- `OutputRule`: `match`/`not_match` without a pattern → error; invalid
  regex → error; `exit_code` without a pattern → OK.
- `Settings`: zero/negative `HistorySize`/`MaxConcurrentActions` → error;
  positive values → OK.
- Polling fields: negative values → error; `FastPollingIntervalSeconds >=
PollingIntervalSeconds` with standard polling enabled → error;
  `PollingIntervalSeconds == 0` with any fast-polling values → OK
  (no effect, see FR-15a).
- A valid minimal `ActionCard` (only a primary action, no status action, no
  polling) → OK.

## Done criteria

See [Definition of Done](../../devops/definition-of-done.md). Specifically:
`go build ./...`, `go vet ./...`, `go test ./...` pass; the new package has
no side effects (no I/O, no global mutable state).
