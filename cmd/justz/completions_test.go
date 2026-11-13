package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const completeRegistry = `{
  "cli": "zzcomp",
  "groups": {
    "core": { "commands": { "build": {"desc": "Build it"} } },
    "tool": { "label": "Tools", "commands": {
        "git": {"desc": "Git helper", "args": [
          {"name": "-v, --level <n>", "desc": "Level", "values": ["low", "high"]},
          {"name": "-q, --quiet", "desc": "Quiet"}]} } }
  }
}`

func completeLines(t *testing.T, args ...string) []string {
	t.Helper()
	r := newRuntimeEnv("build")
	if code := run(append([]string{"__complete"}, args...), r.env); code != 0 {
		t.Fatalf("__complete %v: exit %d, stderr %q", args, code, r.errOut.String())
	}
	out := strings.TrimRight(r.out.String(), "\n")
	if out == "" {
		return nil
	}
	return strings.Split(out, "\n")
}

func completeProject(t *testing.T) string {
	t.Helper()
	dir := gitProject(t)
	writeFile(t, filepath.Join(dir, "config", "commands.json"), completeRegistry)
	t.Chdir(dir)
	t.Setenv("JUSTZ_CONFIG_DIR", t.TempDir()) // never read the real shim registry
	return dir
}

func TestCompleteJustzCommands(t *testing.T) {
	completeProject(t)
	lines := completeLines(t, "--", "ch")
	if len(lines) != 1 || lines[0] != "check\tCheck how the project documents its commands" {
		t.Errorf("lines = %q (value, a tab, a description)", lines)
	}
	if got := completeLines(t, "--", "fmt", ""); len(got) != 1 || got[0] != ":files" {
		t.Errorf("fmt asks for files: %q", got)
	}
}

func TestCompleteRunShim(t *testing.T) {
	completeProject(t)
	lines := completeLines(t, "--shim", "jz", "--", "tool", "git", "")
	got := strings.Join(lines, "|")
	if got != "-q\tQuiet|--quiet\tQuiet|-v\tLevel|--level\tLevel" {
		t.Errorf("options of tool git = %q", got)
	}
	// An enum after its flag, and a bare name.
	if got := strings.Join(completeLines(t, "--shim", "jz", "--", "git", "-v", ""), "|"); got != "low\tLevel|high\tLevel" {
		t.Errorf("enum values = %q", got)
	}
	// jzh: the same lookup plus the help flags.
	if got := strings.Join(completeLines(t, "--shim", "jzh", "--", "-"), "|"); got != "-b\tPick a group or command|-j\tPrint JSON|-w\tAdd example columns" {
		t.Errorf("jzh flags = %q", got)
	}
}

func TestCompleteWithoutAProjectOffersFiles(t *testing.T) {
	bare := t.TempDir()
	if err := os.MkdirAll(filepath.Join(bare, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(bare)
	t.Setenv("JUSTZ_CONFIG_DIR", t.TempDir())
	r := newRuntimeEnv()
	r.runCmd = func(string, ...string) (string, error) { return "", os.ErrNotExist }
	if code := run([]string{"__complete", "--shim", "jz", "--", ""}, r.env); code != 0 || strings.TrimSpace(r.out.String()) != ":files" {
		t.Errorf("code %d, out %q", code, r.out.String())
	}
}

func TestCompletePinnedShimWorksFromAnyDirectory(t *testing.T) {
	proj := completeProject(t)
	shimSetup(t) // a registry and a shim directory of its own
	e, _, errOut := testEnv(true)
	if code := run([]string{"shim", "add", proj}, e); code != 0 {
		t.Fatalf("shim add: %d %s", code, errOut.String())
	}
	t.Chdir(t.TempDir()) // far away from the project

	for _, name := range []string{"zzcomp", "zzcomph"} {
		got := strings.Join(completeLines(t, "--shim", name, "--", "t"), "|")
		if got != "tool\tTools" {
			t.Errorf("%s: groups starting with t = %q", name, got)
		}
	}
	if lines := completeLines(t, "--shim", "nosuchshim", "--", ""); len(lines) != 0 {
		t.Errorf("an unknown shim completes nothing: %q", lines)
	}
}

func TestCompletionScripts(t *testing.T) {
	shimSetup(t)
	proj := project(t, "zzscript")
	e, _, _ := testEnv(true)
	if code := run([]string{"shim", "add", proj}, e); code != 0 {
		t.Fatal("shim add failed")
	}

	r := newRuntimeEnv()
	if code := run([]string{"completions", "zsh"}, r.env); code != 0 {
		t.Fatalf("zsh: exit %d", code)
	}
	zsh := r.out.String()
	for _, want := range []string{"#compdef justz jz jzh zzscript zzscripth", "compdef _justz_complete justz jz jzh zzscript zzscripth", "justz __complete --shim", "_describe", "_files"} {
		if !strings.Contains(zsh, want) {
			t.Errorf("zsh script should contain %q", want)
		}
	}

	r = newRuntimeEnv()
	if code := run([]string{"completions", "bash"}, r.env); code != 0 {
		t.Fatalf("bash: exit %d", code)
	}
	bash := r.out.String()
	for _, want := range []string{"complete -F _justz_complete justz jz jzh zzscript zzscripth", "COMP_WORDS", "compgen -f"} {
		if !strings.Contains(bash, want) {
			t.Errorf("bash script should contain %q", want)
		}
	}
}

func TestCompletionsUsage(t *testing.T) {
	for _, args := range [][]string{{"completions"}, {"completions", "fish"}, {"completions", "zsh", "bash"}} {
		r := newRuntimeEnv()
		if code := run(args, r.env); code != 2 || !strings.Contains(r.errOut.String(), "expected one shell") {
			t.Errorf("%v: code %d, stderr %q", args, code, r.errOut.String())
		}
	}
	r := newRuntimeEnv()
	if code := run([]string{"completions", "--help"}, r.env); code != 0 || !strings.Contains(r.out.String(), "eval") {
		t.Errorf("--help: code %d", code)
	}
}
