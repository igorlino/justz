package schema

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/igorlino/justz/internal/generate"
	"github.com/igorlino/justz/internal/orderedjson"
)

// The schema is checked with a small validator that knows only the keywords
// the schema uses. A test fails when the schema uses another one, so a
// constraint can never be skipped without notice.

var annotations = map[string]bool{"$schema": true, "$id": true, "$comment": true, "title": true, "description": true, "definitions": true}

type validator struct{ root map[string]any }

func newValidator(t *testing.T) *validator {
	t.Helper()
	var root map[string]any
	if err := json.Unmarshal([]byte(Commands), &root); err != nil {
		t.Fatalf("the schema is not valid JSON: %v", err)
	}
	return &validator{root: root}
}

// validate returns the first violation, naming its path, or nil.
func (v *validator) validate(sch map[string]any, val any, path string) error {
	if ref, ok := sch["$ref"].(string); ok {
		name, found := strings.CutPrefix(ref, "#/definitions/")
		def, _ := v.root["definitions"].(map[string]any)[name].(map[string]any)
		if !found || def == nil {
			return fmt.Errorf("%s: unknown reference %q", path, ref)
		}
		return v.validate(def, val, path)
	}
	if want, ok := sch["type"].(string); ok && jsonType(val) != want {
		return fmt.Errorf("%s: want %s, got %s", path, want, jsonType(val))
	}
	if obj, ok := val.(map[string]any); ok {
		if req, ok := sch["required"].([]any); ok {
			for _, r := range req {
				if _, has := obj[r.(string)]; !has {
					return fmt.Errorf("%s: %q is required", path, r)
				}
			}
		}
		props, _ := sch["properties"].(map[string]any)
		keys := make([]string, 0, len(obj))
		for k := range obj {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			child := path + "." + k
			if p, ok := props[k].(map[string]any); ok {
				if err := v.validate(p, obj[k], child); err != nil {
					return err
				}
				continue
			}
			switch ap := sch["additionalProperties"].(type) {
			case map[string]any:
				if err := v.validate(ap, obj[k], child); err != nil {
					return err
				}
			case bool:
				if !ap {
					return fmt.Errorf("%s: not allowed", child)
				}
			}
		}
	}
	if arr, ok := val.([]any); ok {
		if items, ok := sch["items"].(map[string]any); ok {
			for i, item := range arr {
				if err := v.validate(items, item, fmt.Sprintf("%s[%d]", path, i)); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func jsonType(v any) string {
	switch v.(type) {
	case nil:
		return "null"
	case bool:
		return "boolean"
	case float64:
		return "number"
	case string:
		return "string"
	case []any:
		return "array"
	}
	return "object"
}

func check(t *testing.T, v *validator, doc string) error {
	t.Helper()
	var val any
	if err := json.Unmarshal([]byte(doc), &val); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, doc)
	}
	return v.validate(v.root, val, "$")
}

// unsupported walks the schema and returns every keyword it uses that is neither
// handled by the validator nor an annotation.
func unsupported(node any, path string, out *[]string) {
	supported := map[string]bool{"type": true, "properties": true, "items": true, "required": true, "additionalProperties": true, "$ref": true}
	switch n := node.(type) {
	case map[string]any:
		for k, child := range n {
			switch k {
			case "properties", "definitions":
				// The keys below are names, not keywords.
				for name, sub := range child.(map[string]any) {
					unsupported(sub, path+"."+k+"."+name, out)
				}
			default:
				if !supported[k] && !annotations[k] {
					*out = append(*out, path+"."+k)
				}
				unsupported(child, path+"."+k, out)
			}
		}
	case []any:
		for i, c := range n {
			unsupported(c, fmt.Sprintf("%s[%d]", path, i), out)
		}
	}
}

func TestSchemaUsesOnlyKeywordsTheValidatorKnows(t *testing.T) {
	v := newValidator(t)
	var bad []string
	unsupported(v.root, "$", &bad)
	// Strings such as "type" and "description" inside properties are names;
	// the walk above skips them. Anything left is a keyword the tests would not enforce.
	if len(bad) > 0 {
		sort.Strings(bad)
		t.Errorf("keywords the test validator does not enforce: %v", bad)
	}
}

func TestSchemaDeclaresItself(t *testing.T) {
	v := newValidator(t)
	if v.root["$schema"] != "http://json-schema.org/draft-07/schema#" {
		t.Errorf("$schema = %v", v.root["$schema"])
	}
	if id, _ := v.root["$id"].(string); !strings.HasSuffix(id, "/schema/commands.schema.json") {
		t.Errorf("$id = %v", v.root["$id"])
	}
}

// Registries the repository already has must validate. The empty ones are {}
// and have no groups, so they are listed with the wrong shapes.
func TestFixturesValidate(t *testing.T) {
	v := newValidator(t)
	paths := []string{
		"../internal/registry/testdata/full.in.json",
		"../internal/registry/testdata/full.golden.json",
		"../internal/registry/testdata/minimal.in.json",
		"../internal/registry/testdata/minimal.golden.json",
		"../testdata/help-reference/demo/config/commands.json",
		"../testdata/variants/rich/config/commands.json",
		"../testdata/variants/local-rich/.just-local-rich.json",
		"../internal/generate/testdata/native.expected.json",
	}
	for _, p := range paths {
		data, err := os.ReadFile(filepath.FromSlash(p))
		if err != nil {
			t.Fatal(err)
		}
		if err := check(t, v, string(data)); err != nil {
			t.Errorf("%s: %v", p, err)
		}
	}
}

func TestGeneratedRegistryValidates(t *testing.T) {
	dump, err := os.ReadFile("../internal/generate/testdata/native.dump.json")
	if err != nil {
		t.Fatal(err)
	}
	res, err := generate.Build(dump, nil)
	if err != nil {
		t.Fatal(err)
	}
	out, err := orderedjson.Marshal(res.Doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := check(t, newValidator(t), string(out)); err != nil {
		t.Error(err)
	}
}

func TestEmptyFixturesLackGroups(t *testing.T) {
	v := newValidator(t)
	for _, p := range []string{"empty.in.json", "empty.golden.json"} {
		data, err := os.ReadFile(filepath.Join("..", "internal", "registry", "testdata", p))
		if err != nil {
			t.Fatal(err)
		}
		if err := check(t, v, string(data)); err == nil || !strings.Contains(err.Error(), `"groups" is required`) {
			t.Errorf("%s: err = %v", p, err)
		}
	}
}

func TestUnknownFieldsAreAllowed(t *testing.T) {
	doc := `{"zeta": 1, "groups": {"core": {"unk": {"a": [1]}, "commands": {"build": {"desc": "x", "future": true,
		"args": [{"name": "-a", "desc": "d", "type": "boolean", "extra": 1}]}}}}}`
	if err := check(t, newValidator(t), doc); err != nil {
		t.Error(err)
	}
}

func TestEveryDocumentedFieldIsTyped(t *testing.T) {
	doc := `{"$schema": "u", "$comment": "c", "name": "n", "title": "t", "tagline": "g", "subtitle": "s", "cli": "tz",
		"tiers": [{"label": "L", "hint": "h", "groups": ["a"]}], "order": ["core", 1], "dispatchers": [{}], "lab": [],
		"groups": {"a": {"label": "A", "title": "T", "desc": "d", "footer": "f", "seeAlso": ["b"],
			"examples": [{"cmd": "just a x", "desc": "d"}],
			"commands": {"x": {"desc": "d", "example": "… e", "args": [{"name": "--env <e>", "desc": "d", "type": "string", "values": ["a", "b"]}]}},
			"extra": {"y": {"desc": "d"}}}}}`
	if err := check(t, newValidator(t), doc); err != nil {
		t.Error(err)
	}
}

func TestWrongShapesFail(t *testing.T) {
	cases := []struct{ name, doc, want string }{
		{"root is a list", `[]`, "$: want object, got array"},
		{"no groups", `{}`, `"groups" is required`},
		{"groups is a list", `{"groups": []}`, "$.groups: want object, got array"},
		{"group is a string", `{"groups": {"a": "x"}}`, "$.groups.a: want object, got string"},
		{"desc is a number", `{"groups": {"a": {"commands": {"x": {"desc": 3}}}}}`, "$.groups.a.commands.x.desc: want string, got number"},
		{"commands is a list", `{"groups": {"a": {"commands": []}}}`, "$.groups.a.commands: want object, got array"},
		{"extra command shape", `{"groups": {"a": {"extra": {"y": {"args": {}}}}}}`, "$.groups.a.extra.y.args: want array, got object"},
		{"args is an object", `{"groups": {"a": {"commands": {"x": {"args": {}}}}}}`, "$.groups.a.commands.x.args: want array, got object"},
		{"arg entry is a string", `{"groups": {"a": {"commands": {"x": {"args": ["-a"]}}}}}`, "$.groups.a.commands.x.args[0]: want object, got string"},
		{"values has a number", `{"groups": {"a": {"commands": {"x": {"args": [{"values": ["a", 1]}]}}}}}`, "$.groups.a.commands.x.args[0].values[1]: want string, got number"},
		{"example entry", `{"groups": {"a": {"examples": [1]}}}`, "$.groups.a.examples[0]: want object, got number"},
		{"seeAlso entry", `{"groups": {"a": {"seeAlso": [{}]}}}`, "$.groups.a.seeAlso[0]: want string, got object"},
		{"tier groups", `{"groups": {}, "tiers": [{"groups": "a"}]}`, "$.tiers[0].groups: want array, got string"},
		{"cli is a number", `{"cli": 1, "groups": {}}`, "$.cli: want string, got number"},
		{"order is a string", `{"groups": {}, "order": "core"}`, "$.order: want array, got string"},
	}
	v := newValidator(t)
	for _, c := range cases {
		err := check(t, v, c.doc)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err = %v, want it to contain %q", c.name, err, c.want)
		}
	}
}
