package webui

import (
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
)

func TestIndexFallsBackToThePlaceholderWithoutAUIBuild(t *testing.T) {
	data, err := fs.ReadFile(withPlaceholder{fstest.MapFS{}}, "index.html")
	if err != nil || !strings.Contains(string(data), "make ui-build") {
		t.Fatalf("index.html = %q, %v", data, err)
	}
}

func TestDistServesAnIndexPage(t *testing.T) {
	data, err := fs.ReadFile(Dist(), "index.html")
	if err != nil || !strings.Contains(string(data), "<html") {
		t.Fatalf("index.html = %q, %v", data, err)
	}
}
