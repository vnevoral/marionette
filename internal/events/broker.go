// Package events fans card status transitions out to in-process subscribers
// (the SSE endpoint) without blocking the producer.
package events

import (
	"sync"

	"marionette/internal/config"
)

// subscriberBuffer bounds how many events a slow subscriber may lag behind.
const subscriberBuffer = 8

// Event is one published status transition.
type Event struct {
	ID       uint64
	CardID   string
	Snapshot config.StatusSnapshot
}

// Broker delivers status transitions to every current subscriber. Publish
// never blocks: a subscriber that cannot keep up loses its oldest buffered
// event, so it always ends with the newest snapshot.
type Broker struct {
	mu          sync.Mutex
	nextID      uint64
	subscribers map[chan Event]struct{}
	done        chan struct{}
	closed      bool
}

// NewBroker creates an open broker with no subscribers.
func NewBroker() *Broker {
	return &Broker{
		subscribers: make(map[chan Event]struct{}),
		done:        make(chan struct{}),
	}
}

// Publish sends a transition to subscribers without waiting for any of them.
// It has the signature of config.Store.OnStatusChange so the store can be
// wired directly to the broker. Events published after Close are dropped.
func (broker *Broker) Publish(cardID string, snapshot config.StatusSnapshot) {
	broker.mu.Lock()
	defer broker.mu.Unlock()
	if broker.closed {
		return
	}
	broker.nextID++
	event := Event{ID: broker.nextID, CardID: cardID, Snapshot: snapshot}
	for subscriber := range broker.subscribers {
		select {
		case subscriber <- event:
		default:
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

// Subscribe registers a new subscriber and returns its channel together with
// an idempotent unsubscribe function. After Close the channel never receives.
func (broker *Broker) Subscribe() (<-chan Event, func()) {
	subscriber := make(chan Event, subscriberBuffer)
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

// Close drops every subscriber and closes Done so that streaming consumers
// end at once. It is idempotent and safe to call concurrently with Publish.
func (broker *Broker) Close() {
	broker.mu.Lock()
	defer broker.mu.Unlock()
	if broker.closed {
		return
	}
	broker.closed = true
	close(broker.done)
	broker.subscribers = make(map[chan Event]struct{})
}

// Done is closed once the broker has been closed.
func (broker *Broker) Done() <-chan struct{} {
	return broker.done
}
