package help

import (
	"encoding/json"
	"strings"
	"testing"
)

const envRegistry = `{
  "cli": "dm",
  "groups": {
    "ops": {
      "label": "Operations",
      "commands": {
        "deploy": {
          "desc": "Deploy",
          "detail": "A detail.",
          "args": [ {"name": "--dry", "desc": "Do not copy"} ],
          "exits": { "0": "ok" },
          "env": [
            { "name": "STORAGE_HOST", "desc": "Host of the storage box, which is long enough to wrap onto a second line", "required": true },
            { "name": "DEPLOY_LEGACY_STATIC", "desc": "Also copy the legacy static files", "default": "unset" },
            { "name": "BAD\u001b[31m", "desc": "Control\u0007 char", "default": "x\u001b]0;t\u0007" },
            { "name": "", "desc": "no name" },
            { "name": "PLAIN", "desc": "No marker" }
          ]
        },
        "plain": { "desc": "Plain" }
      }
    }
  }
}`

func envView(t *testing.T, name string, o Options) (string, *Project, *Group, *Command) {
	t.Helper()
	p := load(t, envRegistry)
	_, rows := p.Lookup(strings.Fields(name)...)
	if len(rows) != 1 {
		t.Fatalf("lookup %q: %d rows", name, len(rows))
	}
	g := p.GroupByKey(rows[0].Group)
	c := g.FindCommand(rows[0].Command)
	return CommandView(p, g, c, o), p, g, c
}

func TestZoomShowsEnvironmentAfterExitCodesBeforeDetails(t *testing.T) {
	out, _, _, _ := envView(t, "ops deploy", Options{Width: 60})
	last := -1
	for _, s := range []string{"Options", "Exit codes", "Environment", "BAD", "DEPLOY_LEGACY_STATIC", "PLAIN", "STORAGE_HOST", "Details"} {
		i := strings.Index(out, s)
		if i < 0 || i < last {
			t.Fatalf("%q missing or out of order:\n%s", s, out)
		}
		last = i
	}
	for _, want := range []string{"(required)", "(default: unset)"} {
		if !strings.Contains(out, want) {
			t.Errorf("zoom should contain %q:\n%s", want, out)
		}
	}
	for _, bad := range []string{"no name", "\x1b", "\x07"} {
		if strings.Contains(out, bad) {
			t.Errorf("zoom shows %q:\n%q", bad, out)
		}
	}
}

func TestZoomEnvGoldenAndColors(t *testing.T) {
	out, _, _, _ := envView(t, "ops deploy", Options{Width: 60})
	golden(t, "zoom-command-env", out)
	colored, _, _, _ := envView(t, "ops deploy", Options{Width: 60, Color: true})
	if got := ansi.ReplaceAllString(colored, ""); got != out {
		t.Errorf("colors must only add escape codes:\n%q\n%q", got, out)
	}
}

func TestZoomWithoutEnvIsUnchanged(t *testing.T) {
	out, _, _, _ := envView(t, "ops plain", Options{Width: 80})
	if strings.Contains(out, "Environment") || out != "\n  dm ops plain\n  "+strings.Repeat("─", 76)+"\n  Plain\n\n" {
		t.Errorf("zoom = %q", out)
	}
}

func TestCommandJSONHasEnvInNameOrder(t *testing.T) {
	_, p, g, c := envView(t, "ops deploy", Options{})
	b, err := json.Marshal(CommandJSON(p, g, c))
	if err != nil {
		t.Fatal(err)
	}
	var n struct {
		Env []struct {
			Name     string
			Desc     string
			Required bool
			Default  string
		}
	}
	if err := json.Unmarshal(b, &n); err != nil || len(n.Env) != 4 {
		t.Fatalf("env = %+v (%v) %s", n.Env, err, b)
	}
	names := []string{}
	for _, e := range n.Env {
		names = append(names, e.Name)
	}
	if got := strings.Join(names[1:], ","); got != "DEPLOY_LEGACY_STATIC,PLAIN,STORAGE_HOST" {
		t.Errorf("names = %v", names)
	}
	if !n.Env[3].Required || n.Env[1].Default != "unset" {
		t.Errorf("fields lost: %+v", n.Env)
	}
	if strings.ContainsAny(string(b), "\x1b\x07") || strings.Contains(string(b), `\u001b`) || strings.Contains(string(b), `\u0007`) {
		t.Errorf("control characters in json: %s", b)
	}
	_, p, g, c = envView(t, "ops plain", Options{})
	if b, _ := json.Marshal(CommandJSON(p, g, c)); strings.Contains(string(b), "env") {
		t.Errorf("env must be left out when empty: %s", b)
	}
}

func TestEnvOnlyInTheCommandZoom(t *testing.T) {
	p := load(t, envRegistry)
	g := p.GroupByKey("ops")
	for name, view := range map[string]string{
		"summary": Summary(p, Options{Width: 100}), "expanded": Expanded(p, Options{Width: 100}), "group": GroupView(p, g, Options{Width: 100}),
	} {
		if strings.Contains(view, "STORAGE_HOST") || strings.Contains(view, "Environment") {
			t.Errorf("%s view shows env:\n%s", name, view)
		}
	}
	if b, _ := json.Marshal(GroupJSON(p, g)); strings.Contains(string(b), "STORAGE_HOST") || strings.Contains(string(b), `"env"`) {
		t.Errorf("group JSON shows env: %s", b)
	}
}
