// Package control is the local control socket of the service (FR-58,
// ADR-0013). The running service listens on a Unix domain socket that only
// root and the service user can open; the `marionette pair` command connects
// to it to get a pairing code for a new device without the web UI. The file
// permissions of the socket are the authorization: whoever can open it can
// already read the service log on the host (ADR-0011).
package control

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"syscall"
	"time"

	"marionette/internal/access"
)

// DefaultSocketPath is where the service listens when
// MARIONETTE_CONTROL_SOCKET is not set. The systemd unit creates its
// directory (RuntimeDirectory=marionette, mode 0700).
const DefaultSocketPath = "/run/marionette/control.sock"

// pairingCodeRoute is the only route of the control socket.
const pairingCodeRoute = "/pairing-code"

// socketMode keeps the socket to its owner (the service user) and root.
const socketMode = 0o600

const readHeaderTimeout = 5 * time.Second

// Errors returned by RequestCode, so the command can tell the operator what
// to do.
var (
	// ErrNotRunning means the socket does not exist or nobody listens on it.
	ErrNotRunning = errors.New("the Marionette service is not running or has no control socket")
	// ErrPermission means the caller may not open the socket.
	ErrPermission = errors.New("permission denied on the control socket")
	// ErrAccessDisabled means the service runs with MARIONETTE_AUTH=off.
	ErrAccessDisabled = errors.New("access control is disabled (MARIONETTE_AUTH=off), no pairing is needed")
)

// PairingCode is a code issued over the control socket.
type PairingCode struct {
	// Code is formatted for reading, e.g. "K7QM-2XHD".
	Code      string    `json:"code"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// Server serves the control socket.
type Server struct {
	server   *http.Server
	listener net.Listener
}

// Listen creates the control socket at path with mode 0600. A socket left
// behind by a previous run is replaced; any other file at path is an error
// and is never removed. registry is nil when access control is off.
func Listen(path string, registry *access.Registry, logger *slog.Logger) (*Server, error) {
	if info, err := os.Lstat(path); err == nil {
		if info.Mode().Type() != fs.ModeSocket {
			return nil, fmt.Errorf("control socket %s: a file that is not a socket is in the way", path)
		}
		if err := os.Remove(path); err != nil {
			return nil, fmt.Errorf("remove stale control socket: %w", err)
		}
	}
	listener, err := net.Listen("unix", path)
	if err != nil {
		return nil, fmt.Errorf("listen on control socket: %w", err)
	}
	// Under systemd the UMask (0077) and the 0700 runtime directory already
	// keep others out; the explicit mode covers every other start.
	if err := os.Chmod(path, socketMode); err != nil {
		_ = listener.Close()
		return nil, fmt.Errorf("restrict control socket: %w", err)
	}
	return &Server{
		server: &http.Server{
			Handler:           NewHandler(registry, logger),
			ReadHeaderTimeout: readHeaderTimeout,
		},
		listener: listener,
	}, nil
}

// Path returns the socket path.
func (s *Server) Path() string { return s.listener.Addr().String() }

// Serve accepts connections until Shutdown; it returns nil after Shutdown.
func (s *Server) Serve() error {
	if err := s.server.Serve(s.listener); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Shutdown stops accepting connections and removes the socket file.
func (s *Server) Shutdown(ctx context.Context) error {
	if err := s.server.Shutdown(ctx); err != nil {
		_ = s.server.Close()
		return err
	}
	return nil
}

// NewHandler returns the handler of the control socket. It is not part of
// the public HTTP API and must never be mounted on a TCP listener.
func NewHandler(registry *access.Registry, logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST "+pairingCodeRoute, func(w http.ResponseWriter, _ *http.Request) {
		if registry == nil {
			writeJSON(w, http.StatusConflict, errorBody{Error: ErrAccessDisabled.Error()})
			return
		}
		code, expiresAt, err := registry.NewCode()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorBody{Error: err.Error()})
			return
		}
		// Never the code itself (NFR-13): the log is readable by more people
		// than the socket.
		logger.Info("pairing code issued from the host", "expiresAt", expiresAt.UTC().Format(time.RFC3339))
		writeJSON(w, http.StatusCreated, PairingCode{Code: access.FormatCode(code), ExpiresAt: expiresAt})
	})
	return mux
}

type errorBody struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// RequestCode asks the service listening on the socket at path for a new
// pairing code.
func RequestCode(ctx context.Context, path string) (PairingCode, error) {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			var dialer net.Dialer
			return dialer.DialContext(ctx, "unix", path)
		},
	}
	defer transport.CloseIdleConnections()
	// The host part is ignored by the dialer above.
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://marionette"+pairingCodeRoute, nil)
	if err != nil {
		return PairingCode{}, err
	}
	response, err := (&http.Client{Transport: transport}).Do(request)
	switch {
	case errors.Is(err, fs.ErrNotExist), errors.Is(err, syscall.ECONNREFUSED):
		return PairingCode{}, ErrNotRunning
	case errors.Is(err, fs.ErrPermission):
		return PairingCode{}, ErrPermission
	case err != nil:
		return PairingCode{}, err
	}
	defer func() { _ = response.Body.Close() }()
	switch response.StatusCode {
	case http.StatusCreated:
		var code PairingCode
		if err := json.NewDecoder(response.Body).Decode(&code); err != nil || code.Code == "" {
			return PairingCode{}, fmt.Errorf("unexpected answer from the service: %v", err)
		}
		return code, nil
	case http.StatusConflict:
		return PairingCode{}, ErrAccessDisabled
	default:
		var body errorBody
		_ = json.NewDecoder(response.Body).Decode(&body)
		return PairingCode{}, fmt.Errorf("the service answered %d: %s", response.StatusCode, body.Error)
	}
}
