package config

import (
	"testing"
	"time"
)

// waitingStore returns a store with one card whose confirmed state is fail,
// the time of that check and a channel of the announced states.
func waitingStore(t *testing.T) (*Store, string, time.Time, chan StatusState) {
	t.Helper()
	store := NewStore(validSettings())
	card, err := store.CreateCard(validCard())
	if err != nil {
		t.Fatalf("CreateCard() error = %v", err)
	}
	base := time.Date(2026, time.October, 21, 12, 0, 0, 0, time.UTC)
	if err := store.UpdateStatus(card.ID, StatusSnapshot{State: StatusStateFail, CheckedAt: base}); err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}
	announced := make(chan StatusState, 8)
	store.OnStatusChange = func(_ string, snapshot StatusSnapshot) { announced <- snapshot.State }
	return store, card.ID, base, announced
}

func expectAnnounced(t *testing.T, announced chan StatusState, want ...StatusState) {
	t.Helper()
	for _, state := range want {
		select {
		case got := <-announced:
			if got != state {
				t.Fatalf("announced %q, want %q", got, state)
			}
		default:
			t.Fatalf("no announcement, want %q", state)
		}
	}
	select {
	case extra := <-announced:
		t.Fatalf("unexpected announcement %q", extra)
	default:
	}
}

func TestWaitingKeepsFailedChecksOutOfTheHistoryUntilAHealthyCheck(t *testing.T) {
	store, cardID, base, announced := waitingStore(t)
	until := base.Add(2 * time.Minute)
	if err := store.BeginWaiting(cardID, until); err != nil {
		t.Fatalf("BeginWaiting() error = %v", err)
	}
	expectAnnounced(t, announced, StatusStateWaiting)

	if err := store.UpdateStatus(cardID, StatusSnapshot{State: StatusStateFail, CheckedAt: base.Add(10 * time.Second)}); err != nil {
		t.Fatalf("UpdateStatus(fail) error = %v", err)
	}
	expectAnnounced(t, announced)
	snapshot, _ := store.GetStatus(cardID)
	if snapshot.State != StatusStateWaiting || !snapshot.CheckedAt.Equal(base.Add(10*time.Second)) || snapshot.WaitingUntil == nil || !snapshot.WaitingUntil.Equal(until) {
		t.Fatalf("snapshot while waiting = %#v", snapshot)
	}

	if err := store.UpdateStatus(cardID, StatusSnapshot{State: StatusStateOK, CheckedAt: base.Add(20 * time.Second)}); err != nil {
		t.Fatalf("UpdateStatus(ok) error = %v", err)
	}
	expectAnnounced(t, announced, StatusStateOK)
	if _, waiting := store.WaitingUntil(cardID); waiting {
		t.Fatal("card still waits after a healthy check")
	}
	changes, _ := store.GetStatusChanges(cardID)
	if len(changes) != 2 || changes[0].State != StatusStateOK || changes[1].State != StatusStateFail {
		t.Fatalf("status changes = %#v", changes)
	}
}

func TestWaitingStoresAFailedCheckOnceTheWindowHasClosed(t *testing.T) {
	store, cardID, base, announced := waitingStore(t)
	if err := store.UpdateStatus(cardID, StatusSnapshot{State: StatusStateOK, CheckedAt: base.Add(time.Second)}); err != nil {
		t.Fatalf("UpdateStatus(ok) error = %v", err)
	}
	expectAnnounced(t, announced, StatusStateOK)
	until := base.Add(time.Minute)
	if err := store.BeginWaiting(cardID, until); err != nil {
		t.Fatalf("BeginWaiting() error = %v", err)
	}
	expectAnnounced(t, announced, StatusStateWaiting)

	if err := store.UpdateStatus(cardID, StatusSnapshot{State: StatusStateFail, CheckedAt: until}); err != nil {
		t.Fatalf("UpdateStatus(fail) error = %v", err)
	}
	expectAnnounced(t, announced, StatusStateFail)
	changes, _ := store.GetStatusChanges(cardID)
	if len(changes) != 3 || changes[0].State != StatusStateFail || !changes[0].StartedAt.Equal(until) {
		t.Fatalf("status changes = %#v", changes)
	}
}

func TestEndWaitingStoresTheLatestCheckOrRestoresTheConfirmedState(t *testing.T) {
	store, cardID, base, announced := waitingStore(t)
	if err := store.BeginWaiting(cardID, base.Add(time.Minute)); err != nil {
		t.Fatalf("BeginWaiting() error = %v", err)
	}
	if err := store.EndWaiting(cardID); err != nil {
		t.Fatalf("EndWaiting() error = %v", err)
	}
	expectAnnounced(t, announced, StatusStateWaiting, StatusStateFail)
	if snapshot, _ := store.GetStatus(cardID); snapshot.State != StatusStateFail || !snapshot.CheckedAt.Equal(base) || snapshot.WaitingUntil != nil {
		t.Fatalf("snapshot after an empty wait = %#v", snapshot)
	}

	if err := store.BeginWaiting(cardID, base.Add(time.Minute)); err != nil {
		t.Fatalf("BeginWaiting() error = %v", err)
	}
	if err := store.UpdateStatus(cardID, StatusSnapshot{State: StatusStateFail, CheckedAt: base.Add(10 * time.Second)}); err != nil {
		t.Fatalf("UpdateStatus(fail) error = %v", err)
	}
	if err := store.EndWaiting(cardID); err != nil {
		t.Fatalf("EndWaiting() error = %v", err)
	}
	expectAnnounced(t, announced, StatusStateWaiting, StatusStateFail)
	if snapshot, _ := store.GetStatus(cardID); snapshot.State != StatusStateFail || !snapshot.CheckedAt.Equal(base.Add(10*time.Second)) {
		t.Fatalf("snapshot after a wait with a check = %#v", snapshot)
	}
	if changes, _ := store.GetStatusChanges(cardID); len(changes) != 1 {
		t.Fatalf("status changes = %#v", changes)
	}
	if err := store.EndWaiting(cardID); err != nil {
		t.Fatalf("EndWaiting() without a wait error = %v", err)
	}
	expectAnnounced(t, announced)
}

func TestWaitingIsNotPersisted(t *testing.T) {
	store, cardID, base, _ := waitingStore(t)
	if err := store.BeginWaiting(cardID, base.Add(time.Minute)); err != nil {
		t.Fatalf("BeginWaiting() error = %v", err)
	}
	if status := store.snapshot(true).Status[cardID]; status.State != StatusStateFail || status.WaitingUntil != nil {
		t.Fatalf("persisted status = %#v", status)
	}
	if err := store.DeleteCard(cardID); err != nil {
		t.Fatalf("DeleteCard() error = %v", err)
	}
	if _, waiting := store.WaitingUntil(cardID); waiting {
		t.Fatal("deleted card still waits")
	}
}

func TestBeginWaitingOnAnUncheckedCardMarshalsTheWindow(t *testing.T) {
	store := NewStore(validSettings())
	card, err := store.CreateCard(validCard())
	if err != nil {
		t.Fatalf("CreateCard() error = %v", err)
	}
	until := time.Date(2026, time.October, 21, 12, 2, 0, 0, time.UTC)
	if err := store.BeginWaiting(card.ID, until); err != nil {
		t.Fatalf("BeginWaiting() error = %v", err)
	}
	snapshot, _ := store.GetStatus(card.ID)
	data, err := snapshot.MarshalJSON()
	if err != nil || string(data) != `{"state":"waiting","waitingUntil":"2026-10-21T12:02:00Z"}` {
		t.Fatalf("MarshalJSON() = %s, %v", data, err)
	}
	if err := store.BeginWaiting("missing", until); err == nil {
		t.Fatal("BeginWaiting() on a missing card succeeded")
	}
}
