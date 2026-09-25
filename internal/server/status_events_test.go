package server

import (
	"context"
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
