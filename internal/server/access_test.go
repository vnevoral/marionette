package server

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"marionette/internal/access"
	"marionette/internal/config"
)

type accessFixture struct {
	handler  http.Handler
	registry *access.Registry
	logs     *bytes.Buffer
}

func newAccessFixture(t *testing.T, secure CookieSecurity) accessFixture {
	t.Helper()
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	registry, err := access.Open(filepath.Join(t.TempDir(), "devices.json"), access.Options{Logger: logger})
	if err != nil {
		t.Fatalf("access.Open() error = %v", err)
	}
	store := config.NewStore(config.Settings{HistorySize: 5, MaxConcurrentActions: 1})
	createServerCard(t, store, "guarded")
	queue := &fakeActionQueue{primaryCalls: make(chan string, 16), statusCalls: make(chan string, 16)}
	handler := NewRouter(Dependencies{
		Store:   store,
		Actions: queue,
		Logger:  logger,
		Access:  &Access{Registry: registry, Secure: secure},
	})
	return accessFixture{handler: handler, registry: registry, logs: &logs}
}

func (fixture accessFixture) do(method, path, body string, cookie *http.Cookie) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" || method != http.MethodGet {
		request.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		request.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	fixture.handler.ServeHTTP(response, request)
	return response
}

// pairDevice pairs through the API and returns the device cookie.
func (fixture accessFixture) pairDevice(t *testing.T, name string) *http.Cookie {
	t.Helper()
	code, _, err := fixture.registry.NewCode()
	if err != nil {
		t.Fatalf("NewCode() error = %v", err)
	}
	response := fixture.do(http.MethodPost, "/api/pairing", `{"code":"`+code+`","name":"`+name+`"}`, nil)
	if response.Code != http.StatusCreated {
		t.Fatalf("pairing = %d %s", response.Code, response.Body.String())
	}
	return deviceCookie(t, response)
}

func deviceCookie(t *testing.T, response *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == DeviceCookie {
			return cookie
		}
	}
	t.Fatalf("no %s cookie in %v", DeviceCookie, response.Header().Values("Set-Cookie"))
	return nil
}

var protectedRoutes = []struct{ method, path string }{
	{http.MethodGet, "/api/cards"},
	{http.MethodPost, "/api/cards"},
	{http.MethodGet, "/api/cards/guarded"},
	{http.MethodPut, "/api/cards/guarded"},
	{http.MethodDelete, "/api/cards/guarded"},
	{http.MethodGet, "/api/cards/guarded/runs"},
	{http.MethodGet, "/api/cards/guarded/status"},
	{http.MethodGet, "/api/cards/guarded/status/history"},
	{http.MethodPost, "/api/cards/guarded/actions/primary"},
	{http.MethodPost, "/api/cards/guarded/actions/status/check"},
	{http.MethodGet, "/api/events"},
	{http.MethodGet, "/api/devices"},
	{http.MethodPost, "/api/pairing/code"},
	{http.MethodDelete, "/api/devices/some-id"},
	{http.MethodGet, "/api/unknown"},
}

func TestAccessRejectsEveryProtectedRouteWithoutADevice(t *testing.T) {
	fixture := newAccessFixture(t, CookieSecureAuto)
	forged := &http.Cookie{Name: DeviceCookie, Value: "forged-token"}
	for _, route := range protectedRoutes {
		for _, cookie := range []*http.Cookie{nil, forged} {
			response := fixture.do(route.method, route.path, "", cookie)
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("%s %s (cookie %v) = %d, want 401", route.method, route.path, cookie != nil, response.Code)
			}
			var body map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body["code"] != "pairing_required" || body["error"] == "" {
				t.Fatalf("%s %s body = %s", route.method, route.path, response.Body.String())
			}
		}
	}
}

func TestAccessKeepsHealthAndSPAPublic(t *testing.T) {
	fixture := newAccessFixture(t, CookieSecureAuto)
	for _, path := range []string{"/api/health", "/", "/cards/guarded", "/pair"} {
		if response := fixture.do(http.MethodGet, path, "", nil); response.Code != http.StatusOK {
			t.Fatalf("GET %s = %d, want 200", path, response.Code)
		}
	}
}

func TestSessionLogsABootstrapCodeOnlyWhileNoDeviceIsPaired(t *testing.T) {
	fixture := newAccessFixture(t, CookieSecureAuto)
	codePattern := regexp.MustCompile(`code=([0-9A-Z]{4}-[0-9A-Z]{4})`)

	response := fixture.do(http.MethodGet, "/api/session", "", nil)
	var body map[string]any
	_ = json.Unmarshal(response.Body.Bytes(), &body)
	if response.Code != http.StatusUnauthorized || body["bootstrap"] != true {
		t.Fatalf("session without devices = %d %s", response.Code, response.Body.String())
	}
	match := codePattern.FindStringSubmatch(fixture.logs.String())
	if match == nil {
		t.Fatalf("no pairing code in the log: %s", fixture.logs.String())
	}
	// A second visit keeps the valid code instead of logging a new one.
	fixture.do(http.MethodGet, "/api/session", "", nil)
	if count := len(codePattern.FindAllString(fixture.logs.String(), -1)); count != 1 {
		t.Fatalf("pairing code logged %d times", count)
	}

	pairing := fixture.do(http.MethodPost, "/api/pairing", `{"code":"`+strings.ToLower(match[1])+`","name":"Laptop"}`, nil)
	if pairing.Code != http.StatusCreated {
		t.Fatalf("pairing with the logged code = %d %s", pairing.Code, pairing.Body.String())
	}
	cookie := deviceCookie(t, pairing)
	if strings.Contains(fixture.logs.String(), cookie.Value) {
		t.Fatal("the device token was written to the log")
	}
	if strings.Contains(pairing.Body.String(), cookie.Value) {
		t.Fatal("the device token was returned in the response body")
	}

	session := fixture.do(http.MethodGet, "/api/session", "", cookie)
	if session.Code != http.StatusOK || !strings.Contains(session.Body.String(), `"name":"Laptop"`) || !strings.Contains(session.Body.String(), `"expiryDays":60`) {
		t.Fatalf("session with a device = %d %s", session.Code, session.Body.String())
	}
	other := fixture.do(http.MethodGet, "/api/session", "", nil)
	_ = json.Unmarshal(other.Body.Bytes(), &body)
	if other.Code != http.StatusUnauthorized || body["bootstrap"] != false {
		t.Fatalf("session of another browser = %d %s", other.Code, other.Body.String())
	}
	if count := len(codePattern.FindAllString(fixture.logs.String(), -1)); count != 1 {
		t.Fatal("a pairing code was logged although a device is paired")
	}
}

func TestPairingSetsAStrictHttpOnlyCookie(t *testing.T) {
	cases := []struct {
		name   string
		mode   CookieSecurity
		tls    bool
		secure bool
	}{
		{"auto over http", CookieSecureAuto, false, false},
		{"auto over https", CookieSecureAuto, true, true},
		{"always", CookieSecureAlways, false, true},
		{"never over https", CookieSecureNever, true, false},
	}
	for _, testCase := range cases {
		fixture := newAccessFixture(t, testCase.mode)
		code, _, _ := fixture.registry.NewCode()
		request := httptest.NewRequest(http.MethodPost, "/api/pairing", strings.NewReader(`{"code":"`+code+`","name":"Phone"}`))
		request.Header.Set("Content-Type", "application/json")
		if testCase.tls {
			request.TLS = &tls.ConnectionState{}
		}
		response := httptest.NewRecorder()
		fixture.handler.ServeHTTP(response, request)
		cookie := deviceCookie(t, response)
		if !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode || cookie.Path != "/" {
			t.Fatalf("%s: cookie attributes = %#v", testCase.name, cookie)
		}
		if cookie.MaxAge != int(access.DefaultExpiry/time.Second) {
			t.Fatalf("%s: Max-Age = %d", testCase.name, cookie.MaxAge)
		}
		if cookie.Secure != testCase.secure {
			t.Fatalf("%s: Secure = %v, want %v", testCase.name, cookie.Secure, testCase.secure)
		}
	}
}

func TestPairingErrors(t *testing.T) {
	fixture := newAccessFixture(t, CookieSecureAuto)
	if _, _, err := fixture.registry.NewCode(); err != nil {
		t.Fatal(err)
	}
	wrong := fixture.do(http.MethodPost, "/api/pairing", `{"code":"ZZZZ-ZZZZ","name":"Phone"}`, nil)
	if wrong.Code != http.StatusBadRequest || !strings.Contains(wrong.Body.String(), "invalid or has expired") {
		t.Fatalf("wrong code = %d %s", wrong.Code, wrong.Body.String())
	}
	if len(wrong.Result().Cookies()) != 0 {
		t.Fatal("a wrong code set a cookie")
	}
	if !strings.Contains(fixture.logs.String(), "pairing attempt with an invalid code") || strings.Contains(fixture.logs.String(), "ZZZZ") {
		t.Fatalf("failed attempt log = %s", fixture.logs.String())
	}
	blank := fixture.do(http.MethodPost, "/api/pairing", `{"code":"ZZZZ-ZZZZ","name":" "}`, nil)
	if blank.Code != http.StatusUnprocessableEntity || !strings.Contains(blank.Body.String(), `"name"`) {
		t.Fatalf("blank name = %d %s", blank.Code, blank.Body.String())
	}
	if malformed := fixture.do(http.MethodPost, "/api/pairing", `{"code":1}`, nil); malformed.Code != http.StatusBadRequest {
		t.Fatalf("malformed body = %d", malformed.Code)
	}
	// Pairing stays subject to the cross-site protection (NFR-12).
	request := httptest.NewRequest(http.MethodPost, "/api/pairing", strings.NewReader(`{"code":"x","name":"y"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "https://evil.example")
	response := httptest.NewRecorder()
	fixture.handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("cross-site pairing = %d, want 403", response.Code)
	}
}

func TestPairedDeviceUsesTheAPIAndManagesDevices(t *testing.T) {
	fixture := newAccessFixture(t, CookieSecureAuto)
	laptop := fixture.pairDevice(t, "Laptop")

	if response := fixture.do(http.MethodGet, "/api/cards", "", laptop); response.Code != http.StatusOK {
		t.Fatalf("GET /api/cards with a device = %d", response.Code)
	}
	if response := fixture.do(http.MethodPost, "/api/cards/guarded/actions/primary", "", laptop); response.Code != http.StatusAccepted {
		t.Fatalf("run with a device = %d %s", response.Code, response.Body.String())
	}

	codeResponse := fixture.do(http.MethodPost, "/api/pairing/code", "", laptop)
	var created struct {
		Code      string    `json:"code"`
		ExpiresAt time.Time `json:"expiresAt"`
	}
	if err := json.Unmarshal(codeResponse.Body.Bytes(), &created); codeResponse.Code != http.StatusCreated || err != nil || len(created.Code) != 9 || created.ExpiresAt.IsZero() {
		t.Fatalf("pairing code = %d %s", codeResponse.Code, codeResponse.Body.String())
	}
	phonePairing := fixture.do(http.MethodPost, "/api/pairing", `{"code":"`+created.Code+`","name":"Phone"}`, nil)
	if phonePairing.Code != http.StatusCreated {
		t.Fatalf("pairing with a code from Devices = %d", phonePairing.Code)
	}
	phone := deviceCookie(t, phonePairing)

	list := fixture.do(http.MethodGet, "/api/devices", "", laptop)
	var devices []struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		Current   bool   `json:"current"`
		TokenHash string `json:"tokenHash"`
	}
	if err := json.Unmarshal(list.Body.Bytes(), &devices); err != nil || len(devices) != 2 {
		t.Fatalf("devices = %s", list.Body.String())
	}
	var laptopID, phoneID string
	for _, device := range devices {
		if device.TokenHash != "" {
			t.Fatal("the device list exposes the token hash")
		}
		switch device.Name {
		case "Laptop":
			laptopID = device.ID
			if !device.Current {
				t.Fatal("the requesting device is not marked current")
			}
		case "Phone":
			phoneID = device.ID
			if device.Current {
				t.Fatal("another device is marked current")
			}
		}
	}

	// Removing another device keeps the own cookie; the removed one stops working.
	removed := fixture.do(http.MethodDelete, "/api/devices/"+phoneID, "", laptop)
	if removed.Code != http.StatusNoContent || len(removed.Result().Cookies()) != 0 {
		t.Fatalf("remove other device = %d, cookies %v", removed.Code, removed.Result().Cookies())
	}
	if response := fixture.do(http.MethodGet, "/api/cards", "", phone); response.Code != http.StatusUnauthorized {
		t.Fatalf("removed device = %d, want 401", response.Code)
	}
	if response := fixture.do(http.MethodDelete, "/api/devices/"+phoneID, "", laptop); response.Code != http.StatusNotFound {
		t.Fatalf("remove unknown device = %d, want 404", response.Code)
	}

	// Removing the own device is a sign-out: the cookie is cleared.
	self := fixture.do(http.MethodDelete, "/api/devices/"+laptopID, "", laptop)
	if self.Code != http.StatusNoContent {
		t.Fatalf("remove own device = %d", self.Code)
	}
	if cleared := deviceCookie(t, self); cleared.MaxAge >= 0 || cleared.Value != "" {
		t.Fatalf("own removal did not clear the cookie: %#v", cleared)
	}
	if response := fixture.do(http.MethodGet, "/api/cards", "", laptop); response.Code != http.StatusUnauthorized {
		t.Fatalf("after signing out = %d, want 401", response.Code)
	}
}

func TestEventStreamEndsWhenItsDeviceIsRemoved(t *testing.T) {
	previous := statusEventHeartbeat
	statusEventHeartbeat = 20 * time.Millisecond
	t.Cleanup(func() { statusEventHeartbeat = previous })

	fixture := newAccessFixture(t, CookieSecureAuto)
	cookie := fixture.pairDevice(t, "Laptop")
	server := httptest.NewServer(fixture.handler)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/api/events", nil)
	request.AddCookie(cookie)
	response, err := http.DefaultClient.Do(request)
	if err != nil || response.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/events = %v, %v", response, err)
	}
	defer func() { _ = response.Body.Close() }()
	reader := bufio.NewReader(response.Body)
	if line, _ := reader.ReadString('\n'); !strings.HasPrefix(line, ": connected") {
		t.Fatalf("first line = %q", line)
	}

	id := fixture.registry.List()[0].ID
	if err := fixture.registry.Remove(id); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if _, err := io.ReadAll(reader); err != nil {
		t.Fatalf("stream did not end cleanly after the device was removed: %v", err)
	}
	if ctx.Err() != nil {
		t.Fatal("stream kept running after the device was removed")
	}
}

func TestRouterWithoutAccessKeepsTheAPIOpen(t *testing.T) {
	store := config.NewStore(config.Settings{HistorySize: 5, MaxConcurrentActions: 1})
	handler := NewRouter(Dependencies{Store: store, Actions: &fakeActionQueue{}})
	for _, path := range []string{"/api/cards", "/api/session", "/api/devices"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		want := http.StatusOK
		if path != "/api/cards" {
			want = http.StatusNotFound
		}
		if response.Code != want {
			t.Fatalf("GET %s without access control = %d, want %d", path, response.Code, want)
		}
	}
}
