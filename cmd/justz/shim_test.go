package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// shimSetup points justz at temporary registry and shim directories, and puts
// the shim directory on the PATH, so tests never touch the real ones.
func shimSetup(t *testing.T) (shimDir string) {
	t.Helper()
	root := t.TempDir()
	cfg := filepath.Join(root, "cfg")
	shimDir = filepath.Join(root, "bin")
	if err := os.MkdirAll(shimDir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("JUSTZ_CONFIG_DIR", cfg)
	t.Setenv("JUSTZ_SHIM_DIR", shimDir)
	t.Setenv("PATH", shimDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return shimDir
}

// project makes a directory with a justfile and, if cli is not empty, a
// registry that names the shim.
func project(t *testing.T, cli string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "justfile"), "default:\n")
	if cli != "" {
		writeFile(t, filepath.Join(dir, "config", "commands.json"), `{"cli":"`+cli+`","groups":{}}`)
	}
	return dir
}

func TestShimAddListRemove(t *testing.T) {
	shimDir := shimSetup(t)
	proj := project(t, "zzdemo")

	e, out, errOut := testEnv(true)
	if code := run([]string{"shim", "add", proj}, e); code != 0 {
		t.Fatalf("add: exit %d, stderr %q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "added shim zzdemo (and zzdemoh)") {
		t.Errorf("add output = %q", out.String())
	}
	for _, n := range []string{"zzdemo", "zzdemoh"} {
		name := n
		if _, err := os.Stat(filepath.Join(shimDir, name)); err != nil {
			// On Windows the stub is a .cmd file.
			if _, err2 := os.Stat(filepath.Join(shimDir, name+".cmd")); err2 != nil {
				t.Errorf("stub %s was not written", name)
			}
		}
	}

	e, out, _ = testEnv(true)
	if code := run([]string{"shim", "list"}, e); code != 0 || !strings.Contains(out.String(), "zzdemo") {
		t.Errorf("list: code %d, out %q", code, out.String())
	}

	e, out, _ = testEnv(true)
	if code := run([]string{"shim", "list", "--json"}, e); code != 0 {
		t.Fatalf("list --json: exit %d", code)
	}
	var reg struct {
		Shims []struct {
			Name, Justfile, Suffix string
			Soft                   bool
		} `json:"shims"`
	}
	if err := json.Unmarshal(out.Bytes(), &reg); err != nil || len(reg.Shims) != 1 || reg.Shims[0].Name != "zzdemo" || reg.Shims[0].Suffix != "h" {
		t.Errorf("list --json = %s (%v)", out.String(), err)
	}

	e, out, _ = testEnv(true)
	if code := run([]string{"shim", "remove", "zzdemo"}, e); code != 0 || !strings.Contains(out.String(), "removed shim zzdemo") {
		t.Errorf("remove: code %d, out %q", code, out.String())
	}
	entries, _ := os.ReadDir(shimDir)
	if len(entries) != 0 {
		t.Errorf("stubs were left behind: %v", entries)
	}
}

func TestShimAddNameFromFlag(t *testing.T) {
	shimSetup(t)
	proj := project(t, "") // no registry, so no "cli" field
	e, _, errOut := testEnv(true)
	if code := run([]string{"shim", "add", proj}, e); code != 1 || !strings.Contains(errOut.String(), "--name") {
		t.Errorf("without a name: code %d, stderr %q", code, errOut.String())
	}
	e, out, errOut := testEnv(true)
	if code := run([]string{"shim", "add", proj, "--name", "zzflag", "--suffix", "-help", "--soft"}, e); code != 0 {
		t.Fatalf("with --name: exit %d, stderr %q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "zzflag (and zzflag-help)") {
		t.Errorf("output = %q", out.String())
	}
	e, out, _ = testEnv(true)
	run([]string{"shim", "list"}, e)
	if !strings.Contains(out.String(), "(soft)") {
		t.Errorf("list should show the shim as soft: %q", out.String())
	}
}

func TestShimAddDryRunWritesNothing(t *testing.T) {
	shimDir := shimSetup(t)
	proj := project(t, "zzdry")
	e, out, _ := testEnv(true)
	if code := run([]string{"shim", "add", proj, "--dry-run"}, e); code != 0 || !strings.Contains(out.String(), "would add") {
		t.Fatalf("dry run: code %d, out %q", code, out.String())
	}
	if entries, _ := os.ReadDir(shimDir); len(entries) != 0 {
		t.Errorf("a dry run wrote files: %v", entries)
	}
	e, out, _ = testEnv(true)
	run([]string{"shim", "list"}, e)
	if !strings.Contains(out.String(), "No shims registered.") {
		t.Errorf("a dry run registered a shim: %q", out.String())
	}
}

func TestShimAddCollisionBetweenProjects(t *testing.T) {
	shimSetup(t)
	a, b := project(t, "zzsame"), project(t, "zzsame")
	e, _, _ := testEnv(true)
	if code := run([]string{"shim", "add", a}, e); code != 0 {
		t.Fatalf("first add: exit %d", code)
	}
	e, _, errOut := testEnv(true)
	if code := run([]string{"shim", "add", b}, e); code != 1 {
		t.Fatalf("second add: exit %d, want 1", code)
	}
	if !strings.Contains(errOut.String(), "--replace") || !strings.Contains(errOut.String(), "already points at") {
		t.Errorf("stderr should explain and mention --replace: %q", errOut.String())
	}
	e, out, _ := testEnv(true)
	if code := run([]string{"shim", "add", b, "--replace"}, e); code != 0 || !strings.Contains(out.String(), "updated shim") {
		t.Errorf("--replace: code %d, out %q", code, out.String())
	}
}

func TestShimAddJSON(t *testing.T) {
	shimSetup(t)
	proj := project(t, "zzjson")
	e, out, _ := testEnv(true)
	if code := run([]string{"shim", "add", proj, "--json"}, e); code != 0 {
		t.Fatalf("exit %d", code)
	}
	var got struct {
		Entry    struct{ Name string }
		Files    []string
		Updated  bool
		Warnings []string
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil || got.Entry.Name != "zzjson" || len(got.Files) != 2 {
		t.Errorf("add --json = %s (%v)", out.String(), err)
	}
}

func TestShimRemoveUnknown(t *testing.T) {
	shimSetup(t)
	e, _, errOut := testEnv(true)
	if code := run([]string{"shim", "remove", "zznone"}, e); code != 1 || !strings.Contains(errOut.String(), "no shim named") {
		t.Errorf("code %d, stderr %q", code, errOut.String())
	}
}

func TestShimUsageErrors(t *testing.T) {
	shimSetup(t)
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"add bad flag", []string{"shim", "add", "--wat"}, `unknown flag "--wat"`},
		{"add two paths", []string{"shim", "add", "a", "b"}, "only one path"},
		{"add name without value", []string{"shim", "add", "--name"}, "--name needs a value"},
		{"remove without name", []string{"shim", "remove"}, "expected a shim name"},
		{"remove bad flag", []string{"shim", "remove", "--wat"}, `unknown flag "--wat"`},
		{"remove two names", []string{"shim", "remove", "a", "b"}, "only one name"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, _, errOut := testEnv(true)
			if code := run(tt.args, e); code != 2 {
				t.Errorf("exit %d, want 2", code)
			}
			if !strings.Contains(errOut.String(), tt.want) {
				t.Errorf("stderr = %q, want %q", errOut.String(), tt.want)
			}
		})
	}
}

// A file that is not ours gets guidance that works: --replace does not cover
// it, so the message must not suggest it, and the file is reported once.
func TestShimAddNotOursHint(t *testing.T) {
	shimDir := shimSetup(t)
	proj := registryProject(t)
	name := "zzrun"
	file := filepath.Join(shimDir, name)
	if runtime.GOOS == "windows" {
		file += ".cmd"
	}
	if err := os.WriteFile(file, []byte("#!/bin/sh\nexec just\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	e, _, errOut := testEnv(true)
	if code := run([]string{"shim", "add", proj}, e); code != 1 {
		t.Fatalf("want exit 1, got %d: %s", code, errOut.String())
	}
	out := errOut.String()
	if strings.Count(out, "was not written by justz") != 1 {
		t.Errorf("want the file reported once: %q", out)
	}
	if strings.Contains(out, "would hide it") {
		t.Errorf("the same file must not also be reported as hidden: %q", out)
	}
	if strings.Contains(out, "use --replace") {
		t.Errorf("--replace does not help here, so it must not be suggested: %q", out)
	}
	if !strings.Contains(out, "move or delete that file") {
		t.Errorf("want guidance that works: %q", out)
	}
}
