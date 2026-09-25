// Package config contains the domain model for action cards and their runs.
package config

import (
	"fmt"
	"regexp"
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
func (rule OutputRule) Validate() error {
	ruleType := rule.Type
	if ruleType == "" {
		ruleType = OutputRuleExitCode
	}

	switch ruleType {
	case OutputRuleExitCode:
		return nil
	case OutputRuleMatch, OutputRuleNotMatch:
		if rule.Pattern == "" {
			return fmt.Errorf("output rule pattern is required for type %q", ruleType)
		}
		if _, err := regexp.Compile(rule.Pattern); err != nil {
			return fmt.Errorf("output rule pattern is invalid: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("unsupported output rule type %q", rule.Type)
	}
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

// Validate checks that an action can be executed safely by the execution engine.
func (action Action) Validate() error {
	if strings.TrimSpace(action.Command) == "" {
		return fmt.Errorf("action command is required")
	}
	if action.TimeoutSec <= 0 {
		return fmt.Errorf("action timeout must be positive")
	}
	if err := action.Rule.Validate(); err != nil {
		return fmt.Errorf("invalid action rule: %w", err)
	}
	return nil
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

// Validate checks the card identity, actions, and polling configuration.
func (card ActionCard) Validate() error {
	if strings.TrimSpace(card.ID) == "" {
		return fmt.Errorf("action card id is required")
	}
	if strings.TrimSpace(card.Name) == "" {
		return fmt.Errorf("action card name is required")
	}
	if err := card.Primary.Validate(); err != nil {
		return fmt.Errorf("invalid primary action: %w", err)
	}
	if card.Status != nil {
		if err := card.Status.Validate(); err != nil {
			return fmt.Errorf("invalid status action: %w", err)
		}
	}
	if card.PollingIntervalSeconds < 0 {
		return fmt.Errorf("polling interval cannot be negative")
	}
	if card.FastPollingIntervalSeconds < 0 {
		return fmt.Errorf("fast polling interval cannot be negative")
	}
	if card.FastPollingWindowSeconds < 0 {
		return fmt.Errorf("fast polling window cannot be negative")
	}
	if card.PollingIntervalSeconds > 0 && card.FastPollingIntervalSeconds >= card.PollingIntervalSeconds {
		return fmt.Errorf("fast polling interval must be less than polling interval")
	}
	return nil
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

// Settings contains global configuration for history and action concurrency.
type Settings struct {
	HistorySize          int `json:"historySize"`
	MaxConcurrentActions int `json:"maxConcurrentActions"`
}

// Validate checks that global settings use positive limits.
func (settings Settings) Validate() error {
	if settings.HistorySize <= 0 {
		return fmt.Errorf("history size must be positive")
	}
	if settings.MaxConcurrentActions <= 0 {
		return fmt.Errorf("max concurrent actions must be positive")
	}
	return nil
}
