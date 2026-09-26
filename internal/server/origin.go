package server

import (
	"errors"
	"mime"
	"net"
	"net/http"
	"net/url"
	"strings"
)

// Errors reported by the cross-site protection middleware (NFR-12).
var (
	ErrCrossSiteRequest   = errors.New("cross-site request rejected")
	ErrUnsupportedContent = errors.New("request body must be application/json")
	ErrHostNotAllowed     = errors.New("request host is not allowed")
)

// requireSameOrigin protects the mutating API endpoints (POST, PUT, PATCH,
// DELETE under /api/) from requests issued by other web origins opened in the
// operator's browser (NFR-12). Read-only requests, the SSE stream and the SPA
// fallback pass through untouched. The rules, in order:
//
//  1. a request carrying a body or a Content-Type header must declare
//     application/json (415 otherwise) — this blocks HTML form submissions,
//     which cannot produce that media type;
//  2. Sec-Fetch-Site: cross-site is rejected (403);
//  3. when an Origin header is present its host must match the request host
//     (403 otherwise); a request without Origin and without Sec-Fetch-Site
//     is accepted so that non-browser clients such as curl keep working;
//  4. when allowedHosts is non-empty the request Host must be listed (403).
//
// Hosts are compared case-insensitively including the port; a missing port
// counts as the scheme default so that "http://pi.local" matches
// "pi.local:80".
func requireSameOrigin(allowedHosts []string, next http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(allowedHosts))
	for _, host := range allowedHosts {
		if host = strings.ToLower(strings.TrimSpace(host)); host != "" {
			allowed[host] = struct{}{}
		}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if !isMutatingAPIRequest(request) {
			next.ServeHTTP(w, request)
			return
		}
		if err := checkContentType(request); err != nil {
			writeError(w, http.StatusUnsupportedMediaType, err)
			return
		}
		if err := checkOrigin(request); err != nil {
			writeError(w, http.StatusForbidden, err)
			return
		}
		if len(allowed) > 0 {
			if _, ok := allowed[strings.ToLower(request.Host)]; !ok {
				writeError(w, http.StatusForbidden, ErrHostNotAllowed)
				return
			}
		}
		next.ServeHTTP(w, request)
	})
}

func isMutatingAPIRequest(request *http.Request) bool {
	switch request.Method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
	default:
		return false
	}
	path := request.URL.Path
	return path == "/api" || strings.HasPrefix(path, "/api/")
}

func checkContentType(request *http.Request) error {
	contentType := request.Header.Get("Content-Type")
	hasBody := request.ContentLength != 0 || len(request.TransferEncoding) > 0
	if contentType == "" && !hasBody {
		return nil
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil || mediaType != "application/json" {
		return ErrUnsupportedContent
	}
	return nil
}

func checkOrigin(request *http.Request) error {
	if strings.EqualFold(request.Header.Get("Sec-Fetch-Site"), "cross-site") {
		return ErrCrossSiteRequest
	}
	origin, present := request.Header["Origin"]
	if !present {
		return nil
	}
	if len(origin) != 1 || !sameOrigin(origin[0], request.Host, request.TLS != nil) {
		return ErrCrossSiteRequest
	}
	return nil
}

// sameOrigin reports whether the Origin header value points at the host that
// received the request. The request scheme is only known from the TLS state,
// so a request behind a TLS-terminating proxy compares as plain HTTP; both
// hosts are normalised with the default port of their scheme.
func sameOrigin(origin, requestHost string, tls bool) bool {
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return false
	}
	requestScheme := "http"
	if tls {
		requestScheme = "https"
	}
	return normaliseHost(parsed.Host, parsed.Scheme) == normaliseHost(requestHost, requestScheme)
}

func normaliseHost(host, scheme string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	if _, _, err := net.SplitHostPort(host); err == nil {
		return host
	}
	port := "80"
	if scheme == "https" {
		port = "443"
	}
	return net.JoinHostPort(strings.Trim(host, "[]"), port)
}
