package help

import (
	"encoding/json"
	"strings"
	"testing"
)

const tagsRegistry = `{
  "cli": "dm",
  "groups": {
    "ops": {
      "label": "Operations",
      "examples": [ {"cmd": "just ops deploy --dry", "desc": "Preview"} ],
      "commands": {
        "deploy": {
          "desc": "Deploy",
          "detail": "A detail.",
          "args": [ {"name": "--dry", "desc": "Do not copy"} ],
          "exits": { "0": "ok" },
          "env": [ {"name": "HOST", "desc": "Host"} ],
          "tags": ["asks-first", "bad\u001b[31m", "", "production", "spends", "x\u0007y"]
        },
        "plain": { "desc": "Plain" }
      },
      "extra": { "alias": { "desc": "Alias", "tags": ["read-only"] } }
    }
  }
}`

func tagsView(t *testing.T, name string, o Options) (string, *Project, *Group, *Command) {
	t.Helper()
	p := load(t, tagsRegistry)
	_, rows := p.Lookup(strings.Fields(name)...)
	if len(rows) != 1 {
		t.Fatalf("lookup %q: %d rows", name, len(rows))
	}
	g := p.GroupByKey(rows[0].Group)
	c := g.FindCommand(rows[0].Command)
	return CommandView(p, g, c, o), p, g, c
}

func TestZoomShowsTagsUnderTheDescriptionBeforeExamples(t *testing.T) {
	out, _, _, _ := tagsView(t, "ops deploy", Options{Width: 80})
	last := -1
	for _, s := range []string{"dm ops deploy", "Deploy", "Tags: asks-first, bad", "Examples", "Options", "Exit codes", "Environment", "Details"} {
		i := strings.Index(out, s)
		if i < 0 || i < last {
			t.Fatalf("%q missing or out of order:\n%s", s, out)
		}
		last = i
	}
	for _, bad := range []string{"\x1b", "\x07"} {
		if strings.Contains(out, bad) {
			t.Errorf("zoom shows %q:\n%q", bad, out)
		}
	}
}

func TestZoomTagsGoldenAndColors(t *testing.T) {
	out, _, _, _ := tagsView(t, "ops deploy", Options{Width: 60})
	golden(t, "zoom-command-tags", out)
	colored, _, _, _ := tagsView(t, "ops deploy", Options{Width: 60, Color: true})
	if got := ansi.ReplaceAllString(colored, ""); got != out {
		t.Errorf("colors must only add escape codes:\n%q\n%q", got, out)
	}
}

func TestZoomWithoutTagsIsUnchanged(t *testing.T) {
	out, _, _, _ := tagsView(t, "ops plain", Options{Width: 80})
	if strings.Contains(out, "Tags") || out != "\n  dm ops plain\n  "+strings.Repeat("─", 76)+"\n  Plain\n\n" {
		t.Errorf("zoom = %q", out)
	}
}

func TestZoomShowsTagsOfAnExtraEntry(t *testing.T) {
	out, _, _, _ := tagsView(t, "ops alias", Options{Width: 80})
	if !strings.Contains(out, "Tags: read-only") {
		t.Errorf("extra zoom = %q", out)
	}
}

func TestCommandJSONHasTagsInRegistryOrder(t *testing.T) {
	_, p, g, c := tagsView(t, "ops deploy", Options{})
	b, err := json.Marshal(CommandJSON(p, g, c))
	if err != nil {
		t.Fatal(err)
	}
	var n struct{ Tags []string }
	if err := json.Unmarshal(b, &n); err != nil || len(n.Tags) != 5 || n.Tags[0] != "asks-first" || n.Tags[2] != "production" || n.Tags[3] != "spends" {
		t.Fatalf("tags = %+v (%v) %s", n.Tags, err, b)
	}
	if strings.ContainsAny(string(b), "\x1b\x07") || strings.Contains(string(b), `\u001b`) || strings.Contains(string(b), `\u0007`) {
		t.Errorf("control characters in json: %s", b)
	}
	_, p, g, c = tagsView(t, "ops plain", Options{})
	if b, _ := json.Marshal(CommandJSON(p, g, c)); strings.Contains(string(b), "tags") {
		t.Errorf("tags must be left out when empty: %s", b)
	}
}

func TestTagsOnlyInTheCommandZoom(t *testing.T) {
	p := load(t, tagsRegistry)
	g := p.GroupByKey("ops")
	for name, view := range map[string]string{
		"summary": Summary(p, Options{Width: 100}), "expanded": Expanded(p, Options{Width: 100}), "group": GroupView(p, g, Options{Width: 100}),
	} {
		if strings.Contains(view, "asks-first") || strings.Contains(view, "Tags") {
			t.Errorf("%s view shows tags:\n%s", name, view)
		}
	}
	if b, _ := json.Marshal(GroupJSON(p, g)); strings.Contains(string(b), "asks-first") || strings.Contains(string(b), `"tags"`) {
		t.Errorf("group JSON shows tags: %s", b)
	}
}
