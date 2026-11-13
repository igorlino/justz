package help

import (
	"encoding/json"
	"strings"
	"testing"
)

func toMap(t *testing.T, v any) map[string]any {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestJSONShapes(t *testing.T) {
	p := loadDemo(t)

	ov := toMap(t, OverviewJSON(p))
	if ov["cli"] != "dm" || len(ov["groups"].([]any)) != 3 || len(ov["core"].([]any)) != 3 {
		t.Errorf("overview = %v", ov)
	}

	g := toMap(t, GroupJSON(p, p.Group("build")))
	for _, k := range []string{"path", "invoke", "label", "desc", "commands", "examples", "seeAlso"} {
		if _, ok := g[k]; !ok {
			t.Errorf("group node lacks %q: %v", k, g)
		}
	}
	if g["path"] != "build" || g["invoke"] != "dm build" {
		t.Errorf("group path/invoke = %v / %v", g["path"], g["invoke"])
	}
	ex := g["examples"].([]any)[0].(map[string]any)
	if ex["cmd"] != "dm build release --level high" {
		t.Errorf("examples must have the command prefix applied: %v", ex)
	}

	build := p.Group("build")
	c := toMap(t, CommandJSON(p, build, build.FindCommand("release")))
	args := c["args"].([]any)
	if c["path"] != "build release" || c["invoke"] != "dm build release" || len(args) != 1 {
		t.Fatalf("command node = %v", c)
	}
	a := args[0].(map[string]any)
	if a["name"] != "--level" || a["desc"] != "How strict to be" {
		t.Errorf("arg = %v (desc must be the plain description)", a)
	}
	if vals := a["values"].([]any); len(vals) != 3 || vals[0] != "low" {
		t.Errorf("values = %v", a["values"])
	}
}

func TestCoreCommandPathHasNoGroupWord(t *testing.T) {
	p := loadDemo(t)
	c := toMap(t, CommandJSON(p, p.Core, p.Core.FindCommand("test")))
	if c["path"] != "test" || c["invoke"] != "dm test" {
		t.Errorf("core command = %v", c)
	}
}

func TestJSONListsAreNeverNull(t *testing.T) {
	p := load(t, `{"groups":{"g":{"label":"G","commands":{"c":{"desc":"d"}}}}}`)
	for name, v := range map[string]any{
		"group":   GroupJSON(p, p.Group("g")),
		"command": CommandJSON(p, p.Group("g"), p.Group("g").FindCommand("c")),
		"empty":   OverviewJSON(load(t, `{"groups":{}}`)),
	} {
		b, _ := json.Marshal(v)
		if strings.Contains(string(b), ":null") {
			t.Errorf("%s node has a null: %s", name, b)
		}
	}
}

func TestGroupCommandsAreSortedAndIncludeExtras(t *testing.T) {
	p := load(t, `{"groups":{"g":{"label":"G","commands":{"b":{"desc":"b"},"a":{"desc":"a"}},"extra":{"c":{"desc":"c"}}}}}`)
	n := GroupJSON(p, p.Group("g"))
	var names []string
	for _, c := range n.Commands {
		names = append(names, c.Name)
	}
	if strings.Join(names, ",") != "a,b,c" {
		t.Errorf("commands = %v, want a,b,c", names)
	}
}
