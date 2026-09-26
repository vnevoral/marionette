package server

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"marionette/internal/config"
)

func TestStatusEventBrokerPublishesToSubscribers(t *testing.T) {
	broker := NewStatusEventBroker()
	subscriber, unsubscribe := broker.subscribe()
	defer unsubscribe()

	snapshot := config.StatusSnapshot{State: config.StatusStateOK}
	broker.Publish("card-a", snapshot)

	select {
	case event := <-subscriber:
		if event.ID != 1 || event.CardID != "card-a" || event.Snapshot.State != config.StatusStateOK {
			t.Fatalf("event = %#v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for status event")
	}
}

func TestStatusEventBrokerKeepsNewestEventForSlowSubscriber(t *testing.T) {
	broker := NewStatusEventBroker()
	subscriber, unsubscribe := broker.subscribe()
	defer unsubscribe()

	for index := 1; index <= 9; index++ {
		broker.Publish("card-a", config.StatusSnapshot{State: config.StatusStateOK})
	}

	var last StatusEvent
	for {
		select {
		case last = <-subscriber:
		default:
			if last.ID != 9 {
				t.Fatalf("last event ID = %d, want 9", last.ID)
			}
			return
		}
	}
}

func TestRouterWiresStoreStatusChangesToBroker(t *testing.T) {
	store := config.NewStore(config.Settings{HistorySize: 5, MaxConcurrentActions: 1})
	card := config.ActionCard{
		ID:      "card-status",
		Name:    "Status card",
		Primary: config.Action{Command: "primary", TimeoutSec: 1},
		Status:  &config.Action{Command: "status", TimeoutSec: 1},
	}
	if _, err := store.CreateCard(card); err != nil {
		t.Fatalf("CreateCard() error = %v", err)
	}
	broker := NewStatusEventBroker()
	subscriber, unsubscribe := broker.subscribe()
	defer unsubscribe()
	_ = NewRouterWithDependencies(RouterDependencies{Store: store, StatusEvents: broker})

	if err := store.UpdateStatus(card.ID, config.StatusSnapshot{State: config.StatusStateOK}); err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}
	select {
	case event := <-subscriber:
		if event.CardID != card.ID || event.Snapshot.State != config.StatusStateOK {
			t.Fatalf("event = %#v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for router-wired status event")
	}
}

func TestStatusEventsEndpointSetsSSEHeadersAndStopsWithRequest(t *testing.T) {
	store := config.NewStore(config.Settings{HistorySize: 5, MaxConcurrentActions: 1})
	broker := NewStatusEventBroker()
	handler := NewRouterWithDependencies(RouterDependencies{Store: store, StatusEvents: broker})
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
	broker := NewStatusEventBroker()
	testServer := httptest.NewServer(NewRouterWithDependencies(RouterDependencies{Store: store, StatusEvents: broker}))
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
	broker := NewStatusEventBroker()
	broker.Close()
	handler := NewRouterWithDependencies(RouterDependencies{Store: store, StatusEvents: broker})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/events", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), ": connected\n\n") {
		t.Fatalf("SSE response = %d %q", response.Code, response.Body.String())
	}
}
