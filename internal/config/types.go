// Package config contains the domain model for action cards and their runs.
package config

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Default values recommended for newly created action cards and settings.
const (
	DefaultHistorySize                = 20
	DefaultMaxConcurrentActions       = 4
	DefaultPollingIntervalSeconds     = 60
	DefaultFastPollingIntervalSeconds = 10
	DefaultFastPollingWindowSeconds   = 120
)

// Limits applied by Validate to user-supplied values (block 0028). They keep
// a single card small enough for the JSON configuration file and the UI, and
// keep one action from occupying an execution slot indefinitely.
const (
	MaxCardIDLength      = 64
	MaxNameLength        = 120
	MaxDescriptionLength = 2000
	MaxIconLength        = 64
	MaxCommandLength     = 512
	MaxArgs              = 64
	MaxArgLength         = 1024
	MaxDirLength         = 1024
	MaxEnvEntries        = 64
	MaxEnvKeyLength      = 128
	MaxEnvValueLength    = 4096
	MaxTimeoutSec        = 3600
	// IconPrefix is the PrimeIcons class prefix accepted for card icons.
	IconPrefix = "pi pi-"
)

var (
	cardIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
	envKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
)

// ValidationError reports every invalid field of a value at once. Keys are
// JSON paths relative to the validated value (for example
// "primary.timeoutSec") and values are human-readable reasons. It matches
// ErrValidation in errors.Is.
type ValidationError struct {
	Fields map[string]string
}

func (err *ValidationError) Error() string {
	keys := make([]string, 0, len(err.Fields))
	for key := range err.Fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+": "+err.Fields[key])
	}
	return "validation failed: " + strings.Join(parts, "; ")
}

// Is makes errors.Is(err, ErrValidation) true for every ValidationError.
func (err *ValidationError) Is(target error) bool {
	return target == ErrValidation
}

// fieldErrors accumulates validation failures keyed by field path.
type fieldErrors map[string]string

func (fields fieldErrors) add(key, reason string) {
	if _, exists := fields[key]; !exists {
		fields[key] = reason
	}
}

// merge copies nested failures under a prefix ("primary" + "command" →
// "primary.command").
func (fields fieldErrors) merge(prefix string, err error) {
	if err == nil {
		return
	}
	var validation *ValidationError
	if !errorsAs(err, &validation) {
		fields.add(prefix, err.Error())
		return
	}
	for key, reason := range validation.Fields {
		fields.add(prefix+"."+key, reason)
	}
}

func (fields fieldErrors) err() error {
	if len(fields) == 0 {
		return nil
	}
	return &ValidationError{Fields: map[string]string(fields)}
}

func errorsAs(err error, target **ValidationError) bool {
	return errors.As(err, target)
}

func lengthReason(limit int) string {
	return fmt.Sprintf("must be at most %d characters", limit)
}

// OutputRuleType describes how an action result is evaluated.
type OutputRuleType string

const (
	// OutputRuleExitCode evaluates an action by its exit code.
	OutputRuleExitCode OutputRuleType = "exit_code"
	// OutputRuleMatch succeeds when output matches Pattern.
	OutputRuleMatch OutputRuleType = "match"
	// OutputRuleNotMatch succeeds when output does not match Pattern.
	OutputRuleNotMatch OutputRuleType = "not_match"
)

// OutputRule defines the rule used to evaluate an action result.
type OutputRule struct {
	Type    OutputRuleType `json:"type"`
	Pattern string         `json:"pattern,omitempty"`
}

// Validate checks that the output rule has a supported type and valid pattern.
// Failures are reported as a ValidationError with the keys "type" and "pattern".
func (rule OutputRule) Validate() error {
	fields := fieldErrors{}
	ruleType := rule.Type
	if ruleType == "" {
		ruleType = OutputRuleExitCode
	}

	switch ruleType {
	case OutputRuleExitCode:
	case OutputRuleMatch, OutputRuleNotMatch:
		if rule.Pattern == "" {
			fields.add("pattern", fmt.Sprintf("output rule pattern is required for type %q", ruleType))
		} else if _, err := regexp.Compile(rule.Pattern); err != nil {
			fields.add("pattern", fmt.Sprintf("output rule pattern is invalid: %v", err))
		}
	default:
		fields.add("type", fmt.Sprintf("unsupported output rule type %q", rule.Type))
	}
	return fields.err()
}

// Action defines one command that can be executed by the execution engine.
type Action struct {
	Command    string            `json:"command"`
	Args       []string          `json:"args,omitempty"`
	Dir        string            `json:"dir,omitempty"`
	Env        map[string]string `json:"env,omitempty"`
	TimeoutSec int               `json:"timeoutSec"`
	Rule       OutputRule        `json:"rule"`
}

// Validate checks that an action can be executed safely by the execution
// engine and that its values respect the documented limits. All failures are
// collected into one ValidationError keyed by field ("command", "args[2]",
// "env.KEY", "timeoutSec", "rule.pattern", ...).
func (action Action) Validate() error {
	fields := fieldErrors{}
	switch {
	case strings.TrimSpace(action.Command) == "":
		fields.add("command", "action command is required")
	case len(action.Command) > MaxCommandLength:
		fields.add("command", lengthReason(MaxCommandLength))
	}
	if len(action.Args) > MaxArgs {
		fields.add("args", fmt.Sprintf("must have at most %d items", MaxArgs))
	}
	for index, arg := range action.Args {
		if len(arg) > MaxArgLength {
			fields.add(fmt.Sprintf("args[%d]", index), lengthReason(MaxArgLength))
		}
	}
	if len(action.Dir) > MaxDirLength {
		fields.add("dir", lengthReason(MaxDirLength))
	}
	if len(action.Env) > MaxEnvEntries {
		fields.add("env", fmt.Sprintf("must have at most %d entries", MaxEnvEntries))
	}
	for key, value := range action.Env {
		switch {
		case !envKeyPattern.MatchString(key):
			fields.add("env."+key, "environment variable name must match ^[A-Za-z_][A-Za-z0-9_]*$")
		case len(key) > MaxEnvKeyLength:
			fields.add("env."+key, "environment variable name "+lengthReason(MaxEnvKeyLength))
		case len(value) > MaxEnvValueLength:
			fields.add("env."+key, lengthReason(MaxEnvValueLength))
		}
	}
	if action.TimeoutSec < 1 || action.TimeoutSec > MaxTimeoutSec {
		fields.add("timeoutSec", fmt.Sprintf("must be between 1 and %d", MaxTimeoutSec))
	}
	fields.merge("rule", action.Rule.Validate())
	return fields.err()
}

// ActionCard groups a primary action with an optional status action and polling settings.
type ActionCard struct {
	ID                         string  `json:"id"`
	Name                       string  `json:"name"`
	Description                string  `json:"description,omitempty"`
	Icon                       string  `json:"icon,omitempty"`
	Primary                    Action  `json:"primary"`
	Status                     *Action `json:"status,omitempty"`
	PollingIntervalSeconds     int     `json:"pollingIntervalSeconds,omitempty"`
	FastPollingIntervalSeconds int     `json:"fastPollingIntervalSeconds,omitempty"`
	FastPollingWindowSeconds   int     `json:"fastPollingWindowSeconds,omitempty"`
}

// Validate checks the card identity, actions, and polling configuration and
// reports every failure at once as a ValidationError keyed by JSON path.
func (card ActionCard) Validate() error {
	fields := fieldErrors{}
	switch {
	case strings.TrimSpace(card.ID) == "":
		fields.add("id", "action card id is required")
	case len(card.ID) > MaxCardIDLength:
		fields.add("id", lengthReason(MaxCardIDLength))
	case !cardIDPattern.MatchString(card.ID):
		fields.add("id", "must contain only letters, digits, '-' and '_'")
	}
	switch {
	case strings.TrimSpace(card.Name) == "":
		fields.add("name", "action card name is required")
	case len(card.Name) > MaxNameLength:
		fields.add("name", lengthReason(MaxNameLength))
	}
	if len(card.Description) > MaxDescriptionLength {
		fields.add("description", lengthReason(MaxDescriptionLength))
	}
	if !ValidIcon(card.Icon) {
		fields.add("icon", fmt.Sprintf("must be empty or a %q class of at most %d characters", IconPrefix, MaxIconLength))
	}
	fields.merge("primary", card.Primary.Validate())
	if card.Status != nil {
		fields.merge("status", card.Status.Validate())
	}
	if card.PollingIntervalSeconds < 0 {
		fields.add("pollingIntervalSeconds", "polling interval cannot be negative")
	}
	if card.FastPollingIntervalSeconds < 0 {
		fields.add("fastPollingIntervalSeconds", "fast polling interval cannot be negative")
	}
	if card.FastPollingWindowSeconds < 0 {
		fields.add("fastPollingWindowSeconds", "fast polling window cannot be negative")
	}
	if card.PollingIntervalSeconds > 0 && card.FastPollingIntervalSeconds >= card.PollingIntervalSeconds {
		fields.add("fastPollingIntervalSeconds", "fast polling interval must be less than polling interval")
	}
	return fields.err()
}

// ValidIcon reports whether an icon value is empty or a PrimeIcons class
// ("pi pi-<name>") within the length limit. The name part may contain only
// lowercase letters, digits and dashes.
func ValidIcon(icon string) bool {
	if icon == "" {
		return true
	}
	if len(icon) > MaxIconLength || !strings.HasPrefix(icon, IconPrefix) {
		return false
	}
	name := strings.TrimPrefix(icon, IconPrefix)
	if name == "" {
		return false
	}
	for _, r := range name {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
			return false
		}
	}
	return true
}

// RunOutcome describes the result of one action execution.
type RunOutcome string

const (
	// RunOutcomeOK indicates successful execution.
	RunOutcomeOK RunOutcome = "ok"
	// RunOutcomeFail indicates execution completed unsuccessfully.
	RunOutcomeFail RunOutcome = "fail"
	// RunOutcomeTimeout indicates execution was forcibly terminated after timeout.
	RunOutcomeTimeout RunOutcome = "timeout"
	// RunOutcomeCanceled indicates the caller canceled execution (for example
	// during shutdown or card reconfiguration) before the action finished.
	RunOutcomeCanceled RunOutcome = "canceled"
)

// Run records one execution of a primary or status action.
type Run struct {
	ActionKind string        `json:"actionKind"`
	StartedAt  time.Time     `json:"startedAt"`
	Duration   time.Duration `json:"duration"`
	ExitCode   int           `json:"exitCode"`
	Output     string        `json:"output"`
	Truncated  bool          `json:"truncated"`
	Outcome    RunOutcome    `json:"outcome"`
}

// StatusState is the interpreted state of a card's status action.
type StatusState string

const (
	// StatusStateUnknown means that no valid status result is currently known.
	StatusStateUnknown StatusState = "unknown"
	// StatusStateOK means that the status action reports a healthy state.
	StatusStateOK StatusState = "ok"
	// StatusStateFail means that the status action reports an unhealthy state.
	StatusStateFail StatusState = "fail"
)

// StatusChange records one interval in a card status state history.
type StatusChange struct {
	State     StatusState   `json:"state"`
	StartedAt time.Time     `json:"startedAt"`
	EndedAt   *time.Time    `json:"endedAt,omitempty"`
	Duration  time.Duration `json:"duration"`
}

// StatusSnapshot contains the latest status check and its interpreted state.
type StatusSnapshot struct {
	State     StatusState `json:"state"`
	CheckedAt time.Time   `json:"checkedAt"`
	LastCheck Run         `json:"lastCheck"`
}

// Settings contains global configuration for history and action concurrency.
type Settings struct {
	HistorySize          int `json:"historySize"`
	MaxConcurrentActions int `json:"maxConcurrentActions"`
}

// Validate checks that global settings use positive limits.
func (settings Settings) Validate() error {
	fields := fieldErrors{}
	if settings.HistorySize <= 0 {
		fields.add("historySize", "history size must be positive")
	}
	if settings.MaxConcurrentActions <= 0 {
		fields.add("maxConcurrentActions", "max concurrent actions must be positive")
	}
	return fields.err()
}
