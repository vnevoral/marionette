// Package server wires HTTP routes for the API and the embedded SPA.
package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"marionette/internal/actions"
	"marionette/internal/config"
	"marionette/internal/events"
	"marionette/internal/webui"
)

const maxJSONBodyBytes = 1 << 20

// ActionQueue accepts actions for background execution.
type ActionQueue interface {
	EnqueuePrimary(string, config.Action) error
	EnqueueStatus(string) error
}

// PrimaryActionNotifier activates fast polling after a primary action is accepted.
type PrimaryActionNotifier interface {
	NotifyPrimaryAction(string) error
}

// CardReconciler updates background workers after card configuration changes.
type CardReconciler interface {
	Reconcile() error
}

// EventSource hands out subscriptions to status transitions for the SSE
// endpoint and reports when no further events will ever be published.
type EventSource interface {
	Subscribe() (<-chan events.Event, func())
	Done() <-chan struct{}
}

// Dependencies are the application services used by the API routes. Only
// Store is required for the card API; without it the router serves the
// health endpoint and the SPA only. The router never mutates the store; the
// composition root wires Store.OnStatusChange to the event broker.
type Dependencies struct {
	Store      *config.Store
	Actions    ActionQueue
	Notifier   PrimaryActionNotifier
	Reconciler CardReconciler
	// Events feeds GET /api/events; nil serves heartbeats only.
	Events EventSource
	// AllowedHosts optionally restricts the Host header accepted by mutating
	// API requests (NFR-12, MARIONETTE_ALLOWED_HOSTS). Empty disables the check.
	AllowedHosts []string
	// Version is reported by GET /api/health; empty means "dev".
	Version string
	// StartedAt is the process start used for the health uptime; zero means now.
	StartedAt time.Time
	// Logger receives handler warnings; nil discards them.
	Logger *slog.Logger
	// Access enables device pairing; nil (or a nil Registry) leaves the API
	// open (MARIONETTE_AUTH=off).
	Access *Access
}

// NewRouter builds the top-level HTTP handler. Mutating API routes are
// wrapped by the cross-site protection (NFR-12) and every API path gets a
// fallback for unsupported methods that answers 405 with an Allow header and
// the JSON error envelope.
func NewRouter(dependencies Dependencies) http.Handler {
	routes := routeTable{}
	routes.add(http.MethodGet, "/api/health", newHealthHandler(dependencies.Version, dependencies.StartedAt))
	logger := dependencies.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	var accessHandler *accessAPI
	if dependencies.Access != nil && dependencies.Access.Registry != nil {
		accessHandler = &accessAPI{registry: dependencies.Access.Registry, secure: dependencies.Access.Secure, logger: logger}
		routes.add(http.MethodGet, "/api/session", accessHandler.session)
		routes.add(http.MethodPost, "/api/pairing", accessHandler.pair)
		routes.add(http.MethodGet, "/api/devices", accessHandler.listDevices)
		routes.add(http.MethodPost, "/api/pairing/code", accessHandler.createPairingCode)
		routes.add(http.MethodDelete, "/api/devices/{id}", accessHandler.removeDevice)
		routes.add(http.MethodPatch, "/api/devices/{id}", accessHandler.renameDevice)
	}
	if dependencies.Store != nil {
		eventSource := dependencies.Events
		if eventSource == nil {
			eventSource = events.NewBroker()
		}
		handler := cardAPI{
			store:       dependencies.Store,
			actions:     dependencies.Actions,
			notifier:    dependencies.Notifier,
			reconciler:  dependencies.Reconciler,
			eventSource: eventSource,
			logger:      logger,
		}
		if accessHandler != nil {
			handler.devices = accessHandler.registry
		}
		routes.add(http.MethodGet, "/api/cards", handler.listCards)
		routes.add(http.MethodPost, "/api/cards", handler.createCard)
		routes.add(http.MethodGet, "/api/cards/{id}", handler.getCard)
		routes.add(http.MethodPut, "/api/cards/{id}", handler.updateCard)
		routes.add(http.MethodDelete, "/api/cards/{id}", handler.deleteCard)
		routes.add(http.MethodGet, "/api/cards/{id}/runs", handler.getRuns)
		routes.add(http.MethodGet, "/api/cards/{id}/status", handler.getStatus)
		routes.add(http.MethodGet, "/api/cards/{id}/status/history", handler.getStatusHistory)
		routes.add(http.MethodGet, "/api/events", handler.events)
		if dependencies.Actions != nil {
			routes.add(http.MethodPost, "/api/cards/{id}/actions/primary", handler.enqueuePrimary)
			routes.add(http.MethodPost, "/api/cards/{id}/actions/status/check", handler.enqueueStatus)
		}
	}

	mux := http.NewServeMux()
	routes.register(mux)
	mux.Handle("/", spaHandler(webui.Dist()))

	var handler http.Handler = mux
	if accessHandler != nil {
		handler = accessHandler.requireDevice(mux)
	}
	return requireSameOrigin(dependencies.AllowedHosts, handler)
}

// routeTable collects API routes so that each path can be registered with a
// method-less fallback answering 405 for the methods it does not support.
type routeTable map[string]map[string]http.HandlerFunc

func (routes routeTable) add(method, path string, handler http.HandlerFunc) {
	if routes[path] == nil {
		routes[path] = map[string]http.HandlerFunc{}
	}
	routes[path][method] = handler
}

func (routes routeTable) register(mux *http.ServeMux) {
	for path, handlers := range routes {
		methods := make([]string, 0, len(handlers)+1)
		for method, handler := range handlers {
			mux.HandleFunc(method+" "+path, handler)
			methods = append(methods, method)
			if method == http.MethodGet {
				methods = append(methods, http.MethodHead)
			}
		}
		sort.Strings(methods)
		mux.HandleFunc(path, methodNotAllowed(strings.Join(methods, ", ")))
	}
}

func methodNotAllowed(allow string) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		w.Header().Set("Allow", allow)
		writeError(w, http.StatusMethodNotAllowed, fmt.Errorf("method %s is not allowed", request.Method))
	}
}

type cardAPI struct {
	store       *config.Store
	actions     ActionQueue
	notifier    PrimaryActionNotifier
	reconciler  CardReconciler
	eventSource EventSource
	logger      *slog.Logger
	// devices ends event streams of removed devices; nil without access control.
	devices interface{ Exists(string) bool }
}

// acceptedAction is the 202 body of the enqueue endpoints. CheckedAt is the
// time of the last status check known when the request was accepted (read
// before the action is queued), so a client waiting for "a check newer than
// the one before my action" has a baseline that includes a scheduled check
// which completed while the request was in flight. It is absent for a card
// that was never checked.
type acceptedAction struct {
	CardID     string     `json:"cardId"`
	ActionKind string     `json:"actionKind"`
	Status     string     `json:"status"`
	CheckedAt  *time.Time `json:"checkedAt,omitempty"`
}

func (api cardAPI) lastCheckedAt(cardID string) *time.Time {
	snapshot, exists := api.store.GetStatus(cardID)
	if !exists || snapshot.CheckedAt.IsZero() {
		return nil
	}
	checkedAt := snapshot.CheckedAt
	return &checkedAt
}

type cardView struct {
	config.ActionCard
	CurrentStatus *config.StatusSnapshot `json:"currentStatus,omitempty"`
}

func (api cardAPI) listCards(w http.ResponseWriter, _ *http.Request) {
	cards := api.store.ListCards()
	views := make([]cardView, 0, len(cards))
	for _, card := range cards {
		views = append(views, api.viewCard(card))
	}
	writeJSON(w, http.StatusOK, views)
}

func (api cardAPI) viewCard(card config.ActionCard) cardView {
	view := cardView{ActionCard: card}
	if snapshot, exists := api.store.GetStatus(card.ID); exists {
		view.CurrentStatus = &snapshot
	} else if card.Status != nil {
		view.CurrentStatus = &config.StatusSnapshot{State: config.StatusStateUnknown}
	}
	return view
}

func (api cardAPI) createCard(w http.ResponseWriter, request *http.Request) {
	var card config.ActionCard
	if err := decodeJSON(w, request, &card); err != nil {
		return
	}
	created, err := api.store.CreateCard(card)
	if err != nil {
		writeStoreError(w, err)
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
	writeJSON(w, http.StatusOK, api.viewCard(card))
}

func (api cardAPI) updateCard(w http.ResponseWriter, request *http.Request) {
	var card config.ActionCard
	if err := decodeJSON(w, request, &card); err != nil {
		return
	}
	updated, err := api.store.UpdateCard(request.PathValue("id"), card)
	if err != nil {
		writeStoreError(w, err)
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
		writeStoreError(w, err)
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
	checkedAt := api.lastCheckedAt(cardID)
	if err := api.actions.EnqueuePrimary(cardID, card.Primary); err != nil && !errors.Is(err, actions.ErrAlreadyQueued) {
		writeQueueError(w, err)
		return
	}
	if api.notifier != nil {
		if err := api.notifier.NotifyPrimaryAction(cardID); err != nil {
			api.logger.Warn("fast polling notification failed", "card", cardID, "error", err)
		}
	}
	writeJSON(w, http.StatusAccepted, acceptedAction{CardID: cardID, ActionKind: "primary", Status: "accepted", CheckedAt: checkedAt})
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
	checkedAt := api.lastCheckedAt(cardID)
	if err := api.actions.EnqueueStatus(cardID); err != nil && !errors.Is(err, actions.ErrAlreadyQueued) {
		writeQueueError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, acceptedAction{CardID: cardID, ActionKind: "status", Status: "accepted", CheckedAt: checkedAt})
}

// decodeJSON reads one JSON value into target. Failures are answered with the
// JSON error envelope: a body over maxJSONBodyBytes → 413, anything that is
// not exactly one JSON value of the expected shape → 400 with a message that
// names the offending field or offset without exposing Go type names.
func decodeJSON(w http.ResponseWriter, request *http.Request, target any) error {
	request.Body = http.MaxBytesReader(w, request.Body, maxJSONBodyBytes)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeDecodeError(w, err)
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("request contains multiple JSON values")
		}
		writeDecodeError(w, err)
		return err
	}
	return nil
}

func writeDecodeError(w http.ResponseWriter, err error) {
	var (
		maxBytes    *http.MaxBytesError
		syntax      *json.SyntaxError
		wrongType   *json.UnmarshalTypeError
		description string
	)
	switch {
	case errors.As(err, &maxBytes):
		writeError(w, http.StatusRequestEntityTooLarge, fmt.Errorf("request body must be at most %d bytes", maxBytes.Limit))
		return
	case errors.As(err, &syntax):
		description = fmt.Sprintf("malformed JSON at offset %d", syntax.Offset)
	case errors.As(err, &wrongType):
		field := wrongType.Field
		if field == "" {
			field = "body"
		}
		description = fmt.Sprintf("unexpected %s value for field %q", wrongType.Value, field)
	case errors.Is(err, io.EOF):
		description = "request body is empty"
	case errors.Is(err, io.ErrUnexpectedEOF):
		description = "unexpected end of JSON body"
	case strings.HasPrefix(err.Error(), "json: unknown field "):
		description = "unknown field " + strings.TrimPrefix(err.Error(), "json: unknown field ")
	default:
		description = err.Error()
	}
	writeError(w, http.StatusBadRequest, errors.New("invalid JSON body: "+description))
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

// writeQueueError maps action queue errors: a full queue → 503 with a
// Retry-After hint (FR-18), a closed queue (shutdown in progress) → 503,
// anything else → 500.
func writeQueueError(w http.ResponseWriter, err error) {
	var full *actions.QueueFullError
	switch {
	case errors.As(err, &full):
		seconds := int(math.Ceil(full.RetryAfter.Seconds()))
		w.Header().Set("Retry-After", strconv.Itoa(max(seconds, 1)))
		writeError(w, http.StatusServiceUnavailable, err)
	case errors.Is(err, actions.ErrQueueFull), errors.Is(err, actions.ErrQueueClosed):
		w.Header().Set("Retry-After", "1")
		writeError(w, http.StatusServiceUnavailable, err)
	default:
		writeError(w, http.StatusInternalServerError, err)
	}
}

// writeStoreError maps config store errors to HTTP statuses: not found → 404,
// duplicate → 409, validation → 422, persistence failure (rolled back) → 500,
// anything else → 400.
func writeStoreError(w http.ResponseWriter, err error) {
	var persistenceErr *config.PersistenceError
	switch {
	case errors.As(err, &persistenceErr):
		writeError(w, http.StatusInternalServerError, err)
	case errors.Is(err, config.ErrNotFound):
		writeError(w, http.StatusNotFound, err)
	case errors.Is(err, config.ErrAlreadyExists):
		writeError(w, http.StatusConflict, err)
	case errors.Is(err, config.ErrValidation):
		writeValidationError(w, err)
	default:
		writeError(w, http.StatusBadRequest, err)
	}
}

// writeValidationError answers 422 with the error envelope extended by a
// "fields" object (JSON path → reason) when the error carries per-field
// details.
func writeValidationError(w http.ResponseWriter, err error) {
	var validation *config.ValidationError
	if !errors.As(err, &validation) {
		writeError(w, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusUnprocessableEntity, struct {
		Error  string            `json:"error"`
		Fields map[string]string `json:"fields"`
	}{Error: err.Error(), Fields: validation.Fields})
}

type healthResponse struct {
	Status    string `json:"status"`
	Version   string `json:"version"`
	UptimeSec int64  `json:"uptimeSec"`
}

func newHealthHandler(version string, startedAt time.Time) http.HandlerFunc {
	if version == "" {
		version = "dev"
	}
	if startedAt.IsZero() {
		startedAt = time.Now()
	}
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, healthResponse{
			Status:    "ok",
			Version:   version,
			UptimeSec: int64(time.Since(startedAt).Seconds()),
		})
	}
}
