package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"marionette/internal/config"
)

const statusEventHeartbeat = 15 * time.Second

// StatusEvent is the payload published to SSE subscribers when a card status changes.
type StatusEvent struct {
	ID       uint64
	CardID   string
	Snapshot config.StatusSnapshot
}

// StatusEventBroker fans status transitions out to connected SSE clients.
type StatusEventBroker struct {
	mu          sync.Mutex
	nextID      uint64
	subscribers map[chan StatusEvent]struct{}
	// done is closed by Close so that every events handler returns at once
	// instead of keeping its connection open until the HTTP server gives up.
	done   chan struct{}
	closed bool
}

// NewStatusEventBroker creates an empty broker with no subscribers.
func NewStatusEventBroker() *StatusEventBroker {
	return &StatusEventBroker{
		subscribers: make(map[chan StatusEvent]struct{}),
		done:        make(chan struct{}),
	}
}

// Publish sends a transition to subscribers without waiting for any client.
// Events published after Close are dropped.
func (broker *StatusEventBroker) Publish(cardID string, snapshot config.StatusSnapshot) {
	broker.mu.Lock()
	defer broker.mu.Unlock()
	if broker.closed {
		return
	}
	broker.nextID++
	event := StatusEvent{ID: broker.nextID, CardID: cardID, Snapshot: snapshot}
	for subscriber := range broker.subscribers {
		select {
		case subscriber <- event:
		default:
			// Keep the stream non-blocking while ensuring the newest snapshot
			// replaces stale buffered data for a slow client.
			select {
			case <-subscriber:
			default:
			}
			select {
			case subscriber <- event:
			default:
			}
		}
	}
}

// Close disconnects every subscriber and makes the events endpoint end new
// streams immediately. It is idempotent and safe to call concurrently with
// Publish and with connected clients.
func (broker *StatusEventBroker) Close() {
	broker.mu.Lock()
	defer broker.mu.Unlock()
	if broker.closed {
		return
	}
	broker.closed = true
	close(broker.done)
	broker.subscribers = make(map[chan StatusEvent]struct{})
}

// Done is closed once the broker has been closed.
func (broker *StatusEventBroker) Done() <-chan struct{} {
	return broker.done
}

func (broker *StatusEventBroker) subscribe() (<-chan StatusEvent, func()) {
	subscriber := make(chan StatusEvent, 8)
	broker.mu.Lock()
	if !broker.closed {
		broker.subscribers[subscriber] = struct{}{}
	}
	broker.mu.Unlock()

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			broker.mu.Lock()
			delete(broker.subscribers, subscriber)
			broker.mu.Unlock()
		})
	}
	return subscriber, unsubscribe
}

func (api cardAPI) events(w http.ResponseWriter, request *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, errors.New("streaming is not supported"))
		return
	}

	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprint(w, ": connected\n\n")
	flusher.Flush()

	subscriber, unsubscribe := api.statusEvents.subscribe()
	defer unsubscribe()
	heartbeat := time.NewTicker(statusEventHeartbeat)
	defer heartbeat.Stop()

	for {
		select {
		case <-request.Context().Done():
			return
		case <-api.statusEvents.Done():
			return
		case event := <-subscriber:
			payload, err := json.Marshal(struct {
				CardID   string                `json:"cardId"`
				Snapshot config.StatusSnapshot `json:"snapshot"`
			}{CardID: event.CardID, Snapshot: event.Snapshot})
			if err != nil {
				return
			}
			_, _ = fmt.Fprintf(w, "id: %s\nevent: status.changed\ndata: %s\n\n", strconv.FormatUint(event.ID, 10), payload)
			flusher.Flush()
		case <-heartbeat.C:
			_, _ = fmt.Fprint(w, ": heartbeat\n\n")
			flusher.Flush()
		}
	}
}
