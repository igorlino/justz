package help

import (
	"flag"
	"os"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/igorlino/justz/internal/orderedjson"
)

var update = flag.Bool("update", false, "rewrite the golden files")

const demoRegistry = "../../testdata/help-reference/demo/config/commands.json"

func load(t *testing.T, json string) *Project {
	t.Helper()
	doc, err := orderedjson.Parse([]byte(json))
	if err != nil {
		t.Fatal(err)
	}
	p, err := FromRegistry(doc, "fallback")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func loadDemo(t *testing.T) *Project {
	t.Helper()
	data, err := os.ReadFile(demoRegistry)
	if err != nil {
		t.Fatal(err)
	}
	return load(t, string(data))
}

func golden(t *testing.T, name, got string) {
	t.Helper()
	path := "testdata/" + name + ".txt"
	if *update {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Errorf("%s differs from its golden file:\n--- got\n%s\n--- want\n%s", name, got, want)
	}
}

// The golden files are reviewed against the reference output of the old
// renderer (testdata/help-reference). The intended differences: no synthetic
// "help" row in the core list, and the footer and tip name justz help.
func TestSummaryAndExpandedGolden(t *testing.T) {
	p := loadDemo(t)
	golden(t, "summary", Summary(p, Options{Width: 100}))
	golden(t, "expanded", Expanded(p, Options{Width: 100}))
}

var ansi = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func TestColorOnlyWhenAsked(t *testing.T) {
	p := loadDemo(t)
	plain := Summary(p, Options{Color: false})
	if ansi.MatchString(plain) {
		t.Errorf("plain output contains ANSI codes:\n%q", plain)
	}
	colored := Summary(p, Options{Color: true})
	if !ansi.MatchString(colored) {
		t.Error("colored output should contain ANSI codes")
	}
	if got := ansi.ReplaceAllString(colored, ""); got != plain {
		t.Errorf("stripping the colors should give the plain output:\n%q\n%q", got, plain)
	}
	// Expanded too.
	if got := ansi.ReplaceAllString(Expanded(p, Options{Color: true}), ""); got != Expanded(p, Options{}) {
		t.Error("expanded view: colors must only add escape codes")
	}
}

func TestCanonicalOrderDoesNotDependOnTheFile(t *testing.T) {
	a := load(t, `{"groups":{"zeta":{"label":"Z","commands":{"b":{"desc":"b"},"a":{"desc":"a"}}},"core":{"commands":{"y":{"desc":"y"},"x":{"desc":"x"}}},"alpha":{"label":"A","commands":{}}}}`)
	b := load(t, `{"groups":{"alpha":{"label":"A","commands":{}},"core":{"commands":{"x":{"desc":"x"},"y":{"desc":"y"}}},"zeta":{"label":"Z","commands":{"a":{"desc":"a"},"b":{"desc":"b"}}}}}`)
	if Summary(a, Options{}) != Summary(b, Options{}) || Expanded(a, Options{}) != Expanded(b, Options{}) {
		t.Error("the views must be the same however the registry file is ordered")
	}
	out := Summary(a, Options{})
	if strings.Index(out, "x ") > strings.Index(out, "y ") || strings.Index(out, "alpha") > strings.Index(out, "zeta") {
		t.Errorf("core commands and groups should be sorted:\n%s", out)
	}
}

func TestWithoutTiersGroupsFormOneSection(t *testing.T) {
	p := load(t, `{"name":"proj","groups":{"b":{"label":"B group"},"a":{"label":"A group"}}}`)
	out := Summary(p, Options{})
	if !strings.Contains(out, "Commands  (run without args to see sub-commands)") {
		t.Errorf("want a default tier:\n%s", out)
	}
	if strings.Index(out, "A group") > strings.Index(out, "B group") {
		t.Errorf("default tier should be sorted:\n%s", out)
	}
	if !strings.Contains(out, "  proj\n") {
		t.Errorf("header should use the registry name:\n%s", out)
	}
}

func TestNameAndTaglineAndFallback(t *testing.T) {
	if out := Summary(load(t, `{"groups":{}}`), Options{}); !strings.Contains(out, "\n  fallback\n") {
		t.Errorf("fallback name expected:\n%s", out)
	}
	out := Summary(load(t, `{"title":"My Tool","tagline":"does things","groups":{}}`), Options{})
	if !strings.Contains(out, "My Tool  — does things") {
		t.Errorf("title and tagline expected:\n%s", out)
	}
}

func TestWideLayoutShowsExamples(t *testing.T) {
	p := load(t, `{"groups":{"core":{"commands":{"test":{"desc":"Run tests","example":"just test -w"}}}}}`)
	narrow := Expanded(p, Options{Width: 80})
	wide := Expanded(p, Options{Width: 100})
	forced := Expanded(p, Options{Width: 80, Wide: true})
	if strings.Contains(narrow, "just test -w") {
		t.Errorf("no example column below %d columns:\n%s", WideAt, narrow)
	}
	if !strings.Contains(wide, "just test -w") || wide != forced {
		t.Errorf("the example column should show from %d columns, or with Wide:\n%s", WideAt, wide)
	}
}

func TestArgumentDescriptions(t *testing.T) {
	tests := []struct {
		arg  Arg
		want string
	}{
		{Arg{Desc: "How strict"}, "How strict"},
		{Arg{Desc: "How strict", Values: []string{"low", "high"}}, "How strict (low|high)"},
		{Arg{Desc: "One of low|high", Values: []string{"low", "high"}}, "One of low|high"},
		{Arg{Values: []string{"a", "b"}}, "(a|b)"},
		{Arg{}, ""},
	}
	for _, tt := range tests {
		if got := argDescription(tt.arg); got != tt.want {
			t.Errorf("argDescription(%+v) = %q, want %q", tt.arg, got, tt.want)
		}
	}
}

func TestPadCountsCharactersNotBytes(t *testing.T) {
	if got := pad("café", 6); got != "café  " {
		t.Errorf("pad = %q", got)
	}
	if got := pad("longer-than-width", 4); got != "longer-than-width" {
		t.Errorf("a long name must not be cut, got %q", got)
	}
}

func TestFromRegistryErrors(t *testing.T) {
	for _, in := range []string{`[]`, `"x"`, `{}`, `{"groups":[]}`} {
		doc, err := orderedjson.Parse([]byte(in))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := FromRegistry(doc, "x"); err == nil {
			t.Errorf("FromRegistry(%s) should fail", in)
		}
	}
}

const longDesc = "Install the global tz and s-*/sbx-* shims, zsh completion, missing AI agent CLIs, VS Code settings, and 7 bundled agent skills"

const wrapRegistry = `{"name":"proj","groups":{"core":{"label":"Core","commands":{"setup":{"desc":"` + longDesc + `","args":[{"name":"-i, --install","desc":"` + longDesc + `"}]}}}}}`

// A long description breaks at the terminal width and the lines after the first
// start under the description, not at the left edge.
func TestWrapKeepsContinuationLinesUnderTheDescription(t *testing.T) {
	p := load(t, wrapRegistry)
	out := Summary(p, Options{Width: 70, Wrap: true})
	const indent = 4 + compactName + 3
	lines := strings.Split(out, "\n")
	var rows []string
	for i, l := range lines {
		if strings.Contains(l, "setup") {
			rows = append(rows, lines[i:]...)
			break
		}
	}
	if len(rows) < 3 {
		t.Fatalf("want the description on several lines:\n%s", out)
	}
	for _, l := range rows {
		if l == "" {
			break
		}
		if utf8.RuneCountInString(l) > 70 {
			t.Errorf("line is wider than 70: %q", l)
		}
	}
	if !strings.HasPrefix(rows[1], strings.Repeat(" ", indent)) || strings.TrimSpace(rows[1]) == "" || rows[1][indent] == ' ' {
		t.Errorf("continuation line is not aligned under the description: %q", rows[1])
	}
}

func TestWrapIsOffByDefault(t *testing.T) {
	p := load(t, wrapRegistry)
	for _, v := range []string{Summary(p, Options{Width: 70}), Expanded(p, Options{Width: 70})} {
		if !strings.Contains(v, longDesc) {
			t.Errorf("without Wrap the description must stay on one line:\n%s", v)
		}
	}
}

func TestWrapAlsoAppliesToExpandedRowsAndArguments(t *testing.T) {
	p := load(t, wrapRegistry)
	out := Expanded(p, Options{Width: 70, Wrap: true})
	for _, l := range strings.Split(out, "\n") {
		if utf8.RuneCountInString(l) > 70 {
			t.Errorf("line is wider than 70: %q", l)
		}
	}
	if !strings.Contains(out, "\n"+strings.Repeat(" ", 6+fullName+1)) {
		t.Errorf("argument description continuation is not aligned:\n%s", out)
	}
}
