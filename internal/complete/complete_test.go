package complete

import (
	"strings"
	"testing"

	"github.com/igorlino/justz/internal/help"
	"github.com/igorlino/justz/internal/orderedjson"
)

// registry mirrors the shape the old completion tests used.
const registry = `{
  "cli": "dm",
  "groups": {
    "core": { "commands": { "build": {"desc": "Build it"}, "ci": {"desc": "Pipeline"} } },
    "snyk": { "label": "Snyk scanning", "commands": {
        "scan": { "desc": "Run a scan", "args": [
            {"name": "-v, --severity <level>", "desc": "Minimum severity", "values": ["low", "medium", "high", "critical"]},
            {"name": "-s, --silent", "desc": "Quiet output"},
            {"name": "-f, --file <path>", "desc": "Config file"} ] },
        "snap": { "desc": "Take a snapshot" } },
      "extra": { "hidden": {"desc": "Documented only"} } },
    "dbx":  { "label": "Database tools", "commands": {
        "doc": { "desc": "Show docs", "args": [
            {"name": "<topic>", "desc": "Topic", "values": ["schema", "indexes"]},
            {"name": "-l, --long", "desc": "Long form"} ] } } },
    "demo":  { "label": "Demo", "commands": { "greet": {"desc": "Greet from demo"} } },
    "other": { "label": "Other", "commands": { "greet": {"desc": "Greet from other"} } }
  }
}`

func testEnv(t *testing.T) Env {
	t.Helper()
	doc, err := orderedjson.Parse([]byte(registry))
	if err != nil {
		t.Fatal(err)
	}
	proj, err := help.FromRegistry(doc, "p")
	if err != nil {
		t.Fatal(err)
	}
	return Env{
		Project:     func() *help.Project { return proj },
		RootRecipes: func() map[string]bool { return map[string]bool{"rootrecipe": true} },
		Shims:       func() []string { return []string{"tz", "tzh", "jz"} },
	}
}

func values(r Result) string {
	var v []string
	for _, c := range r.Candidates {
		v = append(v, c.Value)
	}
	return strings.Join(v, " ")
}

func desc(r Result, value string) string {
	for _, c := range r.Candidates {
		if c.Value == value {
			return c.Desc
		}
	}
	return "<missing>"
}

func TestRunTopLevel(t *testing.T) {
	e := testEnv(t)
	r := Shim(false, []string{""}, e)
	// Core commands first, then the groups sorted A to Z. No bare names yet.
	if got := values(r); got != "build ci dbx demo other snyk" {
		t.Errorf("top level = %q", got)
	}
	if desc(r, "snyk") != "Snyk scanning" || desc(r, "build") != "Build it" {
		t.Errorf("descriptions: %+v", r.Candidates)
	}
}

func TestRunGroupCommands(t *testing.T) {
	e := testEnv(t)
	r := Shim(false, []string{"snyk", ""}, e)
	if got := values(r); got != "scan snap" {
		t.Errorf("group commands = %q (documented extras are not runnable)", got)
	}
	if desc(r, "scan") != "Run a scan" {
		t.Errorf("descriptions: %+v", r.Candidates)
	}
	if got := values(Shim(false, []string{"snyk", "sc"}, e)); got != "scan" {
		t.Errorf("prefix filter = %q", got)
	}
}

func TestRunBareNamesCarryTheirGroup(t *testing.T) {
	e := testEnv(t)
	r := Shim(false, []string{"sc"}, e)
	if got := values(r); got != "scan" || desc(r, "scan") != "snyk: Run a scan" {
		t.Errorf("bare names = %q, %+v", got, r.Candidates)
	}
	// A name in two groups is offered once per group, so the clash is visible.
	r = Shim(false, []string{"gr"}, e)
	if got := values(r); got != "greet greet" {
		t.Errorf("ambiguous bare name = %q", got)
	}
	// Only runnable commands: no extras, no core commands as bare names.
	r = Shim(false, []string{"h"}, e)
	if strings.Contains(values(r), "hidden") {
		t.Errorf("a documented extra is not a bare name: %q", values(r))
	}
}

func TestRunOptionsOfACommand(t *testing.T) {
	e := testEnv(t)
	want := "-f --file -s --silent -v --severity"
	for _, words := range [][]string{
		{"snyk", "scan", ""}, // by group and command
		{"scan", ""},         // by bare name, like just snyk scan
	} {
		r := Shim(false, words, e)
		if got := values(r); got != want {
			t.Errorf("%v options = %q, want %q (short and long forms, canonical order)", words, got, want)
		}
		if desc(r, "--severity") != "Minimum severity" {
			t.Errorf("option description: %+v", r.Candidates)
		}
	}
}

func TestRunEnumValuesAfterAFlag(t *testing.T) {
	e := testEnv(t)
	// The values keep the order the author wrote.
	for _, flag := range []string{"-v", "--severity"} {
		r := Shim(false, []string{"snyk", "scan", flag, ""}, e)
		if got := values(r); got != "low medium high critical" || r.Files {
			t.Errorf("%s values = %q (files %v)", flag, got, r.Files)
		}
	}
	if got := values(Shim(false, []string{"scan", "-v", "h"}, e)); got != "high" {
		t.Errorf("values are filtered by the prefix: %q", got)
	}
}

func TestRunBooleanAndFreeTextFlags(t *testing.T) {
	e := testEnv(t)
	r := Shim(false, []string{"snyk", "scan", "-s", ""}, e)
	if got := values(r); got != "-f --file -v --severity" || r.Files {
		t.Errorf("after a boolean flag: the rest of the options, not enum values or files: %q files=%v", got, r.Files)
	}
	r = Shim(false, []string{"snyk", "scan", "-f", ""}, e)
	if !r.Files || len(r.Candidates) != 0 {
		t.Errorf("a free-text value is a file: %+v", r)
	}
}

func TestRunPositionalValuesComeFirst(t *testing.T) {
	e := testEnv(t)
	r := Shim(false, []string{"dbx", "doc", ""}, e)
	if got := values(r); got != "schema indexes -l --long" {
		t.Errorf("positional values then flags = %q", got)
	}
	r = Shim(false, []string{"dbx", "doc", "-"}, e)
	if got := values(r); got != "-l --long" {
		t.Errorf("a dash asks for options only: %q", got)
	}
}

func TestAFlagWithValuesTakesAValueEvenWithoutAPlaceholder(t *testing.T) {
	doc, err := orderedjson.Parse([]byte(`{"groups":{"g":{"label":"G","commands":{"run":{"desc":"d","args":[{"name":"--level","desc":"How strict","values":["low","high"]},{"name":"--fast","desc":"Fast"}]}}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	proj, _ := help.FromRegistry(doc, "p")
	e := Env{Project: func() *help.Project { return proj }}
	if got := values(Shim(false, []string{"g", "run", "--level", ""}, e)); got != "low high" {
		t.Errorf("values after a flag that lists them = %q", got)
	}
}

func TestRunCoreCommandOptions(t *testing.T) {
	e := testEnv(t)
	if got := values(Shim(false, []string{"build", ""}, e)); got != "" {
		t.Errorf("a core command without options: %q", got)
	}
}

func TestRunUnknownWordsOfferNothing(t *testing.T) {
	e := testEnv(t)
	for _, words := range [][]string{{"nope", ""}, {"snyk", "nope", ""}, {"rootrecipe", ""}} {
		if r := Shim(false, words, e); len(r.Candidates) != 0 || r.Files {
			t.Errorf("%v: %+v", words, r)
		}
	}
}

func TestHelpShim(t *testing.T) {
	e := testEnv(t)
	// The core commands, then the groups; no bare names until something is typed.
	if got := values(Shim(true, []string{""}, e)); got != "build ci dbx demo other snyk" {
		t.Errorf("jzh <TAB> = %q", got)
	}
	// Once typed, bare names join in.
	got := values(Shim(true, []string{"sn"}, e))
	if got != "snyk snap" {
		t.Errorf("jzh sn<TAB> = %q", got)
	}
	if got := values(Shim(true, []string{"snyk", ""}, e)); got != "scan snap" {
		t.Errorf("jzh snyk <TAB> = %q", got)
	}
	// Flags, in this order.
	r := Shim(true, []string{"-"}, e)
	if got := values(r); got != "-b -j -w" {
		t.Errorf("jzh -<TAB> = %q", got)
	}
	// A flag does not count as a word of the lookup.
	if got := values(Shim(true, []string{"-j", "snyk", ""}, e)); got != "scan snap" {
		t.Errorf("jzh -j snyk <TAB> = %q", got)
	}
}

func TestShimWithoutAProjectOffersFiles(t *testing.T) {
	e := Env{Project: func() *help.Project { return nil }}
	if r := Shim(false, []string{""}, e); !r.Files {
		t.Error("with no project, fall back to file names")
	}
	if r := Shim(false, []string{"-"}, e); r.Files {
		t.Error("a dash is not a file")
	}
}

func TestJustzCommands(t *testing.T) {
	e := testEnv(t)
	got := values(Justz([]string{""}, e))
	for _, want := range []string{"assess", "check", "completions", "fmt", "help", "setup", "shim", "--version"} {
		if !strings.Contains(" "+got+" ", " "+want+" ") {
			t.Errorf("top level should offer %q: %s", want, got)
		}
	}
	if got := values(Justz([]string{"ch"}, e)); got != "check" {
		t.Errorf("prefix = %q", got)
	}
	if got := values(Justz([]string{"shim", ""}, e)); got != "add doctor list remove" {
		t.Errorf("shim subcommands = %q", got)
	}
	if got := values(Justz([]string{"completions", ""}, e)); got != "bash zsh" {
		t.Errorf("shells = %q", got)
	}
}

func TestJustzFlagsAndArguments(t *testing.T) {
	e := testEnv(t)
	tests := []struct {
		words []string
		want  string
		files bool
	}{
		{[]string{"check", "-"}, "--json --strict", false},
		{[]string{"fmt", "-"}, "--check", false},
		{[]string{"fmt", ""}, "", true},
		{[]string{"shim", "add", "-"}, "--dry-run --force --json --name --replace --soft --suffix", false},
		{[]string{"shim", "add", ""}, "", true},
		{[]string{"shim", "remove", ""}, "tz tzh jz", false},
		{[]string{"shim", "list", "-"}, "--json", false},
		{[]string{"help", "--b"}, "--browse", false},
		{[]string{"help", ""}, "build ci dbx demo other snyk", false},
		{[]string{"help", "snyk", ""}, "scan snap", false},
		{[]string{"resolve", "sc"}, "scan", false},
		{[]string{"setup", "-"}, "--dry-run --json", false},
	}
	for _, tt := range tests {
		r := Justz(tt.words, e)
		if got := values(r); got != tt.want || r.Files != tt.files {
			t.Errorf("%v = %q (files %v), want %q (files %v)", tt.words, got, r.Files, tt.want, tt.files)
		}
	}
}

func TestFlagNames(t *testing.T) {
	for in, want := range map[string]string{
		"-w, --watch":            "-w --watch",
		"-k <pattern>":           "-k",
		"--level <x>":            "--level",
		"<file>":                 "",
		"*files":                 "",
		"-v, --severity <level>": "-v --severity",
	} {
		if got := strings.Join(flagNames(in), " "); got != want {
			t.Errorf("flagNames(%q) = %q, want %q", in, got, want)
		}
	}
	if !takesValue("-k <pattern>") || takesValue("-w, --watch") || takesValue("<file>") {
		t.Error("takesValue is wrong")
	}
}
