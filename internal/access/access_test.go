package access

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type clock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *clock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func newClock() *clock {
	return &clock{now: time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)}
}

func openRegistry(t *testing.T, path string, c *clock, logs *bytes.Buffer) *Registry {
	t.Helper()
	options := Options{Now: c.Now}
	if logs != nil {
		options.Logger = slog.New(slog.NewTextHandler(logs, nil))
	}
	registry, err := Open(path, options)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	return registry
}

func pair(t *testing.T, registry *Registry, name string) (Device, string) {
	t.Helper()
	code, _, err := registry.NewCode()
	if err != nil {
		t.Fatalf("NewCode() error = %v", err)
	}
	device, token, err := registry.Pair(code, name)
	if err != nil {
		t.Fatalf("Pair() error = %v", err)
	}
	return device, token
}

func TestCodesUseTheUnambiguousAlphabet(t *testing.T) {
	registry := openRegistry(t, filepath.Join(t.TempDir(), "devices.json"), newClock(), nil)
	for range 200 {
		code, _, err := registry.NewCode()
		if err != nil {
			t.Fatalf("NewCode() error = %v", err)
		}
		if len(code) != CodeLength {
			t.Fatalf("code %q has %d characters", code, len(code))
		}
		for _, r := range code {
			if !strings.ContainsRune(codeAlphabet, r) {
				t.Fatalf("code %q contains %q outside the alphabet", code, r)
			}
		}
	}
	if got := NormalizeCode(" k7qm-3xrd "); got != "K7QM3XRD" {
		t.Fatalf("NormalizeCode() = %q", got)
	}
	if got := NormalizeCode("0oIl-abcd"); got != "0011ABCD" {
		t.Fatalf("NormalizeCode() look-alikes = %q", got)
	}
	if got := FormatCode("K7QM3XRD"); got != "K7QM-3XRD" {
		t.Fatalf("FormatCode() = %q", got)
	}
}

func TestPairingIssuesATokenThatAuthenticates(t *testing.T) {
	c := newClock()
	path := filepath.Join(t.TempDir(), "devices.json")
	registry := openRegistry(t, path, c, nil)
	if !registry.Empty() {
		t.Fatal("new registry is not empty")
	}
	code, expiresAt, err := registry.NewCode()
	if err != nil {
		t.Fatalf("NewCode() error = %v", err)
	}
	if !expiresAt.Equal(c.Now().Add(CodeLifetime)) {
		t.Fatalf("expiresAt = %v", expiresAt)
	}
	device, token, err := registry.Pair(strings.ToLower(FormatCode(code)), "  Work laptop ")
	if err != nil {
		t.Fatalf("Pair() error = %v", err)
	}
	if device.Name != "Work laptop" || device.ID == "" || len(token) < 40 {
		t.Fatalf("Pair() = %#v, token %q", device, token)
	}
	session, ok := registry.Authenticate(token)
	if !ok || session.Device.ID != device.ID || session.Renew {
		t.Fatalf("Authenticate() = %#v, %v", session, ok)
	}
	if _, ok := registry.Authenticate(token + "x"); ok {
		t.Fatal("a different token authenticated")
	}
	if _, ok := registry.Authenticate(""); ok {
		t.Fatal("an empty token authenticated")
	}
	// The code is single-use.
	if _, _, err := registry.Pair(code, "Second"); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("second Pair() with the same code error = %v", err)
	}
}

func TestPairingRejectsExpiredReplacedAndGuessedCodes(t *testing.T) {
	c := newClock()
	registry := openRegistry(t, filepath.Join(t.TempDir(), "devices.json"), c, nil)

	if _, _, err := registry.Pair("ABCDEFGH", "Phone"); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("Pair() without a code error = %v", err)
	}

	code, _, _ := registry.NewCode()
	c.Advance(CodeLifetime)
	if _, _, err := registry.Pair(code, "Phone"); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("Pair() with an expired code error = %v", err)
	}

	first, _, _ := registry.NewCode()
	second, _, _ := registry.NewCode()
	if first != second {
		if _, _, err := registry.Pair(first, "Phone"); !errors.Is(err, ErrInvalidCode) {
			t.Fatalf("Pair() with a replaced code error = %v", err)
		}
	}

	code, _, _ = registry.NewCode()
	for attempt := 1; attempt <= MaxCodeAttempts; attempt++ {
		if _, _, err := registry.Pair("ZZZZZZZZ", "Phone"); !errors.Is(err, ErrInvalidCode) {
			t.Fatalf("attempt %d error = %v", attempt, err)
		}
	}
	if registry.HasValidCode() {
		t.Fatal("code still valid after the maximum number of wrong attempts")
	}
	if _, _, err := registry.Pair(code, "Phone"); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("Pair() with the right code after too many attempts error = %v", err)
	}

	if _, _, err := registry.NewCode(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := registry.Pair("whatever", " "); !errors.Is(err, ErrInvalidName) {
		t.Fatalf("Pair() with a blank name error = %v", err)
	}
	if _, _, err := registry.Pair("whatever", strings.Repeat("n", MaxDeviceNameLength+1)); !errors.Is(err, ErrInvalidName) {
		t.Fatalf("Pair() with a long name error = %v", err)
	}
	if !registry.HasValidCode() {
		t.Fatal("an invalid name consumed an attempt")
	}
}

func TestDevicesPersistOnlyTokenHashes(t *testing.T) {
	c := newClock()
	path := filepath.Join(t.TempDir(), "devices.json")
	registry := openRegistry(t, path, c, nil)
	device, token := pair(t, registry, "Laptop")

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("devices file not written: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Fatalf("devices file mode = %o, want 600", mode)
	}
	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), token) {
		t.Fatal("the devices file contains the token")
	}
	var file devicesFile
	if err := json.Unmarshal(data, &file); err != nil || len(file.Devices) != 1 || file.Devices[0].TokenHash != hashToken(token) {
		t.Fatalf("devices file = %s, %v", data, err)
	}

	reopened := openRegistry(t, path, c, nil)
	if session, ok := reopened.Authenticate(token); !ok || session.Device.ID != device.ID {
		t.Fatal("token not accepted after reopening")
	}
}

func TestUseRenewsOncePerDayAndUnusedDevicesExpire(t *testing.T) {
	c := newClock()
	path := filepath.Join(t.TempDir(), "devices.json")
	registry, err := Open(path, Options{Now: c.Now, Expiry: 10 * 24 * time.Hour})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	_, token := pair(t, registry, "Laptop")

	c.Advance(time.Hour)
	if session, _ := registry.Authenticate(token); session.Renew {
		t.Fatal("renewed within the first day")
	}
	c.Advance(seenPersistInterval)
	session, ok := registry.Authenticate(token)
	if !ok || !session.Renew {
		t.Fatalf("not renewed after a day: %#v, %v", session, ok)
	}
	if session, _ := registry.Authenticate(token); session.Renew {
		t.Fatal("renewed twice in a row")
	}

	// A device in use never expires.
	for range 15 {
		c.Advance(24 * time.Hour)
		if _, ok := registry.Authenticate(token); !ok {
			t.Fatal("a device in daily use expired")
		}
	}

	c.Advance(10*24*time.Hour + time.Second)
	if _, ok := registry.Authenticate(token); ok {
		t.Fatal("an unused device did not expire")
	}
	if !registry.Empty() {
		t.Fatal("the expired device is still listed")
	}
	reopened := openRegistry(t, path, c, nil)
	if !reopened.Empty() {
		t.Fatal("the expired device came back after reopening")
	}
}

func TestExpiredDevicesAreDroppedWhenOpening(t *testing.T) {
	c := newClock()
	path := filepath.Join(t.TempDir(), "devices.json")
	registry := openRegistry(t, path, c, nil)
	pair(t, registry, "Old laptop")
	c.Advance(DefaultExpiry + time.Minute)
	var logs bytes.Buffer
	reopened := openRegistry(t, path, c, &logs)
	if !reopened.Empty() || !strings.Contains(logs.String(), "paired device expired") {
		t.Fatalf("expired device kept: %s", logs.String())
	}
}

func TestLastUseIsSavedOnSave(t *testing.T) {
	c := newClock()
	path := filepath.Join(t.TempDir(), "devices.json")
	registry := openRegistry(t, path, c, nil)
	_, token := pair(t, registry, "Laptop")
	c.Advance(time.Hour)
	registry.Authenticate(token)
	if err := registry.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	reopened := openRegistry(t, path, c, nil)
	if devices := reopened.List(); len(devices) != 1 || !devices[0].LastSeenAt.Equal(c.Now()) {
		t.Fatalf("last use not saved: %#v", devices)
	}

	empty := openRegistry(t, filepath.Join(t.TempDir(), "devices.json"), c, nil)
	if err := empty.Save(); err != nil {
		t.Fatalf("Save() of an empty registry error = %v", err)
	}
	if _, err := os.Stat(empty.path); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("an empty registry created a devices file")
	}
}

func TestRemoveListAndExists(t *testing.T) {
	c := newClock()
	registry := openRegistry(t, filepath.Join(t.TempDir(), "devices.json"), c, nil)
	laptop, laptopToken := pair(t, registry, "Laptop")
	c.Advance(time.Minute)
	phone, _ := pair(t, registry, "Phone")

	devices := registry.List()
	if len(devices) != 2 || devices[0].ID != phone.ID || devices[1].ID != laptop.ID {
		t.Fatalf("List() = %#v, want most recently used first", devices)
	}
	if err := registry.Remove(laptop.ID); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if registry.Exists(laptop.ID) || !registry.Exists(phone.ID) {
		t.Fatal("Exists() does not follow Remove()")
	}
	if _, ok := registry.Authenticate(laptopToken); ok {
		t.Fatal("a removed device still authenticates")
	}
	if err := registry.Remove(laptop.ID); !errors.Is(err, ErrDeviceNotFound) {
		t.Fatalf("second Remove() error = %v", err)
	}
}

func TestCorruptDevicesFileIsQuarantined(t *testing.T) {
	c := newClock()
	dir := t.TempDir()
	path := filepath.Join(dir, "devices.json")
	for _, content := range []string{"{not json", `{"devices":[{"id":"a","tokenHash":""}]}`} {
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		var logs bytes.Buffer
		registry := openRegistry(t, path, c, &logs)
		if !registry.Empty() || !strings.Contains(logs.String(), "quarantined") {
			t.Fatalf("corrupt file %q not quarantined: %s", content, logs.String())
		}
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("corrupt file still in place")
		}
		c.Advance(time.Second)
	}
	matches, _ := filepath.Glob(path + ".corrupt-*")
	if len(matches) != 2 {
		t.Fatalf("quarantined files = %v", matches)
	}
}

func TestUnreadableDevicesFileFailsClosed(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can read any file")
	}
	path := filepath.Join(t.TempDir(), "devices.json")
	if err := os.WriteFile(path, []byte(`{"devices":[]}`), 0o000); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(path, Options{}); err == nil {
		t.Fatal("Open() of an unreadable file succeeded")
	}
}

func TestRegistryIsSafeForConcurrentUse(t *testing.T) {
	c := newClock()
	registry := openRegistry(t, filepath.Join(t.TempDir(), "devices.json"), c, nil)
	_, token := pair(t, registry, "Laptop")
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 50 {
				registry.Authenticate(token)
				registry.List()
				registry.Exists("x")
				registry.HasValidCode()
				c.Advance(time.Hour)
			}
		}()
	}
	wg.Wait()
}
