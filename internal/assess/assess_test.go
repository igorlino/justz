package assess

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/igorlino/justz/internal/orderedjson"
)

// rc builds one recipe of a dump.
func rc(doc string, private, shebang bool, bodyLines, helped int) map[string]any {
	r := map[string]any{"private": private, "shebang": shebang}
	if doc != "" {
		r["doc"] = doc
	} else {
		r["doc"] = nil
	}
	body := make([]any, bodyLines)
	for i := range body {
		body[i] = []any{"echo"}
	}
	r["body"] = body
	var params []any
	for i := 0; i < helped; i++ {
		params = append(params, map[string]any{"name": fmt.Sprintf("p%d", i), "help": "h"})
	}
	r["parameters"] = params
	return r
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestCounts(t *testing.T) {
	dump := mustJSON(t, map[string]any{
		"recipes": map[string]any{
			"a":       rc("Documented", false, false, 1, 0),
			"b":       rc("", false, true, 60, 3),
			"_hidden": rc("", true, false, 1, 0),
		},
		"modules": map[string]any{
			"tool": map[string]any{
				"recipes": map[string]any{
					"c": rc("Documented", false, true, 150, 0),
					"d": rc("", false, false, 2, 1),
				},
				"modules": map[string]any{},
			},
		},
	})
	r, err := Assess(Input{Dump: dump, Variant: "native"})
	if err != nil {
		t.Fatal(err)
	}
	want := Counts{Total: 5, Public: 4, Private: 1, Documented: 2, Undocumented: 2}
	if r.Recipes != want {
		t.Errorf("recipes = %+v, want %+v", r.Recipes, want)
	}
	if r.Modules != 1 || r.ShebangRecipes != 2 || r.ManyOptions != 1 {
		t.Errorf("modules %d, shebang %d, manyOptions %d", r.Modules, r.ShebangRecipes, r.ManyOptions)
	}
	if r.Long.Over40 != 2 || r.Long.Over100 != 1 || r.Long.Longest != 150 || r.Long.LongestName != "tool c" {
		t.Errorf("long = %+v", r.Long)
	}
}

func parse(t *testing.T, s string) *orderedjson.Value {
	t.Helper()
	v, err := orderedjson.Parse([]byte(s))
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestRegistryCoverage(t *testing.T) {
	dump := mustJSON(t, map[string]any{
		"recipes": map[string]any{"build": rc("B", false, false, 1, 0), "test": rc("T", false, false, 1, 0)},
		"modules": map[string]any{"tool": map[string]any{
			"recipes": map[string]any{"clean": rc("C", false, false, 1, 0), "doctor": rc("D", false, false, 1, 0)},
		}},
	})
	reg := parse(t, `{"groups":{
		"core":{"commands":{"build":{},"gone":{}}},
		"tool":{"commands":{"clean":{},"retired":{}}}}}`)
	r, err := Assess(Input{Dump: dump, Variant: "rich", RegistryPath: "config/commands.json", Registry: reg})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(r.Registry.Missing, ","); got != "test,tool doctor" {
		t.Errorf("missing = %q, want test,tool doctor", got)
	}
	if got := strings.Join(r.Registry.Stale, ","); got != "gone,tool retired" {
		t.Errorf("stale = %q, want gone,tool retired", got)
	}
	if r.RegistryPath != "config/commands.json" || r.Variant != "rich" {
		t.Errorf("variant/path = %q %q", r.Variant, r.RegistryPath)
	}
}

func TestDefaultRecipesAreNotReportedMissing(t *testing.T) {
	dump := mustJSON(t, map[string]any{
		"recipes": map[string]any{"default": rc("", false, false, 1, 0), "build": rc("B", false, false, 1, 0)},
		"modules": map[string]any{"tool": map[string]any{"recipes": map[string]any{"default": rc("", false, false, 1, 0)}}},
	})
	r, err := Assess(Input{Dump: dump, Registry: parse(t, `{"groups":{"core":{"commands":{"build":{}}}}}`)})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Registry.Missing) != 0 {
		t.Errorf("default recipes are dispatchers, but missing = %v", r.Registry.Missing)
	}
}

func TestRegistryArgsCountAsOptions(t *testing.T) {
	dump := mustJSON(t, map[string]any{"recipes": map[string]any{"scan": rc("S", false, false, 1, 0)}})
	reg := parse(t, `{"groups":{"core":{"commands":{"scan":{"args":[{"name":"a"},{"name":"b"},{"name":"c"}]}}}}}`)
	r, err := Assess(Input{Dump: dump, Registry: reg})
	if err != nil {
		t.Fatal(err)
	}
	if r.ManyOptions != 1 {
		t.Errorf("manyOptions = %d, want 1 (three registry args)", r.ManyOptions)
	}
}

func TestNoRegistryMeansNoCoverage(t *testing.T) {
	r, _ := Assess(Input{Dump: []byte(`{"recipes":{},"modules":{}}`), Variant: "native"})
	if r.Registry != nil {
		t.Error("coverage should be nil without a registry")
	}
}

func TestRecommendation(t *testing.T) {
	modules := func(n int) map[string]any {
		m := map[string]any{}
		for i := 0; i < n; i++ {
			recipes := map[string]any{}
			for j := 0; j < 10; j++ {
				recipes[fmt.Sprintf("r%d", j)] = rc("d", false, false, 1, 0)
			}
			m[fmt.Sprintf("m%d", i)] = map[string]any{"recipes": recipes}
		}
		return m
	}
	tests := []struct {
		name string
		dump map[string]any
		want string
	}{
		{"small project", map[string]any{"recipes": map[string]any{"a": rc("d", false, false, 1, 0)}}, "native"},
		{"large but flat", map[string]any{"recipes": func() map[string]any {
			m := map[string]any{}
			for i := 0; i < 40; i++ {
				m[fmt.Sprintf("r%d", i)] = rc("d", false, false, 1, 0)
			}
			return m
		}()}, "native"},
		{"large, namespaced, no options", map[string]any{"recipes": map[string]any{}, "modules": modules(4)}, "native"},
		{"large, namespaced, with options", map[string]any{"recipes": map[string]any{"x": rc("d", false, false, 1, 3)}, "modules": modules(4)}, "rich"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := Assess(Input{Dump: mustJSON(t, tt.dump), Variant: "native"})
			if err != nil {
				t.Fatal(err)
			}
			if r.Recommendation.Variant != tt.want || r.Recommendation.Reason == "" {
				t.Errorf("recommendation = %+v, want %s with a reason", r.Recommendation, tt.want)
			}
		})
	}
}

func TestInvalidDump(t *testing.T) {
	if _, err := Assess(Input{Dump: []byte("nope")}); err == nil {
		t.Error("an invalid dump should be an error")
	}
}
