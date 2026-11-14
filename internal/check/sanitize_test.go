package check

import (
	"encoding/json"
	"strings"
	"testing"
)

const evilJSON = `x\u001b[31mR\u001b]0;t\u0007\u0000\r\u007f\u009b\u202e\u2066y`

func assertNoControl(t *testing.T, what, s string) {
	t.Helper()
	for _, r := range s {
		switch {
		case r == '\n':
		case r < 0x20, r == 0x7f, r >= 0x80 && r <= 0x9f, r >= 0x202a && r <= 0x202e, r >= 0x2066 && r <= 0x2069:
			t.Errorf("%s: U+%04X in %q", what, r, s)
			return
		}
	}
}

func TestFindingsOfADirtyProjectAreSafe(t *testing.T) {
	var doc string
	if err := json.Unmarshal([]byte(`"Doc `+evilJSON+`"`), &doc); err != nil {
		t.Fatal(err)
	}
	dump := dumpOf(t, map[string]any{"a": rc(doc, grp("core"))}, nil)
	src := `{"groups": {
		"core": {"commands": {
			"a": {"desc": "Different\n  error   just forged: row ` + evilJSON + `", "args": [{"name": "--f <v> ` + evilJSON + `", "values": ["` + evilJSON + `"]}]},
			"stale ` + evilJSON + `": {"desc": "x"},
			"long": {"desc": "` + strings.Repeat("word ", 40) + evilJSON + `"}}},
		"gr` + evilJSON + `": {"commands": {"k": {"desc": "x", "args": [{"name": "--e <v>", "values": ["a", "b"]}]}},
			"examples": [{"cmd": "just gr k", "desc": "bare"}]}}}`
	res := run(t, Input{Dump: dump, Variant: "rich", Registry: parse(t, src), RegistrySource: []byte(src)})
	if len(res.Findings) == 0 {
		t.Fatal("no findings to look at")
	}
	for _, f := range res.Findings {
		assertNoControl(t, f.Rule+" subject", f.Subject)
		assertNoControl(t, f.Rule+" message", f.Message)
		assertNoControl(t, f.Rule+" fix", f.Fix)
		for _, l := range strings.Split(f.Message, "\n") {
			if strings.HasPrefix(strings.TrimSpace(l), "error   just forged") {
				t.Errorf("a forged row is in the message of %s:\n%s", f.Rule, f.Message)
			}
		}
	}
	b, err := json.Marshal(res.Findings)
	if err != nil {
		t.Fatal(err)
	}
	var back []Finding
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	for _, f := range back {
		assertNoControl(t, "decoded JSON "+f.Rule, f.Subject+f.Message+f.Fix)
	}
}
