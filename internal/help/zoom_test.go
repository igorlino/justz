package help

import (
	"strings"
	"testing"
)

// lookupRegistry mirrors the shape the existing lookup tests used: a command
// found in two groups, a core command, and groups that share a prefix.
const lookupRegistry = `{
  "cli": "dm",
  "groups": {
    "core": { "commands": { "features": {"desc": "List features"}, "ci": {"desc": "Pipeline"} } },
    "tool": { "label": "Tools", "commands": { "git": {"desc": "Git helper"}, "doctor": {"desc": "Check tools"} },
              "extra": { "hidden": {"desc": "Documented only"} } },
    "sbx":  { "label": "Sandboxes", "commands": { "doctor": {"desc": "Check sandboxes"} } },
    "ai":   { "label": "AI agents", "commands": { "claude": {"desc": "Claude"}, "codex": {"desc": "Codex"} } }
  }
}`

func TestLookup(t *testing.T) {
	p := load(t, lookupRegistry)
	tests := []struct {
		name  string
		terms []string
		want  Outcome
		paths []string
	}{
		{"exact command", []string{"git"}, Found, []string{"tool git"}},
		{"unique prefix", []string{"gi"}, Found, []string{"tool git"}},
		{"two words", []string{"tool", "git"}, Found, []string{"tool git"}},
		{"group exact, command prefix", []string{"ai", "cla"}, Found, []string{"ai claude"}},
		{"group prefix, command prefix", []string{"to", "gi"}, Found, []string{"tool git"}},
		{"core command", []string{"features"}, Found, []string{"features"}},
		{"core as a group word", []string{"core", "ci"}, Found, []string{"ci"}},
		{"exact group wins", []string{"tool"}, Found, []string{"tool"}},
		{"ambiguous command", []string{"doctor"}, Ambiguous, []string{"sbx doctor", "tool doctor"}},
		{"documented extra is findable", []string{"hidden"}, Found, []string{"tool hidden"}},
		{"no match", []string{"zzz-nothing"}, NotFound, nil},
		{"unknown group", []string{"nogroup", "git"}, NotFound, nil},
		{"ambiguous group prefix", []string{"s", "doctor"}, Found, []string{"sbx doctor"}},
		{"command missing in group", []string{"tool", "nope"}, NotFound, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, rows := p.Lookup(tt.terms...)
			if got != tt.want {
				t.Fatalf("outcome = %v, want %v (rows %v)", got, tt.want, rows)
			}
			var paths []string
			for _, r := range rows {
				paths = append(paths, r.Path())
			}
			if strings.Join(paths, "|") != strings.Join(tt.paths, "|") {
				t.Errorf("paths = %v, want %v", paths, tt.paths)
			}
		})
	}
}

func TestLookupAmbiguousGroupPrefixIsNotFound(t *testing.T) {
	p := load(t, `{"groups":{"alpha":{"label":"A","commands":{"x":{}}},"alps":{"label":"B","commands":{"x":{}}}}}`)
	if got, _ := p.Lookup("al", "x"); got != NotFound {
		t.Errorf("a prefix shared by two groups must not resolve, got %v", got)
	}
}

func TestZoomViewsGolden(t *testing.T) {
	p := loadDemo(t)
	o := Options{Width: 100}
	build := p.Group("build")
	tool := p.Group("tool")
	golden(t, "zoom-group-build", GroupView(p, build, o))
	golden(t, "zoom-group-tool", GroupView(p, tool, o))
	golden(t, "zoom-command-build-docker", CommandView(p, build, build.FindCommand("docker"), o))
	golden(t, "zoom-command-core-test", CommandView(p, p.Core, p.Core.FindCommand("test"), o))
	golden(t, "overview", Overview(p, o))
}

func TestWrap(t *testing.T) {
	tests := []struct {
		width int
		in    string
		want  []string
	}{
		{10, "short", []string{"short"}},
		{10, "one two three four", []string{"one two", "three four"}},
		{5, "abcdefghij", []string{"abcde", "fghij"}},
		{10, "line one\nline two", []string{"line one", "line two"}},
		{10, "", []string{""}},
		{10, "  spaced   out  ", []string{"spaced out"}},
	}
	for _, tt := range tests {
		got := wrap(tt.width, tt.in)
		if strings.Join(got, "|") != strings.Join(tt.want, "|") {
			t.Errorf("wrap(%d, %q) = %q, want %q", tt.width, tt.in, got, tt.want)
		}
	}
}

func TestColorizeCmd(t *testing.T) {
	on := style{on: true}
	const g, c, y, r = "\x1b[32m", "\x1b[36m", "\x1b[33m", "\x1b[0m"
	tests := []struct{ name, in, want string }{
		{"plain flags", "dm run -v --level", g + "dm run " + c + "-v" + g + " " + c + "--level" + g + r},
		{"flag with value colors only the name", "--level=high", g + c + "--level" + g + "=high" + r},
		{"placeholder", "dm run <file>", g + "dm run " + y + "<file>" + g + r},
		{"quoted text stays plain", `dm say "hello -x"`, g + `dm say "hello -x"` + r},
		{"negative number is not a flag", "dm add -5", g + "dm add -5" + r},
		{"redirect is not a placeholder", "dm run > out.txt", g + "dm run > out.txt" + r},
		{"tab separates words", "dm\t-v", g + "dm\t" + c + "-v" + g + r},
	}
	for _, tt := range tests {
		if got := on.colorizeCmd(tt.in); got != tt.want {
			t.Errorf("%s: colorizeCmd(%q)\n got %q\nwant %q", tt.name, tt.in, got, tt.want)
		}
	}
	if got := (style{}).colorizeCmd("dm -v <x>"); got != "dm -v <x>" {
		t.Errorf("no color without a palette: %q", got)
	}
}

func TestWithCLI(t *testing.T) {
	for in, want := range map[string]string{
		"just build docker":        "dm build docker",
		"echo x; just y":           "echo x; dm y",
		"justify this":             "justify this",
		"just":                     "just",
		"rm -rf; just a && just b": "rm -rf; dm a && dm b",
	} {
		if got := withCLI("dm", in); got != in && got != want {
			t.Errorf("withCLI(%q) = %q, want %q", in, got, want)
		}
		if got := withCLI("dm", in); got != want {
			t.Errorf("withCLI(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCommandExamplesFilterAndFallback(t *testing.T) {
	p := load(t, `{"cli":"dm","groups":{
		"build":{"label":"B","commands":{"docker":{"desc":"d","example":"…  --push"},"release":{"desc":"r"}},
		  "examples":[{"cmd":"just build release --level high","desc":"strict"},{"cmd":"just build docker","desc":"plain"},{"cmd":"just build docker --push"}]}}}`)
	g := p.Group("build")
	got := commandExamples(p, g, g.FindCommand("docker"), "build docker")
	if len(got) != 2 || got[0].Cmd != "dm build docker" || got[1].Cmd != "dm build docker --push" {
		t.Errorf("examples = %+v (want only this command's, sorted, with the prefix)", got)
	}
	// No group example mentions release docker's sibling; fall back to its own example.
	fb := commandExamples(p, g, &Command{Name: "x", Example: "… -w"}, "build x")
	if len(fb) != 1 || fb[0].Cmd != "dm build x -w" {
		t.Errorf("fallback example = %+v", fb)
	}
}

func TestFindCommandIncludesExtras(t *testing.T) {
	p := load(t, lookupRegistry)
	g := p.Group("tool")
	if g.FindCommand("hidden") == nil || g.FindCommand("git") == nil || g.FindCommand("nope") != nil {
		t.Error("FindCommand should find commands and extras only")
	}
	if p.GroupByKey("core") == nil || p.GroupByKey("tool") == nil || p.GroupByKey("zzz") != nil {
		t.Error("GroupByKey should find core and other groups")
	}
}

func TestColorsDoNotChangeTheText(t *testing.T) {
	p := loadDemo(t)
	for name, f := range map[string]func(Options) string{
		"group": func(o Options) string { return GroupView(p, p.Group("build"), o) },
		"command": func(o Options) string {
			return CommandView(p, p.Group("build"), p.Group("build").FindCommand("docker"), o)
		},
		"overview": func(o Options) string { return Overview(p, o) },
	} {
		if got := ansi.ReplaceAllString(f(Options{Color: true, Width: 100}), ""); got != f(Options{Width: 100}) {
			t.Errorf("%s: colors must only add escape codes", name)
		}
	}
}

func TestCommandViewWrapsLongText(t *testing.T) {
	p := load(t, `{"cli":"dm","groups":{"g":{"label":"G","commands":{"c":{"desc":"`+strings.Repeat("word ", 40)+`","args":[{"name":"--flag","desc":"`+strings.Repeat("text ", 30)+`"}]}}}}}`)
	out := CommandView(p, p.Group("g"), p.Group("g").FindCommand("c"), Options{Width: 60})
	for _, line := range strings.Split(out, "\n") {
		if n := len([]rune(line)); n > 60 {
			t.Errorf("a line is %d columns wide, over the 60 allowed: %q", n, line)
		}
	}
}
