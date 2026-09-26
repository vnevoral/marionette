// Package status checks configured card status actions and updates their projection.
package status

import (
	"errors"
	"fmt"

	"marionette/internal/config"
	execengine "marionette/internal/exec"
)

// ErrStatusActionMissing is returned when a card has no status action configured.
var ErrStatusActionMissing = errors.New("status action is not configured")

type runner interface {
	Run(config.Action) (execengine.Result, error)
}

// StatusCheckService runs status actions and records their interpreted result.
type StatusCheckService struct {
	store  *config.Store
	runner runner
}

// NewStatusCheckService creates a status check service for the supplied store and runner.
func NewStatusCheckService(store *config.Store, actionRunner runner) (*StatusCheckService, error) {
	if store == nil {
		return nil, errors.New("config store is required")
	}
	if actionRunner == nil {
		return nil, errors.New("execution runner is required")
	}
	return &StatusCheckService{store: store, runner: actionRunner}, nil
}

// CheckNow executes the configured status action for one card and updates its status projection.
func (service *StatusCheckService) CheckNow(cardID string) (config.StatusSnapshot, error) {
	card, exists := service.store.GetCard(cardID)
	if !exists {
		return config.StatusSnapshot{}, config.ErrNotFound
	}
	if card.Status == nil {
		return config.StatusSnapshot{}, ErrStatusActionMissing
	}

	result, err := service.runner.Run(*card.Status)
	if err != nil {
		return config.StatusSnapshot{}, fmt.Errorf("run status action: %w", err)
	}
	snapshot := config.StatusSnapshot{
		State:     stateFromOutcome(result.Outcome),
		CheckedAt: result.StartedAt.Add(result.Duration),
		LastCheck: result.ToRun("status"),
	}
	if err := service.store.UpdateStatus(cardID, snapshot); err != nil {
		return config.StatusSnapshot{}, fmt.Errorf("update status: %w", err)
	}
	return snapshot, nil
}

func stateFromOutcome(outcome config.RunOutcome) config.StatusState {
	if outcome == config.RunOutcomeOK {
		return config.StatusStateOK
	}
	return config.StatusStateFail
}
