package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// demoHelpRegistry is read once at load time: the tests change directory later.
var demoHelpRegistry = func() string {
	b, err := os.ReadFile("../../testdata/help-reference/demo/config/commands.json")
	if err != nil {
		panic(err)
	}
	return string(b)
}()

func helpProject(t *testing.T) string {
	t.Helper()
	dir := gitProject(t)
	writeFile(t, filepath.Join(dir, "config", "commands.json"), demoHelpRegistry)
	return dir
}

func helpEnv(color bool, width int) *runtimeEnv {
	r := newRuntimeEnv()
	r.env.color, r.env.width = color, width
	return r
}

func TestHelpSummaryAndExpanded(t *testing.T) {
	dir := helpProject(t)
	sub := filepath.Join(dir, "src")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(sub) // found by walking up

	r := helpEnv(false, 100)
	if code := run([]string{"help"}, r.env); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, r.errOut.String())
	}
	out := r.out.String()
	for _, want := range []string{"  demo\n", "  Core\n", "build               # Compile the project", "Commands  (run without args to see sub-commands)", "justz help --expanded"} {
		if !strings.Contains(out, want) {
			t.Errorf("summary should contain %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "build docker") {
		t.Errorf("the summary must not list group commands:\n%s", out)
	}

	r = helpEnv(false, 100)
	if code := run([]string{"help", "--expanded"}, r.env); code != 0 {
		t.Fatalf("expanded: exit %d", code)
	}
	for _, want := range []string{"build  — Build", "build docker                 # Build the container image", "-k <pattern>"} {
		if !strings.Contains(r.out.String(), want) {
			t.Errorf("expanded should contain %q:\n%s", want, r.out.String())
		}
	}
}

func TestHelpColorFollowsTheEnvironment(t *testing.T) {
	t.Chdir(helpProject(t))
	ansi := regexp.MustCompile(`\x1b\[`)

	r := helpEnv(false, 80)
	run([]string{"help"}, r.env)
	if ansi.MatchString(r.out.String()) {
		t.Error("no colors when the output is not a terminal")
	}
	r = helpEnv(true, 80)
	run([]string{"help"}, r.env)
	if !ansi.MatchString(r.out.String()) {
		t.Error("colors expected on a terminal")
	}
}

func TestHelpWideFlag(t *testing.T) {
	dir := gitProject(t)
	writeFile(t, filepath.Join(dir, "config", "commands.json"), `{"groups":{"core":{"commands":{"test":{"desc":"Run tests","example":"just test -w"}}}}}`)
	t.Chdir(dir)
	r := helpEnv(false, 80)
	run([]string{"help", "--expanded"}, r.env)
	if strings.Contains(r.out.String(), "just test -w") {
		t.Error("no example column at 80 columns")
	}
	r = helpEnv(false, 80)
	run([]string{"help", "--expanded", "-w"}, r.env)
	if !strings.Contains(r.out.String(), "just test -w") {
		t.Errorf("-w should add the example column:\n%s", r.out.String())
	}
}

func TestHelpUsesTheLocalOverlay(t *testing.T) {
	dir := gitProject(t)
	writeFile(t, filepath.Join(dir, ".just-local-rich.json"), `{"groups":{"core":{"commands":{"build":{"desc":"From the overlay"}}}}}`)
	t.Chdir(dir)
	r := helpEnv(false, 80)
	if code := run([]string{"help"}, r.env); code != 0 || !strings.Contains(r.out.String(), "From the overlay") {
		t.Errorf("code %d, out %q, err %q", code, r.out.String(), r.errOut.String())
	}
}

func TestHelpErrors(t *testing.T) {
	t.Chdir(gitProject(t)) // no registry, no overlay
	r := helpEnv(false, 80)
	if code := run([]string{"help"}, r.env); code != 1 || !strings.Contains(r.errOut.String(), "justz generate") {
		t.Errorf("no registry: code %d, stderr %q", code, r.errOut.String())
	}

	dir := gitProject(t)
	writeFile(t, filepath.Join(dir, "config", "commands.json"), `{not json`)
	t.Chdir(dir)
	r = helpEnv(false, 80)
	if code := run([]string{"help"}, r.env); code != 1 || !strings.Contains(r.errOut.String(), "not valid JSON") {
		t.Errorf("invalid registry: code %d, stderr %q", code, r.errOut.String())
	}

	r = helpEnv(false, 80)
	if code := run([]string{"help", "--wat"}, r.env); code != 2 || !strings.Contains(r.errOut.String(), `unknown flag "--wat"`) {
		t.Errorf("bad flag: code %d, stderr %q", code, r.errOut.String())
	}
	r = helpEnv(false, 80)
	if code := run([]string{"help", "--help"}, r.env); code != 0 || !strings.Contains(r.out.String(), "Usage: justz help") {
		t.Errorf("--help: code %d", code)
	}
}

func TestTerminalWidth(t *testing.T) {
	t.Setenv("COLUMNS", "120")
	if got := terminalWidth(os.Stdout); got != 120 {
		t.Errorf("COLUMNS=120 gave %d", got)
	}
	for _, bad := range []string{"", "abc", "0", "-5"} {
		t.Setenv("COLUMNS", bad)
		if got := terminalWidth(os.Stdin); got <= 0 {
			t.Errorf("COLUMNS=%q gave %d, want a positive fallback", bad, got)
		}
	}
}

func TestNoColorWins(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	if useColor(os.Stdout) {
		t.Error("NO_COLOR must turn colors off")
	}
}

const zoomRegistry = `{
  "cli": "dm",
  "groups": {
    "core": { "commands": { "features": {"desc": "List features"} } },
    "tool": { "label": "Tools", "commands": { "git": {"desc": "Git helper", "args": [{"name": "-v", "desc": "Verbose"}]}, "doctor": {"desc": "Check tools"} } },
    "sbx":  { "label": "Sandboxes", "commands": { "doctor": {"desc": "Check sandboxes"} } },
    "ai":   { "label": "AI agents", "commands": { "claude": {"desc": "Claude"} } }
  }
}`

func TestHelpZoom(t *testing.T) {
	dir := gitProject(t)
	writeFile(t, filepath.Join(dir, "config", "commands.json"), zoomRegistry)
	t.Chdir(dir)

	tests := []struct {
		name     string
		args     []string
		wantCode int
		wantOut  string
		wantErr  string
	}{
		{"bare command", []string{"help", "git"}, 0, "dm tool git", ""},
		{"unique prefix", []string{"help", "gi"}, 0, "dm tool git", ""},
		{"two words", []string{"help", "tool", "git"}, 0, "dm tool git", ""},
		{"group exact, command prefix", []string{"help", "ai", "cla"}, 0, "dm ai claude", ""},
		{"group prefix, command prefix", []string{"help", "to", "gi"}, 0, "dm tool git", ""},
		{"core command", []string{"help", "features"}, 0, "dm features", ""},
		{"a group", []string{"help", "tool"}, 0, "tool · Tools", ""},
		{"ambiguous exits 2", []string{"help", "doctor"}, 2, "", "ambiguous: 'doctor' matches several entries:"},
		{"ambiguous lists candidates", []string{"help", "doctor"}, 2, "", "dm sbx doctor — Check sandboxes"},
		{"no match exits 2", []string{"help", "zzz-nothing"}, 2, "", "no group or command matches 'zzz-nothing'"},
		{"unknown group exits 2", []string{"help", "nogroup", "git"}, 2, "", "no group or command matches 'nogroup git'"},
		{"three words exits 1", []string{"help", "a", "b", "c"}, 1, "", "expected [group] <term>, got 3 words"},
		{"unknown flag exits 2", []string{"help", "--wat", "git"}, 2, "", `unknown flag "--wat"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := helpEnv(false, 100)
			code := run(tt.args, r.env)
			if code != tt.wantCode {
				t.Errorf("exit %d, want %d (stderr %q)", code, tt.wantCode, r.errOut.String())
			}
			if !strings.Contains(r.out.String(), tt.wantOut) {
				t.Errorf("stdout should contain %q:\n%s", tt.wantOut, r.out.String())
			}
			if !strings.Contains(r.errOut.String(), tt.wantErr) {
				t.Errorf("stderr should contain %q:\n%s", tt.wantErr, r.errOut.String())
			}
		})
	}
}

func TestHelpJSON(t *testing.T) {
	dir := gitProject(t)
	writeFile(t, filepath.Join(dir, "config", "commands.json"), zoomRegistry)
	t.Chdir(dir)

	decode := func(r *runtimeEnv, v any) {
		t.Helper()
		if err := json.Unmarshal(r.out.Bytes(), v); err != nil {
			t.Fatalf("stdout is not valid JSON: %v\n%s", err, r.out.String())
		}
	}

	r := helpEnv(false, 100)
	if code := run([]string{"help", "--json"}, r.env); code != 0 {
		t.Fatalf("overview: exit %d", code)
	}
	var ov struct {
		CLI    string
		Groups []struct{ Name string }
		Core   []string
	}
	decode(r, &ov)
	if ov.CLI != "dm" || len(ov.Groups) != 3 || len(ov.Core) != 1 {
		t.Errorf("overview = %+v", ov)
	}

	r = helpEnv(false, 100)
	if code := run([]string{"help", "-j", "tool", "git"}, r.env); code != 0 {
		t.Fatalf("command: exit %d", code)
	}
	var cmd struct {
		Path, Invoke, Desc string
		Args               []struct{ Name, Desc string }
	}
	decode(r, &cmd)
	if cmd.Path != "tool git" || cmd.Invoke != "dm tool git" || len(cmd.Args) != 1 || cmd.Args[0].Name != "-v" {
		t.Errorf("command = %+v", cmd)
	}

	r = helpEnv(false, 100)
	if code := run([]string{"help", "--json", "tool"}, r.env); code != 0 {
		t.Fatalf("group: exit %d", code)
	}
	var grp struct {
		Path     string
		Commands []struct{ Name string }
	}
	decode(r, &grp)
	if grp.Path != "tool" || len(grp.Commands) != 2 || grp.Commands[0].Name != "doctor" {
		t.Errorf("group = %+v", grp)
	}

	// Ambiguous: an array of every match, and exit 2.
	r = helpEnv(false, 100)
	if code := run([]string{"help", "--json", "doctor"}, r.env); code != 2 {
		t.Errorf("ambiguous: exit %d, want 2", code)
	}
	var both []struct{ Path string }
	decode(r, &both)
	if len(both) != 2 || both[0].Path != "sbx doctor" || both[1].Path != "tool doctor" {
		t.Errorf("ambiguous array = %+v", both)
	}

	r = helpEnv(false, 100)
	if code := run([]string{"help", "--json", "zzz"}, r.env); code != 2 || r.out.Len() != 0 || !strings.Contains(r.errOut.String(), "no group or command matches 'zzz'") {
		t.Errorf("no match: code %d, stdout %q, stderr %q", code, r.out.String(), r.errOut.String())
	}
}
