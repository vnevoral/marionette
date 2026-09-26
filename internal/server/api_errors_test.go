package server

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"marionette/internal/config"
	"marionette/internal/webui"
)

func decodeErrorEnvelope(t *testing.T, response *httptest.ResponseRecorder) map[string]json.RawMessage {
	t.Helper()
	if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json (body %s)", contentType, response.Body.String())
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil || envelope["error"] == nil {
		t.Fatalf("error envelope = %q, %v", response.Body.String(), err)
	}
	return envelope
}

func TestRouterAnswersMethodNotAllowedWithAllowAndJSON(t *testing.T) {
	store := config.NewStore(config.Settings{HistorySize: 5, MaxConcurrentActions: 1})
	createServerCard(t, store, "m")
	queue := &fakeActionQueue{primaryCalls: make(chan string, 1), statusCalls: make(chan string, 1)}
	handler := NewRouter(Dependencies{Store: store, Actions: queue})
	cases := []struct {
		method, path, allow string
	}{
		{http.MethodDelete, "/api/health", "GET, HEAD"},
		{http.MethodPatch, "/api/cards", "GET, HEAD, POST"},
		{http.MethodPost, "/api/cards/m", "DELETE, GET, HEAD, PUT"},
		{http.MethodPut, "/api/cards/m/runs", "GET, HEAD"},
		{http.MethodGet, "/api/cards/m/actions/primary", "POST"},
		{http.MethodDelete, "/api/cards/m/actions/status/check", "POST"},
		{http.MethodPost, "/api/events", "GET, HEAD"},
	}
	for _, testCase := range cases {
		request := httptest.NewRequest(testCase.method, testCase.path, nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusMethodNotAllowed {
			t.Fatalf("%s %s: status = %d, body %s", testCase.method, testCase.path, response.Code, response.Body.String())
		}
		if allow := response.Header().Get("Allow"); allow != testCase.allow {
			t.Fatalf("%s %s: Allow = %q, want %q", testCase.method, testCase.path, allow, testCase.allow)
		}
		decodeErrorEnvelope(t, response)
	}
	head := httptest.NewRecorder()
	handler.ServeHTTP(head, httptest.NewRequest(http.MethodHead, "/api/health", nil))
	if head.Code != http.StatusOK {
		t.Fatalf("HEAD /api/health status = %d", head.Code)
	}
}

func TestRouterRejectsOversizedBodyWithJSON413(t *testing.T) {
	store := config.NewStore(config.Settings{HistorySize: 5, MaxConcurrentActions: 1})
	handler := NewRouter(Dependencies{Store: store})
	body := bytes.NewReader(append([]byte(`{"name":"`), append(bytes.Repeat([]byte("x"), maxJSONBodyBytes+1), []byte(`"}`)...)...))
	request := httptest.NewRequest(http.MethodPost, "/api/cards", body)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413 (body %s)", response.Code, response.Body.String())
	}
	decodeErrorEnvelope(t, response)
}

func TestRouterDecodeErrorsHideGoTypes(t *testing.T) {
	store := config.NewStore(config.Settings{HistorySize: 5, MaxConcurrentActions: 1})
	handler := NewRouter(Dependencies{Store: store})
	cases := []struct {
		name, body, want string
	}{
		{"wrong type", `{"id":"x","name":"x","primary":{"command":"x","timeoutSec":"soon"}}`, `unexpected string value for field "primary.timeoutSec"`},
		{"syntax", `{"id": nope}`, "malformed JSON at offset"},
		{"unknown field", `{"id":"x","surprise":true}`, `unknown field "surprise"`},
		{"empty", ``, "request body is empty"},
		{"truncated", `{"id":"x"`, "unexpected end of JSON body"},
		{"multiple values", `{"id":"x","name":"x","primary":{"command":"x","timeoutSec":1}} {}`, "multiple JSON values"},
		{"array instead of object", `[]`, `unexpected array value for field "body"`},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/cards", strings.NewReader(testCase.body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, body %s", response.Code, response.Body.String())
			}
			envelope := decodeErrorEnvelope(t, response)
			var message string
			_ = json.Unmarshal(envelope["error"], &message)
			if !strings.HasPrefix(message, "invalid JSON body: ") || !strings.Contains(message, testCase.want) {
				t.Fatalf("error = %q, want prefix and %q", message, testCase.want)
			}
			if strings.Contains(message, "Go struct") || strings.Contains(message, "config.") {
				t.Fatalf("error leaks Go types: %q", message)
			}
		})
	}
}

func TestRouterReturnsValidationFields(t *testing.T) {
	store := config.NewStore(config.Settings{HistorySize: 5, MaxConcurrentActions: 1})
	handler := NewRouter(Dependencies{Store: store})
	card := validServerCard("bad id")
	card.Primary.TimeoutSec = config.MaxTimeoutSec + 1
	response := requestJSON(t, handler, http.MethodPost, "/api/cards", card)
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body %s", response.Code, response.Body.String())
	}
	envelope := decodeErrorEnvelope(t, response)
	var fields map[string]string
	if err := json.Unmarshal(envelope["fields"], &fields); err != nil {
		t.Fatalf("fields = %s, %v", envelope["fields"], err)
	}
	if fields["id"] == "" || fields["primary.timeoutSec"] != "must be between 1 and 3600" || len(fields) != 2 {
		t.Fatalf("fields = %v", fields)
	}
}

func TestHealthReportsVersionAndUptime(t *testing.T) {
	handler := NewRouter(Dependencies{Version: "v1.2.3", StartedAt: time.Now().Add(-90 * time.Second)})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/health", nil))
	var health healthResponse
	if err := json.Unmarshal(response.Body.Bytes(), &health); err != nil {
		t.Fatalf("health body = %s, %v", response.Body.String(), err)
	}
	if health.Status != "ok" || health.Version != "v1.2.3" || health.UptimeSec < 90 || health.UptimeSec > 100 {
		t.Fatalf("health = %#v", health)
	}
	defaults := httptest.NewRecorder()
	NewRouter(Dependencies{}).ServeHTTP(defaults, httptest.NewRequest(http.MethodGet, "/api/health", nil))
	if err := json.Unmarshal(defaults.Body.Bytes(), &health); err != nil || health.Version != "dev" || health.UptimeSec < 0 {
		t.Fatalf("default health = %#v, %v", health, err)
	}
}

func TestSPACacheHeadersAndDirectories(t *testing.T) {
	handler := NewRouter(Dependencies{})
	entries, err := fs.ReadDir(webui.Dist(), "assets")
	if err != nil || len(entries) == 0 {
		t.Skipf("embedded assets unavailable: %v", err)
	}
	get := func(path string) *httptest.ResponseRecorder {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		return response
	}
	asset := get("/assets/" + entries[0].Name())
	if asset.Code != http.StatusOK || asset.Header().Get("Cache-Control") != assetsCacheControl {
		t.Fatalf("asset = %d %q", asset.Code, asset.Header().Get("Cache-Control"))
	}
	for _, path := range []string{"/", "/assets", "/assets/", "/cards/x"} {
		response := get(path)
		if response.Code != http.StatusOK || response.Header().Get("Cache-Control") != documentCacheControl ||
			!strings.Contains(response.Header().Get("Content-Type"), "text/html") || !strings.Contains(response.Body.String(), "<html") {
			t.Fatalf("%s = %d %q %q", path, response.Code, response.Header().Get("Cache-Control"), response.Header().Get("Content-Type"))
		}
		if strings.Contains(response.Body.String(), "<pre>") {
			t.Fatalf("%s returned a directory listing", path)
		}
	}
}
