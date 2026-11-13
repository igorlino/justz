// Package check finds problems in how a project documents its commands. It
// reads the justfile through the JSON of just --dump and, when there is one,
// the command registry, and reports findings that a person or a CI job can act
// on.
package check

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/igorlino/justz/internal/orderedjson"
	"github.com/igorlino/justz/internal/registry"
)

// Severity of a finding.
type Severity string

const (
	Error   Severity = "error"
	Warning Severity = "warning"
)

// Rule ids. They are part of the output and stay stable.
const (
	RuleDocMissing       = "doc-missing"
	RuleGroupMissing     = "group-missing"
	RuleVariadic         = "variadic-interpolation"
	RuleRegistryMissing  = "registry-missing"
	RuleRegistryStale    = "registry-stale"
	RuleRegistryDesc     = "registry-desc-diverges"
	RuleKeySignature     = "registry-key-signature"
	RuleDescLong         = "registry-desc-long"
	RuleUnsorted         = "registry-unsorted"
	RuleExampleBare      = "example-bare-command"
	RuleExampleSingle    = "example-single-flag"
	RuleExampleLongFlag  = "example-long-flag"
	RuleExampleEnumUnset = "example-enum-missing"
)

// MaxDescLength is the length above which a registry description is a warning.
const MaxDescLength = 120

// Finding is one problem.
type Finding struct {
	Rule     string   `json:"rule"`
	Severity Severity `json:"severity"`
	Subject  string   `json:"subject"`
	Message  string   `json:"message"`
	Fix      string   `json:"fix,omitempty"`
}

// Input is everything the checks need.
type Input struct {
	Dump           []byte
	Variant        string             // native, rich or local-rich
	Registry       *orderedjson.Value // nil for a native project
	RegistrySource []byte             // the registry file as written, for the order check
	// Sources maps a module path ("" for the root justfile, a module name for a
	// top-level module) to the text that just --dump prints for its file. It is
	// optional: without it the comment-block rule is not applied.
	Sources map[string]string
}

// Result is the outcome of Run.
type Result struct {
	Checked  int       `json:"checked"` // public recipes looked at
	Findings []Finding `json:"findings"`
}

// Errors counts the findings of a severity.
func (r Result) Count(s Severity) int {
	n := 0
	for _, f := range r.Findings {
		if f.Severity == s {
			n++
		}
	}
	return n
}

type dump struct {
	Recipes map[string]recipe `json:"recipes"`
	Modules map[string]dump   `json:"modules"`
}

type recipe struct {
	Private    bool              `json:"private"`
	Doc        *string           `json:"doc"`
	Parameters []param           `json:"parameters"`
	Attributes []json.RawMessage `json:"attributes"`
	Body       []any             `json:"body"`
}

type param struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
}

// item is a public recipe with where it lives.
type item struct {
	Path     string // "build" or "tool clean"
	Group    string // the registry group: "core" or the top-level module
	Name     string
	TopLevel bool // a root recipe
	R        recipe
}

// Run checks a project.
func Run(in Input) (Result, error) {
	var d dump
	if err := json.Unmarshal(in.Dump, &d); err != nil {
		return Result{}, fmt.Errorf("could not understand the output of just --dump: %v", err)
	}
	items := collect(d, "", "")
	var res Result
	res.Checked = len(items)

	for _, it := range items {
		res.Findings = append(res.Findings, recipeRules(it, in.Variant)...)
	}
	if len(in.Sources) > 0 {
		res.Findings = append(res.Findings, docMultilineFindings(items, in.Sources)...)
	}
	if in.Registry != nil {
		res.Findings = append(res.Findings, registryRules(in, items)...)
		res.Findings = append(res.Findings, exampleRules(in.Registry)...)
	}
	sortFindings(res.Findings)
	if res.Findings == nil {
		res.Findings = []Finding{}
	}
	return res, nil
}

// collect lists the public recipes. A module's default recipe is its
// dispatcher, so it is left out.
func collect(d dump, prefix, group string) []item {
	var out []item
	for name, r := range d.Recipes {
		if r.Private || name == "default" {
			continue
		}
		path := name
		g := "core"
		if prefix != "" {
			path = prefix + " " + name
			g = group
		}
		out = append(out, item{Path: path, Group: g, Name: name, TopLevel: prefix == "", R: r})
	}
	for mod, sub := range d.Modules {
		p := mod
		if prefix != "" {
			p = prefix + " " + mod
		}
		g := group
		if prefix == "" {
			g = mod
		}
		out = append(out, collect(sub, p, g)...)
	}
	return out
}

func recipeRules(it item, variant string) []Finding {
	var out []Finding
	subject := "just " + it.Path

	if it.R.Doc == nil || strings.TrimSpace(*it.R.Doc) == "" {
		out = append(out, Finding{RuleDocMissing, Error, subject, "no [doc]", "add [doc('one-line description')] above the recipe"})
	}
	if variant == "native" && it.TopLevel && !hasGroup(it.R) {
		out = append(out, Finding{RuleGroupMissing, Error, subject, "no [group]", "add [group('name')] above the recipe"})
	}
	if v := interpolatedVariadic(it.R); v != "" {
		out = append(out, Finding{RuleVariadic, Error, subject,
			fmt.Sprintf("interpolates {{ %s }}, which splits on whitespace", v),
			`add 'set positional-arguments' and use "$@"`})
	}
	return out
}

// hasAttribute reports whether the recipe has an attribute with this name. An
// attribute is a JSON object such as {"group": "core"}, or a plain string for
// one without a value such as "no-cd".
func hasAttribute(r recipe, name string) bool {
	for _, raw := range r.Attributes {
		var obj map[string]json.RawMessage
		if json.Unmarshal(raw, &obj) != nil {
			continue
		}
		if _, ok := obj[name]; ok {
			return true
		}
	}
	return false
}

func hasGroup(r recipe) bool { return hasAttribute(r, "group") }

// interpolatedVariadic returns the name of a variadic (* or +) parameter that
// the recipe body interpolates, or "".
func interpolatedVariadic(r recipe) string {
	vars := map[string]bool{}
	for _, p := range r.Parameters {
		if p.Kind == "star" || p.Kind == "plus" {
			vars[p.Name] = true
		}
	}
	if len(vars) == 0 {
		return ""
	}
	found := ""
	var walk func(v any)
	walk = func(v any) {
		arr, ok := v.([]any)
		if !ok {
			return
		}
		if len(arr) == 2 {
			if tag, ok := arr[0].(string); ok && tag == "variable" {
				if name, ok := arr[1].(string); ok && vars[name] && found == "" {
					found = name
				}
			}
		}
		for _, e := range arr {
			walk(e)
		}
	}
	for _, line := range r.Body {
		walk(line)
	}
	return found
}

// registryEntry is one command of the registry.
type registryEntry struct {
	Group, Key string
	Desc       string
	HasDesc    bool
	Extra      bool
	Args       []argEntry
}

type argEntry struct {
	Name   string
	Values []string
}

func registryEntries(reg *orderedjson.Value) []registryEntry {
	var out []registryEntry
	groups := reg.Get("groups")
	if groups == nil || groups.Kind != orderedjson.Object {
		return nil
	}
	for _, g := range groups.Obj {
		if g.Val == nil || g.Val.Kind != orderedjson.Object {
			continue
		}
		for _, kind := range []string{"commands", "extra"} {
			cmds := g.Val.Get(kind)
			if cmds == nil || cmds.Kind != orderedjson.Object {
				continue
			}
			for _, c := range cmds.Obj {
				e := registryEntry{Group: g.Key, Key: c.Key, Extra: kind == "extra"}
				if d := c.Val.Get("desc"); d != nil && d.Kind == orderedjson.String {
					e.Desc, e.HasDesc = d.Str, true
				}
				if args := c.Val.Get("args"); args != nil && args.Kind == orderedjson.Array {
					for _, a := range args.Arr {
						ae := argEntry{}
						if n := a.Get("name"); n != nil && n.Kind == orderedjson.String {
							ae.Name = n.Str
						}
						if vals := a.Get("values"); vals != nil && vals.Kind == orderedjson.Array {
							for _, v := range vals.Arr {
								if v.Kind == orderedjson.String {
									ae.Values = append(ae.Values, v.Str)
								}
							}
						}
						e.Args = append(e.Args, ae)
					}
				}
				out = append(out, e)
			}
		}
	}
	return out
}

func registryRules(in Input, items []item) []Finding {
	var out []Finding
	entries := registryEntries(in.Registry)

	byPath := map[string]registryEntry{}
	for _, e := range entries {
		byPath[pathOf(e.Group, e.Key)] = e
	}
	for _, it := range items {
		if !it.TopLevel && strings.Count(it.Path, " ") > 1 {
			continue // nested modules have no registry group
		}
		e, ok := byPath[pathOf(it.Group, it.Name)]
		subject := "just " + it.Path
		if !ok {
			out = append(out, Finding{RuleRegistryMissing, Error, subject, "missing from the registry",
				"add it to commands.json (or regenerate the overlay with: justz generate)"})
			continue
		}
		if it.R.Doc != nil && e.HasDesc && strings.TrimSpace(*it.R.Doc) != "" && strings.TrimSpace(*it.R.Doc) != strings.TrimSpace(e.Desc) {
			out = append(out, Finding{RuleRegistryDesc, Error, subject,
				fmt.Sprintf("[doc] diverges from the registry description\n      justfile: %s\n      registry: %s", *it.R.Doc, e.Desc),
				"make them the same, or keep the long text in a runbook and a one-line [doc] in both"})
		}
	}

	recipePaths := map[string]bool{}
	for _, it := range items {
		recipePaths[pathOf(it.Group, it.Name)] = true
	}
	for _, e := range entries {
		subject := fmt.Sprintf("%s.%s", e.Group, e.Key)
		if strings.ContainsAny(e.Key, " <>[]*+") {
			out = append(out, Finding{RuleKeySignature, Error, subject, "the key carries an argument signature, not a bare recipe name",
				"use the recipe name as the key and put the arguments in args"})
		}
		if !e.Extra && !recipePaths[pathOf(e.Group, e.Key)] && e.Key != "default" {
			out = append(out, Finding{RuleRegistryStale, Error, subject, "in the registry, but the justfile has no such recipe",
				"remove the entry, or restore the recipe"})
		}
		if n := utf8.RuneCountInString(e.Desc); n > MaxDescLength {
			out = append(out, Finding{RuleDescLong, Warning, subject,
				fmt.Sprintf("the description is %d characters long (over %d)", n, MaxDescLength),
				"keep a one-line description and move the rest to a runbook"})
		}
	}

	if f, ok := orderFinding(in.RegistrySource); ok {
		out = append(out, f)
	}
	return out
}

func pathOf(group, key string) string {
	if group == "core" {
		return key
	}
	return group + " " + key
}

// orderFinding reports a registry whose keys are not in canonical order. It
// compares the file with its canonical form, ignoring whitespace.
func orderFinding(src []byte) (Finding, bool) {
	if len(src) == 0 {
		return Finding{}, false
	}
	a, err := orderedjson.Parse(src)
	if err != nil {
		return Finding{}, false
	}
	b, _ := orderedjson.Parse(src)
	registry.Canonicalize(b)
	as, err1 := orderedjson.Marshal(a)
	bs, err2 := orderedjson.Marshal(b)
	if err1 != nil || err2 != nil || string(as) == string(bs) {
		return Finding{}, false
	}
	return Finding{RuleUnsorted, Error, "registry", "the registry is not in canonical order", "run: justz fmt"}, true
}

func sortFindings(f []Finding) {
	rank := map[Severity]int{Error: 0, Warning: 1}
	sort.SliceStable(f, func(i, j int) bool {
		if rank[f[i].Severity] != rank[f[j].Severity] {
			return rank[f[i].Severity] < rank[f[j].Severity]
		}
		if f[i].Rule != f[j].Rule {
			return f[i].Rule < f[j].Rule
		}
		return f[i].Subject < f[j].Subject
	})
}
