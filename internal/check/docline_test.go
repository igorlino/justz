package check

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var docLineDescs = []string{
	"Plain text",
	"It's quoted",
	`Say "hi"`,
	`Both ' and " together`,
	`A back\slash and a trailing one\`,
	`Literal \n and \u{41} and \" and \\ text`,
	"Unicode: café, → arrow, 日本語",
	"Braces {{ not }} interpolated",
	"Quote and braces: it's {{x}}",
	"  Padded  ",
}

func TestDocLineForm(t *testing.T) {
	for in, want := range map[string]string{
		"Plain":    `[doc('Plain')]`,
		"It's":     `[doc("It's")]`,
		`a\b`:      `[doc('a\b')]`,
		`it's "x"`: `[doc("it's \"x\"")]`,
		`it's a\b`: `[doc("it's a\\b")]`,
		" padded ": `[doc('padded')]`,
	} {
		if got := DocLine(in); got != want {
			t.Errorf("DocLine(%q) = %s, want %s", in, got, want)
		}
	}
}

// TestDocLineRoundTrips pastes each line into a real justfile and reads the
// doc back through just --dump, as check does.
func TestDocLineRoundTrips(t *testing.T) {
	just, err := exec.LookPath("just")
	if err != nil {
		t.Skip("just is not installed: the round trip needs the real parser (CI installs it)")
	}
	for _, desc := range docLineDescs {
		dir := t.TempDir()
		src := DocLine(desc) + "\nfoo:\n  echo\n"
		if err := os.WriteFile(filepath.Join(dir, "justfile"), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command(just, "--dump", "--dump-format", "json")
		cmd.Dir = dir
		out, err := cmd.Output()
		if err != nil {
			t.Errorf("%q: just rejected %s: %v", desc, DocLine(desc), err)
			continue
		}
		var d struct {
			Recipes map[string]struct {
				Doc *string `json:"doc"`
			} `json:"recipes"`
		}
		if err := json.Unmarshal(out, &d); err != nil {
			t.Fatal(err)
		}
		doc := d.Recipes["foo"].Doc
		if doc == nil || *doc != strings.TrimSpace(desc) {
			t.Errorf("%q: %s became doc %v", desc, DocLine(desc), doc)
			continue
		}
		// Pasting it makes check clean for the command.
		reg, _ := json.Marshal(map[string]any{"groups": map[string]any{"core": map[string]any{"commands": map[string]any{"foo": map[string]any{"desc": desc}}}}})
		res, err := Run(Input{Dump: out, Variant: "rich", Registry: parse(t, string(reg))})
		if err != nil {
			t.Fatal(err)
		}
		if has(res, RuleRegistryDesc, "") {
			t.Errorf("%q: still diverges after the paste: %v", desc, rules(res))
		}
	}
}

func TestDescDivergesFindingCarriesTheLine(t *testing.T) {
	dump := dumpOf(t, map[string]any{"build": rc("Old text", nil)}, nil)
	reg := `{"groups":{"core":{"commands":{"build":{"desc":"It's new"}}}}}`
	res := run(t, Input{Dump: dump, Variant: "rich", Registry: parse(t, reg), RegistrySource: []byte(reg)})
	var f *Finding
	for i := range res.Findings {
		if res.Findings[i].Rule == RuleRegistryDesc {
			f = &res.Findings[i]
		}
	}
	if f == nil || f.Paste != `[doc("It's new")]` || strings.Contains(f.Message, "paste") || strings.Contains(f.Message, f.Paste) {
		t.Fatalf("finding = %+v", f)
	}
	b, _ := json.Marshal(f)
	if !strings.Contains(string(b), `"paste":"[doc(\"It's new\")]"`) {
		t.Errorf("json = %s", b)
	}
	// An overlay is regenerated, not pasted into the justfile.
	res = run(t, Input{Dump: dump, Variant: "local-rich", Registry: parse(t, reg), RegistrySource: []byte(reg)})
	for _, f := range res.Findings {
		if f.Rule == RuleRegistryDesc && f.Paste != "" {
			t.Errorf("local-rich finding has a paste line: %+v", f)
		}
	}
}

func TestDescDivergesHasNoPasteWhenItCannotBeExact(t *testing.T) {
	dump := dumpOf(t, map[string]any{"build": rc("Old text", nil)}, nil)
	for name, desc := range map[string]string{"blank": `"   "`, "control": `"a\u0007b"`, "newline": `"a\nb"`, "tab": `"a\tb"`} {
		reg := `{"groups":{"core":{"commands":{"build":{"desc":` + desc + `}}}}}`
		res := run(t, Input{Dump: dump, Variant: "rich", Registry: parse(t, reg), RegistrySource: []byte(reg)})
		for _, f := range res.Findings {
			if f.Rule == RuleRegistryDesc && name != "blank" && f.Paste != "" {
				t.Errorf("%s: paste = %q", name, f.Paste)
			}
			if f.Paste != "" && strings.TrimSpace(f.Paste) == "[doc('')]" {
				t.Errorf("%s: blank paste line", name)
			}
		}
	}
}
