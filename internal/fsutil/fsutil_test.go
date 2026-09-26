package fsutil

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWriteFileAtomicCreatesWithModeAndKeepsExistingMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := WriteFileAtomic(path, []byte("one"), 0o600); err != nil {
		t.Fatalf("WriteFileAtomic() error = %v", err)
	}
	if info, _ := os.Stat(path); info.Mode().Perm() != 0o600 {
		t.Fatalf("new file mode = %o", info.Mode().Perm())
	}
	if err := os.Chmod(path, 0o640); err != nil {
		t.Fatal(err)
	}
	if err := WriteFileAtomic(path, []byte("two"), 0o600); err != nil {
		t.Fatalf("second WriteFileAtomic() error = %v", err)
	}
	data, _ := os.ReadFile(path)
	info, _ := os.Stat(path)
	if string(data) != "two" || info.Mode().Perm() != 0o640 {
		t.Fatalf("after replace: %q mode %o", data, info.Mode().Perm())
	}
	if leftovers, _ := filepath.Glob(path + ".*.tmp"); len(leftovers) != 0 {
		t.Fatalf("temporary files left: %v", leftovers)
	}
	if err := WriteFileAtomic(filepath.Join(t.TempDir(), "missing", "x.json"), nil, 0o600); err == nil {
		t.Fatal("write into a missing directory succeeded")
	}
}

func TestQuarantineKeepsContentAndAvoidsCollisions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	now := time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)
	var targets []string
	for range 2 {
		if err := os.WriteFile(path, []byte("broken"), 0o600); err != nil {
			t.Fatal(err)
		}
		target, err := Quarantine(path, now)
		if err != nil {
			t.Fatalf("Quarantine() error = %v", err)
		}
		targets = append(targets, target)
	}
	if targets[0] != path+".corrupt-20260926T120000Z" || targets[1] != targets[0]+"-1" {
		t.Fatalf("targets = %v", targets)
	}
	if data, _ := os.ReadFile(targets[1]); string(data) != "broken" {
		t.Fatal("quarantined content changed")
	}
}
