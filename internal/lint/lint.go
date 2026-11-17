// Package lint finds antipatterns that can be seen in the structure of a
// justfile, as advice. It reads the JSON of just --dump, never parses justfile
// text, and never runs a recipe. Every rule reports a warning.
package lint

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/igorlino/justz/internal/check"
	"github.com/igorlino/justz/internal/generate"
)

// Rule ids. They are part of the output and stay stable.
const (
	RuleRecipeLong       = "recipe-long"
	RuleSecretParameter  = "secret-parameter"
	RuleSecretInterpol   = "secret-interpolated"
	RuleModuleShellUnset = "module-shell-unset"
	RuleIgnoreUnknown    = "ignore-unknown-rule"
	RuleModuleNoDispatch = "module-dispatcher-missing"
	RuleModuleListsRoot  = "module-dispatcher-lists-root"
)

// Rules lists every rule id, for --rule and its completion. A test keeps it in
// step with the constants above.
var Rules = []string{
	RuleRecipeLong, RuleSecretParameter, RuleSecretInterpol, RuleModuleShellUnset,
	RuleIgnoreUnknown, RuleModuleNoDispatch, RuleModuleListsRoot,
}

// MaxBodyLines is the length of a recipe body, in entries of the dump, above
// which it is reported. It is the number justz assess counts as long.
const MaxBodyLines = 40

// IgnoreKey is the first value of the [metadata] attribute that silences rules
// on one recipe: [metadata('justz-ignore', 'recipe-long')].
const IgnoreKey = "justz-ignore"

// ignorable lists the rules a recipe may silence, in the order they are shown.
var ignorable = append([]string{RuleRecipeLong, RuleSecretParameter, RuleSecretInterpol}, check.IgnorableRules...)

// Finding is one piece of advice. It has the fields of a check finding.
type Finding struct {
	Rule     string         `json:"rule"`
	Severity check.Severity `json:"severity"`
	Subject  string         `json:"subject"`
	Message  string         `json:"message"`
	Fix      string         `json:"fix,omitempty"`
	Line     int            `json:"line,omitempty"` // 1-based index into the recipe body as dumped
	// The next three are on recipe-long only: how many {{ }} the body has, how
	// many of them are in single quotes or a quoted here-document, and how many
	// escapes ({{{{, {{"{{"}}) it has.
	Interpolations       *int `json:"interpolations,omitempty"`
	QuotedInterpolations *int `json:"quotedInterpolations,omitempty"`
	EscapedBraces        *int `json:"escapedBraces,omitempty"`
}

// Result is the outcome of Run.
type Result struct {
	Checked  int       `json:"checked"` // recipes looked at, private ones included
	Findings []Finding `json:"findings"`
}

// Count returns how many findings have the severity.
func (r Result) Count(s check.Severity) int {
	n := 0
	for _, f := range r.Findings {
		if f.Severity == s {
			n++
		}
	}
	return n
}

type dump struct {
	Assignments map[string]struct {
		Value any `json:"value"`
	} `json:"assignments"`
	First    *string           `json:"first"` // the recipe a bare call runs; null for no recipes
	Recipes  map[string]recipe `json:"recipes"`
	Modules  map[string]dump   `json:"modules"`
	Settings struct {
		Shell json.RawMessage `json:"shell"`
	} `json:"settings"`
}

type recipe struct {
	Private    bool              `json:"private"`
	Attributes []json.RawMessage `json:"attributes"`
	Body       []any             `json:"body"`
	Parameters []struct {
		Name    string `json:"name"`
		Default any    `json:"default"`
	} `json:"parameters"`
	Dependencies []struct {
		Recipe    string `json:"recipe"`
		Arguments []any  `json:"arguments"`
	} `json:"dependencies"`
	Shebang bool `json:"shebang"` // also true for [script] recipes
}

// isSet reports whether a setting of the dump has a value: it is null when the
// justfile does not set it.
func isSet(raw json.RawMessage) bool {
	s := strings.TrimSpace(string(raw))
	return s != "" && s != "null"
}

// Run lints a project from the output of just --dump --dump-format json.
func Run(dumpJSON []byte) (Result, error) {
	var d dump
	if err := json.Unmarshal(dumpJSON, &d); err != nil {
		return Result{}, fmt.Errorf("could not understand the output of just --dump: %v", err)
	}
	var res Result
	walk(d, "", &res)
	for name, sub := range d.Modules {
		moduleRules(sub, name, &res)
	}
	sortFindings(res.Findings)
	if res.Findings == nil {
		res.Findings = []Finding{}
	}
	return res, nil
}

// walk looks at every recipe: public, private, default, and those of nested
// modules. A long body or a secret is as real in a private helper.
func walk(d dump, prefix string, res *Result) {
	injected := dependencySecrets(d)
	for name, r := range d.Recipes {
		res.Checked++
		path := name
		if prefix != "" {
			path = prefix + " " + name
		}
		subject := "just " + path
		ignored, unknown := ignoredRules(r)

		for _, msg := range unknown {
			res.Findings = append(res.Findings, Finding{
				Rule: RuleIgnoreUnknown, Severity: check.Warning, Subject: subject,
				Message: msg,
				Fix:     "use one of: " + strings.Join(ignorable, ", ") + ". " + RuleModuleShellUnset + " has no per-recipe opt-out: set a shell in the module",
			})
		}
		if n := len(r.Body); n > MaxBodyLines && !ignored[RuleRecipeLong] {
			in := countInterpolations(r.Body, r.Shebang)
			res.Findings = append(res.Findings, Finding{
				Rule: RuleRecipeLong, Severity: check.Warning, Subject: subject,
				Message: fmt.Sprintf("body has %d lines (limit %d)%s", n, MaxBodyLines, in.describe()),
				Fix: "move the logic to scripts/<group>/<name>.sh, behind a shebang wrapper that execs it (error output stays identical; a one-line wrapper adds \"on line N\"). " +
					"A flat list of invocations that other tests look up in the justfile is the case for [metadata('" + IgnoreKey + "', '" + RuleRecipeLong + "')]",
				Interpolations: &in.Total, QuotedInterpolations: &in.Quoted, EscapedBraces: &in.Escaped,
			})
		}
		secretRules(d, r, subject, ignored, injected[name], res)
	}
	for name, sub := range d.Modules {
		p := name
		if prefix != "" {
			p = prefix + " " + name
		}
		walk(sub, p, res)
	}
}

// moduleRules reports a module that did not choose a shell. just does not pass
// the settings of the root justfile on to a module, so a module that sets none
// runs its linewise recipes with sh -cu: no pipefail. A module of only shebang
// or [script] recipes, or of recipes with nothing to run, is not affected by
// the setting, and one that sets a shell has chosen. set windows-shell does not
// count: it changes nothing on Linux or macOS. The root justfile is left alone:
// a root without set shell is a normal small justfile.
func moduleRules(d dump, path string, res *Result) {
	if !isSet(d.Settings.Shell) {
		for _, r := range d.Recipes {
			if !r.Shebang && len(r.Body) > 0 {
				res.Findings = append(res.Findings, Finding{
					Rule: RuleModuleShellUnset, Severity: check.Warning, Subject: "module " + path,
					Message: "sets no shell, so its recipes run with just's default (sh -cu), without pipefail",
					Fix:     "import a shared settings file with a set shell line (justz init writes just/preamble.just), or add set shell := ['bash', '-eu', '-o', 'pipefail', '-c']",
				})
				break
			}
		}
	}
	dispatcherRules(d, path, res)
	for name, sub := range d.Modules {
		moduleRules(sub, path+" "+name, res)
	}
}

// dispatcherRules checks what a bare `just <module>` runs. just runs the
// recipe the dump calls "first": the one with [default], else the first in the
// file. One named default or _default is the dispatcher; any other, private
// or not, is a command that a bare call would run. A dispatcher
// that lists recipes with just --list and justfile() lists the root justfile,
// because inside a module justfile() is the root. The root is left alone.
func dispatcherRules(d dump, path string, res *Result) {
	if d.First == nil {
		return
	}
	r, ok := d.Recipes[*d.First]
	if !ok {
		return
	}
	subject := "module " + path
	if !generate.IsDispatcherName(*d.First) {
		res.Findings = append(res.Findings, Finding{
			Rule: RuleModuleNoDispatch, Severity: check.Warning, Subject: subject,
			Message: fmt.Sprintf("a bare just %s runs %s, which is not a dispatcher", path, *d.First),
			Fix:     "add a default recipe first in the module (and no [default] on another recipe) that runs justz help " + path + " (see the justfile-authoring skill)",
		})
		return
	}
	if listsRoot(r) {
		res.Findings = append(res.Findings, Finding{
			Rule: RuleModuleListsRoot, Severity: check.Warning, Subject: subject,
			Message: fmt.Sprintf("its %s recipe runs just --list with justfile(), which inside a module is the root justfile, so a bare just %s lists the root recipes", *d.First, path),
			Fix:     "make the default recipe run justz help " + path + " (see the justfile-authoring skill)",
		})
	}
}

// listsRoot reports whether a body line has the text --list and a call to
// justfile() or justfile_directory(), which inside a module are the root's.
func listsRoot(r recipe) bool {
	for _, line := range r.Body {
		parts, ok := line.([]any)
		if !ok {
			continue
		}
		list, call := false, false
		for _, p := range parts {
			switch v := p.(type) {
			case string:
				if strings.Contains(v, "--list") {
					list = true
				}
			default:
				if hasJustfileCall(v) {
					call = true
				}
			}
		}
		if list && call {
			return true
		}
	}
	return false
}

// hasJustfileCall looks for a call to justfile or justfile_directory anywhere in an expression.
func hasJustfileCall(v any) bool {
	arr, ok := v.([]any)
	if !ok {
		return false
	}
	if len(arr) >= 2 {
		if tag, _ := arr[0].(string); tag == "call" {
			if name, _ := arr[1].(string); name == "justfile" || name == "justfile_directory" {
				return true
			}
		}
	}
	for _, e := range arr {
		if hasJustfileCall(e) {
			return true
		}
	}
	return false
}

// ignoredRules reads the [metadata('justz-ignore', rule...)] attributes of a
// recipe. It returns the rules to skip, and a message for each value that is
// not a rule a recipe can silence, so that a typo cannot hide a finding.
func ignoredRules(r recipe) (map[string]bool, []string) {
	ignored := map[string]bool{}
	var unknown []string
	for _, raw := range r.Attributes {
		var obj struct {
			Metadata []string `json:"metadata"`
		}
		if json.Unmarshal(raw, &obj) != nil || len(obj.Metadata) == 0 || obj.Metadata[0] != IgnoreKey {
			continue
		}
		ids := obj.Metadata[1:]
		if len(ids) == 0 {
			unknown = append(unknown, "[metadata('"+IgnoreKey+"')] names no rule")
		}
		for _, id := range ids {
			if isIgnorable(id) {
				ignored[id] = true
			} else {
				unknown = append(unknown, fmt.Sprintf("[metadata('%s')] names %q, which is not a rule a recipe can ignore", IgnoreKey, id))
			}
		}
	}
	sort.Strings(unknown)
	return ignored, unknown
}

func isIgnorable(id string) bool {
	for _, r := range ignorable {
		if r == id {
			return true
		}
	}
	return false
}

func sortFindings(f []Finding) {
	sort.SliceStable(f, func(i, j int) bool {
		if f[i].Rule != f[j].Rule {
			return f[i].Rule < f[j].Rule
		}
		return f[i].Subject < f[j].Subject
	})
}
