package assess

import (
	"encoding/json"
	"testing"

	"github.com/igorlino/justz/internal/orderedjson"
)

func TestStaleKeysAndPathsAreCleaned(t *testing.T) {
	var key string
	if err := json.Unmarshal([]byte(`"gone\u001b]0;t\u0007\u009b\u202e"`), &key); err != nil {
		t.Fatal(err)
	}
	dump := mustJSON(t, map[string]any{"recipes": map[string]any{"a": rc("Doc", false, false, 1, 0)}, "modules": map[string]any{}})
	reg, err := orderedjson.Parse([]byte(`{"groups": {"core": {"commands": {"a": {"desc": "x"}, ` + jsonQuote(key) + `: {"desc": "y"}}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	r, err := Assess(Input{Dump: dump, Variant: "rich", RegistryPath: "dir\u001b[31m/commands.json", Registry: reg})
	if err != nil {
		t.Fatal(err)
	}
	all := r.RegistryPath
	for _, s := range r.Registry.Stale {
		all += s
	}
	for _, c := range all {
		if c < 0x20 || c == 0x7f || (c >= 0x80 && c <= 0x9f) || (c >= 0x202a && c <= 0x202e) {
			t.Errorf("U+%04X is still in %q", c, all)
		}
	}
	if len(r.Registry.Stale) != 1 {
		t.Errorf("stale = %v", r.Registry.Stale)
	}
}

func jsonQuote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
