package main

import (
	"bytes"
	"context"
	"log/slog"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"marionette/internal/access"
	"marionette/internal/control"
)

func TestRunCommandUsage(t *testing.T) {
	for _, tc := range []struct {
		args       []string
		want       int
		wantStdout bool
		message    string
	}{
		{[]string{"help"}, 0, true, ""},
		{[]string{"--help"}, 0, true, ""},
		{[]string{"serve"}, 2, false, `unknown command "serve"`},
		{[]string{"pair", "now"}, 2, false, "takes no arguments"},
	} {
		var stdout, stderr bytes.Buffer
		if got := runCommand(tc.args, func(string) string { return "" }, &stdout, &stderr); got != tc.want {
			t.Errorf("runCommand(%v) = %d, want %d", tc.args, got, tc.want)
		}
		output := stderr.String()
		if tc.wantStdout {
			output = stdout.String()
		}
		if !strings.Contains(output, "marionette pair") || !strings.Contains(output, tc.message) {
			t.Errorf("runCommand(%v) output = %q", tc.args, output)
		}
	}
}

func TestRunPairWithoutAService(t *testing.T) {
	path := filepath.Join(t.TempDir(), "control.sock")
	var stdout, stderr bytes.Buffer
	getenv := func(key string) string {
		if key == "MARIONETTE_CONTROL_SOCKET" {
			return path
		}
		return ""
	}
	if got := runCommand([]string{"pair"}, getenv, &stdout, &stderr); got != 1 {
		t.Fatalf("runCommand(pair) = %d, want 1", got)
	}
	if stdout.Len() != 0 || !strings.Contains(stderr.String(), "not running") ||
		!strings.Contains(stderr.String(), path) || !strings.Contains(stderr.String(), "systemctl status marionette") {
		t.Fatalf("stdout = %q, stderr = %q", stdout.String(), stderr.String())
	}
}

func TestLoadEnvironmentControlSocket(t *testing.T) {
	env, err := loadEnvironment(func(string) string { return "" })
	if err != nil {
		t.Fatal(err)
	}
	if env.ControlSocket != control.DefaultSocketPath || env.ControlSocketSet {
		t.Fatalf("default control socket = %q, set = %v", env.ControlSocket, env.ControlSocketSet)
	}
	env, err = loadEnvironment(func(key string) string {
		if key == "MARIONETTE_CONTROL_SOCKET" {
			return " ./marionette.sock "
		}
		return ""
	})
	if err != nil {
		t.Fatal(err)
	}
	if env.ControlSocket != "./marionette.sock" || !env.ControlSocketSet {
		t.Fatalf("explicit control socket = %q, set = %v", env.ControlSocket, env.ControlSocketSet)
	}
}

func TestOpenControlWithoutItsDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "control.sock")
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))

	server, err := openControl(environment{ControlSocket: path}, nil, logger)
	if err != nil || server != nil {
		t.Fatalf("default path without its directory = %v, %v; want no socket and no error", server, err)
	}
	if !strings.Contains(logs.String(), "marionette pair") {
		t.Fatalf("missing warning: %s", logs.String())
	}

	if _, err := openControl(environment{ControlSocket: path, ControlSocketSet: true}, nil, logger); err == nil {
		t.Fatal("an explicit path that cannot be used did not stop the start-up")
	}
}

// TestPairCommandAddsADeviceWhileOthersStayPaired runs the service with one
// paired device, gets a code with `marionette pair` and pairs a second
// device over HTTP (FR-58).
func TestPairCommandAddsADeviceWhileOthersStayPaired(t *testing.T) {
	directory := t.TempDir()
	devicesPath := filepath.Join(directory, "devices.json")
	registry, err := access.Open(devicesPath, access.Options{})
	if err != nil {
		t.Fatal(err)
	}
	first, _, err := registry.NewCode()
	if err != nil {
		t.Fatal(err)
	}
	_, laptopToken, err := registry.Pair(first, "Laptop")
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.Save(); err != nil {
		t.Fatal(err)
	}

	socketDir, err := os.MkdirTemp("", "mpair")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(socketDir) })
	socket := filepath.Join(socketDir, "control.sock")

	var logs safeBuffer
	listening := make(chan net.Addr, 1)
	app := application{
		env: environment{
			ConfigPath:       filepath.Join(directory, "marionette.json"),
			Addr:             "127.0.0.1:0",
			ShutdownTimeout:  2 * time.Second,
			AuthEnabled:      true,
			DevicesPath:      devicesPath,
			DeviceExpiry:     access.DefaultExpiry,
			ControlSocket:    socket,
			ControlSocketSet: true,
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

	var stdout, stderr bytes.Buffer
	getenv := func(key string) string {
		if key == "MARIONETTE_CONTROL_SOCKET" {
			return socket
		}
		return ""
	}
	if got := runCommand([]string{"pair"}, getenv, &stdout, &stderr); got != 0 {
		t.Fatalf("runCommand(pair) = %d, stderr = %s", got, stderr.String())
	}
	match := regexp.MustCompile(`Pairing code: ([0-9A-Z]{4}-[0-9A-Z]{4})\n`).FindStringSubmatch(stdout.String())
	if match == nil || !strings.Contains(stdout.String(), "Valid until:") {
		t.Fatalf("pair output = %q", stdout.String())
	}

	transport := &http.Transport{}
	defer transport.CloseIdleConnections()
	jar, _ := cookiejar.New(nil)
	phone := &http.Client{Jar: jar, Transport: transport}
	pairing, err := phone.Post(baseURL+"/api/pairing", "application/json",
		strings.NewReader(`{"code":"`+match[1]+`","name":"Phone"}`))
	if err != nil || pairing.StatusCode != http.StatusCreated {
		t.Fatalf("pairing with the code from the command = %v, %v", pairing, err)
	}
	_ = pairing.Body.Close()
	expectStatus(t, phone, baseURL+"/api/cards", http.StatusOK)

	laptopJar, _ := cookiejar.New(nil)
	base, _ := url.Parse(baseURL)
	laptopJar.SetCookies(base, []*http.Cookie{{Name: "marionette_device", Value: laptopToken}})
	expectStatus(t, &http.Client{Jar: laptopJar, Transport: transport}, baseURL+"/api/cards", http.StatusOK)

	// See TestRunRequiresAPairedDevice: close spare client connections so
	// the 2 s shutdown does not wait for them.
	transport.CloseIdleConnections()
	cancel()
	select {
	case err := <-finished:
		if err != nil {
			t.Fatalf("run() error = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("run() did not shut down")
	}
	if strings.Contains(logs.String(), match[1]) || strings.Contains(logs.String(), access.NormalizeCode(match[1])) {
		t.Fatalf("the code from the command leaked into the log: %s", logs.String())
	}
	for _, want := range []string{"control socket listening", "pairing code issued from the host", `step="stop control socket"`} {
		if !strings.Contains(logs.String(), want) {
			t.Fatalf("log misses %q: %s", want, logs.String())
		}
	}
	if _, err := os.Lstat(socket); !os.IsNotExist(err) {
		t.Fatalf("control socket after shutdown: %v", err)
	}
}
