package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"marionette/internal/actions"
	"marionette/internal/config"
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
	handler := NewRouter(Dependencies{Store: store})

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
	handler := NewRouter(Dependencies{Store: store})
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
	trailing.Header.Set("Content-Type", "application/json")
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
	handler := NewRouter(Dependencies{Store: store})

	listResponse := httptest.NewRecorder()
	handler.ServeHTTP(listResponse, httptest.NewRequest(http.MethodGet, "/api/cards", nil))
	if listResponse.Code != http.StatusOK {
		t.Fatalf("list response = %d", listResponse.Code)
	}
	var views []cardView
	if err := json.NewDecoder(listResponse.Body).Decode(&views); err != nil {
		t.Fatalf("decode card views: %v", err)
	}
	if len(views) != 1 || views[0].CurrentStatus == nil || views[0].CurrentStatus.State != config.StatusStateOK {
		t.Fatalf("card views = %#v", views)
	}

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
	handler := NewRouter(Dependencies{Store: store})
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
	handler := NewRouter(Dependencies{})
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

type fakePrimaryNotifier struct {
	calls chan string
}

func (notifier *fakePrimaryNotifier) NotifyPrimaryAction(cardID string) error {
	notifier.calls <- cardID
	return nil
}

type failingPrimaryNotifier struct{}

func (failingPrimaryNotifier) NotifyPrimaryAction(string) error {
	return errors.New("scheduler unavailable")
}

func TestRouterEnqueuesActionsWithoutWaiting(t *testing.T) {
	store := config.NewStore(config.Settings{HistorySize: 5, MaxConcurrentActions: 1})
	createServerCard(t, store, "enqueue")
	queue := &fakeActionQueue{primaryCalls: make(chan string, 1), statusCalls: make(chan string, 1)}
	notifier := &fakePrimaryNotifier{calls: make(chan string, 1)}
	handler := NewRouter(Dependencies{Store: store, Actions: queue, Notifier: notifier})

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

	// Never checked: the baseline is absent.
	var accepted map[string]any
	if err := json.Unmarshal(primary.Body.Bytes(), &accepted); err != nil {
		t.Fatalf("decode 202 body: %v", err)
	}
	if _, present := accepted["checkedAt"]; present {
		t.Fatalf("checkedAt present for an unchecked card: %s", primary.Body.String())
	}

	checkedAt := time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)
	if err := store.UpdateStatus("enqueue", config.StatusSnapshot{State: config.StatusStateOK, CheckedAt: checkedAt}); err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}
	status := httptest.NewRecorder()
	handler.ServeHTTP(status, httptest.NewRequest(http.MethodPost, "/api/cards/enqueue/actions/status/check", nil))
	if status.Code != http.StatusAccepted {
		t.Fatalf("status enqueue response = %d, body = %s", status.Code, status.Body.String())
	}
	if cardID := <-queue.statusCalls; cardID != "enqueue" {
		t.Fatalf("status enqueue card ID = %q", cardID)
	}
	var acceptedStatus struct {
		CheckedAt time.Time `json:"checkedAt"`
	}
	if err := json.Unmarshal(status.Body.Bytes(), &acceptedStatus); err != nil {
		t.Fatalf("decode 202 body: %v", err)
	}
	if !acceptedStatus.CheckedAt.Equal(checkedAt) {
		t.Fatalf("202 checkedAt = %v, want %v (body %s)", acceptedStatus.CheckedAt, checkedAt, status.Body.String())
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
	handler := NewRouter(Dependencies{Store: store, Actions: queue})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/cards/without-status/actions/status/check", nil))
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status enqueue response = %d", response.Code)
	}
}

func TestRouterKeepsAcceptedPrimaryActionWhenFastPollingNotificationFails(t *testing.T) {
	store := config.NewStore(config.Settings{HistorySize: 5, MaxConcurrentActions: 1})
	createServerCard(t, store, "notify-failure")
	queue := &fakeActionQueue{primaryCalls: make(chan string, 1), statusCalls: make(chan string, 1)}
	handler := NewRouter(Dependencies{
		Store:    store,
		Actions:  queue,
		Notifier: failingPrimaryNotifier{},
	})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/cards/notify-failure/actions/primary", nil))
	if response.Code != http.StatusAccepted {
		t.Fatalf("primary enqueue response = %d", response.Code)
	}
	if cardID := <-queue.primaryCalls; cardID != "notify-failure" {
		t.Fatalf("primary enqueue card ID = %q", cardID)
	}
}

func TestRouterReturnsJSONNotFoundForUnknownAPIPath(t *testing.T) {
	handler := NewRouter(Dependencies{Store: config.NewStore(config.Settings{HistorySize: 5, MaxConcurrentActions: 1})})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/not-a-route", nil))
	if response.Code != http.StatusNotFound || response.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("unknown API response = %d %q", response.Code, response.Header().Get("Content-Type"))
	}
}

func TestRouterMapsPersistenceFailureToInternalServerError(t *testing.T) {
	store := config.NewStore(config.Settings{HistorySize: 5, MaxConcurrentActions: 1})
	store.OnChange = func(*config.Store) error {
		return errors.New("disk full")
	}
	handler := NewRouter(Dependencies{Store: store})
	response := requestJSON(t, handler, http.MethodPost, "/api/cards", validServerCard("persist-failure"))
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("persistence failure response = %d", response.Code)
	}
	if _, exists := store.GetCard("persist-failure"); exists {
		t.Fatal("card mutation was not rolled back after the persistence failure")
	}
	getResponse := httptest.NewRecorder()
	handler.ServeHTTP(getResponse, httptest.NewRequest(http.MethodGet, "/api/cards/persist-failure", nil))
	if getResponse.Code != http.StatusNotFound {
		t.Fatalf("GET after failed create = %d, want 404", getResponse.Code)
	}
}

type stubActionQueue struct {
	err error
}

func (queue stubActionQueue) EnqueuePrimary(string, config.Action) error { return queue.err }
func (queue stubActionQueue) EnqueueStatus(string) error                 { return queue.err }

func TestRouterMapsQueueErrorsToAcceptedOrServiceUnavailable(t *testing.T) {
	store := config.NewStore(config.Settings{HistorySize: 5, MaxConcurrentActions: 1})
	createServerCard(t, store, "queue")
	cases := []struct {
		name       string
		err        error
		wantStatus int
		wantRetry  string
	}{
		{name: "already queued", err: actions.ErrAlreadyQueued, wantStatus: http.StatusAccepted},
		{name: "full", err: &actions.QueueFullError{RetryAfter: 2500 * time.Millisecond}, wantStatus: http.StatusServiceUnavailable, wantRetry: "3"},
		{name: "closed", err: actions.ErrQueueClosed, wantStatus: http.StatusServiceUnavailable, wantRetry: "1"},
		{name: "other", err: errors.New("boom"), wantStatus: http.StatusInternalServerError},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			handler := NewRouter(Dependencies{Store: store, Actions: stubActionQueue{err: testCase.err}})
			for _, path := range []string{"/api/cards/queue/actions/primary", "/api/cards/queue/actions/status/check"} {
				response := requestJSON(t, handler, http.MethodPost, path, nil)
				if response.Code != testCase.wantStatus {
					t.Fatalf("%s status = %d, want %d (body %s)", path, response.Code, testCase.wantStatus, response.Body.String())
				}
				if retry := response.Header().Get("Retry-After"); retry != testCase.wantRetry {
					t.Fatalf("%s Retry-After = %q, want %q", path, retry, testCase.wantRetry)
				}
			}
		})
	}
}

func TestRouterMapsValidationErrorsToUnprocessableEntity(t *testing.T) {
	store := config.NewStore(config.Settings{HistorySize: 5, MaxConcurrentActions: 1})
	handler := NewRouter(Dependencies{Store: store})
	invalid := validServerCard("invalid")
	invalid.Name = ""
	response := requestJSON(t, handler, http.MethodPost, "/api/cards", invalid)
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("invalid create response = %d, want 422: %s", response.Code, response.Body.String())
	}
	createServerCard(t, store, "existing")
	invalid.ID = "existing"
	response = requestJSON(t, handler, http.MethodPut, "/api/cards/existing", invalid)
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("invalid update response = %d, want 422: %s", response.Code, response.Body.String())
	}
}
