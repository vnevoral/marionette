package control

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"marionette/internal/access"
)

// socketDir returns a short temporary directory: a Unix socket path is
// limited to about 100 bytes, which t.TempDir() can exceed.
func socketDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "mctl")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}

func openRegistry(t *testing.T) *access.Registry {
	t.Helper()
	registry, err := access.Open(filepath.Join(t.TempDir(), "devices.json"), access.Options{})
	if err != nil {
		t.Fatalf("access.Open() error = %v", err)
	}
	return registry
}

// lockedBuffer collects the log of the server goroutine.
type lockedBuffer struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.String()
}

func serve(t *testing.T, path string, registry *access.Registry, logs io.Writer) *Server {
	t.Helper()
	server, err := Listen(path, registry, slog.New(slog.NewTextHandler(logs, nil)))
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	served := make(chan error, 1)
	go func() { served <- server.Serve() }()
	t.Cleanup(func() {
		_ = server.Shutdown(context.Background())
		if err := <-served; err != nil {
			t.Errorf("Serve() error = %v", err)
		}
	})
	return server
}

func TestRequestCodeIssuesACodeThatPairsADevice(t *testing.T) {
	path := filepath.Join(socketDir(t), "control.sock")
	registry := openRegistry(t)
	var logs lockedBuffer
	server := serve(t, path, registry, &logs)
	if server.Path() != path {
		t.Fatalf("Path() = %q, want %q", server.Path(), path)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Fatalf("socket mode = %o, want 600", mode)
	}

	before := time.Now()
	code, err := RequestCode(context.Background(), path)
	if err != nil {
		t.Fatalf("RequestCode() error = %v", err)
	}
	if len(code.Code) != access.CodeLength+1 || code.Code[4] != '-' {
		t.Fatalf("code = %q, want XXXX-XXXX", code.Code)
	}
	if lifetime := code.ExpiresAt.Sub(before); lifetime < access.CodeLifetime-time.Minute || lifetime > access.CodeLifetime+time.Minute {
		t.Fatalf("expiresAt = %s, want about %s from now", code.ExpiresAt, access.CodeLifetime)
	}
	if _, _, err := registry.Pair(code.Code, "Phone"); err != nil {
		t.Fatalf("Pair() with the code from the socket error = %v", err)
	}
	if !strings.Contains(logs.String(), "pairing code issued from the host") {
		t.Fatalf("issuing is not logged: %s", logs.String())
	}
	if strings.Contains(logs.String(), code.Code) || strings.Contains(logs.String(), access.NormalizeCode(code.Code)) {
		t.Fatalf("the code leaked into the log: %s", logs.String())
	}
}

func TestANewCodeReplacesTheCodeFromTheUI(t *testing.T) {
	path := filepath.Join(socketDir(t), "control.sock")
	registry := openRegistry(t)
	serve(t, path, registry, io.Discard)
	fromUI, _, err := registry.NewCode()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := RequestCode(context.Background(), path); err != nil {
		t.Fatalf("RequestCode() error = %v", err)
	}
	if _, _, err := registry.Pair(fromUI, "Laptop"); !errors.Is(err, access.ErrInvalidCode) {
		t.Fatalf("Pair() with the replaced code error = %v, want ErrInvalidCode", err)
	}
}

func TestListenReplacesAStaleSocketButNoOtherFile(t *testing.T) {
	dir := socketDir(t)
	path := filepath.Join(dir, "control.sock")
	stale, err := Listen(path, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	// Simulate a crash: the socket file stays behind without a listener.
	stale.listener.(interface{ SetUnlinkOnClose(bool) }).SetUnlinkOnClose(false)
	_ = stale.listener.Close()
	if _, err := os.Lstat(path); err != nil {
		t.Fatalf("stale socket missing: %v", err)
	}
	serve(t, path, openRegistry(t), io.Discard)
	if _, err := RequestCode(context.Background(), path); err != nil {
		t.Fatalf("RequestCode() after replacing a stale socket error = %v", err)
	}

	regular := filepath.Join(dir, "regular")
	if err := os.WriteFile(regular, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Listen(regular, nil, slog.New(slog.NewTextHandler(io.Discard, nil))); err == nil {
		t.Fatal("Listen() over a regular file succeeded")
	}
	if data, err := os.ReadFile(regular); err != nil || string(data) != "keep" {
		t.Fatalf("regular file changed: %q, %v", data, err)
	}
}

func TestShutdownRemovesTheSocket(t *testing.T) {
	path := filepath.Join(socketDir(t), "control.sock")
	server, err := Listen(path, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	served := make(chan error, 1)
	go func() { served <- server.Serve() }()
	if err := server.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	if err := <-served; err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("socket after shutdown: %v", err)
	}
}

func TestRequestCodeErrors(t *testing.T) {
	dir := socketDir(t)

	if _, err := RequestCode(context.Background(), filepath.Join(dir, "missing.sock")); !errors.Is(err, ErrNotRunning) {
		t.Fatalf("missing socket error = %v, want ErrNotRunning", err)
	}

	disabled := filepath.Join(dir, "disabled.sock")
	serve(t, disabled, nil, io.Discard)
	if _, err := RequestCode(context.Background(), disabled); !errors.Is(err, ErrAccessDisabled) {
		t.Fatalf("access off error = %v, want ErrAccessDisabled", err)
	}

	if os.Geteuid() != 0 {
		locked := filepath.Join(dir, "locked")
		if err := os.Mkdir(locked, 0o700); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(locked, "control.sock")
		serve(t, path, openRegistry(t), io.Discard)
		if err := os.Chmod(locked, 0o000); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(locked, 0o700) })
		if _, err := RequestCode(context.Background(), path); !errors.Is(err, ErrPermission) {
			t.Fatalf("unreadable directory error = %v, want ErrPermission", err)
		}
	}
}

func TestHandlerOffersOnlyThePairingCodeRoute(t *testing.T) {
	handler := NewHandler(openRegistry(t), slog.New(slog.NewTextHandler(io.Discard, nil)))
	for _, tc := range []struct {
		method, path string
		want         int
	}{
		{http.MethodPost, "/pairing-code", http.StatusCreated},
		{http.MethodGet, "/pairing-code", http.StatusMethodNotAllowed},
		{http.MethodPost, "/api/pairing/code", http.StatusNotFound},
		{http.MethodGet, "/", http.StatusNotFound},
	} {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(tc.method, tc.path, nil))
		if recorder.Code != tc.want {
			t.Errorf("%s %s = %d, want %d", tc.method, tc.path, recorder.Code, tc.want)
		}
	}
}
