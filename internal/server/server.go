// Package server wires HTTP routes for the API and the embedded SPA.
package server

import (
	"encoding/json"
	"io/fs"
	"net/http"

	"marionette/internal/webui"
)

// NewRouter builds the top-level HTTP handler for the application.
func NewRouter() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", handleHealth)

	mux.Handle("/", spaHandler(webui.Dist()))

	return mux
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// spaHandler serves static assets from the embedded filesystem and falls back
// to index.html for unknown paths so client-side (Vue Router) routes resolve.
func spaHandler(root fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(root))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path != "/" {
			if _, err := fs.Stat(root, path[1:]); err == nil {
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		data, err := fs.ReadFile(root, "index.html")
		if err != nil {
			http.Error(w, "index.html not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(data)
	})
}
