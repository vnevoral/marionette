package server

import (
	"io/fs"
	"net/http"
	"strings"

	"marionette/internal/config"
)

const (
	// Hashed Vite assets never change under the same name.
	assetsCacheControl = "public, max-age=31536000, immutable"
	// index.html and other unhashed files must be revalidated on every load
	// so a new release is picked up immediately.
	documentCacheControl = "no-cache"
)

// spaHandler serves static assets from the embedded filesystem and falls back
// to index.html for unknown paths so client-side (Vue Router) routes resolve.
// Directories are never listed: they resolve to index.html like any other
// client route. Unknown /api paths answer 404 with the JSON error envelope.
func spaHandler(root fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(root))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path != "/" {
			if strings.HasPrefix(path, "/api/") || path == "/api" {
				writeError(w, http.StatusNotFound, config.ErrNotFound)
				return
			}
			if info, err := fs.Stat(root, strings.TrimPrefix(path, "/")); err == nil && !info.IsDir() {
				if strings.HasPrefix(path, "/assets/") {
					w.Header().Set("Cache-Control", assetsCacheControl)
				} else {
					w.Header().Set("Cache-Control", documentCacheControl)
				}
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		data, err := fs.ReadFile(root, "index.html")
		if err != nil {
			writeError(w, http.StatusNotFound, config.ErrNotFound)
			return
		}
		w.Header().Set("Cache-Control", documentCacheControl)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(data)
	})
}
