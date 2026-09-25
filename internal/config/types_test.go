package config

import (
	"strings"
	"testing"
)

func validAction() Action {
	return Action{
		Command:    "echo",
		TimeoutSec: 10,
		Rule:       OutputRule{Type: OutputRuleExitCode},
	}
}

func validCard() ActionCard {
	return ActionCard{
		ID:      "card-1",
		Name:    "Test card",
		Primary: validAction(),
	}
}

func validSettings() Settings {
	return Settings{
		HistorySize:          DefaultHistorySize,
		MaxConcurrentActions: DefaultMaxConcurrentActions,
	}
}

func TestActionCardValidate(t *testing.T) {
	tests := []struct {
		name    string
		card    ActionCard
		wantErr string
	}{
		{name: "valid minimum", card: validCard()},
		{name: "missing id", card: func() ActionCard { card := validCard(); card.ID = ""; return card }(), wantErr: "id"},
		{name: "missing name", card: func() ActionCard { card := validCard(); card.Name = ""; return card }(), wantErr: "name"},
		{name: "missing primary command", card: func() ActionCard { card := validCard(); card.Primary.Command = ""; return card }(), wantErr: "command"},
		{name: "invalid status action", card: func() ActionCard { card := validCard(); card.Status = &Action{}; return card }(), wantErr: "status"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.card.Validate()
			if test.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("Validate() error = %v, want substring %q", err, test.wantErr)
			}
		})
	}
}

func TestActionValidateTimeoutAndRule(t *testing.T) {
	tests := []struct {
		name    string
		action  Action
		wantErr string
	}{
		{name: "valid", action: validAction()},
		{name: "zero timeout", action: func() Action { action := validAction(); action.TimeoutSec = 0; return action }(), wantErr: "timeout"},
		{name: "negative timeout", action: func() Action { action := validAction(); action.TimeoutSec = -1; return action }(), wantErr: "timeout"},
		{name: "missing command", action: Action{TimeoutSec: 1}, wantErr: "command"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.action.Validate()
			if test.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("Validate() error = %v, want substring %q", err, test.wantErr)
			}
		})
	}
}

func TestOutputRuleValidate(t *testing.T) {
	tests := []struct {
		name    string
		rule    OutputRule
		wantErr bool
	}{
		{name: "empty type defaults to exit code", rule: OutputRule{}},
		{name: "exit code ignores pattern", rule: OutputRule{Type: OutputRuleExitCode, Pattern: "["}},
		{name: "match valid", rule: OutputRule{Type: OutputRuleMatch, Pattern: `ready`}},
		{name: "not match valid", rule: OutputRule{Type: OutputRuleNotMatch, Pattern: `ready`}},
		{name: "match missing pattern", rule: OutputRule{Type: OutputRuleMatch}, wantErr: true},
		{name: "not match missing pattern", rule: OutputRule{Type: OutputRuleNotMatch}, wantErr: true},
		{name: "invalid regex", rule: OutputRule{Type: OutputRuleMatch, Pattern: "["}, wantErr: true},
		{name: "unsupported type", rule: OutputRule{Type: OutputRuleType("other")}, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.rule.Validate()
			if (err != nil) != test.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %t", err, test.wantErr)
			}
		})
	}
}

func TestSettingsValidate(t *testing.T) {
	tests := []struct {
		name     string
		settings Settings
		wantErr  bool
	}{
		{name: "valid", settings: validSettings()},
		{name: "zero history", settings: func() Settings { settings := validSettings(); settings.HistorySize = 0; return settings }(), wantErr: true},
		{name: "negative history", settings: func() Settings { settings := validSettings(); settings.HistorySize = -1; return settings }(), wantErr: true},
		{name: "zero concurrency", settings: func() Settings { settings := validSettings(); settings.MaxConcurrentActions = 0; return settings }(), wantErr: true},
		{name: "negative concurrency", settings: func() Settings { settings := validSettings(); settings.MaxConcurrentActions = -1; return settings }(), wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.settings.Validate()
			if (err != nil) != test.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %t", err, test.wantErr)
			}
		})
	}
}

func TestActionCardValidatePolling(t *testing.T) {
	tests := []struct {
		name    string
		card    ActionCard
		wantErr bool
	}{
		{name: "polling disabled allows fast fields", card: func() ActionCard {
			card := validCard()
			card.FastPollingIntervalSeconds = 30
			card.FastPollingWindowSeconds = 120
			return card
		}()},
		{name: "valid polling", card: func() ActionCard {
			card := validCard()
			card.PollingIntervalSeconds = 60
			card.FastPollingIntervalSeconds = 10
			card.FastPollingWindowSeconds = 120
			return card
		}()},
		{name: "negative standard interval", card: func() ActionCard { card := validCard(); card.PollingIntervalSeconds = -1; return card }(), wantErr: true},
		{name: "negative fast interval", card: func() ActionCard { card := validCard(); card.FastPollingIntervalSeconds = -1; return card }(), wantErr: true},
		{name: "negative fast window", card: func() ActionCard { card := validCard(); card.FastPollingWindowSeconds = -1; return card }(), wantErr: true},
		{name: "fast interval equal to standard", card: func() ActionCard {
			card := validCard()
			card.PollingIntervalSeconds = 10
			card.FastPollingIntervalSeconds = 10
			return card
		}(), wantErr: true},
		{name: "fast interval greater than standard", card: func() ActionCard {
			card := validCard()
			card.PollingIntervalSeconds = 10
			card.FastPollingIntervalSeconds = 11
			return card
		}(), wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.card.Validate()
			if (err != nil) != test.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %t", err, test.wantErr)
			}
		})
	}
}
