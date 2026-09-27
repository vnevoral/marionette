package server

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"marionette/internal/config"
	"marionette/internal/events"
)

func TestNewRouterDoesNotMutateStoreAndStreamsBrokerEvents(t *testing.T) {
	store := config.NewStore(config.Settings{HistorySize: 5, MaxConcurrentActions: 1})
	card := createServerCard(t, store, "wired")
	broker := events.NewBroker()
	handler := NewRouter(Dependencies{Store: store, Events: broker})
	if store.OnStatusChange != nil || store.OnRunAppended != nil {
		t.Fatal("NewRouter must not install store callbacks; that is the composition root's job")
	}
	// Composition root wiring, as done in cmd/marionette.
	store.OnStatusChange = broker.Publish
	store.OnRunAppended = broker.PublishRun

	testServer := httptest.NewServer(handler)
	defer testServer.Close()
	response, err := http.Get(testServer.URL + "/api/events")
	if err != nil {
		t.Fatalf("GET /api/events error = %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	reader := bufio.NewReader(response.Body)
	if line, err := reader.ReadString('\n'); err != nil || line != ": connected\n" {
		t.Fatalf("first SSE line = %q, %v", line, err)
	}
	if err := store.UpdateStatus(card.ID, config.StatusSnapshot{State: config.StatusStateOK}); err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}
	var lines []string
	for len(lines) < 4 {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("reading SSE stream: %v (got %q)", err, lines)
		}
		lines = append(lines, line)
	}
	body := strings.Join(lines, "")
	if !strings.Contains(body, "id: 1\n") || !strings.Contains(body, "event: status.changed\n") ||
		!strings.Contains(body, `"cardId":"wired"`) || !strings.Contains(body, `"state":"ok"`) {
		t.Fatalf("SSE body = %q", body)
	}

	// A recorded primary run follows as run.recorded (FR-42a) with the run in
	// the shape of GET /api/cards/{id}/runs.
	run := config.Run{ActionKind: "primary", ExitCode: 2, Outcome: config.RunOutcomeFail, Output: "down"}
	if err := store.AppendRun(card.ID, run); err != nil {
		t.Fatalf("AppendRun() error = %v", err)
	}
	lines = nil
	for len(lines) < 4 {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("reading SSE stream: %v (got %q)", err, lines)
		}
		lines = append(lines, line)
	}
	body = strings.Join(lines, "")
	runJSON, _ := json.Marshal(run)
	if !strings.Contains(body, "id: 2\n") || !strings.Contains(body, "event: run.recorded\n") ||
		!strings.Contains(body, `data: {"cardId":"wired","run":`+string(runJSON)+"}\n") {
		t.Fatalf("SSE body = %q", body)
	}
}

func TestStatusEventsEndpointSetsSSEHeadersAndStopsWithRequest(t *testing.T) {
	store := config.NewStore(config.Settings{HistorySize: 5, MaxConcurrentActions: 1})
	broker := events.NewBroker()
	handler := NewRouter(Dependencies{Store: store, Events: broker})
	requestContext, cancel := context.WithCancel(context.Background())
	request := httptest.NewRequest(http.MethodGet, "/api/events", nil).WithContext(requestContext)
	response := httptest.NewRecorder()
	cancel()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Header().Get("Content-Type") != "text/event-stream" {
		t.Fatalf("SSE response = %d %q", response.Code, response.Header().Get("Content-Type"))
	}
	if !strings.Contains(response.Body.String(), ": connected\n\n") {
		t.Fatalf("SSE response did not contain connection comment: %q", response.Body.String())
	}
}

func TestBrokerCloseEndsEventsHandler(t *testing.T) {
	store := config.NewStore(config.Settings{HistorySize: 5, MaxConcurrentActions: 1})
	broker := events.NewBroker()
	testServer := httptest.NewServer(NewRouter(Dependencies{Store: store, Events: broker}))
	defer testServer.Close()

	response, err := http.Get(testServer.URL + "/api/events")
	if err != nil {
		t.Fatalf("GET /api/events error = %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	reader := bufio.NewReader(response.Body)
	if line, err := reader.ReadString('\n'); err != nil || line != ": connected\n" {
		t.Fatalf("first SSE line = %q, %v", line, err)
	}

	finished := make(chan error, 1)
	go func() {
		_, err := io.ReadAll(reader)
		finished <- err
	}()
	started := time.Now()
	broker.Close()
	select {
	case err := <-finished:
		if err != nil {
			t.Fatalf("stream ended with error %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("events handler kept the stream open after Close()")
	}
	if elapsed := time.Since(started); elapsed > 100*time.Millisecond {
		t.Fatalf("stream took %s to end after Close()", elapsed)
	}
	broker.Close()                                                             // idempotent
	broker.Publish("card", config.StatusSnapshot{State: config.StatusStateOK}) // dropped, no panic
}

func TestStatusEventsEndpointEndsImmediatelyWhenBrokerClosed(t *testing.T) {
	store := config.NewStore(config.Settings{HistorySize: 5, MaxConcurrentActions: 1})
	broker := events.NewBroker()
	broker.Close()
	handler := NewRouter(Dependencies{Store: store, Events: broker})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/events", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), ": connected\n\n") {
		t.Fatalf("SSE response = %d %q", response.Code, response.Body.String())
	}
}
