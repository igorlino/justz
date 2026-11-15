package check

import (
	"fmt"
	"strings"

	"github.com/igorlino/justz/internal/orderedjson"
	"github.com/igorlino/justz/schema"
)

// shapeRules reports a registry whose fields have the wrong type. It walks the
// same schema that justz schema prints, so the check and the schema cannot
// disagree: the schema is read for type, properties, items,
// additionalProperties, required and $ref. Fields the schema does not name are
// allowed, as the schema allows them. justz reads a field of the wrong type as
// absent, so without this a mistyped args or examples drops documentation
// without a word.
func shapeRules(reg *orderedjson.Value) []Finding {
	root, err := orderedjson.Parse([]byte(schema.Commands))
	if err != nil {
		return nil // the embedded schema is tested to be valid JSON
	}
	w := shapeWalker{root: root}
	w.walk(root, reg, nil)
	return capShape(w.out)
}

// maxShapePerSubject is how many type mistakes one command or group reports
// before they are summed up, so a long array of bad rows stays readable.
const maxShapePerSubject = 10

func capShape(in []Finding) []Finding {
	var out []Finding
	seen := map[string]int{}
	for _, f := range in {
		seen[f.Subject]++
		if seen[f.Subject] <= maxShapePerSubject {
			out = append(out, f)
		}
	}
	for _, f := range in { // one summary per subject, in first-seen order
		if n := seen[f.Subject] - maxShapePerSubject; n > 0 {
			out = append(out, Finding{RuleRegistryShape, Error, f.Subject,
				fmt.Sprintf("%d more type mistakes in %s", n, f.Subject), f.Fix})
			seen[f.Subject] = 0
		}
	}
	return out
}

type shapeWalker struct {
	root *orderedjson.Value
	out  []Finding
}

func (w *shapeWalker) walk(sch, val *orderedjson.Value, path []string) {
	if ref := sch.Get("$ref"); ref != nil && ref.Kind == orderedjson.String {
		sch = w.resolve(ref.Str)
		if sch == nil {
			return
		}
	}
	if t := sch.Get("type"); t != nil && t.Kind == orderedjson.String && kindName(val) != t.Str {
		w.report(path, fmt.Sprintf("must be %s, not %s", article(t.Str), article(kindName(val))))
		return
	}
	switch val.Kind {
	case orderedjson.Object:
		if req := sch.Get("required"); req != nil && req.Kind == orderedjson.Array {
			for _, r := range req.Arr {
				if r.Kind == orderedjson.String && val.Get(r.Str) == nil {
					w.report(append(path, r.Str), "is required")
				}
			}
		}
		props := sch.Get("properties")
		extra := sch.Get("additionalProperties")
		for _, m := range val.Obj {
			child := append(append([]string{}, path...), m.Key)
			if props != nil && props.Get(m.Key) != nil {
				w.walk(props.Get(m.Key), m.Val, child)
			} else if extra != nil && extra.Kind == orderedjson.Object {
				w.walk(extra, m.Val, child)
			}
		}
	case orderedjson.Array:
		if items := sch.Get("items"); items != nil && items.Kind == orderedjson.Object {
			for i, e := range val.Arr {
				w.walk(items, e, append(append([]string{}, path...), fmt.Sprintf("[%d]", i)))
			}
		}
	}
}

// resolve follows a local reference such as #/definitions/command.
func (w *shapeWalker) resolve(ref string) *orderedjson.Value {
	cur := w.root
	for _, p := range strings.Split(strings.TrimPrefix(ref, "#/"), "/") {
		if cur == nil {
			return nil
		}
		cur = cur.Get(p)
	}
	return cur
}

// report names the command or group, then the field below it.
func (w *shapeWalker) report(path []string, what string) {
	subject, field := "registry", path
	if len(path) >= 2 && path[0] == "groups" {
		subject, field = path[1], path[2:]
		if len(path) >= 4 && (path[2] == "commands" || path[2] == "extra") {
			subject, field = path[1]+"."+path[3], path[4:]
		}
	}
	name := joinPath(field)
	if name == "" {
		name = "the entry"
		if subject == "registry" {
			name = "the registry"
		}
	} else {
		name = fmt.Sprintf("%q", name)
	}
	w.out = append(w.out, Finding{RuleRegistryShape, Error, subject,
		fmt.Sprintf("%s %s", name, what),
		"justz schema prints the expected shape (unknown fields are allowed)"})
}

func joinPath(p []string) string {
	var b strings.Builder
	for i, s := range p {
		if i > 0 && !strings.HasPrefix(s, "[") {
			b.WriteString(".")
		}
		b.WriteString(s)
	}
	return b.String()
}

func kindName(v *orderedjson.Value) string {
	switch v.Kind {
	case orderedjson.Bool:
		return "boolean"
	case orderedjson.Number:
		return "number"
	case orderedjson.String:
		return "string"
	case orderedjson.Array:
		return "array"
	case orderedjson.Object:
		return "object"
	}
	return "null"
}

func article(kind string) string {
	if kind == "array" || kind == "object" {
		return "an " + kind
	}
	return "a " + kind
}
