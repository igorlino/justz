package help

import (
	"encoding/json"
	"strings"
	"testing"
)

const seeAlsoRegistry = `{
  "cli": "dm",
  "groups": {
    "core": { "commands": { "build": {"desc": "Build it"} } },
    "ops": {
      "label": "Operations",
      "seeAlso": ["core"],
      "commands": {
        "deploy": {"desc": "Deploy the report"},
        "deploy-staging": {
          "desc": "Deploy to staging",
          "seeAlso": ["ops.deploy", "core.build", "ops.hidden", "gone.cmd", "bad\u001b[31mkey"]
        },
        "lone": {"desc": "Lone", "seeAlso": ["core.build"]}
      },
      "extra": { "hidden": {"desc": "Documented only"} }
    },
    "dev": { "label": "Dev", "commands": { "plain": {"desc": "Plain"}, "ref": {"desc": "Ref", "seeAlso": ["ops.deploy"]} } }
  }
}`

func seeAlsoView(t *testing.T, name string, o Options) (string, *Project, *Group, *Command) {
	t.Helper()
	p := load(t, seeAlsoRegistry)
	_, rows := p.Lookup(strings.Fields(name)...)
	if len(rows) != 1 {
		t.Fatalf("lookup %q: %d rows", name, len(rows))
	}
	g := p.GroupByKey(rows[0].Group)
	c := g.FindCommand(rows[0].Command)
	return CommandView(p, g, c, o), p, g, c
}

func TestZoomShowsCommandSeeAlsoSortedAndResolved(t *testing.T) {
	out, _, _, _ := seeAlsoView(t, "ops deploy-staging", Options{Width: 80})
	// Sorted by key, then written: invocations for keys that resolve (core and
	// extra included), the rest as written with control characters replaced, and
	// the group line directly under it.
	want := "  See also: bad\uFFFD[31mkey, dm build, gone.cmd, dm ops deploy, dm ops hidden\n  See also: dm core\n"
	if !strings.Contains(out, want) {
		t.Errorf("zoom should contain %q:\n%q", want, out)
	}
	if strings.Contains(out, "\x1b") {
		t.Errorf("a key must not carry an escape: %q", out)
	}
	if !strings.HasSuffix(out, "\n\n") || strings.Contains(out, "\n\n\n") {
		t.Errorf("blank lines around the section: %q", out)
	}
}

func TestZoomSeeAlsoStyles(t *testing.T) {
	// The line of a command in a group without seeAlso, and in core.
	out, _, _, _ := seeAlsoView(t, "dev ref", Options{Width: 80})
	if !strings.Contains(out, "\n  See also: dm ops deploy\n\n") {
		t.Errorf("zoom:\n%s", out)
	}
	colored, _, _, _ := seeAlsoView(t, "dev ref", Options{Width: 80, Color: true})
	if got := ansi.ReplaceAllString(colored, ""); got != out {
		t.Errorf("colors must only add escape codes:\n%q\n%q", got, out)
	}
}

func TestZoomWithoutSeeAlsoIsUnchanged(t *testing.T) {
	out, _, _, _ := seeAlsoView(t, "dev plain", Options{Width: 80})
	if strings.Contains(out, "See also") {
		t.Errorf("no seeAlso, no line:\n%s", out)
	}
	if out != "\n  dm dev plain\n  "+strings.Repeat("─", 76)+"\n  Plain\n\n" {
		t.Errorf("zoom = %q", out)
	}
}

func TestZoomSeeAlsoGolden(t *testing.T) {
	out, _, _, _ := seeAlsoView(t, "ops deploy-staging", Options{Width: 80})
	golden(t, "zoom-command-see-also", out)
}

func TestCommandJSONHasSeeAlsoWhenSet(t *testing.T) {
	_, p, g, c := seeAlsoView(t, "ops deploy-staging", Options{})
	b, _ := json.Marshal(CommandJSON(p, g, c))
	var n struct {
		SeeAlso []string `json:"seeAlso"`
	}
	if err := json.Unmarshal(b, &n); err != nil || len(n.SeeAlso) != 5 || n.SeeAlso[3] != "ops.deploy" {
		t.Errorf("json = %s", b)
	}
	if strings.Contains(string(b), "\x1b") || strings.Contains(string(b), `\u001b`) {
		t.Errorf("a key must be clean in JSON: %s", b)
	}
	_, p, g, c = seeAlsoView(t, "dev plain", Options{})
	if b, _ := json.Marshal(CommandJSON(p, g, c)); strings.Contains(string(b), "seeAlso") {
		t.Errorf("seeAlso must be left out when empty: %s", b)
	}
}

func TestSeeAlsoIsOnlyInTheZoom(t *testing.T) {
	p := load(t, seeAlsoRegistry)
	g := p.GroupByKey("ops")
	for name, view := range map[string]string{
		"summary":  Summary(p, Options{Width: 100}),
		"expanded": Expanded(p, Options{Width: 100}),
		"group":    GroupView(p, g, Options{Width: 100}),
	} {
		if strings.Contains(view, "ops.deploy") || strings.Contains(view, "gone.cmd") {
			t.Errorf("%s view shows a command's seeAlso:\\n%s", name, view)
		}
	}
	b, _ := json.Marshal(GroupJSON(p, g))
	if strings.Contains(string(b), "gone.cmd") {
		t.Errorf("group JSON shows a command's seeAlso: %s", b)
	}
}

func TestSeeAlsoOrderIsByKeyInZoomAndJSON(t *testing.T) {
	p := load(t, `{"cli":"dm","groups":{"core":{"commands":{"z":{"desc":"Z"},"m":{"desc":"M"},"x":{"desc":"X","seeAlso":["ops.a","core.z","core.m"]}}},"ops":{"label":"O","commands":{"a":{"desc":"A"}}}}}`)
	g := p.Core
	c := g.FindCommand("x")
	out := CommandView(p, g, c, Options{Width: 80})
	// By key: core.m, core.z, ops.a. By rendered text it would be dm m, dm ops a, dm z.
	if !strings.Contains(out, "See also: dm m, dm z, dm ops a\n") {
		t.Errorf("zoom order:\n%s", out)
	}
	b, _ := json.Marshal(CommandJSON(p, g, c))
	if !strings.Contains(string(b), `"seeAlso":["core.m","core.z","ops.a"]`) {
		t.Errorf("json order: %s", b)
	}
}

func TestSeeAlsoDuplicatesAndSelfAreLeftOut(t *testing.T) {
	p := load(t, `{"cli":"dm","groups":{"core":{"commands":{"b":{"desc":"B"},"x":{"desc":"X","seeAlso":["core.b","core.x","core.b","gone","gone"]}}}}}`)
	g := p.Core
	c := g.FindCommand("x")
	out := CommandView(p, g, c, Options{Width: 80})
	if !strings.Contains(out, "See also: dm b, gone\n") {
		t.Errorf("zoom:\n%s", out)
	}
	if strings.Count(out, "dm b") != 1 || strings.Contains(out, "dm x,") || strings.Contains(out, ", dm x") {
		t.Errorf("a duplicate or a self reference is shown:\n%s", out)
	}
	b, _ := json.Marshal(CommandJSON(p, g, c))
	if !strings.Contains(string(b), `"seeAlso":["core.b","gone"]`) {
		t.Errorf("json: %s", b)
	}
	// Only a self reference: no section, no JSON field.
	q := load(t, `{"groups":{"core":{"commands":{"x":{"desc":"X","seeAlso":["core.x"]}}}}}`)
	qc := q.Core.FindCommand("x")
	if strings.Contains(CommandView(q, q.Core, qc, Options{Width: 80}), "See also") {
		t.Errorf("a lone self reference must show nothing")
	}
	if b, _ := json.Marshal(CommandJSON(q, q.Core, qc)); strings.Contains(string(b), "seeAlso") {
		t.Errorf("json: %s", b)
	}
}
