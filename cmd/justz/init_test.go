package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitWritesNativeProject(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	e, out, errOut := testEnv(false)
	if code := run([]string{"init", "--module", "build", "--module=ops", dir}, e); code != 0 {
		t.Fatalf("code %d: %s", code, errOut)
	}
	for _, p := range []string{"justfile", "just/preamble.just", "just/build.just", "just/ops.just"} {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(p))); err != nil {
			t.Errorf("missing %s", p)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "config")); err == nil {
		t.Error("native must not write config/")
	}
	if !strings.Contains(readFile(t, filepath.Join(dir, "justfile")), "$'\\n  demo\\n\\n'") {
		t.Error("name should default to the directory name")
	}
	if !strings.Contains(out.String(), "wrote just/build.just") {
		t.Errorf("stdout: %s", out)
	}
}

func TestInitRichWritesRegistry(t *testing.T) {
	dir := t.TempDir()
	e, _, errOut := testEnv(false)
	if code := run([]string{"init", "--variant", "rich", "--name", "My App", dir}, e); code != 0 {
		t.Fatalf("code %d: %s", code, errOut)
	}
	if !strings.Contains(readFile(t, filepath.Join(dir, "config", "commands.json")), `"name": "My App"`) {
		t.Error("registry misses the name")
	}
}

func TestInitRefusesToOverwriteAndWritesNothing(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "just", "build.just"), "mine\n")
	e, _, errOut := testEnv(false)
	if code := run([]string{"init", "--module", "build", dir}, e); code != 1 {
		t.Fatalf("code %d, want 1", code)
	}
	if !strings.Contains(errOut.String(), "just/build.just") {
		t.Errorf("stderr: %s", errOut)
	}
	if _, err := os.Stat(filepath.Join(dir, "justfile")); err == nil {
		t.Error("a refusal must write nothing")
	}
	if readFile(t, filepath.Join(dir, "just", "build.just")) != "mine\n" {
		t.Error("existing file was changed")
	}
}

func TestInitRefusesExistingJustfileInAnyCase(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".justfile"), "x:\n")
	e, _, errOut := testEnv(false)
	if code := run([]string{"init", dir}, e); code != 1 || !strings.Contains(errOut.String(), ".justfile") {
		t.Fatalf("code %d: %s", code, errOut)
	}
}

func TestInitRefusesJustfileInOtherCase(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "JUSTFILE"), "x:\n")
	e, _, errOut := testEnv(false)
	if code := run([]string{"init", dir}, e); code != 1 || !strings.Contains(errOut.String(), "JUSTFILE") {
		t.Fatalf("code %d: %s", code, errOut)
	}
}

func TestInitRefusesFileWhereDirectoryIsNeeded(t *testing.T) {
	for _, tc := range []struct{ blocker, variant string }{{"just", "native"}, {"config", "rich"}} {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, tc.blocker), "a file\n")
		e, _, errOut := testEnv(false)
		if code := run([]string{"init", "--variant", tc.variant, dir}, e); code != 1 || !strings.Contains(errOut.String(), "not a directory") {
			t.Fatalf("%s: code %d: %s", tc.blocker, code, errOut)
		}
		if _, err := os.Stat(filepath.Join(dir, "justfile")); err == nil {
			t.Errorf("%s: a refusal must write nothing", tc.blocker)
		}
	}
}

func TestInitRefusesSymlinkedDirectory(t *testing.T) {
	root := t.TempDir()
	dir, outside := filepath.Join(root, "p"), filepath.Join(root, "outside")
	for _, d := range []string{dir, outside} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(outside, filepath.Join(dir, "just")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	e, _, errOut := testEnv(false)
	if code := run([]string{"init", dir}, e); code != 1 || !strings.Contains(errOut.String(), "symbolic link") {
		t.Fatalf("code %d: %s", code, errOut)
	}
	if entries, _ := os.ReadDir(outside); len(entries) != 0 {
		t.Error("init wrote through the symbolic link")
	}
}

func TestInitHintsAtNameWhenDirectoryNameIsUnusable(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "my+proj")
	e, _, errOut := testEnv(false)
	if code := run([]string{"init", dir}, e); code != 2 || !strings.Contains(errOut.String(), "--name") {
		t.Fatalf("code %d: %s", code, errOut)
	}
}

func TestInitDryRunWritesNothing(t *testing.T) {
	dir := t.TempDir()
	e, out, _ := testEnv(false)
	if code := run([]string{"init", "--dry-run", dir}, e); code != 0 {
		t.Fatalf("code %d", code)
	}
	if !strings.Contains(out.String(), "would write justfile") {
		t.Errorf("stdout: %s", out)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("dry run wrote %d entries", len(entries))
	}
}

func TestInitUsageErrors(t *testing.T) {
	for _, args := range [][]string{
		{"init", "--nope"},
		{"init", "--variant"},
		{"init", "--module"},
		{"init", "a", "b"},
		{"init", "--variant", "fancy", "x"},
		{"init", "--name", "", "x"},
		{"init", "--name", "it's", "x"},
		{"init", "--module", "Bad", "x"},
	} {
		e, _, _ := testEnv(false)
		if code := run(args, e); code != 2 {
			t.Errorf("%v: code %d, want 2", args, code)
		}
	}
}
