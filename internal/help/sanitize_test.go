package help

import (
	"encoding/json"
	"strings"
	"testing"
)

// evil is a JSON string value full of what a terminal acts on: an SGR code, an
// OSC title, BEL, NUL, a carriage return, DEL, U+009B (CSI), a bidirectional
// override and isolate, and a tag character.
const evil = `x\u001b[31mR\u001b]0;t\u0007\u0000\r\u007f\u009b‮⁦󠁁y`

// assertSafe fails when s holds a character the terminal would act on. A
// newline is the only control character the views write themselves (colors are
// off in these tests).
func assertSafe(t *testing.T, what, s string) {
	t.Helper()
	for i, r := range s {
		switch {
		case r == '\n':
		case r < 0x20, r == 0x7f, r >= 0x80 && r <= 0x9f,
			r >= 0x202a && r <= 0x202e, r >= 0x2066 && r <= 0x2069,
			r >= 0xe0000 && r <= 0xe007f, r == 0x2028, r == 0x2029:
			t.Errorf("%s: U+%04X at byte %d of %q", what, r, i, s)
			return
		}
	}
}

const dirtyRegistry = `{
  "name": "` + evil + `", "tagline": "` + evil + `", "cli": "` + evil + `",
  "tiers": [{"label": "` + evil + `", "hint": "` + evil + `", "groups": ["g"]}],
  "groups": {
    "core": {"commands": {"c": {"desc": "` + evil + `"}}},
    "g": {
      "label": "` + evil + `", "title": "` + evil + `", "desc": "` + evil + `", "footer": "` + evil + `\nsecond",
      "seeAlso": ["` + evil + `"],
      "examples": [{"cmd": "just g k ` + evil + `", "desc": "` + evil + `"}],
      "commands": {
        "k": {"desc": "` + evil + `", "detail": "` + evil + `\nmore", "example": "` + evil + `",
              "args": [{"name": "-a, --` + evil + `", "desc": "` + evil + `", "values": ["` + evil + `"]}]}
      },
      "extra": {"x": {"desc": "` + evil + `"}}
    }
  }
}`

func TestEveryViewOfADirtyRegistryIsSafe(t *testing.T) {
	p := load(t, dirtyRegistry)
	o := Options{Width: 100}
	views := map[string]string{
		"summary":  Summary(p, o),
		"expanded": Expanded(p, o),
		"overview": Overview(p, o),
	}
	for _, g := range append([]*Group{p.Core}, p.Groups...) {
		views["group "+g.Key] = GroupView(p, g, o)
		for _, c := range append(append([]Command{}, g.Commands...), g.Extra...) {
			c := c
			views["command "+c.Name] = CommandView(p, g, &c, o)
			b, err := json.Marshal(CommandJSON(p, g, &c))
			if err != nil {
				t.Fatal(err)
			}
			var back CommandNode
			if err := json.Unmarshal(b, &back); err != nil {
				t.Fatal(err)
			}
			views["command json "+c.Name] = back.Desc + back.Detail + back.Path + back.Invoke
			for _, a := range back.Args {
				views["arg json"] = a.Name + a.Desc + strings.Join(a.Values, "")
			}
		}
		if g != p.Core {
			b, _ := json.Marshal(GroupJSON(p, g))
			var back GroupNode
			if err := json.Unmarshal(b, &back); err != nil {
				t.Fatal(err)
			}
			views["group json"] = back.Label + back.Desc + strings.Join(back.SeeAlso, "")
		}
	}
	for name, v := range views {
		assertSafe(t, name, v)
	}
	if !strings.Contains(views["summary"], "�") {
		t.Errorf("the residue of a replaced character is not visible:\n%s", views["summary"])
	}
}

func TestADescriptionWithANewlineCannotForgeARow(t *testing.T) {
	p := load(t, `{"groups": {"core": {"commands": {"a": {"desc": "one\n    forged   # row"}}}}}`)
	for _, v := range []string{Summary(p, Options{Width: 100}), Expanded(p, Options{Width: 100})} {
		for _, l := range strings.Split(v, "\n") {
			if strings.HasPrefix(strings.TrimSpace(l), "forged") {
				t.Errorf("a forged row appears:\n%s", v)
			}
		}
	}
}

func TestDetailAndFooterKeepTheirNewlines(t *testing.T) {
	p := load(t, dirtyRegistry)
	g := p.Group("g")
	if !strings.Contains(g.Footer, "\nsecond") || !strings.Contains(g.FindCommand("k").Detail, "\nmore") {
		t.Errorf("footer %q, detail %q", g.Footer, g.FindCommand("k").Detail)
	}
}

func TestSanitizeLeavesOrdinaryTextAlone(t *testing.T) {
	p := load(t, `{"name": "日本語 🚀", "groups": {"core": {"commands": {"a": {"desc": "Run — “tests” ünï", "detail": "line\n\n  two"}}}}}`)
	if p.Name != "日本語 🚀" || p.Core.FindCommand("a").Desc != "Run — “tests” ünï" || p.Core.FindCommand("a").Detail != "line\n\n  two" {
		t.Errorf("text changed: %q / %q / %q", p.Name, p.Core.FindCommand("a").Desc, p.Core.FindCommand("a").Detail)
	}
}

func TestANativeProjectIsCleanedToo(t *testing.T) {
	dump := `{"recipes": {"a": {"name": "a", "doc": "doc ` + evil + `", "attributes": [{"group": "head ` + evil + `"}],
	  "parameters": [], "body": [], "dependencies": [], "private": false, "quiet": false, "shebang": false}},
	  "modules": {}, "settings": {}, "assignments": {}, "aliases": {}, "groups": [], "warnings": []}`
	var name string
	if err := json.Unmarshal([]byte(`"dir `+evil+`"`), &name); err != nil {
		t.Fatal(err)
	}
	p, err := FromDump([]byte(dump), name)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range []string{Summary(p, Options{Width: 100}), Expanded(p, Options{Width: 100})} {
		assertSafe(t, "native view", v)
	}
	assertSafe(t, "native name", p.Name)
}
