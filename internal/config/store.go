package config

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
)

// Errors returned by store operations. They are wrapped with context, so
// callers classify them with errors.Is.
var (
	// ErrNotFound indicates that a requested card does not exist in the store.
	ErrNotFound = errors.New("config item not found")
	// ErrAlreadyExists indicates that a card with the same ID is already stored.
	ErrAlreadyExists = errors.New("config item already exists")
	// ErrValidation indicates that the supplied configuration is invalid.
	ErrValidation = errors.New("invalid configuration")
)

// PersistenceError indicates that a store mutation could not be persisted to
// the configured backing file. The in-memory change has been rolled back, so
// the store still matches the last successfully persisted state.
type PersistenceError struct {
	Operation string
	Err       error
}

func (err *PersistenceError) Error() string {
	return fmt.Sprintf("persist %s: %v", err.Operation, err.Err)
}

func (err *PersistenceError) Unwrap() error {
	return err.Err
}

// Store keeps action-card configuration and run history in memory.
type Store struct {
	mu sync.RWMutex
	// persistMu serialises persisted mutations from the in-memory change up to
	// and including OnChange, so a failed OnChange can be rolled back without
	// interleaving with another persisted mutation.
	persistMu     sync.Mutex
	settings      Settings
	cards         map[string]ActionCard
	history       map[string]map[string][]Run
	statuses      map[string]StatusSnapshot
	statusHistory map[string][]StatusChange
	// waiting holds the cards that wait for a status check to confirm a
	// primary action (FR-15b). statuses keeps the last confirmed state, so
	// a waiting card is never persisted or recorded in the status history.
	waiting map[string]waitingStatus
	// changes counts every in-memory mutation; savedChanges is the value
	// captured by the last successful SaveFileWithHistory. They differ while
	// the file on disk lags behind memory (see Dirty).
	changes      uint64
	savedChanges uint64

	// OnChange is called after a successful configuration mutation. It is not
	// called for AppendRun because run history is saved only at shutdown.
	OnChange func(*Store) error

	// OnStatusChange is called after a status transition is stored. It runs
	// outside the store lock and is intended for transient runtime consumers;
	// the composition root wires it to the event broker. StatusSnapshot is a
	// value type without references, so callers receive an independent copy.
	OnStatusChange func(string, StatusSnapshot)

	// OnRunAppended is called after a run is added to the history (FR-42a).
	// Like OnStatusChange it runs outside the store lock and receives a copy;
	// the composition root wires it to the event broker.
	OnRunAppended func(string, Run)
}

// NewStore creates an empty store. Invalid settings are replaced with the
// recommended default settings because the constructor cannot return an error.
func NewStore(settings Settings) *Store {
	if err := settings.Validate(); err != nil {
		settings = Settings{
			HistorySize:          DefaultHistorySize,
			MaxConcurrentActions: DefaultMaxConcurrentActions,
		}
	}

	return &Store{
		settings:      settings,
		cards:         make(map[string]ActionCard),
		history:       make(map[string]map[string][]Run),
		statuses:      make(map[string]StatusSnapshot),
		statusHistory: make(map[string][]StatusChange),
		waiting:       make(map[string]waitingStatus),
	}
}

// waitingStatus is the confirmation window of one card and the latest
// check that did not confirm the action yet.
type waitingStatus struct {
	until time.Time
	check *StatusSnapshot
}

// ListCards returns all cards sorted by ID. Returned cards are independent
// copies and can be safely mutated by the caller.
func (store *Store) ListCards() []ActionCard {
	store.mu.RLock()
	defer store.mu.RUnlock()

	cards := make([]ActionCard, 0, len(store.cards))
	for _, card := range store.cards {
		cards = append(cards, cloneCard(card))
	}
	sort.Slice(cards, func(i, j int) bool { return cards[i].ID < cards[j].ID })
	return cards
}

// GetCard returns a card copy and whether the card exists.
func (store *Store) GetCard(id string) (ActionCard, bool) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	card, ok := store.cards[id]
	if !ok {
		return ActionCard{}, false
	}
	return cloneCard(card), true
}

// mutate runs one persisted configuration change under the store's
// protocol: apply the change in memory (under mu), count it, run OnChange
// outside mu, and on failure undo the change, uncount it and report a
// PersistenceError for operation. apply returns the undo step; an error from
// apply (validation, not found, …) is returned unchanged and nothing is
// counted.
func (store *Store) mutate(operation string, apply func() (undo func(), err error)) error {
	store.persistMu.Lock()
	defer store.persistMu.Unlock()
	store.mu.Lock()
	undo, err := apply()
	if err != nil {
		store.mu.Unlock()
		return err
	}
	store.changes++
	store.mu.Unlock()
	if err := store.notifyChange(); err != nil {
		store.mu.Lock()
		undo()
		store.changes--
		store.mu.Unlock()
		return &PersistenceError{Operation: operation, Err: err}
	}
	return nil
}

// CreateCard validates and stores a card. A missing ID is generated.
func (store *Store) CreateCard(card ActionCard) (ActionCard, error) {
	var created ActionCard
	err := store.mutate("card creation", func() (func(), error) {
		if card.ID == "" {
			id, err := generateCardID()
			if err != nil {
				return nil, fmt.Errorf("generate card id: %w", err)
			}
			card.ID = id
		}
		if err := card.Validate(); err != nil {
			return nil, err
		}
		if _, exists := store.cards[card.ID]; exists {
			return nil, fmt.Errorf("card %q: %w", card.ID, ErrAlreadyExists)
		}
		card = cloneCard(card)
		store.cards[card.ID] = card
		created = cloneCard(card)
		return func() { delete(store.cards, card.ID) }, nil
	})
	if err != nil {
		return ActionCard{}, err
	}
	return created, nil
}

// UpdateCard validates and replaces an existing card while retaining the
// identifier supplied by the caller.
func (store *Store) UpdateCard(id string, card ActionCard) (ActionCard, error) {
	var updated ActionCard
	err := store.mutate("card update", func() (func(), error) {
		previous, exists := store.cards[id]
		if !exists {
			return nil, ErrNotFound
		}
		card.ID = id
		if err := card.Validate(); err != nil {
			return nil, err
		}
		card = cloneCard(card)
		store.cards[id] = card
		updated = cloneCard(card)
		return func() { store.cards[id] = previous }, nil
	})
	if err != nil {
		return ActionCard{}, err
	}
	return updated, nil
}

// DeleteCard removes a card and all run history associated with it.
func (store *Store) DeleteCard(id string) error {
	return store.mutate("card deletion", func() (func(), error) {
		previousCard, exists := store.cards[id]
		if !exists {
			return nil, ErrNotFound
		}
		previousHistory, hadHistory := store.history[id]
		previousStatus, hadStatus := store.statuses[id]
		previousStatusHistory, hadStatusHistory := store.statusHistory[id]
		delete(store.cards, id)
		delete(store.history, id)
		previousWaiting, wasWaiting := store.waiting[id]
		delete(store.statuses, id)
		delete(store.statusHistory, id)
		delete(store.waiting, id)
		return func() {
			if wasWaiting {
				store.waiting[id] = previousWaiting
			}
			store.cards[id] = previousCard
			if hadHistory {
				store.history[id] = previousHistory
			}
			if hadStatus {
				store.statuses[id] = previousStatus
			}
			if hadStatusHistory {
				store.statusHistory[id] = previousStatusHistory
			}
		}, nil
	})
}

// Dirty reports whether the in-memory state (configuration, run history or
// status projection) changed since the last successful SaveFileWithHistory.
// A freshly loaded store is clean. Shutdown uses it to skip a second history
// save when nothing changed while the queue and scheduler were stopping.
func (store *Store) Dirty() bool {
	store.mu.RLock()
	defer store.mu.RUnlock()
	return store.changes != store.savedChanges
}

// GetSettings returns the current global settings.
func (store *Store) GetSettings() Settings {
	store.mu.RLock()
	defer store.mu.RUnlock()
	return store.settings
}

// AppendRun adds a run to the card and action history, retaining only the
// newest Settings.HistorySize records.
func (store *Store) AppendRun(cardID string, run Run) error {
	store.mu.Lock()
	if _, exists := store.cards[cardID]; !exists {
		store.mu.Unlock()
		return ErrNotFound
	}
	if store.history[cardID] == nil {
		store.history[cardID] = make(map[string][]Run)
	}

	runs := store.history[cardID][run.ActionKind]
	runs = append([]Run{run}, runs...)
	store.history[cardID][run.ActionKind] = trimRuns(runs, store.settings.HistorySize)
	store.changes++
	onRunAppended := store.OnRunAppended
	store.mu.Unlock()
	if onRunAppended != nil {
		onRunAppended(cardID, run)
	}
	return nil
}

// GetRuns returns the newest run first. The returned slice is an independent
// copy and an unknown action kind returns an empty slice without an error.
func (store *Store) GetRuns(cardID string, actionKind string) ([]Run, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	if _, exists := store.cards[cardID]; !exists {
		return nil, ErrNotFound
	}
	runs := store.history[cardID][actionKind]
	// Never nil: an empty history is served as [] in the API, not null.
	return append(make([]Run, 0, len(runs)), runs...), nil
}

// GetStatus returns the latest status snapshot for a card. A waiting card
// reports the waiting state with its latest check.
func (store *Store) GetStatus(cardID string) (StatusSnapshot, bool) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	if _, exists := store.cards[cardID]; !exists {
		return StatusSnapshot{}, false
	}
	if _, waiting := store.waiting[cardID]; waiting {
		return store.visibleStatusLocked(cardID), true
	}
	snapshot, exists := store.statuses[cardID]
	return snapshot, exists
}

// visibleStatusLocked is the snapshot clients see: the confirmed one, or
// while the card waits, its latest check in the waiting state. The caller
// holds mu.
func (store *Store) visibleStatusLocked(cardID string) StatusSnapshot {
	snapshot := store.statuses[cardID]
	if snapshot.State == "" {
		snapshot.State = StatusStateUnknown
	}
	wait, waiting := store.waiting[cardID]
	if !waiting {
		return snapshot
	}
	if wait.check != nil {
		snapshot = *wait.check
	}
	until := wait.until
	snapshot.State = StatusStateWaiting
	snapshot.WaitingUntil = &until
	return snapshot
}

// WaitingUntil returns the end of the card's confirmation window while it
// waits for a status check to confirm a primary action.
func (store *Store) WaitingUntil(cardID string) (time.Time, bool) {
	store.mu.RLock()
	defer store.mu.RUnlock()
	wait, waiting := store.waiting[cardID]
	return wait.until, waiting
}

// BeginWaiting puts the card into the waiting state until the given time,
// or moves the end of a window that is already open (FR-15b). Checks that
// do not confirm the action keep the card waiting; see UpdateStatus.
func (store *Store) BeginWaiting(cardID string, until time.Time) error {
	store.mu.Lock()
	if _, exists := store.cards[cardID]; !exists {
		store.mu.Unlock()
		return ErrNotFound
	}
	wait := store.waiting[cardID]
	wait.until = until
	store.waiting[cardID] = wait
	snapshot := store.visibleStatusLocked(cardID)
	onStatusChange := store.OnStatusChange
	store.mu.Unlock()
	if onStatusChange != nil {
		onStatusChange(cardID, snapshot)
	}
	return nil
}

// ExtendWaiting moves the end of the card's confirmation window if the card
// still waits; a wait that a healthy check already ended is not reopened.
func (store *Store) ExtendWaiting(cardID string, until time.Time) error {
	store.mu.Lock()
	wait, waiting := store.waiting[cardID]
	if !waiting {
		store.mu.Unlock()
		return nil
	}
	wait.until = until
	store.waiting[cardID] = wait
	snapshot := store.visibleStatusLocked(cardID)
	onStatusChange := store.OnStatusChange
	store.mu.Unlock()
	if onStatusChange != nil {
		onStatusChange(cardID, snapshot)
	}
	return nil
}

// EndWaiting closes the card's confirmation window: the latest check made
// while waiting becomes the card's state, as if it had just been stored, and
// without one the card returns to its confirmed state. It does nothing for
// a card that is not waiting.
func (store *Store) EndWaiting(cardID string) error {
	store.mu.Lock()
	wait, waiting := store.waiting[cardID]
	if !waiting {
		store.mu.Unlock()
		return nil
	}
	if wait.check != nil {
		// A closed window makes UpdateStatus store the check and announce
		// the end of the wait.
		store.waiting[cardID] = waitingStatus{}
		store.mu.Unlock()
		return store.UpdateStatus(cardID, *wait.check)
	}
	delete(store.waiting, cardID)
	snapshot := store.visibleStatusLocked(cardID)
	onStatusChange := store.OnStatusChange
	store.mu.Unlock()
	if onStatusChange != nil {
		onStatusChange(cardID, snapshot)
	}
	return nil
}

// GetStatusChanges returns the newest status transitions first.
func (store *Store) GetStatusChanges(cardID string) ([]StatusChange, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	if _, exists := store.cards[cardID]; !exists {
		return nil, ErrNotFound
	}
	changes := store.statusHistory[cardID]
	return cloneStatusChanges(changes), nil
}

// UpdateStatus stores the latest status check and records a transition only
// when the interpreted state differs from the previous state. While the card
// waits (FR-15b), a check that does not confirm the action and ends before
// the window closes is only kept as the latest check; a healthy check, or
// one that ends when the window has closed, ends the wait and is stored as
// usual.
func (store *Store) UpdateStatus(cardID string, snapshot StatusSnapshot) error {
	if !validStatusState(snapshot.State) {
		return fmt.Errorf("%w: unsupported status state %q", ErrValidation, snapshot.State)
	}
	if snapshot.CheckedAt.IsZero() {
		snapshot.CheckedAt = time.Now()
	}
	snapshot.WaitingUntil = nil

	store.mu.Lock()
	if _, exists := store.cards[cardID]; !exists {
		store.mu.Unlock()
		return ErrNotFound
	}

	// A wait that ends shows a new state even when the confirmed one is
	// unchanged, so it is always announced.
	endedWait := false
	if wait, waiting := store.waiting[cardID]; waiting {
		if snapshot.State != StatusStateOK && snapshot.CheckedAt.Before(wait.until) {
			check := snapshot
			wait.check = &check
			store.waiting[cardID] = wait
			store.mu.Unlock()
			return nil
		}
		delete(store.waiting, cardID)
		endedWait = true
	}

	previous, exists := store.statuses[cardID]
	if exists && previous.State == snapshot.State {
		store.statuses[cardID] = snapshot
		store.changes++
		onStatusChange := store.OnStatusChange
		store.mu.Unlock()
		if endedWait && onStatusChange != nil {
			onStatusChange(cardID, snapshot)
		}
		return nil
	}

	changes := store.statusHistory[cardID]
	if len(changes) > 0 {
		last := &changes[0]
		if snapshot.CheckedAt.Before(last.StartedAt) {
			store.mu.Unlock()
			return fmt.Errorf("status check time precedes current status change")
		}
		endedAt := snapshot.CheckedAt
		last.EndedAt = &endedAt
		last.Duration = endedAt.Sub(last.StartedAt)
	}
	change := StatusChange{State: snapshot.State, StartedAt: snapshot.CheckedAt}
	store.statusHistory[cardID] = trimStatusChanges(append([]StatusChange{change}, changes...), store.settings.HistorySize)
	store.statuses[cardID] = snapshot
	store.changes++
	onStatusChange := store.OnStatusChange
	store.mu.Unlock()
	if onStatusChange != nil {
		onStatusChange(cardID, snapshot)
	}
	return nil
}

// notifyChange runs OnChange. The caller must hold persistMu and must not
// hold mu, because OnChange typically snapshots the store under a read lock.
func (store *Store) notifyChange() error {
	if store.OnChange == nil {
		return nil
	}
	return store.OnChange(store)
}

func validStatusState(state StatusState) bool {
	return state == StatusStateUnknown || state == StatusStateOK || state == StatusStateFail
}

func generateCardID() (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	return "card-" + hex.EncodeToString(bytes[:]), nil
}

func trimRuns(runs []Run, limit int) []Run {
	if len(runs) <= limit {
		return runs
	}
	return runs[:limit]
}

func trimStatusChanges(changes []StatusChange, limit int) []StatusChange {
	if len(changes) <= limit {
		return changes
	}
	return changes[:limit]
}

func cloneStatusChanges(changes []StatusChange) []StatusChange {
	cloned := make([]StatusChange, len(changes))
	copy(cloned, changes)
	for index := range cloned {
		if changes[index].EndedAt != nil {
			endedAt := *changes[index].EndedAt
			cloned[index].EndedAt = &endedAt
		}
	}
	return cloned
}

func cloneCard(card ActionCard) ActionCard {
	card.Primary = cloneAction(card.Primary)
	if card.Status != nil {
		status := cloneAction(*card.Status)
		card.Status = &status
	}
	return card
}

func cloneAction(action Action) Action {
	action.Args = append([]string(nil), action.Args...)
	if action.Env != nil {
		clonedEnv := make(map[string]string, len(action.Env))
		for key, value := range action.Env {
			clonedEnv[key] = value
		}
		action.Env = clonedEnv
	}
	return action
}
