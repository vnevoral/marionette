package config

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"sync"
)

// ErrNotFound indicates that a requested card does not exist in the store.
var ErrNotFound = errors.New("config item not found")

// Store keeps action-card configuration and run history in memory.
type Store struct {
	mu        sync.RWMutex
	persistMu sync.Mutex
	settings  Settings
	cards     map[string]ActionCard
	history   map[string]map[string][]Run

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
		settings: settings,
		cards:    make(map[string]ActionCard),
		history:  make(map[string]map[string][]Run),
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
		return created, fmt.Errorf("persist card creation: %w", err)
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
		return updated, fmt.Errorf("persist card update: %w", err)
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
	store.mu.Unlock()
	if err := store.notifyChange(); err != nil {
		return fmt.Errorf("persist card deletion: %w", err)
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
	store.mu.Unlock()
	if err := store.notifyChange(); err != nil {
		return fmt.Errorf("persist settings update: %w", err)
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
