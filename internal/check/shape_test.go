package check

import (
	"strings"

	"github.com/igorlino/justz/internal/orderedjson"
	"github.com/igorlino/justz/schema"
	"testing"
)

func shapeFindings(t *testing.T, reg string) []Finding {
	t.Helper()
	var out []Finding
	for _, f := range shapeRules(parse(t, reg)) {
		out = append(out, f)
	}
	return out
}

func TestRegistryShapeValidRegistryHasNoFinding(t *testing.T) {
	reg := `{"$schema":"x","name":"n","futureField":{"a":1},"groups":{"core":{"label":"L","futureField":[1],
	  "commands":{"build":{"desc":"d","detail":"x","example":"e","extra":3,"args":[{"name":"-s","desc":"d","type":"string","values":["a","b"],"more":true}]}},
	  "extra":{"old":{"desc":"d"}},
	  "examples":[{"cmd":"just build -x","desc":"d","more":1}],"seeAlso":["tool"]}},
	  "tiers":[{"label":"t","hint":"h","groups":["core"]}],"order":["core",1],"dispatchers":[],"lab":[]}`
	if got := shapeFindings(t, reg); len(got) != 0 {
		t.Errorf("findings = %+v", got)
	}
}

func TestRegistryShapeNamesCommandAndField(t *testing.T) {
	tests := []struct {
		name, reg, subject, message string
	}{
		{"args is a string", `{"groups":{"tool":{"commands":{"clean":{"args":"-s"}}}}}`, "tool.clean", `"args" must be an array, not a string`},
		{"desc is a number", `{"groups":{"core":{"commands":{"build":{"desc":3}}}}}`, "core.build", `"desc" must be a string, not a number`},
		{"arg row is a string", `{"groups":{"core":{"commands":{"build":{"args":["x"]}}}}}`, "core.build", `"args[0]" must be an object, not a string`},
		{"arg name is a number", `{"groups":{"core":{"commands":{"build":{"args":[{"name":1}]}}}}}`, "core.build", `"args[0].name" must be a string, not a number`},
		{"values has a non-string", `{"groups":{"core":{"commands":{"build":{"args":[{"values":["a",2]}]}}}}}`, "core.build", `"args[0].values[1]" must be a string, not a number`},
		{"extra entry", `{"groups":{"core":{"extra":{"old":{"args":{}}}}}}`, "core.old", `"args" must be an array, not an object`},
		{"examples is an object", `{"groups":{"tool":{"examples":{"cmd":"x"}}}}`, "tool", `"examples" must be an array, not an object`},
		{"example cmd is a list", `{"groups":{"tool":{"examples":[{"cmd":[]}]}}}`, "tool", `"examples[0].cmd" must be a string, not an array`},
		{"commands is a list", `{"groups":{"tool":{"commands":[]}}}`, "tool", `"commands" must be an object, not an array`},
		{"group is not an object", `{"groups":{"tool":"x"}}`, "tool", `the entry must be an object, not a string`},
		{"null is a wrong type", `{"groups":{"core":{"commands":{"build":{"desc":null}}}}}`, "core.build", `"desc" must be a string, not a null`},
		{"tier groups", `{"groups":{},"tiers":[{"groups":"core"}]}`, "registry", `"tiers[0].groups" must be an array, not a string`},
		{"root name", `{"groups":{},"name":1}`, "registry", `"name" must be a string, not a number`},
		{"groups missing", `{"name":"n"}`, "registry", `"groups" is required`},
		{"groups is a list", `{"groups":[]}`, "registry", `"groups" must be an object, not an array`},
		{"root is not an object", `[]`, "registry", `the registry must be an object, not an array`},
	}
	for _, tt := range tests {
		got := shapeFindings(t, tt.reg)
		if len(got) != 1 {
			t.Errorf("%s: findings = %+v", tt.name, got)
			continue
		}
		f := got[0]
		if f.Rule != RuleRegistryShape || f.Severity != Error || f.Subject != tt.subject || !strings.Contains(f.Message, tt.message) {
			t.Errorf("%s: finding = %+v, want %s %q", tt.name, f, tt.subject, tt.message)
		}
	}
}

func TestRunReportsRegistryShape(t *testing.T) {
	d := dumpOf(t, map[string]any{"build": rc("B", grp("core"))}, nil)
	reg := `{"groups":{"core":{"commands":{"build":{"desc":"B","args":"-s"}}}}}`
	res := run(t, Input{Dump: d, Variant: "rich", Registry: parse(t, reg)})
	if !has(res, RuleRegistryShape, "core.build") {
		t.Errorf("findings = %v", rules(res))
	}
}

func TestRegistryShapeIsCapped(t *testing.T) {
	rows := strings.TrimSuffix(strings.Repeat(`1,`, 25), ",")
	got := shapeFindings(t, `{"groups":{"core":{"commands":{"build":{"args":[`+rows+`]}}}}}`)
	if len(got) != 11 {
		t.Fatalf("findings = %d, want 10 and a summary", len(got))
	}
	if last := got[10]; last.Subject != "core.build" || !strings.Contains(last.Message, "15 more type mistakes in core.build") {
		t.Errorf("summary = %+v", last)
	}
	if strings.Contains(got[0].Fix, "fix the type") {
		t.Errorf("fix repeats itself: %q", got[0].Fix)
	}
}

// The walker handles only some schema keywords. A new keyword in the schema
// that it ignores would make check accept what the schema rejects.
func TestShapeWalkerHandlesEverySchemaKeyword(t *testing.T) {
	handled := map[string]bool{"type": true, "properties": true, "items": true, "additionalProperties": true,
		"required": true, "$ref": true, "$schema": true, "$id": true, "title": true, "description": true,
		"definitions": true, "$defs": true, "examples": true, "default": true}
	root, err := orderedjson.Parse([]byte(schema.Commands))
	if err != nil {
		t.Fatal(err)
	}
	var visit func(node *orderedjson.Value, where string)
	visit = func(node *orderedjson.Value, where string) {
		if node == nil || node.Kind != orderedjson.Object {
			return
		}
		for _, m := range node.Obj {
			if !handled[m.Key] {
				t.Errorf("%s: schema keyword %q is not handled by the shape check", where, m.Key)
			}
			switch m.Key {
			case "properties", "definitions", "$defs": // maps of name to schema
				for _, e := range m.Val.Obj {
					visit(e.Val, where+"/"+m.Key+"/"+e.Key)
				}
			case "items", "additionalProperties":
				visit(m.Val, where+"/"+m.Key)
			}
		}
	}
	visit(root, "#")
}
