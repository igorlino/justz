package help

import (
	"encoding/json"
	"strings"
	"testing"
)

const exitsRegistry = `{
  "cli": "dm",
  "groups": {
    "ops": {
      "label": "Operations",
      "commands": {
        "check": {
          "desc": "Check the docs",
          "detail": "A detail.",
          "args": [ {"name": "--strict", "desc": "Warnings fail"} ],
          "exits": { "10": "a rare code with a long meaning that has to wrap onto a second line at the zoom width", "2": "usage error", "0": "no errors", "x": "not a code", "007": "leading zeros", "256": "too big", "1": "an error\u001b[31m" }
        },
        "plain": { "desc": "Plain" },
        "blank": { "desc": "Blank detail", "detail": "  \n " }
      }
    }
  }
}`

func exitsView(t *testing.T, name string, o Options) (string, *Project, *Group, *Command) {
	t.Helper()
	p := load(t, exitsRegistry)
	_, rows := p.Lookup(strings.Fields(name)...)
	if len(rows) != 1 {
		t.Fatalf("lookup %q: %d rows", name, len(rows))
	}
	g := p.GroupByKey(rows[0].Group)
	c := g.FindCommand(rows[0].Command)
	return CommandView(p, g, c, o), p, g, c
}

func TestZoomShowsExitCodesInNumericOrderAfterOptions(t *testing.T) {
	out, _, _, _ := exitsView(t, "ops check", Options{Width: 60})
	for _, want := range []string{"\n  Exit codes\n", "    0   no errors\n", "    1   an error\uFFFD[31m\n", "    2   usage error\n", "    10  a rare code"} {
		if !strings.Contains(out, want) {
			t.Errorf("zoom should contain %q:\n%s", want, out)
		}
	}
	last := -1
	for _, s := range []string{"Options", "Exit codes", "no errors", "an error", "usage error", "a rare code", "Details"} {
		i := strings.Index(out, s)
		if i < 0 || i < last {
			t.Fatalf("%q missing or out of order:\n%s", s, out)
		}
		last = i
	}
	for _, bad := range []string{"not a code", "leading zeros", "too big", "\x1b"} {
		if strings.Contains(out, bad) {
			t.Errorf("zoom shows %q:\n%q", bad, out)
		}
	}
	if !strings.Contains(out, "\n        second line") && !strings.Contains(out, "second line at") {
		t.Errorf("the long meaning should wrap:\n%s", out)
	}
}

func TestZoomExitsGoldenAndColors(t *testing.T) {
	out, _, _, _ := exitsView(t, "ops check", Options{Width: 60})
	golden(t, "zoom-command-exits", out)
	colored, _, _, _ := exitsView(t, "ops check", Options{Width: 60, Color: true})
	if got := ansi.ReplaceAllString(colored, ""); got != out {
		t.Errorf("colors must only add escape codes:\n%q\n%q", got, out)
	}
}

func TestZoomWithoutExitsIsUnchanged(t *testing.T) {
	out, _, _, _ := exitsView(t, "ops plain", Options{Width: 80})
	if strings.Contains(out, "Exit codes") || out != "\n  dm ops plain\n  "+strings.Repeat("─", 76)+"\n  Plain\n\n" {
		t.Errorf("zoom = %q", out)
	}
}

func TestCommandJSONHasExitsInNumericOrder(t *testing.T) {
	_, p, g, c := exitsView(t, "ops check", Options{})
	b, err := json.Marshal(CommandJSON(p, g, c))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"exits":{"0":"no errors","1":"an error\uFFFD[31m","2":"usage error","10":"a rare code`) && !strings.Contains(string(b), `"exits":{"0":"no errors","1":"an error`) {
		t.Errorf("json = %s", b)
	}
	if strings.Index(string(b), `"2":`) > strings.Index(string(b), `"10":`) {
		t.Errorf("2 must come before 10: %s", b)
	}
	var n struct {
		Exits map[string]string `json:"exits"`
	}
	if err := json.Unmarshal(b, &n); err != nil || len(n.Exits) != 4 || n.Exits["2"] != "usage error" {
		t.Errorf("exits = %v (%v)", n.Exits, err)
	}
	_, p, g, c = exitsView(t, "ops plain", Options{})
	if b, _ := json.Marshal(CommandJSON(p, g, c)); strings.Contains(string(b), "exits") {
		t.Errorf("exits must be left out when empty: %s", b)
	}
}

func TestGroupListingHasDetailFlag(t *testing.T) {
	p := load(t, exitsRegistry)
	g := p.GroupByKey("ops")
	b, _ := json.Marshal(GroupJSON(p, g))
	var n struct {
		Commands []struct {
			Name      string
			HasDetail *bool `json:"hasDetail"`
		}
	}
	if err := json.Unmarshal(b, &n); err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"check": true, "plain": false, "blank": false}
	for _, c := range n.Commands {
		if c.HasDetail == nil || *c.HasDetail != want[c.Name] {
			t.Errorf("%s: hasDetail = %v, want %v (always present)", c.Name, c.HasDetail, want[c.Name])
		}
	}
	if len(n.Commands) != 3 {
		t.Errorf("commands = %d", len(n.Commands))
	}
}

func TestExitsAndHasDetailOnlyWhereTheyBelong(t *testing.T) {
	p := load(t, exitsRegistry)
	g := p.GroupByKey("ops")
	for name, view := range map[string]string{
		"summary": Summary(p, Options{Width: 100}), "expanded": Expanded(p, Options{Width: 100}), "group": GroupView(p, g, Options{Width: 100}),
	} {
		if strings.Contains(view, "no errors") || strings.Contains(view, "Exit codes") {
			t.Errorf("%s view shows exits:\n%s", name, view)
		}
	}
	if b, _ := json.Marshal(GroupJSON(p, g)); strings.Contains(string(b), "usage error") || strings.Contains(string(b), `"exits"`) {
		t.Errorf("group JSON shows exits: %s", b)
	}
	_, p2, g2, c := exitsView(t, "ops check", Options{})
	if b, _ := json.Marshal(CommandJSON(p2, g2, c)); strings.Contains(string(b), "hasDetail") {
		t.Errorf("hasDetail belongs to the group listing: %s", b)
	}
}
