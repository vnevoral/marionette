// Package access keeps the paired devices and the one-time pairing code
// (FR-50..FR-56, NFR-13, ADR-0011). A device proves itself with a random
// token that the browser keeps in a cookie; only the SHA-256 hash of the
// token is stored, in its own file next to the configuration.
package access

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"marionette/internal/fsutil"
)

const (
	// DefaultExpiry is how long an unused device stays paired (FR-52).
	DefaultExpiry = 60 * 24 * time.Hour
	// MaxExpiry is the upper bound of the expiry; browsers cap cookie
	// lifetimes at 400 days.
	MaxExpiry = 400 * 24 * time.Hour
	// CodeLifetime is how long a pairing code stays valid (FR-51).
	CodeLifetime = 10 * time.Minute
	// MaxCodeAttempts is the number of wrong codes after which the current
	// code is invalidated (FR-51).
	MaxCodeAttempts = 5
	// CodeLength is the number of characters of a pairing code.
	CodeLength = 8
	// MaxDeviceNameLength limits the name given at pairing or renaming.
	MaxDeviceNameLength = 64
	// seenPersistInterval bounds how often the last use of a device is
	// written to disk (and its cookie renewed), to spare SD cards.
	seenPersistInterval = 24 * time.Hour
)

// codeAlphabet is Crockford's Base32: no I, L, O or U, so a code read from a
// log or a screen cannot be mistyped between look-alike characters.
const codeAlphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// Errors returned by the registry.
var (
	// ErrInvalidCode covers a wrong, expired, used or exhausted pairing code;
	// callers show one message for all of them.
	ErrInvalidCode = errors.New("the pairing code is invalid or has expired")
	// ErrDeviceNotFound is returned when removing or renaming an unknown device.
	ErrDeviceNotFound = errors.New("device not found")
	// ErrInvalidName is returned for an empty or too long device name.
	ErrInvalidName = fmt.Errorf("device name is required and must be at most %d characters", MaxDeviceNameLength)
	// ErrDevicesCorrupt means the devices file exists but cannot be decoded.
	ErrDevicesCorrupt = errors.New("devices file is corrupt")
)

// Device is one paired browser. TokenHash is the hex SHA-256 of its token.
type Device struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	TokenHash  string    `json:"tokenHash"`
	PairedAt   time.Time `json:"pairedAt"`
	LastSeenAt time.Time `json:"lastSeenAt"`
}

// Session is the result of a successful authentication.
type Session struct {
	Device Device
	// Renew reports that the cookie should be sent again with a fresh
	// lifetime; it is true at most once per seenPersistInterval.
	Renew bool
}

type devicesFile struct {
	Devices []Device `json:"devices"`
}

type pairingCode struct {
	code      string
	expiresAt time.Time
	attempts  int
}

// Registry is the in-memory set of paired devices backed by a JSON file. It
// is safe for concurrent use.
type Registry struct {
	path   string
	expiry time.Duration
	now    func() time.Time
	logger *slog.Logger
	random io.Reader

	mu        sync.Mutex
	devices   map[string]*Device // by ID
	byHash    map[string]string  // token hash → ID
	persisted map[string]time.Time
	code      *pairingCode
}

// Options configure Open. Zero values select the defaults.
type Options struct {
	Expiry time.Duration
	Now    func() time.Time
	Logger *slog.Logger
	// Random is the entropy source; crypto/rand when nil. Tests only.
	Random io.Reader
}

// Open loads the devices file at path. A missing file yields an empty
// registry; a file that cannot be decoded is quarantined
// ("<path>.corrupt-<time>", logged as a warning) and the registry starts
// empty, so the operator pairs again with the code from the log. A file that
// cannot be read is an error: access must fail closed. Devices unused for
// longer than the expiry are dropped.
func Open(path string, options Options) (*Registry, error) {
	registry := &Registry{
		path:      path,
		expiry:    options.Expiry,
		now:       options.Now,
		logger:    options.Logger,
		random:    options.Random,
		devices:   map[string]*Device{},
		byHash:    map[string]string{},
		persisted: map[string]time.Time{},
	}
	if registry.expiry <= 0 {
		registry.expiry = DefaultExpiry
	}
	if registry.now == nil {
		registry.now = time.Now
	}
	if registry.logger == nil {
		registry.logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	if registry.random == nil {
		registry.random = rand.Reader
	}

	data, err := os.ReadFile(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return registry, nil
	case err != nil:
		return nil, fmt.Errorf("read devices file %q: %w", path, err)
	}
	var file devicesFile
	if err := json.Unmarshal(data, &file); err != nil || !validDevices(file.Devices) {
		quarantined, quarantineErr := fsutil.Quarantine(path, registry.now())
		if quarantineErr != nil {
			return nil, fmt.Errorf("%w (%q) and could not be quarantined: %w", ErrDevicesCorrupt, path, quarantineErr)
		}
		registry.logger.Warn("devices file is corrupt; it was quarantined and every device must pair again", "quarantined", quarantined)
		return registry, nil
	}
	now := registry.now()
	for index := range file.Devices {
		device := file.Devices[index]
		if registry.expired(device, now) {
			registry.logger.Info("paired device expired", "device", device.ID, "name", device.Name)
			continue
		}
		registry.devices[device.ID] = &device
		registry.byHash[device.TokenHash] = device.ID
		registry.persisted[device.ID] = device.LastSeenAt
	}
	return registry, nil
}

func validDevices(devices []Device) bool {
	seen := map[string]bool{}
	for _, device := range devices {
		if device.ID == "" || device.TokenHash == "" || seen[device.ID] {
			return false
		}
		seen[device.ID] = true
	}
	return true
}

// Expiry returns how long an unused device stays paired.
func (registry *Registry) Expiry() time.Duration {
	return registry.expiry
}

// Empty reports whether no device is paired.
func (registry *Registry) Empty() bool {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	registry.dropExpiredLocked(registry.now())
	return len(registry.devices) == 0
}

// NewCode creates a pairing code valid for CodeLifetime and invalidates any
// previous one (FR-51).
func (registry *Registry) NewCode() (string, time.Time, error) {
	code, err := registry.randomCode()
	if err != nil {
		return "", time.Time{}, err
	}
	registry.mu.Lock()
	defer registry.mu.Unlock()
	registry.code = &pairingCode{code: code, expiresAt: registry.now().Add(CodeLifetime)}
	return code, registry.code.expiresAt, nil
}

// HasValidCode reports whether a pairing code can currently be used.
func (registry *Registry) HasValidCode() bool {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	return registry.validCodeLocked(registry.now())
}

func (registry *Registry) validCodeLocked(now time.Time) bool {
	return registry.code != nil && now.Before(registry.code.expiresAt) && registry.code.attempts < MaxCodeAttempts
}

// Pair exchanges a valid pairing code for a new device and its token. The
// code is consumed on success; a wrong code counts as an attempt and the
// code is invalidated after MaxCodeAttempts. Every failure is
// ErrInvalidCode (or ErrInvalidName) so the caller cannot tell which case
// applied.
func (registry *Registry) Pair(code, name string) (Device, string, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > MaxDeviceNameLength {
		return Device{}, "", ErrInvalidName
	}
	token, err := registry.randomToken()
	if err != nil {
		return Device{}, "", err
	}
	id, err := registry.randomID()
	if err != nil {
		return Device{}, "", err
	}

	registry.mu.Lock()
	defer registry.mu.Unlock()
	now := registry.now()
	if !registry.validCodeLocked(now) {
		registry.code = nil
		return Device{}, "", ErrInvalidCode
	}
	given := NormalizeCode(code)
	if subtle.ConstantTimeCompare([]byte(given), []byte(registry.code.code)) != 1 {
		registry.code.attempts++
		if registry.code.attempts >= MaxCodeAttempts {
			registry.code = nil
		}
		return Device{}, "", ErrInvalidCode
	}
	registry.code = nil

	device := Device{ID: id, Name: name, TokenHash: hashToken(token), PairedAt: now, LastSeenAt: now}
	registry.devices[id] = &device
	registry.byHash[device.TokenHash] = id
	if err := registry.saveLocked(); err != nil && !errors.Is(err, fsutil.ErrDirectorySync) {
		delete(registry.devices, id)
		delete(registry.byHash, device.TokenHash)
		return Device{}, "", err
	}
	registry.persisted[id] = now
	return device, token, nil
}

// Authenticate returns the device for a token. It records the use; the last
// use reaches the disk, and Session.Renew is set, at most once per day per
// device. An expired device is removed.
func (registry *Registry) Authenticate(token string) (Session, bool) {
	if token == "" {
		return Session{}, false
	}
	hash := hashToken(token)
	registry.mu.Lock()
	defer registry.mu.Unlock()
	id, found := registry.byHash[hash]
	if !found {
		return Session{}, false
	}
	device := registry.devices[id]
	now := registry.now()
	if registry.expired(*device, now) {
		registry.removeLocked(id)
		registry.logger.Info("paired device expired", "device", id, "name", device.Name)
		registry.saveLoggingLocked()
		return Session{}, false
	}
	device.LastSeenAt = now
	renew := now.Sub(registry.persisted[id]) >= seenPersistInterval
	if renew {
		registry.persisted[id] = now
		registry.saveLoggingLocked()
	}
	return Session{Device: *device, Renew: renew}, true
}

// Exists reports whether a device is still paired (used by long-lived
// streams to end when their device is removed).
func (registry *Registry) Exists(id string) bool {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	_, found := registry.devices[id]
	return found
}

// List returns the paired devices, most recently used first.
func (registry *Registry) List() []Device {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	registry.dropExpiredLocked(registry.now())
	devices := make([]Device, 0, len(registry.devices))
	for _, device := range registry.devices {
		devices = append(devices, *device)
	}
	sort.Slice(devices, func(i, j int) bool {
		if !devices[i].LastSeenAt.Equal(devices[j].LastSeenAt) {
			return devices[i].LastSeenAt.After(devices[j].LastSeenAt)
		}
		return devices[i].ID < devices[j].ID
	})
	return devices
}

// Remove unpairs a device; its token stops working immediately.
func (registry *Registry) Remove(id string) error {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	device, found := registry.devices[id]
	if !found {
		return ErrDeviceNotFound
	}
	registry.removeLocked(id)
	if err := registry.saveLocked(); err != nil && !errors.Is(err, fsutil.ErrDirectorySync) {
		registry.devices[id] = device
		registry.byHash[device.TokenHash] = id
		return err
	}
	return nil
}

// Rename changes the name of a paired device (FR-57) and saves it at once.
// The name is validated like at pairing; the token and the times stay
// unchanged. An unknown or expired device is ErrDeviceNotFound.
func (registry *Registry) Rename(id, name string) (Device, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > MaxDeviceNameLength {
		return Device{}, ErrInvalidName
	}
	registry.mu.Lock()
	defer registry.mu.Unlock()
	registry.dropExpiredLocked(registry.now())
	device, found := registry.devices[id]
	if !found {
		return Device{}, ErrDeviceNotFound
	}
	previous := device.Name
	device.Name = name
	if err := registry.saveLocked(); err != nil && !errors.Is(err, fsutil.ErrDirectorySync) {
		device.Name = previous
		return Device{}, err
	}
	registry.logger.Info("renamed paired device", "device", id, "name", name)
	return *device, nil
}

// Save writes the devices, including the last use kept in memory (called at
// shutdown).
func (registry *Registry) Save() error {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	if len(registry.devices) == 0 {
		if _, err := os.Stat(registry.path); errors.Is(err, os.ErrNotExist) {
			return nil
		}
	}
	return registry.saveLocked()
}

func (registry *Registry) expired(device Device, now time.Time) bool {
	return now.Sub(device.LastSeenAt) > registry.expiry
}

func (registry *Registry) dropExpiredLocked(now time.Time) {
	changed := false
	for id, device := range registry.devices {
		if registry.expired(*device, now) {
			registry.removeLocked(id)
			registry.logger.Info("paired device expired", "device", id, "name", device.Name)
			changed = true
		}
	}
	if changed {
		registry.saveLoggingLocked()
	}
}

func (registry *Registry) removeLocked(id string) {
	if device, found := registry.devices[id]; found {
		delete(registry.byHash, device.TokenHash)
	}
	delete(registry.devices, id)
	delete(registry.persisted, id)
}

func (registry *Registry) saveLoggingLocked() {
	if err := registry.saveLocked(); err != nil {
		registry.logger.Warn("devices file could not be saved", "path", registry.path, "error", err)
	}
}

func (registry *Registry) saveLocked() error {
	file := devicesFile{Devices: make([]Device, 0, len(registry.devices))}
	for _, device := range registry.devices {
		file.Devices = append(file.Devices, *device)
	}
	sort.Slice(file.Devices, func(i, j int) bool { return file.Devices[i].ID < file.Devices[j].ID })
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("encode devices file: %w", err)
	}
	if err := fsutil.WriteFileAtomic(registry.path, data, 0o600); err != nil {
		return fmt.Errorf("save devices file: %w", err)
	}
	return nil
}

// NormalizeCode upper-cases a typed code, drops separators and maps the
// look-alike letters Crockford Base32 excludes (I, L → 1; O → 0).
func NormalizeCode(code string) string {
	var builder strings.Builder
	for _, r := range strings.ToUpper(code) {
		switch r {
		case '-', ' ', '\t':
			continue
		case 'I', 'L':
			r = '1'
		case 'O':
			r = '0'
		}
		builder.WriteRune(r)
	}
	return builder.String()
}

// FormatCode groups a code for display ("K7QM-3XRD").
func FormatCode(code string) string {
	if len(code) != CodeLength {
		return code
	}
	return code[:4] + "-" + code[4:]
}

func (registry *Registry) randomCode() (string, error) {
	buffer := make([]byte, CodeLength)
	if _, err := io.ReadFull(registry.random, buffer); err != nil {
		return "", fmt.Errorf("generate pairing code: %w", err)
	}
	for index, value := range buffer {
		// 256 is a multiple of 32, so the modulo keeps the distribution uniform.
		buffer[index] = codeAlphabet[int(value)%len(codeAlphabet)]
	}
	return string(buffer), nil
}

func (registry *Registry) randomToken() (string, error) {
	buffer := make([]byte, 32)
	if _, err := io.ReadFull(registry.random, buffer); err != nil {
		return "", fmt.Errorf("generate device token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func (registry *Registry) randomID() (string, error) {
	buffer := make([]byte, 8)
	if _, err := io.ReadFull(registry.random, buffer); err != nil {
		return "", fmt.Errorf("generate device id: %w", err)
	}
	return hex.EncodeToString(buffer), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
