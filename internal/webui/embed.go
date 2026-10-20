// Package webui embeds the compiled Vue SPA into the Go binary. Vite builds
// it into internal/webui/dist (make ui-build); the build output is not
// versioned, so a binary built without it serves placeholder.html instead.
package webui

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

//go:embed placeholder.html
var placeholder []byte

// Dist returns the embedded SPA build output rooted at "dist". Without a UI
// build, index.html is the placeholder page.
func Dist() fs.FS {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic(err)
	}
	if _, err := fs.Stat(sub, "index.html"); err == nil {
		return sub
	}
	return withPlaceholder{sub}
}

// withPlaceholder answers fs.ReadFile("index.html") with placeholder.html,
// which is how the SPA handler reads the page.
type withPlaceholder struct{ fs.FS }

func (p withPlaceholder) ReadFile(name string) ([]byte, error) {
	if name == "index.html" {
		return placeholder, nil
	}
	return fs.ReadFile(p.FS, name)
}
