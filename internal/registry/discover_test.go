package registry

import (
	"os"
	"path/filepath"
	"testing"
)

func touch(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDiscoverOrder(t *testing.T) {
	dir := t.TempDir()
	touch(t, filepath.Join(dir, "just", "commands.json"))
	if got, err := Discover(dir); err != nil || got != filepath.Join(dir, "just", "commands.json") {
		t.Errorf("just/ only: got %q, %v", got, err)
	}

	touch(t, filepath.Join(dir, "config", "commands.json"))
	if got, _ := Discover(dir); got != filepath.Join(dir, "config", "commands.json") {
		t.Errorf("config/ should beat just/: got %q", got)
	}

	touch(t, filepath.Join(dir, ".zeta", "config", "commands.json"))
	touch(t, filepath.Join(dir, ".alpha", "config", "commands.json"))
	if got, _ := Discover(dir); got != filepath.Join(dir, ".alpha", "config", "commands.json") {
		t.Errorf("a hidden directory should beat config/, sorted by name: got %q", got)
	}
}

func TestDiscoverNothing(t *testing.T) {
	if _, err := Discover(t.TempDir()); err == nil {
		t.Error("an empty directory should not be a success")
	}
}
