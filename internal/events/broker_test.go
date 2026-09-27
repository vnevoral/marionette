package events

import (
	"testing"
	"time"

	"marionette/internal/config"
)

func TestBrokerPublishesToSubscribers(t *testing.T) {
	broker := NewBroker()
	subscriber, unsubscribe := broker.Subscribe()
	defer unsubscribe()

	broker.Publish("card-a", config.StatusSnapshot{State: config.StatusStateOK})

	select {
	case event := <-subscriber:
		if event.ID != 1 || event.CardID != "card-a" || event.Snapshot.State != config.StatusStateOK {
			t.Fatalf("event = %#v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for status event")
	}
}

func TestBrokerPublishesRunsInTheSameSequence(t *testing.T) {
	broker := NewBroker()
	subscriber, unsubscribe := broker.Subscribe()
	defer unsubscribe()

	broker.Publish("card-a", config.StatusSnapshot{State: config.StatusStateOK})
	run := config.Run{ActionKind: "primary", ExitCode: 3, Outcome: config.RunOutcomeFail}
	broker.PublishRun("card-b", run)
	run.ExitCode = 9 // the published event holds its own copy

	status, recorded := <-subscriber, <-subscriber
	if status.ID != 1 || status.Run != nil {
		t.Fatalf("status event = %#v", status)
	}
	if recorded.ID != 2 || recorded.CardID != "card-b" || recorded.Run == nil ||
		recorded.Run.ExitCode != 3 || recorded.Run.Outcome != config.RunOutcomeFail {
		t.Fatalf("run event = %#v", recorded)
	}
}

func TestBrokerKeepsNewestEventForSlowSubscriber(t *testing.T) {
	broker := NewBroker()
	subscriber, unsubscribe := broker.Subscribe()
	defer unsubscribe()

	for index := range subscriberBuffer + 3 {
		state := config.StatusStateOK
		if index%2 == 1 {
			state = config.StatusStateFail
		}
		broker.Publish("card", config.StatusSnapshot{State: state})
	}
	var last Event
	for len(subscriber) > 0 {
		last = <-subscriber
	}
	if last.ID != uint64(subscriberBuffer+3) {
		t.Fatalf("last buffered event ID = %d, want the newest %d", last.ID, subscriberBuffer+3)
	}
}

func TestBrokerCloseIsIdempotentAndDropsSubscribers(t *testing.T) {
	broker := NewBroker()
	subscriber, unsubscribe := broker.Subscribe()
	defer unsubscribe()
	broker.Close()
	broker.Close()
	select {
	case <-broker.Done():
	default:
		t.Fatal("Done() is not closed after Close()")
	}
	broker.Publish("card", config.StatusSnapshot{State: config.StatusStateOK})
	if len(subscriber) != 0 {
		t.Fatal("event delivered after Close()")
	}
	late, unsubscribeLate := broker.Subscribe()
	defer unsubscribeLate()
	broker.Publish("card", config.StatusSnapshot{State: config.StatusStateOK})
	if len(late) != 0 {
		t.Fatal("late subscriber received an event after Close()")
	}
}
