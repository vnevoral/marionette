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

// ErrNotFound indicates that a requested card does not exist in the store.
var ErrNotFound = errors.New("config item not found")

// PersistenceError indicates that a store mutation succeeded in memory but
// could not be persisted to the configured backing file.
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
	mu            sync.RWMutex
	persistMu     sync.Mutex
	settings      Settings
	cards         map[string]ActionCard
	history       map[string]map[string][]Run
	statuses      map[string]StatusSnapshot
	statusHistory map[string][]StatusChange

	// OnChange is called after a successful configuration mutation. It is not
	// called for AppendRun because run history is saved only at shutdown.
	OnChange func(*Store) error
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
	}
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

// CreateCard validates and stores a card. A missing ID is generated.
func (store *Store) CreateCard(card ActionCard) (ActionCard, error) {
	store.mu.Lock()

	if card.ID == "" {
		id, err := generateCardID()
		if err != nil {
			store.mu.Unlock()
			return ActionCard{}, fmt.Errorf("generate card id: %w", err)
		}
		card.ID = id
	}
	if err := card.Validate(); err != nil {
		store.mu.Unlock()
		return ActionCard{}, err
	}
	if _, exists := store.cards[card.ID]; exists {
		store.mu.Unlock()
		return ActionCard{}, fmt.Errorf("card %q already exists", card.ID)
	}

	card = cloneCard(card)
	store.cards[card.ID] = card
	created := cloneCard(card)
	store.mu.Unlock()
	if err := store.notifyChange(); err != nil {
		return created, &PersistenceError{Operation: "card creation", Err: err}
	}
	return created, nil
}

// UpdateCard validates and replaces an existing card while retaining the
// identifier supplied by the caller.
func (store *Store) UpdateCard(id string, card ActionCard) (ActionCard, error) {
	store.mu.Lock()

	if _, exists := store.cards[id]; !exists {
		store.mu.Unlock()
		return ActionCard{}, ErrNotFound
	}
	card.ID = id
	if err := card.Validate(); err != nil {
		store.mu.Unlock()
		return ActionCard{}, err
	}

	card = cloneCard(card)
	store.cards[id] = card
	updated := cloneCard(card)
	store.mu.Unlock()
	if err := store.notifyChange(); err != nil {
		return updated, &PersistenceError{Operation: "card update", Err: err}
	}
	return updated, nil
}

// DeleteCard removes a card and all run history associated with it.
func (store *Store) DeleteCard(id string) error {
	store.mu.Lock()

	if _, exists := store.cards[id]; !exists {
		store.mu.Unlock()
		return ErrNotFound
	}
	delete(store.cards, id)
	delete(store.history, id)
	delete(store.statuses, id)
	delete(store.statusHistory, id)
	store.mu.Unlock()
	if err := store.notifyChange(); err != nil {
		return &PersistenceError{Operation: "card deletion", Err: err}
	}
	return nil
}

// GetSettings returns the current global settings.
func (store *Store) GetSettings() Settings {
	store.mu.RLock()
	defer store.mu.RUnlock()
	return store.settings
}

// UpdateSettings validates and replaces the global settings. Histories are
// trimmed to the newest records when the history limit is reduced.
func (store *Store) UpdateSettings(settings Settings) error {
	if err := settings.Validate(); err != nil {
		return err
	}

	store.mu.Lock()
	store.settings = settings
	for cardID, histories := range store.history {
		for actionKind, runs := range histories {
			store.history[cardID][actionKind] = trimRuns(runs, settings.HistorySize)
		}
	}
	for cardID, changes := range store.statusHistory {
		store.statusHistory[cardID] = trimStatusChanges(changes, settings.HistorySize)
	}
	store.mu.Unlock()
	if err := store.notifyChange(); err != nil {
		return &PersistenceError{Operation: "settings update", Err: err}
	}
	return nil
}

// AppendRun adds a run to the card and action history, retaining only the
// newest Settings.HistorySize records.
func (store *Store) AppendRun(cardID string, run Run) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	if _, exists := store.cards[cardID]; !exists {
		return ErrNotFound
	}
	if store.history[cardID] == nil {
		store.history[cardID] = make(map[string][]Run)
	}

	runs := store.history[cardID][run.ActionKind]
	runs = append([]Run{run}, runs...)
	store.history[cardID][run.ActionKind] = trimRuns(runs, store.settings.HistorySize)
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
	return append([]Run(nil), runs...), nil
}

// GetStatus returns the latest status snapshot for a card.
func (store *Store) GetStatus(cardID string) (StatusSnapshot, bool) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	if _, exists := store.cards[cardID]; !exists {
		return StatusSnapshot{}, false
	}
	snapshot, exists := store.statuses[cardID]
	return cloneStatusSnapshot(snapshot), exists
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
// when the interpreted state differs from the previous state.
func (store *Store) UpdateStatus(cardID string, snapshot StatusSnapshot) error {
	if snapshot.State != StatusStateUnknown && snapshot.State != StatusStateOK && snapshot.State != StatusStateFail {
		return fmt.Errorf("unsupported status state %q", snapshot.State)
	}
	if snapshot.CheckedAt.IsZero() {
		snapshot.CheckedAt = time.Now()
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	if _, exists := store.cards[cardID]; !exists {
		return ErrNotFound
	}

	previous, exists := store.statuses[cardID]
	if exists && previous.State == snapshot.State {
		store.statuses[cardID] = cloneStatusSnapshot(snapshot)
		return nil
	}

	changes := store.statusHistory[cardID]
	if len(changes) > 0 {
		last := &changes[0]
		if snapshot.CheckedAt.Before(last.StartedAt) {
			return fmt.Errorf("status check time precedes current status change")
		}
		endedAt := snapshot.CheckedAt
		last.EndedAt = &endedAt
		last.Duration = endedAt.Sub(last.StartedAt)
	}
	change := StatusChange{State: snapshot.State, StartedAt: snapshot.CheckedAt}
	store.statusHistory[cardID] = trimStatusChanges(append([]StatusChange{change}, changes...), store.settings.HistorySize)
	store.statuses[cardID] = cloneStatusSnapshot(snapshot)
	return nil
}

func (store *Store) notifyChange() error {
	store.persistMu.Lock()
	defer store.persistMu.Unlock()

	if store.OnChange == nil {
		return nil
	}
	return store.OnChange(store)
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

func cloneStatusSnapshot(snapshot StatusSnapshot) StatusSnapshot {
	return snapshot
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
