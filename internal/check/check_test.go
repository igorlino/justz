package check

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/igorlino/justz/internal/orderedjson"
)

// rc builds one recipe of a dump.
func rc(doc string, attrs []map[string]any, params ...map[string]any) map[string]any {
	r := map[string]any{"private": false, "body": []any{[]any{"echo"}}, "parameters": params, "attributes": attrs}
	if doc == "" {
		r["doc"] = nil
	} else {
		r["doc"] = doc
	}
	return r
}

func grp(name string) []map[string]any { return []map[string]any{{"group": name}} }

func dumpOf(t *testing.T, recipes map[string]any, modules map[string]any) []byte {
	t.Helper()
	if modules == nil {
		modules = map[string]any{}
	}
	b, err := json.Marshal(map[string]any{"recipes": recipes, "modules": modules})
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func module(recipes map[string]any) map[string]any {
	return map[string]any{"recipes": recipes, "modules": map[string]any{}}
}

func parse(t *testing.T, s string) *orderedjson.Value {
	t.Helper()
	v, err := orderedjson.Parse([]byte(s))
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func rules(r Result) []string {
	var out []string
	for _, f := range r.Findings {
		out = append(out, f.Rule+":"+f.Subject)
	}
	return out
}

func has(r Result, rule, subject string) bool {
	for _, f := range r.Findings {
		if f.Rule == rule && (subject == "" || f.Subject == subject) {
			return true
		}
	}
	return false
}

func run(t *testing.T, in Input) Result {
	t.Helper()
	res, err := Run(in)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func TestNativeDocumentationRules(t *testing.T) {
	dump := dumpOf(t, map[string]any{
		"good":    rc("Documented", grp("core")),
		"nodoc":   rc("", grp("core")),
		"nogroup": rc("Documented", nil),
		"default": rc("", nil), // a dispatcher: never checked
		"_hidden": map[string]any{"private": true, "doc": nil, "parameters": nil, "attributes": nil, "body": nil},
	}, map[string]any{"tool": module(map[string]any{
		"clean":   rc("Clean", nil), // a module recipe needs a doc, not a group
		"nodoc2":  rc("", nil),
		"default": rc("", nil),
	})})
	res := run(t, Input{Dump: dump, Variant: "native"})

	if !has(res, RuleDocMissing, "just nodoc") || !has(res, RuleDocMissing, "just tool nodoc2") {
		t.Errorf("missing [doc] not found: %v", rules(res))
	}
	if !has(res, RuleGroupMissing, "just nogroup") {
		t.Errorf("missing [group] not found: %v", rules(res))
	}
	for _, bad := range []string{"just good", "just tool clean", "just default", "just tool default", "just _hidden"} {
		for _, f := range res.Findings {
			if f.Subject == bad {
				t.Errorf("%s should not be reported: %+v", bad, f)
			}
		}
	}
	if has(res, RuleGroupMissing, "just tool clean") {
		t.Error("a module recipe does not need a [group]")
	}
	if res.Checked != 5 {
		t.Errorf("checked = %d, want 5 public recipes (default and private are skipped)", res.Checked)
	}
}

func TestGroupIsOnlyRequiredForNative(t *testing.T) {
	dump := dumpOf(t, map[string]any{"a": rc("Doc", nil)}, nil)
	reg := parse(t, `{"groups":{"core":{"commands":{"a":{"desc":"Doc"}}}}}`)
	res := run(t, Input{Dump: dump, Variant: "rich", Registry: reg, RegistrySource: []byte(`{"groups":{"core":{"commands":{"a":{"desc":"Doc"}}}}}`)})
	if has(res, RuleGroupMissing, "") {
		t.Errorf("a rich project does not need [group]: %v", rules(res))
	}
	if len(res.Findings) != 0 {
		t.Errorf("a clean project should have no findings: %+v", res.Findings)
	}
}

func TestVariadicInterpolation(t *testing.T) {
	interp := rc("Doc", grp("core"), map[string]any{"name": "args", "kind": "star"})
	interp["body"] = []any{[]any{"echo ", []any{[]any{"variable", "args"}}}}
	safe := rc("Doc", grp("core"), map[string]any{"name": "args", "kind": "star"})
	safe["body"] = []any{[]any{"echo \"$@\""}}
	plain := rc("Doc", grp("core"), map[string]any{"name": "x", "kind": "singular"})
	plain["body"] = []any{[]any{"echo ", []any{[]any{"variable", "x"}}}}

	res := run(t, Input{Dump: dumpOf(t, map[string]any{"bad": interp, "safe": safe, "plain": plain}, nil), Variant: "native"})
	if !has(res, RuleVariadic, "just bad") || has(res, RuleVariadic, "just safe") || has(res, RuleVariadic, "just plain") {
		t.Errorf("variadic findings = %v", rules(res))
	}
}

func TestRegistryParity(t *testing.T) {
	dump := dumpOf(t, map[string]any{
		"build": rc("Compile", grp("core")),
		"test":  rc("Run the tests", grp("core")),
		"fresh": rc("Not in the registry", grp("core")),
	}, map[string]any{"tool": module(map[string]any{
		"clean": rc("Clean", nil),
	})})
	src := `{"groups":{
		"core":{"commands":{"build":{"desc":"Compile"},"test":{"desc":"Run all the tests"},"gone":{"desc":"Old"}}},
		"tool":{"commands":{"clean":{"desc":"Clean"},"manifest [n]":{"desc":"Has a signature"}}}}}`
	res := run(t, Input{Dump: dump, Variant: "rich", Registry: parse(t, src), RegistrySource: []byte(src)})

	if !has(res, RuleRegistryMissing, "just fresh") {
		t.Errorf("missing: %v", rules(res))
	}
	if !has(res, RuleRegistryDesc, "just test") {
		t.Errorf("diverging description: %v", rules(res))
	}
	if !has(res, RuleRegistryStale, "core.gone") || !has(res, RuleRegistryStale, "tool.manifest [n]") {
		t.Errorf("stale: %v", rules(res))
	}
	if !has(res, RuleKeySignature, "tool.manifest [n]") {
		t.Errorf("key signature: %v", rules(res))
	}
	for _, f := range res.Findings {
		if f.Subject == "just build" || f.Subject == "just tool clean" {
			t.Errorf("a matching entry must not be reported: %+v", f)
		}
	}
}

func TestExtrasAreNotStaleAndCoverRecipes(t *testing.T) {
	dump := dumpOf(t, map[string]any{"a": rc("Doc", grp("core"))}, nil)
	src := `{"groups":{"core":{"commands":{},"extra":{"a":{"desc":"Doc"},"documented-only":{"desc":"x"}}}}}`
	res := run(t, Input{Dump: dump, Variant: "rich", Registry: parse(t, src), RegistrySource: []byte(src)})
	if has(res, RuleRegistryStale, "") || has(res, RuleRegistryMissing, "") {
		t.Errorf("extras should cover a recipe and never be stale: %v", rules(res))
	}
}

func TestLongDescriptionIsAWarning(t *testing.T) {
	long := strings.Repeat("word ", 40)
	dump := dumpOf(t, map[string]any{"a": rc("Doc", grp("core"))}, nil)
	src := `{"groups":{"core":{"commands":{"a":{"desc":"` + long + `"}}}}}`
	res := run(t, Input{Dump: dump, Variant: "rich", Registry: parse(t, src), RegistrySource: []byte(src)})
	var found *Finding
	for i := range res.Findings {
		if res.Findings[i].Rule == RuleDescLong {
			found = &res.Findings[i]
		}
	}
	if found == nil || found.Severity != Warning {
		t.Fatalf("want a warning, got %+v", res.Findings)
	}
	if res.Count(Error) == 0 || res.Count(Warning) != 1 {
		t.Errorf("counts: %d errors, %d warnings", res.Count(Error), res.Count(Warning))
	}
}

func TestUnsortedRegistry(t *testing.T) {
	dump := dumpOf(t, map[string]any{"a": rc("A", grp("core")), "b": rc("B", grp("core"))}, nil)
	unsorted := `{"groups":{"core":{"commands":{"b":{"desc":"B"},"a":{"desc":"A"}}}}}`
	res := run(t, Input{Dump: dump, Variant: "rich", Registry: parse(t, unsorted), RegistrySource: []byte(unsorted)})
	if !has(res, RuleUnsorted, "registry") {
		t.Errorf("unsorted registry not found: %v", rules(res))
	}

	// Whitespace alone is not an order problem.
	sorted := `{"groups":   {"core":{"commands":{"a":{"desc":"A"},"b":{"desc":"B"}}}}}`
	res = run(t, Input{Dump: dump, Variant: "rich", Registry: parse(t, sorted), RegistrySource: []byte(sorted)})
	if has(res, RuleUnsorted, "") {
		t.Errorf("a sorted registry with odd whitespace must pass: %v", rules(res))
	}
}

func TestExampleRules(t *testing.T) {
	dump := dumpOf(t, map[string]any{}, map[string]any{"snyk": module(map[string]any{
		"auth": rc("Auth", nil), "scan": rc("Scan", nil), "report": rc("Report", nil),
	})})
	commands := `{
		"auth":{"desc":"Auth","args":[{"name":"-l","desc":"Login"}]},
		"scan":{"desc":"Scan","args":[{"name":"-v, --severity <level>","desc":"Severity","values":["low","high"]},{"name":"-s, --silent","desc":"Quiet"}]},
		"report":{"desc":"Report","args":[{"name":"-o, --output <file>","desc":"Where"}]}}`
	withExamples := func(examples string) Result {
		s := `{"groups":{"snyk":{"label":"Snyk","commands":` + commands + `,"examples":` + examples + `}}}`
		return run(t, Input{Dump: dump, Variant: "rich", Registry: parse(t, s), RegistrySource: []byte(s)})
	}

	res := withExamples(`[{"cmd":"just snyk auth"},{"cmd":"just snyk auth -l"},{"cmd":"just snyk scan -v high --silent"},{"cmd":"just snyk scan --severity low -s"}]`)
	if !has(res, RuleExampleBare, "snyk.examples") {
		t.Errorf("bare command: %v", rules(res))
	}
	if !has(res, RuleExampleSingle, "snyk.examples") {
		t.Errorf("single boolean flag: %v", rules(res))
	}
	if !has(res, RuleExampleLongFlag, "snyk.examples") {
		t.Errorf("long flag with a short alias: %v", rules(res))
	}
	if has(res, RuleExampleEnumUnset, "") {
		t.Errorf("the enum flag is shown (-v high), so no finding: %v", rules(res))
	}

	res = withExamples(`[{"cmd":"just snyk scan -s"}]`)
	if !has(res, RuleExampleEnumUnset, "snyk.scan") {
		t.Errorf("enum values never shown: %v", rules(res))
	}
	// A value flag with no alias, shown or not, is fine for the other rules.
	res = withExamples(`[{"cmd":"just snyk report -o out.txt"},{"cmd":"just snyk scan -v low -s"}]`)
	for _, f := range res.Findings {
		if strings.HasPrefix(f.Rule, "example-") {
			t.Errorf("clean examples were reported: %+v", f)
		}
	}
}

func TestExampleRulesForCoreCommandsHaveNoGroupWord(t *testing.T) {
	dump := dumpOf(t, map[string]any{"test": rc("Run tests", grp("core"))}, nil)
	src := `{"groups":{"core":{"commands":{"test":{"desc":"Run tests","args":[{"name":"-w, --watch","desc":"Watch"}]}},"examples":[{"cmd":"just test"},{"cmd":"just test -w"}]}}}`
	res := run(t, Input{Dump: dump, Variant: "rich", Registry: parse(t, src), RegistrySource: []byte(src)})
	if !has(res, RuleExampleBare, "core.examples") || !has(res, RuleExampleSingle, "core.examples") {
		t.Errorf("core examples: %v", rules(res))
	}
	// The old checker never matched core examples, so they are warnings.
	for _, f := range res.Findings {
		if strings.HasPrefix(f.Rule, "example-") && f.Severity != Warning {
			t.Errorf("core example findings must be warnings: %+v", f)
		}
	}
}

func TestFindingsAreSortedAndNeverNull(t *testing.T) {
	res := run(t, Input{Dump: dumpOf(t, map[string]any{"a": rc("Doc", grp("core"))}, nil), Variant: "native"})
	if res.Findings == nil || len(res.Findings) != 0 {
		t.Errorf("no findings should be an empty list, got %#v", res.Findings)
	}
	dump := dumpOf(t, map[string]any{"b": rc("", nil), "a": rc("", nil)}, nil)
	res = run(t, Input{Dump: dump, Variant: "native"})
	got := rules(res)
	want := []string{"doc-missing:just a", "doc-missing:just b", "group-missing:just a", "group-missing:just b"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("order = %v, want %v", got, want)
	}
}

func TestInvalidDump(t *testing.T) {
	if _, err := Run(Input{Dump: []byte("nope")}); err == nil {
		t.Error("an invalid dump should be an error")
	}
}

func TestAttributesMayBePlainStrings(t *testing.T) {
	// just writes an attribute without a value, such as [no-cd], as a string.
	r := rc("Doc", nil)
	r["attributes"] = []any{"no-cd", map[string]any{"group": "core"}, "private-ish"}
	res := run(t, Input{Dump: dumpOf(t, map[string]any{"a": r}, nil), Variant: "native"})
	if len(res.Findings) != 0 {
		t.Errorf("string attributes must not break parsing or hide the group: %+v", res.Findings)
	}
}
