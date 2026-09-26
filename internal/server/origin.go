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
// Hosts (Origin, request Host and the allowlist) are compared through
// canonicalHost: case-insensitively, and the default ports 80 and 443 count
// the same as no port, so "https://pi.local" behind a TLS-terminating proxy
// still matches a request Host of "pi.local" or "pi.local:80". Any other
// explicit port must match exactly.
func requireSameOrigin(allowedHosts []string, next http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(allowedHosts))
	for _, host := range allowedHosts {
		if host = canonicalHost(host); host != "" {
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
			if _, ok := allowed[canonicalHost(request.Host)]; !ok {
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
	if len(origin) != 1 || !sameOrigin(origin[0], request.Host) {
		return ErrCrossSiteRequest
	}
	return nil
}

// sameOrigin reports whether the Origin header value points at the host that
// received the request. The request scheme is not known reliably (a
// TLS-terminating proxy forwards plain HTTP), so the scheme only has to be
// http or https and the hosts are compared with canonicalHost, which treats
// the default ports of both schemes alike.
func sameOrigin(origin, requestHost string) bool {
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return false
	}
	return canonicalHost(parsed.Host) == canonicalHost(requestHost)
}

// canonicalHost lower-cases a host and drops a default port (80 or 443), so
// "PI.LOCAL:443", "pi.local:80" and "pi.local" all become "pi.local" while
// "pi.local:8080" keeps its port. IPv6 literals lose their brackets when the
// port is dropped ("[::1]" and "[::1]:80" become "::1").
func canonicalHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	name, port, err := net.SplitHostPort(host)
	if err != nil {
		return strings.Trim(host, "[]")
	}
	if port == "80" || port == "443" {
		return name
	}
	return net.JoinHostPort(name, port)
}
