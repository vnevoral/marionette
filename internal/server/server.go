// Package server wires HTTP routes for the API and the embedded SPA.
package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strings"

	"marionette/internal/config"
	"marionette/internal/webui"
)

const maxJSONBodyBytes = 1 << 20

// RouterDependencies contains application services used by API routes.
type RouterDependencies struct {
	Store      *config.Store
	Actions    ActionQueue
	Notifier   PrimaryActionNotifier
	Reconciler CardReconciler
}

// NewRouter builds the top-level HTTP handler for the application. A store
// enables the card API; omitting it keeps the health endpoint and SPA fallback
// available during application composition.
func NewRouter(stores ...*config.Store) http.Handler {
	if len(stores) > 0 {
		return NewRouterWithDependencies(RouterDependencies{Store: stores[0]})
	}
	return NewRouterWithDependencies(RouterDependencies{})
}

// NewRouterWithDependencies builds the HTTP handler with application services.
func NewRouterWithDependencies(dependencies RouterDependencies) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", handleHealth)
	if dependencies.Store != nil {
		handler := cardAPI{
			store:      dependencies.Store,
			actions:    dependencies.Actions,
			notifier:   dependencies.Notifier,
			reconciler: dependencies.Reconciler,
		}
		mux.HandleFunc("GET /api/cards", handler.listCards)
		mux.HandleFunc("POST /api/cards", handler.createCard)
		mux.HandleFunc("GET /api/cards/{id}", handler.getCard)
		mux.HandleFunc("PUT /api/cards/{id}", handler.updateCard)
		mux.HandleFunc("DELETE /api/cards/{id}", handler.deleteCard)
		mux.HandleFunc("GET /api/cards/{id}/runs", handler.getRuns)
		mux.HandleFunc("GET /api/cards/{id}/status", handler.getStatus)
		mux.HandleFunc("GET /api/cards/{id}/status/history", handler.getStatusHistory)
		if dependencies.Actions != nil {
			mux.HandleFunc("POST /api/cards/{id}/actions/primary", handler.enqueuePrimary)
			mux.HandleFunc("POST /api/cards/{id}/actions/status/check", handler.enqueueStatus)
		}
	}

	mux.Handle("/", spaHandler(webui.Dist()))

	return mux
}

type cardAPI struct {
	store      *config.Store
	actions    ActionQueue
	notifier   PrimaryActionNotifier
	reconciler CardReconciler
}

func (api cardAPI) listCards(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, api.store.ListCards())
}

func (api cardAPI) createCard(w http.ResponseWriter, request *http.Request) {
	var card config.ActionCard
	if err := decodeJSON(w, request, &card); err != nil {
		return
	}
	created, err := api.store.CreateCard(card)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			writeError(w, http.StatusConflict, err)
			return
		}
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if api.reconciler != nil {
		if err := api.reconciler.Reconcile(); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
	}
	writeJSON(w, http.StatusCreated, created)
}

func (api cardAPI) getCard(w http.ResponseWriter, request *http.Request) {
	card, exists := api.store.GetCard(request.PathValue("id"))
	if !exists {
		writeError(w, http.StatusNotFound, config.ErrNotFound)
		return
	}
	writeJSON(w, http.StatusOK, card)
}

func (api cardAPI) updateCard(w http.ResponseWriter, request *http.Request) {
	var card config.ActionCard
	if err := decodeJSON(w, request, &card); err != nil {
		return
	}
	updated, err := api.store.UpdateCard(request.PathValue("id"), card)
	if err != nil {
		if errors.Is(err, config.ErrNotFound) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if api.reconciler != nil {
		if err := api.reconciler.Reconcile(); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
	}
	writeJSON(w, http.StatusOK, updated)
}

func (api cardAPI) deleteCard(w http.ResponseWriter, request *http.Request) {
	if err := api.store.DeleteCard(request.PathValue("id")); err != nil {
		if errors.Is(err, config.ErrNotFound) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if api.reconciler != nil {
		if err := api.reconciler.Reconcile(); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (api cardAPI) getRuns(w http.ResponseWriter, request *http.Request) {
	runs, err := api.store.GetRuns(request.PathValue("id"), "primary")
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, runs)
}

func (api cardAPI) getStatus(w http.ResponseWriter, request *http.Request) {
	cardID := request.PathValue("id")
	if _, exists := api.store.GetCard(cardID); !exists {
		writeError(w, http.StatusNotFound, config.ErrNotFound)
		return
	}
	snapshot, exists := api.store.GetStatus(cardID)
	if !exists {
		snapshot.State = config.StatusStateUnknown
	}
	writeJSON(w, http.StatusOK, snapshot)
}

func (api cardAPI) getStatusHistory(w http.ResponseWriter, request *http.Request) {
	changes, err := api.store.GetStatusChanges(request.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, changes)
}

func (api cardAPI) enqueuePrimary(w http.ResponseWriter, request *http.Request) {
	cardID := request.PathValue("id")
	card, exists := api.store.GetCard(cardID)
	if !exists {
		writeError(w, http.StatusNotFound, config.ErrNotFound)
		return
	}
	if err := api.actions.EnqueuePrimary(cardID, card.Primary); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if api.notifier != nil {
		if err := api.notifier.NotifyPrimaryAction(cardID); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
	}
	writeJSON(w, http.StatusAccepted, acceptedAction{CardID: cardID, ActionKind: "primary", Status: "accepted"})
}

func (api cardAPI) enqueueStatus(w http.ResponseWriter, request *http.Request) {
	cardID := request.PathValue("id")
	card, exists := api.store.GetCard(cardID)
	if !exists {
		writeError(w, http.StatusNotFound, config.ErrNotFound)
		return
	}
	if card.Status == nil {
		writeError(w, http.StatusUnprocessableEntity, errors.New("status action is not configured"))
		return
	}
	if err := api.actions.EnqueueStatus(cardID); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusAccepted, acceptedAction{CardID: cardID, ActionKind: "status", Status: "accepted"})
}

func decodeJSON(w http.ResponseWriter, request *http.Request, target any) error {
	request.Body = http.MaxBytesReader(w, request.Body, maxJSONBodyBytes)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("decode JSON: %w", err))
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("request contains multiple JSON values")
		}
		writeError(w, http.StatusBadRequest, fmt.Errorf("decode JSON: %w", err))
		return err
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// spaHandler serves static assets from the embedded filesystem and falls back
// to index.html for unknown paths so client-side (Vue Router) routes resolve.
func spaHandler(root fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(root))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path != "/" {
			if _, err := fs.Stat(root, path[1:]); err == nil {
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		data, err := fs.ReadFile(root, "index.html")
		if err != nil {
			http.Error(w, "index.html not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(data)
	})
}
