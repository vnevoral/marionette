package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"marionette/internal/access"
)

// DeviceCookie is the name of the cookie that carries the device token.
const DeviceCookie = "marionette_device"

// CookieSecurity decides the Secure attribute of the device cookie
// (MARIONETTE_COOKIE_SECURE, NFR-13).
type CookieSecurity int

const (
	// CookieSecureAuto sets Secure when the connection itself uses TLS.
	CookieSecureAuto CookieSecurity = iota
	// CookieSecureAlways always sets Secure (behind a TLS-terminating proxy).
	CookieSecureAlways
	// CookieSecureNever never sets Secure.
	CookieSecureNever
)

// Access enables device pairing (FR-50..FR-56, ADR-0011). A nil Registry in
// Dependencies.Access leaves the API open (MARIONETTE_AUTH=off).
type Access struct {
	Registry *access.Registry
	Secure   CookieSecurity
}

type deviceContextKey struct{}

// currentDevice returns the device that made the request, when access
// control is on.
func currentDevice(request *http.Request) (access.Device, bool) {
	device, ok := request.Context().Value(deviceContextKey{}).(access.Device)
	return device, ok
}

type accessAPI struct {
	registry *access.Registry
	secure   CookieSecurity
	logger   *slog.Logger
}

// publicAPIRoute lists the API routes a device can use before pairing.
func publicAPIRoute(request *http.Request) bool {
	switch request.URL.Path {
	case "/api/health":
		return request.Method == http.MethodGet || request.Method == http.MethodHead
	case "/api/session":
		return request.Method == http.MethodGet
	case "/api/pairing":
		return request.Method == http.MethodPost
	}
	return false
}

// requireDevice answers 401 for every API request without a valid device
// token, except the public routes. The SPA files stay public: they hold no
// data and show the pairing screen. A valid device is put in the request
// context, and its cookie is sent again with a fresh lifetime at most once a
// day (FR-52).
func (api *accessAPI) requireDevice(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		path := request.URL.Path
		if (path != "/api" && !strings.HasPrefix(path, "/api/")) || publicAPIRoute(request) {
			next.ServeHTTP(w, request)
			return
		}
		session, token, ok := api.authenticate(request)
		if !ok {
			api.writePairingRequired(w)
			return
		}
		if session.Renew {
			api.setCookie(w, request, token)
		}
		next.ServeHTTP(w, request.WithContext(context.WithValue(request.Context(), deviceContextKey{}, session.Device)))
	})
}

func (api *accessAPI) authenticate(request *http.Request) (access.Session, string, bool) {
	cookie, err := request.Cookie(DeviceCookie)
	if err != nil {
		return access.Session{}, "", false
	}
	session, ok := api.registry.Authenticate(cookie.Value)
	return session, cookie.Value, ok
}

func (api *accessAPI) writePairingRequired(w http.ResponseWriter) {
	writeJSON(w, http.StatusUnauthorized, struct {
		Error string `json:"error"`
		Code  string `json:"code"`
	}{Error: "this device is not paired", Code: "pairing_required"})
}

func (api *accessAPI) setCookie(w http.ResponseWriter, request *http.Request, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     DeviceCookie,
		Value:    token,
		Path:     "/",
		MaxAge:   int(api.registry.Expiry() / time.Second),
		HttpOnly: true,
		Secure:   api.secureFor(request),
		SameSite: http.SameSiteStrictMode,
	})
}

func (api *accessAPI) clearCookie(w http.ResponseWriter, request *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     DeviceCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   api.secureFor(request),
		SameSite: http.SameSiteStrictMode,
	})
}

func (api *accessAPI) secureFor(request *http.Request) bool {
	switch api.secure {
	case CookieSecureAlways:
		return true
	case CookieSecureNever:
		return false
	default:
		return request.TLS != nil
	}
}

type deviceView struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	PairedAt   time.Time `json:"pairedAt"`
	LastSeenAt time.Time `json:"lastSeenAt"`
	Current    bool      `json:"current"`
}

func viewDevice(device access.Device, currentID string) deviceView {
	return deviceView{
		ID:         device.ID,
		Name:       device.Name,
		PairedAt:   device.PairedAt,
		LastSeenAt: device.LastSeenAt,
		Current:    device.ID == currentID,
	}
}

// session reports the paired device, or 401 with "bootstrap": true when no
// device is paired yet. In that case a pairing code is written to the
// service log whenever none is valid, so the first device can always pair
// with the latest line of `journalctl -u marionette` (FR-53).
func (api *accessAPI) session(w http.ResponseWriter, request *http.Request) {
	if session, _, ok := api.authenticate(request); ok {
		writeJSON(w, http.StatusOK, struct {
			Device     deviceView `json:"device"`
			ExpiryDays int        `json:"expiryDays"`
		}{
			Device:     viewDevice(session.Device, session.Device.ID),
			ExpiryDays: int(api.registry.Expiry() / (24 * time.Hour)),
		})
		return
	}
	bootstrap := api.registry.Empty()
	if bootstrap && !api.registry.HasValidCode() {
		if err := LogBootstrapCode(api.registry, api.logger); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
	}
	writeJSON(w, http.StatusUnauthorized, struct {
		Error     string `json:"error"`
		Code      string `json:"code"`
		Bootstrap bool   `json:"bootstrap"`
	}{Error: "this device is not paired", Code: "pairing_required", Bootstrap: bootstrap})
}

// LogBootstrapCode creates a pairing code and writes it to the log for the
// operator of a service without paired devices (FR-53).
func LogBootstrapCode(registry *access.Registry, logger *slog.Logger) error {
	code, expiresAt, err := registry.NewCode()
	if err != nil {
		return err
	}
	logger.Warn("no device is paired; open Marionette in a browser and enter this pairing code",
		"code", access.FormatCode(code), "expiresAt", expiresAt.UTC().Format(time.RFC3339))
	return nil
}

func (api *accessAPI) pair(w http.ResponseWriter, request *http.Request) {
	var body struct {
		Code string `json:"code"`
		Name string `json:"name"`
	}
	if err := decodeJSON(w, request, &body); err != nil {
		return
	}
	device, token, err := api.registry.Pair(body.Code, body.Name)
	switch {
	case errors.Is(err, access.ErrInvalidName):
		writeJSON(w, http.StatusUnprocessableEntity, struct {
			Error  string            `json:"error"`
			Fields map[string]string `json:"fields"`
		}{Error: err.Error(), Fields: map[string]string{"name": err.Error()}})
		return
	case errors.Is(err, access.ErrInvalidCode):
		api.logger.Warn("pairing attempt with an invalid code", "remote", request.RemoteAddr)
		writeError(w, http.StatusBadRequest, err)
		return
	case err != nil:
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	api.logger.Info("device paired", "device", device.ID, "name", device.Name, "remote", request.RemoteAddr)
	api.setCookie(w, request, token)
	writeJSON(w, http.StatusCreated, struct {
		Device deviceView `json:"device"`
	}{Device: viewDevice(device, device.ID)})
}

func (api *accessAPI) createPairingCode(w http.ResponseWriter, request *http.Request) {
	code, expiresAt, err := api.registry.NewCode()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if device, ok := currentDevice(request); ok {
		api.logger.Info("pairing code created", "by", device.ID, "name", device.Name)
	}
	writeJSON(w, http.StatusCreated, struct {
		Code      string    `json:"code"`
		ExpiresAt time.Time `json:"expiresAt"`
	}{Code: access.FormatCode(code), ExpiresAt: expiresAt})
}

func (api *accessAPI) listDevices(w http.ResponseWriter, request *http.Request) {
	current, _ := currentDevice(request)
	devices := api.registry.List()
	views := make([]deviceView, 0, len(devices))
	for _, device := range devices {
		views = append(views, viewDevice(device, current.ID))
	}
	writeJSON(w, http.StatusOK, views)
}

func (api *accessAPI) removeDevice(w http.ResponseWriter, request *http.Request) {
	id := request.PathValue("id")
	if err := api.registry.Remove(id); err != nil {
		if errors.Is(err, access.ErrDeviceNotFound) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	current, _ := currentDevice(request)
	api.logger.Info("device removed", "device", id, "by", current.ID)
	if id == current.ID {
		api.clearCookie(w, request)
	}
	w.WriteHeader(http.StatusNoContent)
}
