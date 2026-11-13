// Package complete works out shell completion candidates. The shell scripts that
// justz generates are tiny: they pass the words typed so far to justz, and
// this package decides what can come next, from the same registry the help
// views use.
package complete

import (
	"regexp"
	"sort"
	"strings"

	"github.com/igorlino/justz/internal/help"
	"github.com/igorlino/justz/internal/resolve"
)

// Candidate is one possible completion.
type Candidate struct {
	Value string
	Desc  string
}

// Result is what to offer. Files means the shell should complete file names
// (for a value that is free text) instead of, or as well as, the candidates.
type Result struct {
	Candidates []Candidate
	Files      bool
}

// Env gives access to the project a command line refers to. Each function is
// called only when needed, so completing the first word never reads a project.
type Env struct {
	// Project returns the help model of the project, or nil if there is none.
	Project func() *help.Project
	// RootRecipes returns the recipes at the top level of the justfile, so a
	// bare name that is a real recipe is not mistaken for a group command.
	RootRecipes func() map[string]bool
	// Shims returns the names of registered shims (and their help names).
	Shims func() []string
}

var flagToken = regexp.MustCompile(`-{1,2}[A-Za-z0-9][A-Za-z0-9-]*`)

// flagNames returns the flags an argument name declares: "-w, --watch" gives
// -w and --watch, "-k <pattern>" gives -k. Positional arguments give none.
func flagNames(name string) []string {
	if !strings.HasPrefix(name, "-") {
		return nil
	}
	if i := strings.Index(name, "<"); i >= 0 {
		name = name[:i]
	}
	return flagToken.FindAllString(name, -1)
}

// takesValue reports whether an option declared as name takes a value, which
// it shows with a <placeholder>.
func takesValue(name string) bool {
	return strings.HasPrefix(name, "-") && strings.Contains(name, "<")
}

// filter keeps the candidates that start with prefix, in order, without
// duplicates.
func filter(cands []Candidate, prefix string) []Candidate {
	// Two candidates with the same value but different descriptions are both
	// kept: a command name found in two groups should show the clash.
	seen := map[[2]string]bool{}
	var out []Candidate
	for _, c := range cands {
		key := [2]string{c.Value, c.Desc}
		if strings.HasPrefix(c.Value, prefix) && !seen[key] {
			seen[key] = true
			out = append(out, c)
		}
	}
	return out
}

// Shim completes the words typed after a shim name. For a run shim (jz, or a
// pinned shim such as tz) the first words are a group and a command, or a
// bare command name, followed by the command's options. For a help shim (jzh)
// they are the same lookup plus the flags of the help command.
func Shim(help_ bool, words []string, e Env) Result {
	if len(words) == 0 {
		words = []string{""}
	}
	cur := words[len(words)-1]
	prior := words[:len(words)-1]

	proj := e.Project()
	if proj == nil {
		return Result{Files: !strings.HasPrefix(cur, "-")}
	}

	if help_ {
		return helpCandidates(proj, prior, cur)
	}
	return runCandidates(proj, e, prior, cur)
}

// helpFlags are the flags of the help command, in this order.
var helpFlags = []Candidate{
	{"-b", "Pick a group or command"},
	{"-j", "Print JSON"},
	{"-w", "Add example columns"},
}

func helpCandidates(proj *help.Project, prior []string, cur string) Result {
	// Flags may appear anywhere; they do not count as words of the lookup.
	var terms []string
	for _, w := range prior {
		if !strings.HasPrefix(w, "-") {
			terms = append(terms, w)
		}
	}
	if strings.HasPrefix(cur, "-") {
		return Result{Candidates: filter(helpFlags, cur)}
	}
	var cands []Candidate
	switch len(terms) {
	case 0:
		cands = groupAndCommandCandidates(proj, cur)
	case 1:
		cands = commandsOfGroup(proj, terms[0])
	}
	return Result{Candidates: filter(cands, cur)}
}

// groupAndCommandCandidates lists the core commands, then the groups, and
// (once something is typed) the bare names of runnable commands, which carry
// their group in the description.
func groupAndCommandCandidates(proj *help.Project, cur string) []Candidate {
	var cands []Candidate
	if proj.Core != nil {
		for _, c := range proj.Core.Commands {
			cands = append(cands, Candidate{c.Name, c.Desc})
		}
	}
	for _, g := range proj.Groups {
		cands = append(cands, Candidate{g.Key, g.Label})
	}
	if cur != "" {
		cands = append(cands, bareNames(proj)...)
	}
	return cands
}

// bareNames lists every runnable command of a group other than core, with its
// group in the description. A name found in several groups appears once for
// each, so the person can see the ambiguity.
func bareNames(proj *help.Project) []Candidate {
	var out []Candidate
	for _, g := range proj.Groups {
		for _, c := range g.Commands {
			out = append(out, Candidate{c.Name, g.Key + ": " + c.Desc})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Value < out[j].Value })
	return out
}

func commandsOfGroup(proj *help.Project, key string) []Candidate {
	g := proj.GroupByKey(key)
	if g == nil {
		return nil
	}
	var out []Candidate
	for _, c := range g.Commands {
		out = append(out, Candidate{c.Name, c.Desc})
	}
	return out
}

func runCandidates(proj *help.Project, e Env, prior []string, cur string) Result {
	// Nothing typed yet: the core commands and the groups.
	if len(prior) == 0 {
		return Result{Candidates: filter(groupAndCommandCandidates(proj, cur), cur)}
	}

	first := prior[0]

	// A group: its commands, then a command's options.
	if g := proj.Group(first); g != nil {
		if len(prior) == 1 {
			return Result{Candidates: filter(commandsOfGroup(proj, first), cur)}
		}
		if c := g.FindCommand(prior[1]); c != nil {
			return optionCandidates(c, prior[2:], cur)
		}
		return Result{}
	}

	// A core command runs without its group.
	if proj.Core != nil {
		if c := proj.Core.FindCommand(first); c != nil {
			return optionCandidates(c, prior[1:], cur)
		}
	}

	// A bare command name that resolves exactly to one group.
	var roots map[string]bool
	if e.RootRecipes != nil {
		roots = e.RootRecipes()
	}
	if doc := proj.Doc; doc != nil {
		if res := resolve.Exact(doc, roots, first); res.Kind == resolve.Found {
			m := res.Matches[0]
			if g := proj.Group(m.Group); g != nil {
				if c := g.FindCommand(m.Command); c != nil {
					return optionCandidates(c, prior[1:], cur)
				}
			}
		}
	}
	return Result{}
}

// optionCandidates completes the options of one command. prior are the words
// already typed after the command.
//
// After an option that takes a value, an enum's values are offered, or file
// names for free text. Otherwise the allowed values of positional arguments
// come first, then the options not used yet, each with both its short and
// long form.
func optionCandidates(c *help.Command, prior []string, cur string) Result {
	// The word before the cursor may be an option that wants a value.
	if n := len(prior); n > 0 {
		last := prior[n-1]
		for _, a := range c.Args {
			// An option takes a value when it shows a <placeholder>, or when it
			// lists the values it allows.
			if !takesValue(a.Name) && !(strings.HasPrefix(a.Name, "-") && len(a.Values) > 0) {
				continue
			}
			for _, f := range flagNames(a.Name) {
				if f == last {
					if len(a.Values) > 0 {
						var vals []Candidate
						for _, v := range a.Values {
							vals = append(vals, Candidate{v, a.Desc})
						}
						return Result{Candidates: filter(vals, cur)}
					}
					return Result{Files: true}
				}
			}
		}
	}

	// An option is used when any of its forms was typed: -s and --silent are
	// the same option.
	typed := map[string]bool{}
	for _, w := range prior {
		typed[w] = true
	}
	used := map[string]bool{}
	for _, a := range c.Args {
		forms := flagNames(a.Name)
		for _, f := range forms {
			if typed[f] {
				for _, g := range forms {
					used[g] = true
				}
				break
			}
		}
	}

	var cands []Candidate
	if !strings.HasPrefix(cur, "-") {
		for _, a := range c.Args {
			if !strings.HasPrefix(a.Name, "-") {
				for _, v := range a.Values {
					cands = append(cands, Candidate{v, a.Desc})
				}
			}
		}
	}
	for _, a := range c.Args {
		for _, f := range flagNames(a.Name) {
			if !used[f] {
				cands = append(cands, Candidate{f, a.Desc})
			}
		}
	}
	return Result{Candidates: filter(cands, cur)}
}
