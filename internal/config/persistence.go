package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

type persistedFile struct {
	Settings Settings        `json:"settings"`
	Cards    []ActionCard    `json:"cards"`
	History  json.RawMessage `json:"history,omitempty"`
}

// LoadFile loads settings and cards from a JSON file. A missing file creates an
// empty store with default settings. Invalid configuration is returned with an
// error; invalid history is ignored with a warning.
func LoadFile(path string) (*Store, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewStore(Settings{
				HistorySize:          DefaultHistorySize,
				MaxConcurrentActions: DefaultMaxConcurrentActions,
			}), nil
		}
		return NewStore(Settings{}), fmt.Errorf("read config file %q: %w", path, err)
	}

	var file persistedFile
	if err := json.Unmarshal(data, &file); err != nil {
		return NewStore(Settings{}), fmt.Errorf("decode config file %q: %w", path, err)
	}
	if err := file.Settings.Validate(); err != nil {
		return NewStore(Settings{}), fmt.Errorf("invalid settings in config file %q: %w", path, err)
	}

	store := NewStore(file.Settings)
	for _, card := range file.Cards {
		if err := card.Validate(); err != nil {
			return NewStore(Settings{}), fmt.Errorf("invalid card %q in config file %q: %w", card.ID, path, err)
		}
		if _, exists := store.cards[card.ID]; exists {
			return NewStore(Settings{}), fmt.Errorf("duplicate card %q in config file %q", card.ID, path)
		}
		store.cards[card.ID] = cloneCard(card)
	}

	loadHistory(store, file.History, path)
	return store, nil
}

// SaveFile atomically saves settings and cards without run history.
func (store *Store) SaveFile(path string) error {
	return writePersistedFile(path, store.snapshot(false))
}

// SaveFileWithHistory atomically saves settings, cards, and run history.
func (store *Store) SaveFileWithHistory(path string) error {
	return writePersistedFile(path, store.snapshot(true))
}

func (store *Store) snapshot(includeHistory bool) persistedFile {
	store.mu.RLock()
	defer store.mu.RUnlock()

	cards := make([]ActionCard, 0, len(store.cards))
	for _, card := range store.cards {
		cards = append(cards, cloneCard(card))
	}
	file := persistedFile{
		Settings: store.settings,
		Cards:    cards,
	}
	if includeHistory {
		file.History = marshalHistory(store.history)
	}
	return file
}

func marshalHistory(history map[string]map[string][]Run) json.RawMessage {
	if len(history) == 0 {
		return json.RawMessage(`{}`)
	}

	copyHistory := make(map[string]map[string][]Run, len(history))
	for cardID, histories := range history {
		copyHistory[cardID] = make(map[string][]Run, len(histories))
		for actionKind, runs := range histories {
			copyHistory[cardID][actionKind] = append([]Run(nil), runs...)
		}
	}
	data, err := json.Marshal(copyHistory)
	if err != nil {
		return nil
	}
	return data
}

func loadHistory(store *Store, raw json.RawMessage, path string) {
	if len(bytes.TrimSpace(raw)) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return
	}

	var history map[string]map[string][]Run
	if err := json.Unmarshal(raw, &history); err != nil {
		log.Printf("warning: ignoring invalid history in config file %q: %v", path, err)
		return
	}
	for cardID, histories := range history {
		if _, exists := store.cards[cardID]; !exists {
			log.Printf("warning: ignoring history for unknown card %q in config file %q", cardID, path)
			continue
		}
		store.history[cardID] = make(map[string][]Run, len(histories))
		for actionKind, runs := range histories {
			store.history[cardID][actionKind] = trimRuns(append([]Run(nil), runs...), store.settings.HistorySize)
		}
	}
}

func writePersistedFile(path string, file persistedFile) error {
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config file %q: %w", path, err)
	}

	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, "marionette-*.json")
	if err != nil {
		return fmt.Errorf("create temporary config file in %q: %w", directory, err)
	}
	temporaryName := temporary.Name()
	cleanup := func() {
		_ = temporary.Close()
		_ = os.Remove(temporaryName)
	}

	if _, err := temporary.Write(data); err != nil {
		cleanup()
		return fmt.Errorf("write temporary config file %q: %w", temporaryName, err)
	}
	if err := temporary.Sync(); err != nil {
		cleanup()
		return fmt.Errorf("sync temporary config file %q: %w", temporaryName, err)
	}
	if err := temporary.Close(); err != nil {
		_ = os.Remove(temporaryName)
		return fmt.Errorf("close temporary config file %q: %w", temporaryName, err)
	}
	if err := os.Rename(temporaryName, path); err != nil {
		_ = os.Remove(temporaryName)
		return fmt.Errorf("replace config file %q: %w", path, err)
	}
	return nil
}
