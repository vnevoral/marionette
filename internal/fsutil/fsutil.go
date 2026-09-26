// Package fsutil holds the file helpers shared by the stores that persist
// JSON next to each other on the host (configuration, paired devices).
package fsutil

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ErrDirectorySync is returned by WriteFileAtomic when the file was already
// replaced but its directory entry could not be fsynced. The new content is
// on disk; only its durability across a crash is uncertain. Callers should
// log it and must not undo the change in memory.
var ErrDirectorySync = errors.New("directory could not be synced")

// WriteFileAtomic replaces path atomically: data is written to a unique
// "<name>.<random>.tmp" file next to the target, fsynced, renamed over the
// target and the directory entry is fsynced. The unique name keeps two
// writers from truncating each other's temporary file; the last rename wins
// with a complete file. An existing file keeps its permission bits; a new
// file gets newMode. A failure after the rename wraps ErrDirectorySync.
func WriteFileAtomic(path string, data []byte, newMode os.FileMode) error {
	mode := newMode
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary file for %q: %w", path, err)
	}
	temporaryName := temporary.Name()
	cleanup := func() {
		_ = temporary.Close()
		_ = os.Remove(temporaryName)
	}

	if _, err := temporary.Write(data); err != nil {
		cleanup()
		return fmt.Errorf("write temporary file %q: %w", temporaryName, err)
	}
	if err := temporary.Sync(); err != nil {
		cleanup()
		return fmt.Errorf("sync temporary file %q: %w", temporaryName, err)
	}
	if err := temporary.Close(); err != nil {
		_ = os.Remove(temporaryName)
		return fmt.Errorf("close temporary file %q: %w", temporaryName, err)
	}
	// CreateTemp always uses 0600; apply the intended permission bits.
	if err := os.Chmod(temporaryName, mode); err != nil {
		_ = os.Remove(temporaryName)
		return fmt.Errorf("set permissions of temporary file %q: %w", temporaryName, err)
	}
	if err := os.Rename(temporaryName, path); err != nil {
		_ = os.Remove(temporaryName)
		return fmt.Errorf("replace file %q: %w", path, err)
	}
	if err := syncDirectory(filepath.Dir(path)); err != nil {
		return fmt.Errorf("sync directory for %q: %w: %w", path, ErrDirectorySync, err)
	}
	return nil
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	if err := directory.Sync(); err != nil {
		_ = directory.Close()
		return err
	}
	return directory.Close()
}

// Quarantine renames a file that could not be loaded to
// "<path>.corrupt-<UTC timestamp>" so that a later save never overwrites it.
// A numeric suffix is appended when that name is already taken. The new path
// is returned.
func Quarantine(path string, now time.Time) (string, error) {
	base := path + ".corrupt-" + now.UTC().Format("20060102T150405Z")
	target := base
	for suffix := 1; ; suffix++ {
		_, err := os.Lstat(target)
		if errors.Is(err, os.ErrNotExist) {
			break
		}
		if err != nil {
			return "", fmt.Errorf("inspect quarantine target %q: %w", target, err)
		}
		target = fmt.Sprintf("%s-%d", base, suffix)
	}
	if err := os.Rename(path, target); err != nil {
		return "", fmt.Errorf("quarantine file %q: %w", path, err)
	}
	return target, nil
}
