package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const unsortedRegistry = `{"groups":{"zeta":{"commands":{"b":{"desc":"b"},"a":{"desc":"a"}}},"core":{"commands":{}}}}`

const sortedRegistry = "{\n  \"groups\": {\n    \"core\": {\n      \"commands\": {}\n    },\n    \"zeta\": {\n      \"commands\": {\n        \"a\": {\n          \"desc\": \"a\"\n        },\n        \"b\": {\n          \"desc\": \"b\"\n        }\n      }\n    }\n  }\n}\n"

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestFmtRewritesInPlace(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "commands.json")
	writeFile(t, p, unsortedRegistry)

	e, out, _ := testEnv(true)
	if code := run([]string{"fmt", p}, e); code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if got := readFile(t, p); got != sortedRegistry {
		t.Errorf("file not canonical:\n%s", got)
	}
	if !strings.Contains(out.String(), "formatted "+p) {
		t.Errorf("stdout = %q", out.String())
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("temporary files were left behind: %v", entries)
	}

	// A second run finds nothing to do.
	e, out, _ = testEnv(true)
	if code := run([]string{"fmt", p}, e); code != 0 || !strings.Contains(out.String(), "already formatted") {
		t.Errorf("second run: code=%d out=%q", code, out.String())
	}
}

func TestFmtCheck(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "commands.json")
	writeFile(t, p, unsortedRegistry)

	e, out, _ := testEnv(true)
	if code := run([]string{"fmt", "--check", p}, e); code != 1 {
		t.Errorf("unformatted file: exit code = %d, want 1", code)
	}
	if !strings.Contains(out.String(), "not formatted") {
		t.Errorf("stdout = %q", out.String())
	}
	if got := readFile(t, p); got != unsortedRegistry {
		t.Error("--check must not modify the file")
	}

	writeFile(t, p, sortedRegistry)
	e, _, _ = testEnv(true)
	if code := run([]string{"fmt", "--check", p}, e); code != 0 {
		t.Errorf("formatted file: exit code = %d, want 0", code)
	}
}

func TestFmtKeepsFileMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file modes are not meaningful on Windows")
	}
	p := filepath.Join(t.TempDir(), "commands.json")
	writeFile(t, p, unsortedRegistry)
	if err := os.Chmod(p, 0o640); err != nil {
		t.Fatal(err)
	}
	e, _, _ := testEnv(true)
	if code := run([]string{"fmt", p}, e); code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	info, _ := os.Stat(p)
	if info.Mode().Perm() != 0o640 {
		t.Errorf("mode = %v, want 0640", info.Mode().Perm())
	}
}

func TestFmtRefusesSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need extra privileges on Windows")
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "real.json")
	link := filepath.Join(dir, "commands.json")
	writeFile(t, target, unsortedRegistry)
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	e, _, errOut := testEnv(true)
	if code := run([]string{"fmt", link}, e); code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(errOut.String(), "symbolic link") {
		t.Errorf("stderr = %q", errOut.String())
	}
	if got := readFile(t, target); got != unsortedRegistry {
		t.Error("the symlink target was modified")
	}
}

func TestFmtErrors(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.json")
	writeFile(t, bad, `{"a":`)
	tests := []struct {
		name     string
		args     []string
		wantCode int
		wantErr  string
	}{
		{"invalid JSON", []string{"fmt", bad}, 1, "not valid JSON"},
		{"missing file", []string{"fmt", filepath.Join(dir, "nope.json")}, 1, "no such file"},
		{"unknown flag", []string{"fmt", "--wat"}, 2, `unknown flag "--wat"`},
		{"two paths", []string{"fmt", "a.json", "b.json"}, 2, "only one path"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, _, errOut := testEnv(true)
			code := run(tt.args, e)
			if code != tt.wantCode {
				t.Errorf("exit code = %d, want %d", code, tt.wantCode)
			}
			msg := strings.ToLower(errOut.String())
			if !strings.Contains(msg, strings.ToLower(tt.wantErr)) && !(tt.name == "missing file" && strings.Contains(msg, "cannot find")) {
				t.Errorf("stderr = %q, want it to contain %q", errOut.String(), tt.wantErr)
			}
		})
	}
	if got := readFile(t, bad); got != `{"a":` {
		t.Error("an invalid file must not be modified")
	}
}

func TestFmtDiscoversRegistry(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "config", "commands.json"), unsortedRegistry)
	t.Chdir(dir)

	e, _, _ := testEnv(true)
	if code := run([]string{"fmt"}, e); code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if got := readFile(t, filepath.Join(dir, "config", "commands.json")); got != sortedRegistry {
		t.Errorf("discovered registry was not formatted:\n%s", got)
	}
}

func TestFmtUsesJustzConfig(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "elsewhere.json")
	writeFile(t, p, unsortedRegistry)
	t.Setenv("JUSTZ_CONFIG", p)

	e, _, _ := testEnv(true)
	if code := run([]string{"fmt"}, e); code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if got := readFile(t, p); got != sortedRegistry {
		t.Errorf("JUSTZ_CONFIG file was not formatted:\n%s", got)
	}
}

func TestFmtWithoutRegistryFails(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("JUSTZ_CONFIG", "")
	e, _, errOut := testEnv(true)
	if code := run([]string{"fmt"}, e); code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(errOut.String(), "no commands.json found") {
		t.Errorf("stderr = %q", errOut.String())
	}
}
