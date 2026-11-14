package help

import (
	"encoding/json"
	"strings"
	"testing"
)

const detailRegistry = `{
  "cli": "dm",
  "groups": {
    "core": { "commands": { "plain": {"desc": "No detail here"} } },
    "ops": {
      "label": "Operations",
      "seeAlso": ["core"],
      "commands": {
        "deploy": {
          "desc": "Deploy the report",
          "detail": "\n\nBuilds the day directory, runs the doctor, uploads and reindexes it on the cloud host, and sends a roll-up.\nSecond line.\n\n  indented   text\n\n",
          "example": "… --dry-run",
          "args": [ {"name": "[--dry-run]", "desc": "Print the plan"} ]
        },
        "long-word": { "desc": "Long word", "detail": "see https://example.invalid/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa now" },
        "dup": { "desc": "Dup one", "detail": "first" }
      },
      "extra": { "hidden": { "desc": "Documented only", "detail": "Extra entries have detail too." } }
    },
    "dev": { "label": "Dev", "commands": { "dup": { "desc": "Dup two", "detail": "second" } } }
  }
}`

func detailView(t *testing.T, name string, o Options) string {
	t.Helper()
	p := load(t, detailRegistry)
	_, rows := p.Lookup(strings.Fields(name)...)
	if len(rows) != 1 {
		t.Fatalf("lookup %q: %d rows", name, len(rows))
	}
	g := p.GroupByKey(rows[0].Group)
	return CommandView(p, g, g.FindCommand(rows[0].Command), o)
}

func TestZoomShowsDetailAfterOptions(t *testing.T) {
	out := detailView(t, "ops deploy", Options{Width: 60})
	order := []string{"dm ops deploy", "Deploy the report", "Examples", "Options", "Details", "Builds the day directory", "See also"}
	last := -1
	for _, want := range order {
		i := strings.Index(out, want)
		if i < 0 || i < last {
			t.Fatalf("%q missing or out of order in:\n%s", want, out)
		}
		last = i
	}
}

func TestZoomDetailLayout(t *testing.T) {
	out := detailView(t, "ops deploy", Options{Width: 60})
	_, section, _ := strings.Cut(out, "  Details\n")
	section, _, _ = strings.Cut(section, "\n  See also")
	want := "    Builds the day directory, runs the doctor, uploads and\n" +
		"    reindexes it on the cloud host, and sends a roll-up.\n" +
		"    Second line.\n" +
		"\n" +
		"    indented text\n"
	if !strings.HasPrefix(section, want) {
		t.Errorf("details section =\n%q\nwant it to start with\n%q", section, want)
	}
	for _, l := range strings.Split(out, "\n") {
		if l != strings.TrimRight(l, " ") {
			t.Errorf("trailing spaces in %q", l)
		}
		if len([]rune(l)) > 60 {
			t.Errorf("line over the width: %q", l)
		}
	}
}

func TestZoomDetailTrimsNewlinesAndKeepsOneBlankLineBefore(t *testing.T) {
	out := detailView(t, "ops deploy", Options{Width: 60})
	if strings.Contains(out, "Details\n\n") || strings.Contains(out, "\n\n\n") {
		t.Errorf("a blank line starts the section or two follow each other:\n%q", out)
	}
	p := load(t, `{"groups": {"core": {"commands": {"a": {"desc": "A", "detail": "\r\n\r\ntext\r\n"}}}}}`)
	g := p.GroupByKey("core")
	got := CommandView(p, g, g.FindCommand("a"), Options{})
	if !strings.Contains(got, "  Details\n    text\n") {
		t.Errorf("CRLF detail not cleaned:\n%q", got)
	}
}

func TestZoomDetailCutsAWordLongerThanTheLine(t *testing.T) {
	out := detailView(t, "ops long-word", Options{Width: 40})
	for _, l := range strings.Split(out, "\n") {
		if len([]rune(l)) > 40 {
			t.Errorf("line over the width: %q", l)
		}
	}
	if !strings.Contains(out, "now") {
		t.Errorf("text after the long word is lost:\n%s", out)
	}
	if got := strings.Count(out, "a"); got < 100 {
		t.Errorf("characters of the long word were lost: %d a's in\n%s", got, out)
	}
}

func TestZoomWithoutDetailIsUnchanged(t *testing.T) {
	out := detailView(t, "plain", Options{Width: 60})
	if strings.Contains(out, "Details") {
		t.Errorf("a command without detail shows a Details section:\n%s", out)
	}
	p := load(t, `{"groups": {"core": {"commands": {"a": {"desc": "A", "detail": "  \n\n"}}}}}`)
	g := p.GroupByKey("core")
	if strings.Contains(CommandView(p, g, g.FindCommand("a"), Options{}), "Details") {
		t.Error("a detail of only blanks shows a Details section")
	}
}

func TestZoomDetailWorksForExtraEntries(t *testing.T) {
	out := detailView(t, "hidden", Options{Width: 60})
	if !strings.Contains(out, "Extra entries have detail too.") {
		t.Errorf("extra entry lacks its detail:\n%s", out)
	}
}

func TestZoomDetailColorIsOnlyOnTheHeading(t *testing.T) {
	plain := detailView(t, "ops deploy", Options{Width: 60})
	colored := detailView(t, "ops deploy", Options{Width: 60, Color: true})
	if got := ansi.ReplaceAllString(colored, ""); got != plain {
		t.Errorf("colored view differs from the plain one once the escapes are gone:\n%q\n%q", got, plain)
	}
	if !strings.Contains(colored, "\x1b[1mDetails\x1b[0m") {
		t.Errorf("the Details heading is not bold in:\n%q", colored)
	}
}

func TestDetailIsOnlyInTheZoomJSONAndNotInListings(t *testing.T) {
	p := load(t, detailRegistry)
	g := p.Group("ops")
	n := toMap(t, CommandJSON(p, g, g.FindCommand("deploy")))
	if d, _ := n["detail"].(string); !strings.Contains(d, "Builds the day directory") {
		t.Errorf("zoom JSON lacks detail: %v", n)
	}
	core := toMap(t, CommandJSON(p, p.Core, p.Core.FindCommand("plain")))
	if _, ok := core["detail"]; ok {
		t.Errorf("a command without detail has a detail key: %v", core)
	}
	list, _ := json.Marshal(GroupJSON(p, g))
	if strings.Contains(string(list), "Builds the day directory") || strings.Contains(string(list), `"detail"`) {
		t.Errorf("the group listing carries detail: %s", list)
	}
	for _, view := range []string{Summary(p, Options{Width: 120}), Expanded(p, Options{Width: 120}), GroupView(p, g, Options{Width: 120})} {
		if strings.Contains(view, "Builds the day directory") {
			t.Errorf("a listing view carries detail:\n%s", view)
		}
	}
}

func TestZoomDetailAtAWidthBelowTheMinimum(t *testing.T) {
	out := detailView(t, "ops deploy", Options{Width: 10})
	for _, l := range strings.Split(out, "\n") {
		if len([]rune(l)) > 40 {
			t.Errorf("line over the 40-column minimum: %q", l)
		}
	}
}

func TestDetailIgnoresBlankLinesAtTheEdges(t *testing.T) {
	p := load(t, `{"groups": {"core": {"commands": {"a": {"desc": "A", "detail": " \nbody\n \t\n"}}}}}`)
	g := p.GroupByKey("core")
	got := CommandView(p, g, g.FindCommand("a"), Options{})
	if !strings.Contains(got, "  Details\n    body\n\n") || strings.Contains(got, "\n\n\n") {
		t.Errorf("blank-only lines at the edges were kept:\n%q", got)
	}
}

func TestDetailOfTheWrongTypeIsIgnored(t *testing.T) {
	for _, v := range []string{`3`, `null`, `[]`, `{}`, `""`} {
		p := load(t, `{"groups": {"core": {"commands": {"a": {"desc": "A", "detail": `+v+`}}}}}`)
		g := p.GroupByKey("core")
		if c := g.FindCommand("a"); c.Detail != "" {
			t.Errorf("detail %s: Detail = %q", v, c.Detail)
		}
		if strings.Contains(CommandView(p, g, g.FindCommand("a"), Options{}), "Details") {
			t.Errorf("detail %s shows a section", v)
		}
	}
}

func TestExtraEntryDetailIsInTheJSON(t *testing.T) {
	p := load(t, detailRegistry)
	g := p.Group("ops")
	n := toMap(t, CommandJSON(p, g, g.FindCommand("hidden")))
	if n["detail"] != "Extra entries have detail too." {
		t.Errorf("extra entry JSON = %v", n)
	}
}
