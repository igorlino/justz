package main

import (
	"encoding/json"
	"errors"
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
	// No registry, no overlay, no justfile at all.
	bare := t.TempDir()
	if err := os.MkdirAll(filepath.Join(bare, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(bare)
	r := helpEnv(false, 80)
	if code := run([]string{"help"}, r.env); code != 1 || !strings.Contains(r.errOut.String(), "no justfile") {
		t.Errorf("nothing to read: code %d, stderr %q", code, r.errOut.String())
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
		{"three words is a usage error", []string{"help", "a", "b", "c"}, 2, "", "expected [group] <term>, got 3 words"},
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

// A native project has a justfile and no registry: help is rendered from the
// output of just --dump.
func TestHelpNativeProject(t *testing.T) {
	dir := gitProject(t) // a justfile, no registry
	t.Chdir(dir)
	nativeEnv := func() *runtimeEnv {
		r := helpEnv(false, 100)
		r.runCmd = func(name string, args ...string) (string, error) {
			if name == "just" {
				return string(nativeDump), nil
			}
			return "", errors.New("unexpected command")
		}
		return r
	}

	r := nativeEnv()
	if code := run([]string{"help"}, r.env); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, r.errOut.String())
	}
	out := r.out.String()
	// [group] attributes become headings; recipes keep their own path. The
	// module of the fixture has no default recipe, so its heading has no hint.
	for _, want := range []string{"  core\n", "  deploy\n", "Compile the project", "Deploy to an environment", "  Commands\n"} {
		if !strings.Contains(out, want) {
			t.Errorf("native summary should contain %q:\n%s", want, out)
		}
	}

	r = nativeEnv()
	if code := run([]string{"help", "test"}, r.env); code != 0 {
		t.Fatalf("zoom: exit %d, stderr %q", code, r.errOut.String())
	}
	zoomed := r.out.String()
	for _, want := range []string{"just test", "Run the test suite", "-k <pattern>", "-w, --watch", "$ just test -w"} {
		if !strings.Contains(zoomed, want) {
			t.Errorf("native zoom should contain %q:\n%s", want, zoomed)
		}
	}

	r = nativeEnv()
	if code := run([]string{"help", "tool", "clean"}, r.env); code != 0 || !strings.Contains(r.out.String(), "just tool clean") {
		t.Errorf("module command: code %d, out %q", code, r.out.String())
	}

}

func TestHelpNativeJSON(t *testing.T) {
	t.Chdir(gitProject(t))
	r := helpEnv(false, 100)
	r.runCmd = func(name string, args ...string) (string, error) { return string(nativeDump), nil }
	if code := run([]string{"help", "--json", "test"}, r.env); code != 0 || !strings.Contains(r.out.String(), `"invoke": "just test"`) {
		t.Errorf("native JSON: code %d, out %q", code, r.out.String())
	}
}

const detailCLIRegistry = `{
  "cli": "dm",
  "groups": {
    "tool": { "label": "Tools", "commands": {
      "doctor": {"desc": "Check tools", "detail": "Walks the PATH and reports what is missing."} } },
    "sbx": { "label": "Sandboxes", "commands": {
      "doctor": {"desc": "Check sandboxes", "detail": "Starts each sandbox and stops it again."} } }
  }
}`

func TestHelpShowsDetailOnlyInTheZoom(t *testing.T) {
	dir := gitProject(t)
	writeFile(t, filepath.Join(dir, "config", "commands.json"), detailCLIRegistry)
	t.Chdir(dir)

	r := helpEnv(false, 80)
	if code := run([]string{"help", "tool", "doctor"}, r.env); code != 0 || !strings.Contains(r.out.String(), "Details") || !strings.Contains(r.out.String(), "Walks the PATH") {
		t.Errorf("zoom: exit %d, out %q", code, r.out.String())
	}
	for _, args := range [][]string{{"help"}, {"help", "--expanded"}, {"help", "tool"}} {
		r = helpEnv(false, 80)
		if code := run(args, r.env); code != 0 || strings.Contains(r.out.String(), "Walks the PATH") {
			t.Errorf("%v: exit %d, detail leaked into the listing:\n%s", args, code, r.out.String())
		}
	}

	r = helpEnv(false, 80)
	if code := run([]string{"help", "-j", "tool", "doctor"}, r.env); code != 0 || !strings.Contains(r.out.String(), `"detail": "Walks the PATH and reports what is missing."`) {
		t.Errorf("zoom JSON: exit %d, out %q", code, r.out.String())
	}
	r = helpEnv(false, 80)
	if code := run([]string{"help", "-j", "tool"}, r.env); code != 0 || strings.Contains(r.out.String(), "detail") {
		t.Errorf("group JSON carries detail: exit %d, out %q", code, r.out.String())
	}
	// An ambiguous lookup is a choice, not a read: the array leaves the text out.
	r = helpEnv(false, 80)
	if code := run([]string{"help", "-j", "doctor"}, r.env); code != 2 || strings.Contains(r.out.String(), "detail") || !strings.Contains(r.out.String(), `"path": "sbx doctor"`) {
		t.Errorf("ambiguous JSON: exit %d, out %q", code, r.out.String())
	}
}

// A justfile with no attributes is the only source of a native project, so a
// parameter with no help text is still shown, and a recipe with no doc has no
// dangling hash in the listing.
func TestHelpNativeShowsUndocumentedParameters(t *testing.T) {
	t.Chdir(gitProject(t))
	const dump = `{"recipes": {
	  "deploy": {"private": false, "doc": null, "parameters": [{"name": "env", "kind": "singular"}], "attributes": []},
	  "test": {"private": false, "doc": "Run the tests", "parameters": [], "attributes": []}}, "modules": {}}`
	env := func() *runtimeEnv {
		r := helpEnv(false, 100)
		r.runCmd = func(name string, args ...string) (string, error) { return dump, nil }
		return r
	}

	r := env()
	if code := run([]string{"help"}, r.env); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, r.errOut.String())
	}
	if out := r.out.String(); !strings.Contains(out, "    deploy\n") || strings.Contains(out, "deploy              #") || !strings.Contains(out, "# Run the tests") {
		t.Errorf("an undocumented recipe is a bare name in the listing:\n%s", out)
	}

	r = env()
	if code := run([]string{"help", "deploy"}, r.env); code != 0 || !strings.Contains(r.out.String(), "<env>") {
		t.Errorf("zoom should show the parameter: exit %d\n%s", code, r.out.String())
	}

	r = env()
	if code := run([]string{"help", "-j", "deploy"}, r.env); code != 0 || !strings.Contains(r.out.String(), `"name": "\u003cenv\u003e"`) {
		t.Errorf("JSON should list the parameter: exit %d\n%s", code, r.out.String())
	}
}

// help -j --expanded is the inventory: every command with what it claims to
// touch, in one document. Without --expanded the overview has no commands.
func TestHelpJSONInventory(t *testing.T) {
	dir := gitProject(t)
	writeFile(t, filepath.Join(dir, "config", "commands.json"), `{"groups":{
	  "core":{"commands":{"build":{"desc":"Build it"}}},
	  "tool":{"commands":{"clean":{"desc":"Remove output","detail":"Deletes dist.","tags":["destructive"]}}}}}`)
	t.Chdir(dir)

	r := helpEnv(false, 80)
	if code := run([]string{"help", "-j", "--expanded"}, r.env); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, r.errOut.String())
	}
	var got struct {
		Groups   []struct{ Name string }
		Core     []string
		Commands []struct {
			Path, Desc string
			HasDetail  bool
			Tags       []string
		}
	}
	if err := json.Unmarshal(r.out.Bytes(), &got); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, r.out.String())
	}
	if len(got.Groups) != 1 || len(got.Core) != 1 || len(got.Commands) != 2 {
		t.Fatalf("inventory = %+v", got)
	}
	if c := got.Commands[0]; c.Path != "build" || c.Desc != "Build it" || c.HasDetail || c.Tags != nil {
		t.Errorf("core command = %+v", c)
	}
	if c := got.Commands[1]; c.Path != "tool clean" || !c.HasDetail || len(c.Tags) != 1 || c.Tags[0] != "destructive" {
		t.Errorf("group command = %+v", c)
	}

	r = helpEnv(false, 80)
	if code := run([]string{"help", "-j"}, r.env); code != 0 || strings.Contains(r.out.String(), `"commands": [`) {
		t.Errorf("the plain overview should not list commands: exit %d\n%s", code, r.out.String())
	}
}

// Positional arguments are read in order, so a documented parameter must not
// jump in front of an undocumented one. A module recipe is covered too, and
// the expanded reference and the group view have no dangling hash either.
func TestHelpNativeKeepsTheOrderOfParameters(t *testing.T) {
	t.Chdir(gitProject(t))
	const dump = `{"recipes": {
	  "release": {"private": false, "doc": null, "attributes": [], "parameters": [
	    {"name": "env", "kind": "singular"}, {"name": "tag", "kind": "singular", "help": "Release tag"}, {"name": "extra", "kind": "singular"}]}},
	  "modules": {"tool": {"recipes": {"bare": {"private": false, "doc": null, "attributes": [], "parameters": [{"name": "path", "kind": "singular"}]}}, "modules": {}}}}`
	env := func() *runtimeEnv {
		r := helpEnv(false, 100)
		r.runCmd = func(name string, args ...string) (string, error) { return dump, nil }
		return r
	}

	r := env()
	if code := run([]string{"help", "-j", "release"}, r.env); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, r.errOut.String())
	}
	var got struct{ Args []struct{ Name, Desc string } }
	if err := json.Unmarshal(r.out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Args) != 3 || got.Args[0].Name != "<env>" || got.Args[1].Name != "<tag>" || got.Args[1].Desc != "Release tag" || got.Args[2].Name != "<extra>" {
		t.Errorf("args = %+v, want env, tag, extra", got.Args)
	}

	r = env()
	if code := run([]string{"help", "-j", "tool", "bare"}, r.env); code != 0 || !strings.Contains(r.out.String(), `"name": "\u003cpath\u003e"`) {
		t.Errorf("module recipe: exit %d\n%s", code, r.out.String())
	}

	for _, args := range [][]string{{"help", "--expanded"}, {"help", "tool"}, {"help", "release"}} {
		r = env()
		if code := run(args, r.env); code != 0 {
			t.Fatalf("%v: exit %d, stderr %q", args, code, r.errOut.String())
		}
		for _, l := range strings.Split(r.out.String(), "\n") {
			if strings.HasSuffix(l, "#") || strings.HasSuffix(l, "# ") || (l != strings.TrimRight(l, " ") && strings.TrimSpace(l) != "") {
				t.Errorf("%v: a line ends in a hash or in spaces: %q", args, l)
			}
		}
	}
}

// The inventory is always there with --expanded, also when it is empty, and
// it has the documented extras of the core group, marked.
func TestHelpJSONInventoryEdges(t *testing.T) {
	dir := gitProject(t)
	writeFile(t, filepath.Join(dir, "config", "commands.json"), `{"groups":{"core":{"commands":{},"extra":{"aaa":{"desc":"Not runnable","tags":["destructive"]}}}}}`)
	t.Chdir(dir)
	r := helpEnv(false, 80)
	if code := run([]string{"help", "-j", "--expanded"}, r.env); code != 0 || !strings.Contains(r.out.String(), `"path": "aaa"`) || !strings.Contains(r.out.String(), `"extra": true`) {
		t.Errorf("core extra: exit %d\n%s", code, r.out.String())
	}

	writeFile(t, filepath.Join(dir, "config", "commands.json"), `{"groups":{}}`)
	r = helpEnv(false, 80)
	if code := run([]string{"help", "-j", "--expanded"}, r.env); code != 0 || !strings.Contains(r.out.String(), `"commands": []`) {
		t.Errorf("empty project: exit %d\n%s", code, r.out.String())
	}
}

// The summary invites the reader to run a group with no arguments. That is
// only true where every module has a default recipe: without one, just runs
// the first recipe of the module, so the hint is left out. It is also left
// out when the dump does not say what a bare call runs.
func TestHelpNativeHintNeedsADispatcher(t *testing.T) {
	t.Chdir(gitProject(t))
	module := func(first string) string {
		if first != "" {
			first = `"first": "` + first + `", `
		}
		return `{` + first + `"modules": {}, "recipes": {
		  "_default": {"private": true, "doc": null, "attributes": [], "parameters": []},
		  "clean": {"private": false, "doc": "Remove output", "attributes": [], "parameters": []}}}`
	}
	summary := func(modules string) string {
		r := helpEnv(false, 100)
		r.runCmd = func(name string, args ...string) (string, error) {
			return `{"recipes": {}, "modules": {` + modules + `}}`, nil
		}
		if code := run([]string{"help"}, r.env); code != 0 {
			t.Fatalf("exit %d, stderr %q", code, r.errOut.String())
		}
		return r.out.String()
	}
	const hint = "Commands  (run without args to see sub-commands)"
	if out := summary(`"tool": ` + module("_default")); !strings.Contains(out, hint) {
		t.Errorf("a module with a dispatcher keeps the hint:\n%s", out)
	}
	for name, modules := range map[string]string{
		"a bare call runs clean":       `"tool": ` + module("clean"),
		"one of two has no dispatcher": `"a": ` + module("_default") + `, "b": ` + module("clean"),
		"the dump does not say":        `"tool": ` + module(""),
	} {
		if out := summary(modules); strings.Contains(out, "run without args") || !strings.Contains(out, "  Commands\n") {
			t.Errorf("%s, so there is no hint:\n%s", name, out)
		}
	}
}

// The overlay is generated from the justfile, so the same rule holds for it.
func TestHelpOverlayHintNeedsADispatcher(t *testing.T) {
	dir := gitProject(t)
	writeFile(t, filepath.Join(dir, ".just-local-rich.json"), `{"groups":{"tool":{"commands":{"clean":{"desc":"Remove output"}}}}}`)
	t.Chdir(dir)
	summary := func(first string) string {
		r := helpEnv(false, 100)
		r.runCmd = func(name string, args ...string) (string, error) {
			return `{"recipes": {}, "modules": {"tool": {"first": "` + first + `", "modules": {}, "recipes": {}}}}`, nil
		}
		if code := run([]string{"help"}, r.env); code != 0 {
			t.Fatalf("exit %d, stderr %q", code, r.errOut.String())
		}
		return r.out.String()
	}
	if out := summary("_default"); !strings.Contains(out, "run without args to see sub-commands") {
		t.Errorf("a module with a dispatcher keeps the hint:\n%s", out)
	}
	if out := summary("clean"); strings.Contains(out, "run without args") {
		t.Errorf("a bare call would run clean, so there is no hint:\n%s", out)
	}
}
