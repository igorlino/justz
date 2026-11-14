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
)

// Rule ids. They are part of the output and stay stable.
const (
	RuleRecipeLong       = "recipe-long"
	RuleModuleShellUnset = "module-shell-unset"
	RuleIgnoreUnknown    = "ignore-unknown-rule"
)

// MaxBodyLines is the length of a recipe body, in entries of the dump, above
// which it is reported. It is the number justz assess counts as long.
const MaxBodyLines = 40

// IgnoreKey is the first value of the [metadata] attribute that silences rules
// on one recipe: [metadata('justz-ignore', 'recipe-long')].
const IgnoreKey = "justz-ignore"

// ignorable lists the rules a recipe may silence, in the order they are shown.
var ignorable = []string{RuleRecipeLong}

// Finding is one piece of advice. It has the fields of a check finding.
type Finding struct {
	Rule     string         `json:"rule"`
	Severity check.Severity `json:"severity"`
	Subject  string         `json:"subject"`
	Message  string         `json:"message"`
	Fix      string         `json:"fix,omitempty"`
	Line     int            `json:"line,omitempty"` // 1-based index into the recipe body as dumped
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
	Recipes  map[string]recipe `json:"recipes"`
	Modules  map[string]dump   `json:"modules"`
	Settings struct {
		Shell json.RawMessage `json:"shell"`
	} `json:"settings"`
}

type recipe struct {
	Attributes []json.RawMessage `json:"attributes"`
	Body       []any             `json:"body"`
	Shebang    bool              `json:"shebang"` // also true for [script] recipes
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
			res.Findings = append(res.Findings, Finding{
				Rule: RuleRecipeLong, Severity: check.Warning, Subject: subject,
				Message: fmt.Sprintf("body has %d lines (limit %d)", n, MaxBodyLines),
				Fix:     "move the logic to scripts/<group>/<name>.sh and keep the recipe thin, or add [metadata('" + IgnoreKey + "', '" + RuleRecipeLong + "')]",
			})
		}
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
	for name, sub := range d.Modules {
		moduleRules(sub, path+" "+name, res)
	}
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
