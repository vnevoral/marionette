package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"marionette/internal/config"
	execengine "marionette/internal/exec"
)

func validServerCard(id string) config.ActionCard {
	return config.ActionCard{
		ID:      id,
		Name:    "Server card",
		Primary: config.Action{Command: "primary", TimeoutSec: 1},
		Status:  &config.Action{Command: "status", TimeoutSec: 1},
	}
}

func createServerCard(t *testing.T, store *config.Store, id string) config.ActionCard {
	t.Helper()
	card, err := store.CreateCard(validServerCard(id))
	if err != nil {
		t.Fatalf("CreateCard() error = %v", err)
	}
	return card
}

func requestJSON(t *testing.T, handler http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	request := httptest.NewRequest(method, path, bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func TestRouterHealthAndCardCRUD(t *testing.T) {
	store := config.NewStore(config.Settings{HistorySize: 5, MaxConcurrentActions: 1})
	handler := NewRouter(store)

	health := httptest.NewRecorder()
	handler.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/api/health", nil))
	if health.Code != http.StatusOK || health.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("health response = %d %q", health.Code, health.Header().Get("Content-Type"))
	}

	createdResponse := requestJSON(t, handler, http.MethodPost, "/api/cards", validServerCard("card-a"))
	if createdResponse.Code != http.StatusCreated {
		t.Fatalf("create response = %d, body = %s", createdResponse.Code, createdResponse.Body.String())
	}
	var created config.ActionCard
	if err := json.NewDecoder(createdResponse.Body).Decode(&created); err != nil {
		t.Fatalf("decode created card: %v", err)
	}
	if created.ID != "card-a" {
		t.Fatalf("created card ID = %q", created.ID)
	}

	listResponse := httptest.NewRecorder()
	handler.ServeHTTP(listResponse, httptest.NewRequest(http.MethodGet, "/api/cards", nil))
	if listResponse.Code != http.StatusOK {
		t.Fatalf("list response = %d", listResponse.Code)
	}
	var cards []config.ActionCard
	if err := json.NewDecoder(listResponse.Body).Decode(&cards); err != nil || len(cards) != 1 {
		t.Fatalf("list cards = %#v, error = %v", cards, err)
	}

	updated := validServerCard("ignored-path-id")
	updated.Name = "Updated"
	updateResponse := requestJSON(t, handler, http.MethodPut, "/api/cards/card-a", updated)
	if updateResponse.Code != http.StatusOK {
		t.Fatalf("update response = %d, body = %s", updateResponse.Code, updateResponse.Body.String())
	}
	var updatedCard config.ActionCard
	if err := json.NewDecoder(updateResponse.Body).Decode(&updatedCard); err != nil {
		t.Fatalf("decode updated card: %v", err)
	}
	if updatedCard.ID != "card-a" || updatedCard.Name != "Updated" {
		t.Fatalf("updated card = %#v", updatedCard)
	}

	deleteResponse := httptest.NewRecorder()
	handler.ServeHTTP(deleteResponse, httptest.NewRequest(http.MethodDelete, "/api/cards/card-a", nil))
	if deleteResponse.Code != http.StatusNoContent {
		t.Fatalf("delete response = %d", deleteResponse.Code)
	}
	missingResponse := httptest.NewRecorder()
	handler.ServeHTTP(missingResponse, httptest.NewRequest(http.MethodGet, "/api/cards/card-a", nil))
	if missingResponse.Code != http.StatusNotFound {
		t.Fatalf("missing response = %d", missingResponse.Code)
	}
}

func TestRouterRejectsInvalidJSONAndDuplicateCard(t *testing.T) {
	store := config.NewStore(config.Settings{HistorySize: 5, MaxConcurrentActions: 1})
	handler := NewRouter(store)
	createServerCard(t, store, "duplicate")

	duplicate := requestJSON(t, handler, http.MethodPost, "/api/cards", validServerCard("duplicate"))
	if duplicate.Code != http.StatusConflict {
		t.Fatalf("duplicate response = %d, body = %s", duplicate.Code, duplicate.Body.String())
	}
	unknownField := httptest.NewRequest(http.MethodPost, "/api/cards", bytes.NewBufferString(`{"id":"x","name":"x","primary":{"command":"x","timeoutSec":1},"unexpected":true}`))
	unknownField.Header.Set("Content-Type", "application/json")
	unknownResponse := httptest.NewRecorder()
	handler.ServeHTTP(unknownResponse, unknownField)
	if unknownResponse.Code != http.StatusBadRequest {
		t.Fatalf("unknown field response = %d", unknownResponse.Code)
	}
	trailing := httptest.NewRequest(http.MethodPost, "/api/cards", bytes.NewBufferString(`{"id":"x","name":"x","primary":{"command":"x","timeoutSec":1}} {}`))
	trailingResponse := httptest.NewRecorder()
	handler.ServeHTTP(trailingResponse, trailing)
	if trailingResponse.Code != http.StatusBadRequest {
		t.Fatalf("trailing JSON response = %d", trailingResponse.Code)
	}
}

func TestRouterReadsRunsAndStatusProjectionWithoutExecutingStatus(t *testing.T) {
	store := config.NewStore(config.Settings{HistorySize: 5, MaxConcurrentActions: 1})
	card := createServerCard(t, store, "card-status")
	if err := store.AppendRun(card.ID, config.Run{ActionKind: "primary", Outcome: config.RunOutcomeOK}); err != nil {
		t.Fatalf("AppendRun() error = %v", err)
	}
	checkedAt := time.Date(2026, time.September, 25, 12, 0, 0, 0, time.UTC)
	if err := store.UpdateStatus(card.ID, config.StatusSnapshot{
		State:     config.StatusStateOK,
		CheckedAt: checkedAt,
		LastCheck: config.Run{ActionKind: "status", StartedAt: checkedAt, Outcome: config.RunOutcomeOK},
	}); err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}
	handler := NewRouter(store)

	statusResponse := httptest.NewRecorder()
	handler.ServeHTTP(statusResponse, httptest.NewRequest(http.MethodGet, "/api/cards/card-status/status", nil))
	if statusResponse.Code != http.StatusOK {
		t.Fatalf("status response = %d", statusResponse.Code)
	}
	var snapshot config.StatusSnapshot
	if err := json.NewDecoder(statusResponse.Body).Decode(&snapshot); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if snapshot.State != config.StatusStateOK || !snapshot.CheckedAt.Equal(checkedAt) {
		t.Fatalf("snapshot = %#v", snapshot)
	}

	runsResponse := httptest.NewRecorder()
	handler.ServeHTTP(runsResponse, httptest.NewRequest(http.MethodGet, "/api/cards/card-status/runs", nil))
	if runsResponse.Code != http.StatusOK {
		t.Fatalf("runs response = %d", runsResponse.Code)
	}
	var runs []config.Run
	if err := json.NewDecoder(runsResponse.Body).Decode(&runs); err != nil || len(runs) != 1 || runs[0].ActionKind != "primary" {
		t.Fatalf("runs = %#v, error = %v", runs, err)
	}

	historyResponse := httptest.NewRecorder()
	handler.ServeHTTP(historyResponse, httptest.NewRequest(http.MethodGet, "/api/cards/card-status/status/history", nil))
	if historyResponse.Code != http.StatusOK {
		t.Fatalf("history response = %d", historyResponse.Code)
	}
}

func TestRouterReturnsUnknownStatusBeforeFirstCheck(t *testing.T) {
	store := config.NewStore(config.Settings{HistorySize: 5, MaxConcurrentActions: 1})
	createServerCard(t, store, "unknown")
	handler := NewRouter(store)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/cards/unknown/status", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status response = %d", response.Code)
	}
	var snapshot config.StatusSnapshot
	if err := json.NewDecoder(response.Body).Decode(&snapshot); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if snapshot.State != config.StatusStateUnknown {
		t.Fatalf("unknown snapshot state = %q", snapshot.State)
	}
}

func TestRouterWithoutStoreKeepsHealthAvailable(t *testing.T) {
	handler := NewRouter()
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/health", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("health response = %d", response.Code)
	}
}

type fakeActionQueue struct {
	primaryCalls chan string
	statusCalls  chan string
}

func (queue *fakeActionQueue) EnqueuePrimary(cardID string, _ config.Action) error {
	queue.primaryCalls <- cardID
	return nil
}

func (queue *fakeActionQueue) EnqueueStatus(cardID string) error {
	queue.statusCalls <- cardID
	return nil
}

func (queue *fakeActionQueue) Wait() {}

type fakePrimaryNotifier struct {
	calls chan string
}

func (notifier *fakePrimaryNotifier) NotifyPrimaryAction(cardID string) error {
	notifier.calls <- cardID
	return nil
}

func TestRouterEnqueuesActionsWithoutWaiting(t *testing.T) {
	store := config.NewStore(config.Settings{HistorySize: 5, MaxConcurrentActions: 1})
	createServerCard(t, store, "enqueue")
	queue := &fakeActionQueue{primaryCalls: make(chan string, 1), statusCalls: make(chan string, 1)}
	notifier := &fakePrimaryNotifier{calls: make(chan string, 1)}
	handler := NewRouterWithDependencies(RouterDependencies{Store: store, Actions: queue, Notifier: notifier})

	primary := httptest.NewRecorder()
	handler.ServeHTTP(primary, httptest.NewRequest(http.MethodPost, "/api/cards/enqueue/actions/primary", nil))
	if primary.Code != http.StatusAccepted {
		t.Fatalf("primary enqueue response = %d, body = %s", primary.Code, primary.Body.String())
	}
	if cardID := <-queue.primaryCalls; cardID != "enqueue" {
		t.Fatalf("primary enqueue card ID = %q", cardID)
	}
	if cardID := <-notifier.calls; cardID != "enqueue" {
		t.Fatalf("primary notification card ID = %q", cardID)
	}

	status := httptest.NewRecorder()
	handler.ServeHTTP(status, httptest.NewRequest(http.MethodPost, "/api/cards/enqueue/actions/status/check", nil))
	if status.Code != http.StatusAccepted {
		t.Fatalf("status enqueue response = %d, body = %s", status.Code, status.Body.String())
	}
	if cardID := <-queue.statusCalls; cardID != "enqueue" {
		t.Fatalf("status enqueue card ID = %q", cardID)
	}
}

func TestRouterRejectsStatusEnqueueWithoutStatusAction(t *testing.T) {
	store := config.NewStore(config.Settings{HistorySize: 5, MaxConcurrentActions: 1})
	card := validServerCard("without-status")
	card.Status = nil
	if _, err := store.CreateCard(card); err != nil {
		t.Fatalf("CreateCard() error = %v", err)
	}
	queue := &fakeActionQueue{primaryCalls: make(chan string, 1), statusCalls: make(chan string, 1)}
	handler := NewRouterWithDependencies(RouterDependencies{Store: store, Actions: queue})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/cards/without-status/actions/status/check", nil))
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status enqueue response = %d", response.Code)
	}
}

type fakeBackgroundRunner struct {
	called chan config.Action
	result execengine.Result
}

func (runner *fakeBackgroundRunner) Run(action config.Action) (execengine.Result, error) {
	runner.called <- action
	return runner.result, nil
}

type fakeStatusChecker struct {
	called chan string
}

func (checker *fakeStatusChecker) CheckNow(cardID string) (config.StatusSnapshot, error) {
	checker.called <- cardID
	return config.StatusSnapshot{State: config.StatusStateOK}, nil
}

func TestBackgroundActionsRunAndDrainAcceptedWork(t *testing.T) {
	store := config.NewStore(config.Settings{HistorySize: 5, MaxConcurrentActions: 1})
	card := createServerCard(t, store, "background")
	runner := &fakeBackgroundRunner{
		called: make(chan config.Action, 1),
		result: execengine.Result{Outcome: config.RunOutcomeOK},
	}
	checker := &fakeStatusChecker{called: make(chan string, 1)}
	actions, err := NewBackgroundActions(store, runner, checker)
	if err != nil {
		t.Fatalf("NewBackgroundActions() error = %v", err)
	}
	if err := actions.EnqueuePrimary(card.ID, card.Primary); err != nil {
		t.Fatalf("EnqueuePrimary() error = %v", err)
	}
	if err := actions.EnqueueStatus(card.ID); err != nil {
		t.Fatalf("EnqueueStatus() error = %v", err)
	}
	actions.Wait()

	select {
	case <-runner.called:
	default:
		t.Fatal("primary action was not executed")
	}
	select {
	case cardID := <-checker.called:
		if cardID != card.ID {
			t.Fatalf("status card ID = %q", cardID)
		}
	default:
		t.Fatal("status action was not executed")
	}
	runs, err := store.GetRuns(card.ID, "primary")
	if err != nil || len(runs) != 1 || runs[0].Outcome != config.RunOutcomeOK {
		t.Fatalf("primary runs = %#v, error = %v", runs, err)
	}
	actions.Close()
	if err := actions.EnqueuePrimary(card.ID, card.Primary); !errors.Is(err, ErrActionQueueClosed) {
		t.Fatalf("enqueue after close error = %v", err)
	}
}
