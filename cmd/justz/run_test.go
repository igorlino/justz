package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// runtimeEnv is an env whose just can be dumped (it knows the given root
// recipes) and whose execJust records the arguments instead of running.
type runtimeEnv struct {
	env
	out, errOut *bytes.Buffer
	calls       [][]string
	exitCode    int
}

func newRuntimeEnv(rootRecipes ...string) *runtimeEnv {
	r := &runtimeEnv{out: &bytes.Buffer{}, errOut: &bytes.Buffer{}}
	recipes := map[string]map[string]bool{}
	for _, n := range rootRecipes {
		recipes[n] = map[string]bool{"private": false}
	}
	dump, _ := json.Marshal(map[string]any{"recipes": recipes})
	r.env = env{
		stdout: r.out,
		stderr: r.errOut,
		lookPath: func(file string) (string, error) {
			return "/usr/bin/" + file, nil
		},
		runCmd: func(name string, args ...string) (string, error) {
			for _, a := range args {
				if a == "--dump" {
					return string(dump), nil
				}
			}
			return "", errors.New("unexpected command")
		},
		execJust: func(args []string) int {
			r.calls = append(r.calls, args)
			return r.exitCode
		},
	}
	return r
}

const demoRegistry = `{
  "cli": "zzrun",
  "groups": {
    "core": { "commands": { "ci": {} } },
    "tool": { "commands": { "git": {}, "doctor": {} } },
    "sbx": { "commands": { "doctor": {} } },
    "demo": { "commands": { "greet": {} }, "extra": { "hidden": {} } },
    "other": { "commands": { "greet": {} } }
  }
}`

// registryProject makes a project directory with the demo registry.
func registryProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "justfile"), "default:\n")
	writeFile(t, filepath.Join(dir, "config", "commands.json"), demoRegistry)
	return dir
}

func TestResolveCommand(t *testing.T) {
	proj := registryProject(t)
	t.Chdir(proj)

	tests := []struct {
		name     string
		args     []string
		wantCode int
		wantOut  string
		wantErr  string
	}{
		{"unique command", []string{"resolve", "git"}, 0, "tool git\n", ""},
		{"ambiguous", []string{"resolve", "doctor"}, 2, "", "sbx doctor"},
		{"ambiguous lists all", []string{"resolve", "greet"}, 2, "", "other greet"},
		{"core command", []string{"resolve", "ci"}, 3, "", "core command"},
		{"group word", []string{"resolve", "tool"}, 3, "", "it is a group"},
		{"documented extra", []string{"resolve", "hidden"}, 3, "", "documented extra"},
		{"root recipe wins", []string{"resolve", "build"}, 3, "", "recipe of the justfile"},
		{"prefix never resolves", []string{"resolve", "gi"}, 3, "", "exact name"},
		{"unknown", []string{"resolve", "nope"}, 3, "", "does not resolve"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newRuntimeEnv("build")
			code := run(tt.args, r.env)
			if code != tt.wantCode {
				t.Errorf("exit %d, want %d", code, tt.wantCode)
			}
			if tt.wantOut != "" && r.out.String() != tt.wantOut {
				t.Errorf("stdout = %q, want %q", r.out.String(), tt.wantOut)
			}
			if !strings.Contains(r.errOut.String(), tt.wantErr) {
				t.Errorf("stderr = %q, want it to contain %q", r.errOut.String(), tt.wantErr)
			}
		})
	}
}

func TestResolveJSON(t *testing.T) {
	t.Chdir(registryProject(t))
	r := newRuntimeEnv()
	if code := run([]string{"resolve", "--json", "git"}, r.env); code != 0 {
		t.Fatalf("exit %d", code)
	}
	var got struct {
		Name, Status string
		Matches      []struct{ Group, Command string }
	}
	if err := json.Unmarshal(r.out.Bytes(), &got); err != nil || got.Status != "found" || len(got.Matches) != 1 || got.Matches[0].Group != "tool" {
		t.Errorf("resolve --json = %s (%v)", r.out.String(), err)
	}

	r = newRuntimeEnv()
	if code := run([]string{"resolve", "--json", "doctor"}, r.env); code != 2 {
		t.Errorf("ambiguous --json: exit %d, want 2", code)
	}
	if !strings.Contains(r.out.String(), `"status": "ambiguous"`) {
		t.Errorf("stdout = %q", r.out.String())
	}
}

func TestResolveNeedsARegistry(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "justfile"), "default:\n")
	t.Chdir(dir)
	r := newRuntimeEnv()
	if code := run([]string{"resolve", "git"}, r.env); code != 3 || !strings.Contains(r.errOut.String(), "no commands.json") {
		t.Errorf("code %d, stderr %q", code, r.errOut.String())
	}
}

func TestResolveUsageErrors(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"resolve"}, "expected a command name"},
		{[]string{"resolve", "--wat"}, "unknown flag"},
		{[]string{"resolve", "a", "b"}, "only one name"},
	} {
		r := newRuntimeEnv()
		if code := run(tc.args, r.env); code != 2 || !strings.Contains(r.errOut.String(), tc.want) {
			t.Errorf("%v: code %d, stderr %q", tc.args, code, r.errOut.String())
		}
	}
}

// addDemoShim registers a shim named zzrun for a registry project.
func addDemoShim(t *testing.T, soft bool) (proj string) {
	t.Helper()
	shimSetup(t)
	proj = registryProject(t)
	e, _, errOut := testEnv(true)
	args := []string{"shim", "add", proj}
	if soft {
		args = append(args, "--soft")
	}
	if code := run(args, e); code != 0 {
		t.Fatalf("shim add: %d %s", code, errOut.String())
	}
	return proj
}

func TestShimRunResolvesBareNames(t *testing.T) {
	proj := addDemoShim(t, false)
	jf := filepath.Join(proj, "justfile")
	// The shim works from anywhere.
	t.Chdir(t.TempDir())

	tests := []struct {
		name     string
		args     []string
		wantCall []string // nil means just must not run
		wantCode int
		wantErr  string
	}{
		{"bare name runs without its group", []string{"git", "-v"}, []string{"--justfile", jf, "tool", "git", "-v"}, 0, "zzrun: tool git"},
		{"no arguments", nil, []string{"--justfile", jf}, 0, ""},
		{"dash-first argument passes through", []string{"--list"}, []string{"--justfile", jf, "--list"}, 0, ""},
		{"real root recipe wins", []string{"build"}, []string{"--justfile", jf, "build"}, 0, ""},
		{"core command runs as it is", []string{"ci"}, []string{"--justfile", jf, "ci"}, 0, ""},
		{"explicit group path passes through", []string{"tool", "git"}, []string{"--justfile", jf, "tool", "git"}, 0, ""},
		{"documented extra is not run by bare name", []string{"hidden"}, []string{"--justfile", jf, "hidden"}, 0, ""},
		{"prefix never runs", []string{"gi"}, []string{"--justfile", jf, "gi"}, 0, ""},
		{"ambiguous name exits 2 and lists the candidates", []string{"doctor"}, nil, 2, "zzrun tool doctor"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newRuntimeEnv("build")
			code := run(append([]string{"--as", "zzrun"}, tt.args...), r.env)
			if code != tt.wantCode {
				t.Errorf("exit %d, want %d (stderr %q)", code, tt.wantCode, r.errOut.String())
			}
			if tt.wantCall == nil {
				if len(r.calls) != 0 {
					t.Errorf("just must not run, but got %v", r.calls)
				}
			} else if len(r.calls) != 1 || strings.Join(r.calls[0], "|") != strings.Join(tt.wantCall, "|") {
				t.Errorf("just ran %v, want %v", r.calls, tt.wantCall)
			}
			if !strings.Contains(r.errOut.String(), tt.wantErr) {
				t.Errorf("stderr = %q, want it to contain %q", r.errOut.String(), tt.wantErr)
			}
		})
	}
}

func TestShimRunQuietAndExitCode(t *testing.T) {
	addDemoShim(t, false)
	t.Setenv("JUSTZ_QUIET", "1")
	r := newRuntimeEnv()
	r.exitCode = 7
	if code := run([]string{"--as", "zzrun", "git"}, r.env); code != 7 {
		t.Errorf("exit %d, want the exit code of just (7)", code)
	}
	if r.errOut.Len() != 0 {
		t.Errorf("JUSTZ_QUIET=1 should silence the echo, got %q", r.errOut.String())
	}
}

func TestShimWhich(t *testing.T) {
	addDemoShim(t, false)
	for _, tc := range []struct {
		args     []string
		wantCode int
		wantOut  string
	}{
		{[]string{"--which", "git"}, 0, "tool git\n"},
		{[]string{"--which", "doctor"}, 2, ""},
		{[]string{"--which", "nope"}, 3, ""},
		{[]string{"--which", "tool"}, 3, ""},
		{[]string{"--which"}, 2, ""},
	} {
		r := newRuntimeEnv()
		code := run(append([]string{"--as", "zzrun"}, tc.args...), r.env)
		if code != tc.wantCode || (tc.wantOut != "" && r.out.String() != tc.wantOut) {
			t.Errorf("%v: code %d out %q, want %d %q", tc.args, code, r.out.String(), tc.wantCode, tc.wantOut)
		}
		if len(r.calls) != 0 {
			t.Errorf("%v: --which must never run just: %v", tc.args, r.calls)
		}
	}
}

func TestShimRunErrors(t *testing.T) {
	shimSetup(t)
	r := newRuntimeEnv()
	if code := run([]string{"--as", "zznone"}, r.env); code != 2 || !strings.Contains(r.errOut.String(), "no shim named") {
		t.Errorf("unknown shim: code %d, stderr %q", code, r.errOut.String())
	}
	r = newRuntimeEnv()
	if code := run([]string{"--as"}, r.env); code != 2 {
		t.Errorf("--as without a name: code %d", code)
	}

	proj := addDemoShim(t, false)

	// The project moves away.
	if err := os.Remove(filepath.Join(proj, "justfile")); err != nil {
		t.Fatal(err)
	}
	r = newRuntimeEnv()
	if code := run([]string{"--as", "zzrun", "git"}, r.env); code != 2 || !strings.Contains(r.errOut.String(), "shim doctor") || len(r.calls) != 0 {
		t.Errorf("missing justfile: code %d, stderr %q, calls %v", code, r.errOut.String(), r.calls)
	}
	r = newRuntimeEnv()
	if code := run([]string{"--as", "zzrunh"}, r.env); code != 2 || !strings.Contains(r.errOut.String(), "shim doctor") {
		t.Errorf("help shim with a missing justfile: code %d, stderr %q", code, r.errOut.String())
	}
}

func TestSoftShimFallsThroughWhenTheJustfileIsGone(t *testing.T) {
	proj := addDemoShim(t, true)
	jf := filepath.Join(proj, "justfile")
	if err := os.Remove(jf); err != nil {
		t.Fatal(err)
	}
	r := newRuntimeEnv()
	if code := run([]string{"--as", "zzrun", "git"}, r.env); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, r.errOut.String())
	}
	if len(r.calls) != 1 || strings.Join(r.calls[0], "|") != strings.Join([]string{"--justfile", jf, "git"}, "|") {
		t.Errorf("a soft shim should hand over to plain just, got %v", r.calls)
	}
}

func TestExecJustRemovesJustzVariables(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the recipe below needs a POSIX shell")
	}
	if _, err := os.Stat("/usr/bin/env"); err != nil {
		t.Skip("needs a POSIX system")
	}
	dir := t.TempDir()
	outFile := filepath.Join(dir, "seen.txt")
	writeFile(t, filepath.Join(dir, "justfile"), "set shell := [\"sh\", \"-c\"]\n\nshow:\n    printf '%s|%s' \"${JUSTZ_QUIET-unset}\" \"${KEEP_ME-unset}\" > '"+outFile+"'\n")
	t.Setenv("JUSTZ_QUIET", "1")
	t.Setenv("JUSTZ_CONFIG_DIR", "/somewhere")
	t.Setenv("KEEP_ME", "yes")

	if code := execJust([]string{"--justfile", filepath.Join(dir, "justfile"), "show"}); code != 0 {
		t.Skipf("just is not available or failed (exit %d)", code)
	}
	got, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "unset|yes" {
		t.Errorf("recipe saw %q, want JUSTZ_ variables removed and others kept (unset|yes)", got)
	}
}

func TestInvokedAs(t *testing.T) {
	for in, want := range map[string]string{
		"jz": "jz", "/usr/local/bin/jz": "jz", "jzh": "jzh", "JZ.EXE": "jz", "jzh.exe": "jzh",
		"justz": "", "/usr/bin/just": "", "jzz": "", "": "",
	} {
		if got := invokedAs(in); got != want {
			t.Errorf("invokedAs(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLocalShortcut(t *testing.T) {
	proj := registryProject(t)
	jf := filepath.Join(proj, "justfile")
	sub := filepath.Join(proj, "src", "deep")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(sub) // jz finds the justfile by walking up, like just does

	r := newRuntimeEnv("build")
	if code := run([]string{"--local", "jz", "git", "-v"}, r.env); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, r.errOut.String())
	}
	want := []string{"--justfile", jf, "tool", "git", "-v"}
	if len(r.calls) != 1 || strings.Join(r.calls[0], "|") != strings.Join(want, "|") {
		t.Errorf("just ran %v, want %v", r.calls, want)
	}
	if !strings.Contains(r.errOut.String(), "jz: tool git") {
		t.Errorf("stderr = %q", r.errOut.String())
	}

	// A flag first, a real recipe and an ambiguous name behave like a shim.
	r = newRuntimeEnv("build")
	run([]string{"--local", "jz", "--list"}, r.env)
	run([]string{"--local", "jz", "build"}, r.env)
	if len(r.calls) != 2 || r.calls[0][2] != "--list" || r.calls[1][2] != "build" {
		t.Errorf("calls = %v", r.calls)
	}
	r = newRuntimeEnv("build")
	if code := run([]string{"--local", "jz", "doctor"}, r.env); code != 2 || len(r.calls) != 0 {
		t.Errorf("ambiguous: code %d, calls %v", code, r.calls)
	}
}

func TestLocalShortcutWithoutAJustfile(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	r := newRuntimeEnv()
	if code := run([]string{"--local", "jz", "x"}, r.env); code != 2 || !strings.Contains(r.errOut.String(), "no justfile found") || len(r.calls) != 0 {
		t.Errorf("code %d, stderr %q, calls %v", code, r.errOut.String(), r.calls)
	}
	r = newRuntimeEnv()
	if code := run([]string{"--local", "jzh"}, r.env); code != 1 || !strings.Contains(r.errOut.String(), "no justfile") {
		t.Errorf("jzh without a justfile: code %d, stderr %q", code, r.errOut.String())
	}
	r = newRuntimeEnv()
	if code := run([]string{"--local"}, r.env); code != 2 {
		t.Errorf("--local without a name: code %d", code)
	}
}

func TestShimHelpNames(t *testing.T) {
	addDemoShim(t, false)
	t.Chdir(t.TempDir()) // a pinned shim works from anywhere

	helpRun := func(args ...string) *runtimeEnv {
		r := newRuntimeEnv()
		r.env.width = 100
		code := run(append([]string{"--as", "zzrunh"}, args...), r.env)
		if code != 0 {
			t.Fatalf("zzrunh %v: exit %d, stderr %q", args, code, r.errOut.String())
		}
		return r
	}

	r := helpRun()
	for _, want := range []string{"  Core\n", "Commands  (run without args", "zzrunh --expanded"} {
		if !strings.Contains(r.out.String(), want) {
			t.Errorf("bare help should contain %q:\n%s", want, r.out.String())
		}
	}
	if len(r.calls) != 0 {
		t.Errorf("help must not run just: %v", r.calls)
	}

	r = helpRun("git")
	if !strings.Contains(r.out.String(), "zzrun tool git") {
		t.Errorf("zooming a command:\n%s", r.out.String())
	}
	r = helpRun("-j", "tool")
	if !strings.Contains(r.out.String(), `"path": "tool"`) {
		t.Errorf("-j should print JSON: %q", r.out.String())
	}

	// A miss and an ambiguous name use the shim's own help name in the hint.
	r = newRuntimeEnv()
	if code := run([]string{"--as", "zzrunh", "nope"}, r.env); code != 2 || !strings.Contains(r.errOut.String(), "Try: zzrunh --expanded") {
		t.Errorf("miss: code %d, stderr %q", code, r.errOut.String())
	}
}

func TestJzhHelpsFromASubdirectory(t *testing.T) {
	proj := registryProject(t)
	sub := filepath.Join(proj, "src", "deep")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(sub)

	r := newRuntimeEnv()
	if code := run([]string{"--local", "jzh"}, r.env); code != 0 {
		t.Fatalf("bare jzh: exit %d, stderr %q", code, r.errOut.String())
	}
	if !strings.Contains(r.out.String(), "jzh --expanded") || !strings.Contains(r.out.String(), "Commands  (run without") {
		t.Errorf("bare jzh should print the summary with a usage hint:\n%s", r.out.String())
	}

	r = newRuntimeEnv()
	if code := run([]string{"--local", "jzh", "git"}, r.env); code != 0 || !strings.Contains(r.out.String(), "Verbose") && !strings.Contains(r.out.String(), "zzrun tool git") {
		t.Errorf("jzh <command> should zoom from a subdirectory: code %d, out %q", code, r.out.String())
	}

	r = newRuntimeEnv()
	if code := run([]string{"--local", "jzh", "-j"}, r.env); code != 0 || !strings.HasPrefix(strings.TrimSpace(r.out.String()), "{") {
		t.Errorf("jzh -j: code %d, out %q", code, r.out.String())
	}
}
