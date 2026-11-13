package atomicfile

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWriteCreatesAndReplaces(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f.txt")

	if err := Write(p, []byte("one"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Write(p, []byte("two"), 0o644); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(p); string(b) != "two" {
		t.Errorf("content = %q, want two", b)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("temporary files were left behind: %v", entries)
	}
}

func TestWriteKeepsExistingMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file modes are not meaningful on Windows")
	}
	p := filepath.Join(t.TempDir(), "f.txt")
	if err := os.WriteFile(p, []byte("x"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := Write(p, []byte("y"), 0o755); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(p)
	if info.Mode().Perm() != 0o640 {
		t.Errorf("mode = %v, want 0640 (the existing file's mode)", info.Mode().Perm())
	}
}

func TestWriteUsesGivenModeForNewFiles(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file modes are not meaningful on Windows")
	}
	p := filepath.Join(t.TempDir(), "new.sh")
	if err := Write(p, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(p)
	if info.Mode().Perm() != 0o755 {
		t.Errorf("mode = %v, want 0755", info.Mode().Perm())
	}
}

func TestWriteRefusesSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need extra privileges on Windows")
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "target.txt")
	link := filepath.Join(dir, "link.txt")
	if err := os.WriteFile(target, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	err := Write(link, []byte("overwrite"), 0o644)
	if err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("want a symbolic link error, got %v", err)
	}
	if b, _ := os.ReadFile(target); string(b) != "keep" {
		t.Error("the symlink target was modified")
	}
}

func TestWriteFailsWhenDirectoryIsMissing(t *testing.T) {
	if err := Write(filepath.Join(t.TempDir(), "nope", "f.txt"), []byte("x"), 0o644); err == nil {
		t.Error("writing into a missing directory should fail")
	}
}
