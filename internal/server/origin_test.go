package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"marionette/internal/config"
)

// mutatingRoutes lists every mutating API route registered by the router;
// each one must be covered by the cross-site protection.
var mutatingRoutes = []struct {
	method string
	path   string
}{
	{http.MethodPost, "/api/cards"},
	{http.MethodPut, "/api/cards/protected"},
	{http.MethodDelete, "/api/cards/protected"},
	{http.MethodPost, "/api/cards/protected/actions/primary"},
	{http.MethodPost, "/api/cards/protected/actions/status/check"},
}

func protectedRouter(t *testing.T, allowedHosts ...string) http.Handler {
	t.Helper()
	store := config.NewStore(config.Settings{HistorySize: 5, MaxConcurrentActions: 1})
	createServerCard(t, store, "protected")
	queue := &fakeActionQueue{primaryCalls: make(chan string, 16), statusCalls: make(chan string, 16)}
	return NewRouter(Dependencies{Store: store, Actions: queue, AllowedHosts: allowedHosts})
}

func send(handler http.Handler, method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	var reader *strings.Reader
	if body != "" {
		reader = strings.NewReader(body)
	} else {
		reader = strings.NewReader("")
	}
	request := httptest.NewRequest(method, path, reader)
	request.Host = "pi.local:8080"
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func assertJSONError(t *testing.T, response *httptest.ResponseRecorder, wantStatus int) {
	t.Helper()
	if response.Code != wantStatus {
		t.Fatalf("status = %d, want %d (body %s)", response.Code, wantStatus, response.Body.String())
	}
	var envelope map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil || envelope["error"] == "" {
		t.Fatalf("error envelope = %q, %v", response.Body.String(), err)
	}
}

func TestMutatingRoutesRejectCrossSiteRequests(t *testing.T) {
	handler := protectedRouter(t)
	for _, route := range mutatingRoutes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			assertJSONError(t, send(handler, route.method, route.path, "", map[string]string{"Sec-Fetch-Site": "cross-site"}), http.StatusForbidden)
			assertJSONError(t, send(handler, route.method, route.path, "", map[string]string{"Origin": "http://evil.example"}), http.StatusForbidden)
			assertJSONError(t, send(handler, route.method, route.path, "", map[string]string{"Origin": "null"}), http.StatusForbidden)
			assertJSONError(t, send(handler, route.method, route.path, "x=1", map[string]string{"Content-Type": "application/x-www-form-urlencoded"}), http.StatusUnsupportedMediaType)
			assertJSONError(t, send(handler, route.method, route.path, "{}", map[string]string{"Content-Type": "text/plain"}), http.StatusUnsupportedMediaType)
			assertJSONError(t, send(handler, route.method, route.path, "{}", nil), http.StatusUnsupportedMediaType)
		})
	}
}

func TestMutatingRoutesAcceptSameOriginAndNonBrowserClients(t *testing.T) {
	handler := protectedRouter(t)
	path := "/api/cards/protected/actions/primary"
	accepted := []map[string]string{
		{"Sec-Fetch-Site": "same-origin", "Origin": "http://pi.local:8080", "Content-Type": "application/json"},
		{"Origin": "HTTP://PI.LOCAL:8080", "Content-Type": "application/json; charset=utf-8"},
		{"Sec-Fetch-Site": "none"},
		{}, // curl -X POST without Origin, Sec-Fetch-Site or body
	}
	for index, headers := range accepted {
		if response := send(handler, http.MethodPost, path, "", headers); response.Code != http.StatusAccepted {
			t.Fatalf("case %d headers %v: status = %d, body %s", index, headers, response.Code, response.Body.String())
		}
	}
	if response := send(handler, http.MethodPost, path, "null", map[string]string{"Content-Type": "application/json"}); response.Code != http.StatusAccepted {
		t.Fatalf("JSON body without Origin: status = %d", response.Code)
	}
}

func TestReadOnlyRoutesIgnoreForeignOrigin(t *testing.T) {
	handler := protectedRouter(t, "pi.local:8080")
	headers := map[string]string{"Origin": "http://evil.example", "Sec-Fetch-Site": "cross-site"}
	for _, path := range []string{"/api/cards", "/api/cards/protected", "/api/cards/protected/runs", "/api/health"} {
		if response := send(handler, http.MethodGet, path, "", headers); response.Code != http.StatusOK {
			t.Fatalf("GET %s with foreign origin: status = %d", path, response.Code)
		}
	}
	// The SSE endpoint is read-only as well; use a cancelled request so the
	// handler returns immediately.
	request := httptest.NewRequest(http.MethodGet, "/api/events", nil)
	request.Header.Set("Origin", "http://evil.example")
	request.Header.Set("Sec-Fetch-Site", "cross-site")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request.WithContext(ctx))
	if response.Code != http.StatusOK {
		t.Fatalf("GET /api/events with foreign origin: status = %d", response.Code)
	}
	// The SPA fallback is not an API route.
	if response := send(handler, http.MethodPost, "/some/page", "", headers); response.Code == http.StatusForbidden {
		t.Fatal("SPA fallback must not be protected as an API route")
	}
}

func TestAllowedHostsRestrictMutatingRequests(t *testing.T) {
	handler := protectedRouter(t, "pi.local:8080", " Marionette.Home ", "proxied.local:80")
	path := "/api/cards/protected/actions/primary"
	for _, host := range []string{"192.168.1.5:8080", "pi.local", "pi.local:80", "proxied.local:8080"} {
		request := httptest.NewRequest(http.MethodPost, path, nil)
		request.Host = host
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		assertJSONError(t, response, http.StatusForbidden)
	}

	// Default ports (80, 443, none) are interchangeable on both sides so a
	// TLS-terminating proxy and an allowlist entry with ":80" both work.
	for _, host := range []string{
		"pi.local:8080", "PI.LOCAL:8080", "marionette.home", "marionette.home:80", "marionette.home:443",
		"proxied.local", "proxied.local:443",
	} {
		request := httptest.NewRequest(http.MethodPost, path, nil)
		request.Host = host
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusAccepted {
			t.Fatalf("Host %q: status = %d", host, response.Code)
		}
	}
	// Read-only routes are not restricted by the allowlist.
	request := httptest.NewRequest(http.MethodGet, "/api/cards", nil)
	request.Host = "192.168.1.5:8080"
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("GET with unlisted host: status = %d", response.Code)
	}
}

func TestSameOriginComparison(t *testing.T) {
	cases := []struct {
		origin, host string
		want         bool
	}{
		{"http://pi.local:8080", "pi.local:8080", true},
		{"HTTP://PI.LOCAL:8080", "pi.local:8080", true},
		{"http://pi.local", "pi.local:80", true},
		{"http://pi.local", "pi.local", true},
		{"https://pi.local", "pi.local:443", true},
		// TLS-terminating proxy: the browser sees https, the service plain http.
		{"https://pi.local", "pi.local", true},
		{"https://pi.local", "pi.local:80", true},
		{"http://pi.local", "pi.local:443", true},
		{"https://pi.local:8443", "pi.local", false},
		{"http://pi.local", "pi.local:8080", false},
		{"http://pi.local:8080", "pi.local:9090", false},
		{"http://pi.local:8080", "pi.local", false},
		{"http://[::1]:8080", "[::1]:8080", true},
		{"http://[::1]", "[::1]:80", true},
		{"http://[::1]", "[::1]", true},
		{"http://evil.example:8080", "pi.local:8080", false},
		{"null", "pi.local:8080", false},
		{"", "pi.local:8080", false},
		{"ftp://pi.local:8080", "pi.local:8080", false},
		{"http://pi.local:8080/path", "pi.local:8080", true},
	}
	for _, testCase := range cases {
		if got := sameOrigin(testCase.origin, testCase.host); got != testCase.want {
			t.Errorf("sameOrigin(%q, %q) = %v, want %v", testCase.origin, testCase.host, got, testCase.want)
		}
	}
}

func TestCanonicalHost(t *testing.T) {
	cases := map[string]string{
		"pi.local":        "pi.local",
		" PI.local:80 ":   "pi.local",
		"pi.local:443":    "pi.local",
		"pi.local:8080":   "pi.local:8080",
		"[::1]":           "::1",
		"[::1]:80":        "::1",
		"[::1]:8080":      "[::1]:8080",
		"192.168.1.5:443": "192.168.1.5",
		"":                "",
	}
	for host, want := range cases {
		if got := canonicalHost(host); got != want {
			t.Errorf("canonicalHost(%q) = %q, want %q", host, got, want)
		}
	}
}
