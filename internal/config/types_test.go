package config

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"
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

func TestValidateCollectsAllFieldsAndMatchesErrValidation(t *testing.T) {
	card := validCard()
	card.ID = "bad id"
	card.Name = ""
	card.Primary.TimeoutSec = 0
	card.Status = &Action{Command: "x", TimeoutSec: 5, Rule: OutputRule{Type: OutputRuleMatch}}
	err := card.Validate()
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("Validate() error = %v, want errors.Is ErrValidation", err)
	}
	var validation *ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("Validate() error = %T, want *ValidationError", err)
	}
	for _, key := range []string{"id", "name", "primary.timeoutSec", "status.rule.pattern"} {
		if validation.Fields[key] == "" {
			t.Errorf("missing field %q in %v", key, validation.Fields)
		}
	}
	if len(validation.Fields) != 4 {
		t.Fatalf("fields = %v, want exactly 4", validation.Fields)
	}
	if message := err.Error(); !strings.HasPrefix(message, "validation failed: id: ") || strings.Contains(message, "Go struct") {
		t.Fatalf("message = %q", message)
	}
}

func TestValidateLimits(t *testing.T) {
	long := func(n int) string { return strings.Repeat("a", n) }
	args := func(n int) []string {
		list := make([]string, n)
		for i := range list {
			list[i] = "x"
		}
		return list
	}
	env := func(n int) map[string]string {
		m := make(map[string]string, n)
		for i := range n {
			m["K"+strconv.Itoa(i)] = "v"
		}
		return m
	}
	cases := []struct {
		name   string
		mutate func(*ActionCard)
		field  string // empty = valid
	}{
		{"id at limit", func(c *ActionCard) { c.ID = long(MaxCardIDLength) }, ""},
		{"id over limit", func(c *ActionCard) { c.ID = long(MaxCardIDLength + 1) }, "id"},
		{"id with slash", func(c *ActionCard) { c.ID = "a/b" }, "id"},
		{"id with space", func(c *ActionCard) { c.ID = "a b" }, "id"},
		{"id with diacritics", func(c *ActionCard) { c.ID = "kartá" }, "id"},
		{"id empty", func(c *ActionCard) { c.ID = "  " }, "id"},
		{"id generated shape", func(c *ActionCard) { c.ID = "card-0123456789abcdef0123456789abcdef" }, ""},
		{"name at limit", func(c *ActionCard) { c.Name = long(MaxNameLength) }, ""},
		{"name over limit", func(c *ActionCard) { c.Name = long(MaxNameLength + 1) }, "name"},
		{"description at limit", func(c *ActionCard) { c.Description = long(MaxDescriptionLength) }, ""},
		{"description over limit", func(c *ActionCard) { c.Description = long(MaxDescriptionLength + 1) }, "description"},
		{"icon known", func(c *ActionCard) { c.Icon = "pi pi-exclamation-triangle" }, ""},
		{"icon wrong prefix", func(c *ActionCard) { c.Icon = "fa fa-home" }, "icon"},
		{"icon empty name", func(c *ActionCard) { c.Icon = "pi pi-" }, "icon"},
		{"icon with markup", func(c *ActionCard) { c.Icon = "pi pi-home\"><script>" }, "icon"},
		{"icon over limit", func(c *ActionCard) { c.Icon = IconPrefix + long(MaxIconLength) }, "icon"},
		{"command at limit", func(c *ActionCard) { c.Primary.Command = long(MaxCommandLength) }, ""},
		{"command over limit", func(c *ActionCard) { c.Primary.Command = long(MaxCommandLength + 1) }, "primary.command"},
		{"args at limit", func(c *ActionCard) { c.Primary.Args = args(MaxArgs) }, ""},
		{"args over limit", func(c *ActionCard) { c.Primary.Args = args(MaxArgs + 1) }, "primary.args"},
		{"arg at limit", func(c *ActionCard) { c.Primary.Args = []string{long(MaxArgLength)} }, ""},
		{"arg over limit", func(c *ActionCard) { c.Primary.Args = []string{"ok", long(MaxArgLength + 1)} }, "primary.args[1]"},
		{"dir at limit", func(c *ActionCard) { c.Primary.Dir = long(MaxDirLength) }, ""},
		{"dir over limit", func(c *ActionCard) { c.Primary.Dir = long(MaxDirLength + 1) }, "primary.dir"},
		{"env at limit", func(c *ActionCard) { c.Primary.Env = env(MaxEnvEntries) }, ""},
		{"env over limit", func(c *ActionCard) { c.Primary.Env = env(MaxEnvEntries + 1) }, "primary.env"},
		{"env key with equals", func(c *ActionCard) { c.Primary.Env = map[string]string{"A=B": "x"} }, "primary.env.A=B"},
		{"env key starting with digit", func(c *ActionCard) { c.Primary.Env = map[string]string{"1A": "x"} }, "primary.env.1A"},
		{"env key at limit", func(c *ActionCard) { c.Primary.Env = map[string]string{"K" + long(MaxEnvKeyLength-1): "x"} }, ""},
		{"env key over limit", func(c *ActionCard) { c.Primary.Env = map[string]string{"K" + long(MaxEnvKeyLength): "x"} }, "primary.env.K" + long(MaxEnvKeyLength)},
		{"env value at limit", func(c *ActionCard) { c.Primary.Env = map[string]string{"K": long(MaxEnvValueLength)} }, ""},
		{"env value over limit", func(c *ActionCard) { c.Primary.Env = map[string]string{"K": long(MaxEnvValueLength + 1)} }, "primary.env.K"},
		{"timeout zero", func(c *ActionCard) { c.Primary.TimeoutSec = 0 }, "primary.timeoutSec"},
		{"timeout at limit", func(c *ActionCard) { c.Primary.TimeoutSec = MaxTimeoutSec }, ""},
		{"timeout over limit", func(c *ActionCard) { c.Primary.TimeoutSec = MaxTimeoutSec + 1 }, "primary.timeoutSec"},
		{"status timeout over limit", func(c *ActionCard) {
			status := validAction()
			status.TimeoutSec = MaxTimeoutSec + 1
			c.Status = &status
		}, "status.timeoutSec"},
		{"rule type unknown", func(c *ActionCard) { c.Primary.Rule.Type = "weird" }, "primary.rule.type"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			card := validCard()
			testCase.mutate(&card)
			err := card.Validate()
			if testCase.field == "" {
				if err != nil {
					t.Fatalf("Validate() error = %v, want valid", err)
				}
				return
			}
			var validation *ValidationError
			if !errors.As(err, &validation) {
				t.Fatalf("Validate() error = %v, want ValidationError", err)
			}
			if _, ok := validation.Fields[testCase.field]; !ok || len(validation.Fields) != 1 {
				t.Fatalf("fields = %v, want only %q", validation.Fields, testCase.field)
			}
		})
	}
}

func TestStatusSnapshotJSONOmitsCheckForUncheckedCard(t *testing.T) {
	unchecked, err := json.Marshal(StatusSnapshot{State: StatusStateUnknown})
	if err != nil {
		t.Fatalf("marshal unchecked: %v", err)
	}
	if string(unchecked) != `{"state":"unknown"}` {
		t.Fatalf("unchecked snapshot = %s, want only state", unchecked)
	}

	checkedAt := time.Date(2026, time.September, 26, 10, 0, 0, 0, time.UTC)
	checked := StatusSnapshot{
		State:     StatusStateOK,
		CheckedAt: checkedAt,
		LastCheck: Run{ActionKind: "status", StartedAt: checkedAt, Duration: time.Second, Outcome: RunOutcomeOK},
	}
	encoded, err := json.Marshal(checked)
	if err != nil {
		t.Fatalf("marshal checked: %v", err)
	}
	var decoded StatusSnapshot
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !decoded.CheckedAt.Equal(checkedAt) || decoded.LastCheck.Outcome != RunOutcomeOK || decoded.LastCheck.Duration != time.Second {
		t.Fatalf("round trip lost data: %+v", decoded)
	}
	var fresh StatusSnapshot
	if err := json.Unmarshal(unchecked, &fresh); err != nil {
		t.Fatalf("unmarshal unchecked: %v", err)
	}
	if !fresh.CheckedAt.IsZero() || fresh.State != StatusStateUnknown {
		t.Fatalf("unchecked round trip = %+v", fresh)
	}
}

func TestValidateEssentialIgnoresLimits(t *testing.T) {
	long := func(n int) string { return strings.Repeat("a", n) }
	card := validCard()
	card.ID = "legacy.card"
	card.Name = long(MaxNameLength + 1)
	card.Icon = "pi pi-Home"
	card.Primary.TimeoutSec = MaxTimeoutSec + 1
	card.Primary.Args = []string{long(MaxArgLength + 1)}
	card.PollingIntervalSeconds = 10
	card.FastPollingIntervalSeconds = 20
	if err := card.ValidateEssential(); err != nil {
		t.Fatalf("ValidateEssential() error = %v, want limits ignored", err)
	}
	if err := card.Validate(); !errors.Is(err, ErrValidation) {
		t.Fatalf("Validate() error = %v, want ErrValidation", err)
	}

	essential := map[string]func(*ActionCard){
		"empty id":             func(c *ActionCard) { c.ID = "" },
		"empty command":        func(c *ActionCard) { c.Primary.Command = "" },
		"zero timeout":         func(c *ActionCard) { c.Primary.TimeoutSec = 0 },
		"bad rule":             func(c *ActionCard) { c.Primary.Rule = OutputRule{Type: "bogus"} },
		"bad env name":         func(c *ActionCard) { c.Primary.Env = map[string]string{"1x": "v"} },
		"negative polling":     func(c *ActionCard) { c.PollingIntervalSeconds = -1 },
		"status without cmd":   func(c *ActionCard) { c.Status = &Action{TimeoutSec: 1} },
		"negative fast window": func(c *ActionCard) { c.FastPollingWindowSeconds = -1 },
	}
	for name, mutate := range essential {
		broken := validCard()
		mutate(&broken)
		if err := broken.ValidateEssential(); !errors.Is(err, ErrValidation) {
			t.Errorf("%s: ValidateEssential() error = %v, want ErrValidation", name, err)
		}
	}
}
