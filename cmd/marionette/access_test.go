package main

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/cookiejar"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"marionette/internal/access"
	"marionette/internal/server"
)

func TestLoadEnvironmentAccessSettings(t *testing.T) {
	env, err := loadEnvironment(func(key string) string {
		if key == "MARIONETTE_CONFIG" {
			return "/var/lib/marionette/marionette.json"
		}
		return ""
	})
	if err != nil {
		t.Fatalf("loadEnvironment() error = %v", err)
	}
	if !env.AuthEnabled || env.DeviceExpiry != 60*24*time.Hour || env.CookieSecure != server.CookieSecureAuto {
		t.Fatalf("access defaults = %#v", env)
	}
	if env.DevicesPath != "/var/lib/marionette/devices.json" {
		t.Fatalf("default DevicesPath = %q", env.DevicesPath)
	}

	values := map[string]string{
		"MARIONETTE_AUTH":               " OFF ",
		"MARIONETTE_DEVICES":            "/tmp/devices.json",
		"MARIONETTE_DEVICE_EXPIRY_DAYS": "30",
		"MARIONETTE_COOKIE_SECURE":      "Always",
	}
	env, err = loadEnvironment(func(key string) string { return values[key] })
	if err != nil {
		t.Fatalf("loadEnvironment() error = %v", err)
	}
	if env.AuthEnabled || env.DevicesPath != "/tmp/devices.json" || env.DeviceExpiry != 30*24*time.Hour || env.CookieSecure != server.CookieSecureAlways {
		t.Fatalf("access environment = %#v", env)
	}
	values["MARIONETTE_COOKIE_SECURE"] = "never"
	if env, _ := loadEnvironment(func(key string) string { return values[key] }); env.CookieSecure != server.CookieSecureNever {
		t.Fatalf("CookieSecure never = %v", env.CookieSecure)
	}

	invalid := map[string][]string{
		"MARIONETTE_AUTH":               {"yes", "false"},
		"MARIONETTE_DEVICE_EXPIRY_DAYS": {"0", "401", "-3", "30d", "1.5"},
		"MARIONETTE_COOKIE_SECURE":      {"true"},
	}
	for key, candidates := range invalid {
		for _, candidate := range candidates {
			if _, err := loadEnvironment(func(k string) string {
				if k == key {
					return candidate
				}
				return ""
			}); err == nil {
				t.Fatalf("%s=%q was accepted", key, candidate)
			}
		}
	}
}

func TestOpenAccessLogsACodeOnlyWithoutDevices(t *testing.T) {
	path := filepath.Join(t.TempDir(), "devices.json")
	env := environment{AuthEnabled: true, DevicesPath: path, DeviceExpiry: access.DefaultExpiry}
	var logs bytes.Buffer
	registry, err := openAccess(env, slog.New(slog.NewTextHandler(&logs, nil)))
	if err != nil || registry == nil {
		t.Fatalf("openAccess() = %v, %v", registry, err)
	}
	match := regexp.MustCompile(`code=([0-9A-Z]{4}-[0-9A-Z]{4})`).FindStringSubmatch(logs.String())
	if match == nil {
		t.Fatalf("no pairing code logged: %s", logs.String())
	}
	if _, _, err := registry.Pair(match[1], "Laptop"); err != nil {
		t.Fatalf("Pair() with the logged code error = %v", err)
	}

	logs.Reset()
	if _, err := openAccess(env, slog.New(slog.NewTextHandler(&logs, nil))); err != nil {
		t.Fatalf("second openAccess() error = %v", err)
	}
	if strings.Contains(logs.String(), "code=") || !strings.Contains(logs.String(), "pairedDevices=1") {
		t.Fatalf("log with a paired device = %s", logs.String())
	}

	logs.Reset()
	registry, err = openAccess(environment{AuthEnabled: false}, slog.New(slog.NewTextHandler(&logs, nil)))
	if err != nil || registry != nil || !strings.Contains(logs.String(), "access control is disabled") {
		t.Fatalf("disabled access = %v, %v, %s", registry, err, logs.String())
	}

	if os.Geteuid() != 0 {
		if err := os.Chmod(path, 0o000); err != nil {
			t.Fatal(err)
		}
		if _, err := openAccess(env, slog.New(slog.NewTextHandler(io.Discard, nil))); err == nil {
			t.Fatal("an unreadable devices file did not stop the start-up")
		}
	}
}

func TestRunRequiresAPairedDevice(t *testing.T) {
	directory := t.TempDir()
	var logs safeBuffer
	listening := make(chan net.Addr, 1)
	app := application{
		env: environment{
			ConfigPath:      filepath.Join(directory, "marionette.json"),
			Addr:            "127.0.0.1:0",
			ShutdownTimeout: 2 * time.Second,
			AuthEnabled:     true,
			DevicesPath:     filepath.Join(directory, "devices.json"),
			DeviceExpiry:    access.DefaultExpiry,
		},
		logger:      slog.New(slog.NewTextHandler(&logs, nil)),
		onListening: func(addr net.Addr) { listening <- addr },
	}
	ctx, cancel := context.WithCancel(context.Background())
	finished := make(chan error, 1)
	go func() { finished <- app.run(ctx) }()
	var addr net.Addr
	select {
	case addr = <-listening:
	case err := <-finished:
		t.Fatalf("run() returned early: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("server did not start listening")
	}
	baseURL := "http://" + addr.String()

	if response, err := http.Get(baseURL + "/api/cards"); err != nil || response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("GET /api/cards without a device = %v, %v", response, err)
	}
	if response, err := http.Get(baseURL + "/api/health"); err != nil || response.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/health = %v, %v", response, err)
	}
	match := regexp.MustCompile(`code=([0-9A-Z]{4}-[0-9A-Z]{4})`).FindStringSubmatch(logs.String())
	if match == nil {
		t.Fatalf("no pairing code in the start-up log: %s", logs.String())
	}

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	pairing, err := client.Post(baseURL+"/api/pairing", "application/json",
		strings.NewReader(`{"code":"`+match[1]+`","name":"Integration test"}`))
	if err != nil || pairing.StatusCode != http.StatusCreated {
		t.Fatalf("pairing = %v, %v", pairing, err)
	}
	_ = pairing.Body.Close()
	if response, err := client.Get(baseURL + "/api/cards"); err != nil || response.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/cards as a paired device = %v, %v", response, err)
	}

	cancel()
	select {
	case err := <-finished:
		if err != nil {
			t.Fatalf("run() error = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("run() did not shut down")
	}
	data, err := os.ReadFile(filepath.Join(directory, "devices.json"))
	if err != nil || !strings.Contains(string(data), "Integration test") {
		t.Fatalf("devices file after shutdown = %s, %v", data, err)
	}
	for _, cookie := range jar.Cookies(pairing.Request.URL) {
		if strings.Contains(logs.String(), cookie.Value) || strings.Contains(string(data), cookie.Value) {
			t.Fatal("the device token leaked into the log or the devices file")
		}
	}
	if !strings.Contains(logs.String(), `step="save devices"`) {
		t.Fatalf("devices not saved at shutdown: %s", logs.String())
	}
}

// safeBuffer is a bytes.Buffer that the server goroutines and the test can
// use at the same time.
type safeBuffer struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func (b *safeBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.Write(p)
}

func (b *safeBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.String()
}
