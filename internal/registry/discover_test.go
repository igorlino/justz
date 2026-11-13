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

func TestDiscoverWalksUp(t *testing.T) {
	root := t.TempDir()
	touch(t, filepath.Join(root, "config", "commands.json"))
	deep := filepath.Join(root, "src", "pkg", "deep")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	if got, err := Discover(deep); err != nil || got != filepath.Join(root, "config", "commands.json") {
		t.Errorf("from a subdirectory: got %q, %v", got, err)
	}
}

func TestDiscoverStopsAtTheProjectRoot(t *testing.T) {
	outer := t.TempDir()
	touch(t, filepath.Join(outer, "config", "commands.json")) // an unrelated project above
	project := filepath.Join(outer, "project")
	if err := os.MkdirAll(filepath.Join(project, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(project, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if got, err := Discover(sub); err == nil {
		t.Errorf("should not look past the project root (.git), but found %q", got)
	}
}

func TestDiscoverFindsRegistryNextToGit(t *testing.T) {
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	touch(t, filepath.Join(project, "config", "commands.json"))
	if got, err := Discover(project); err != nil || got != filepath.Join(project, "config", "commands.json") {
		t.Errorf("registry in the project root: got %q, %v", got, err)
	}
}

func TestDiscoverNothing(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil { // bound the search to this directory
		t.Fatal(err)
	}
	if _, err := Discover(dir); err == nil {
		t.Error("an empty directory should not be a success")
	}
}
