// Package resolve turns a bare command name into the group path that runs it.
//
// With a project registry that groups commands, "git" can mean "tool git".
// Only exact names resolve: a prefix never does, because a prefix that is
// unambiguous today stops being so when a command is added, and a shortcut
// that silently changes meaning is worse than none.
package resolve

import (
	"sort"

	"github.com/igorlino/justz/internal/orderedjson"
)

// Kind is the outcome of an exact lookup.
type Kind int

const (
	// Found: exactly one group has a command with this name.
	Found Kind = iota
	// Ambiguous: several groups have a command with this name.
	Ambiguous
	// None: nothing to resolve. Reason says why.
	None
)

// Why a name did not resolve.
const (
	ReasonUnknown = "unknown"          // no group has such a command
	ReasonRoot    = "root-recipe"      // a real recipe of the justfile: it wins
	ReasonGroup   = "group"            // the name is a group, which is a namespace
	ReasonCore    = "core"             // a core command runs without its group
	ReasonExtra   = "documented-extra" // documented but not runnable by bare name
)

// Match is one group and command.
type Match struct {
	Group   string `json:"group"`
	Command string `json:"command"`
}

// Result is the outcome of Exact.
type Result struct {
	Kind    Kind
	Matches []Match
	Reason  string // set when Kind is None
}

// Exact resolves name against the registry doc. rootRecipes are the recipes
// that exist at the top level of the justfile.
func Exact(doc *orderedjson.Value, rootRecipes map[string]bool, name string) Result {
	if rootRecipes[name] {
		return Result{Kind: None, Reason: ReasonRoot}
	}
	groups := doc.Get("groups")
	if groups == nil || groups.Kind != orderedjson.Object {
		return Result{Kind: None, Reason: ReasonUnknown}
	}
	if groups.Get(name) != nil {
		return Result{Kind: None, Reason: ReasonGroup}
	}

	var matches []Match
	for _, g := range groups.Obj {
		if g.Val == nil || g.Val.Kind != orderedjson.Object {
			continue
		}
		if g.Key == "core" {
			if has(g.Val.Get("commands"), name) {
				return Result{Kind: None, Reason: ReasonCore}
			}
			continue
		}
		if has(g.Val.Get("commands"), name) {
			matches = append(matches, Match{Group: g.Key, Command: name})
		} else if has(g.Val.Get("extra"), name) {
			// Documented, but not something a bare name should run.
			continue
		}
	}

	switch len(matches) {
	case 0:
		if isExtra(groups, name) {
			return Result{Kind: None, Reason: ReasonExtra}
		}
		return Result{Kind: None, Reason: ReasonUnknown}
	case 1:
		return Result{Kind: Found, Matches: matches}
	}
	sort.Slice(matches, func(i, j int) bool { return matches[i].Group < matches[j].Group })
	return Result{Kind: Ambiguous, Matches: matches}
}

func has(obj *orderedjson.Value, name string) bool {
	return obj != nil && obj.Kind == orderedjson.Object && obj.Get(name) != nil
}

func isExtra(groups *orderedjson.Value, name string) bool {
	for _, g := range groups.Obj {
		if g.Val != nil && g.Val.Kind == orderedjson.Object && has(g.Val.Get("extra"), name) {
			return true
		}
	}
	return false
}
