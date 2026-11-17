// Package assess reports facts about a justfile, from the JSON that
// just --dump produces. It reports what is there, not opinions, except for one
// stated recommendation of which documentation variant fits.
package assess

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/igorlino/justz/internal/orderedjson"
	"github.com/igorlino/justz/internal/textsafe"
)

type dump struct {
	Recipes map[string]recipe `json:"recipes"`
	Modules map[string]dump   `json:"modules"`
}

type recipe struct {
	Private    bool              `json:"private"`
	Doc        *string           `json:"doc"`
	Shebang    bool              `json:"shebang"`
	Body       []json.RawMessage `json:"body"`
	Parameters []struct {
		Help *string `json:"help"`
	} `json:"parameters"`
}

// Counts of recipes.
type Counts struct {
	Total        int `json:"total"`
	Public       int `json:"public"`
	Private      int `json:"private"`
	Documented   int `json:"documented"`   // public recipes with a doc
	Undocumented int `json:"undocumented"` // public recipes without one
	Dispatchers  int `json:"dispatchers"`  // public recipes named default: counted in Total, not in Public
}

// Long counts recipes by the length of their body, in lines.
type Long struct {
	Over40      int    `json:"over40"`
	Over100     int    `json:"over100"`
	Longest     int    `json:"longest"`
	LongestName string `json:"longestName"`
}

// Coverage compares the public recipes with the registry.
type Coverage struct {
	Missing []string `json:"missing"` // public recipes the registry does not list
	Stale   []string `json:"stale"`   // registry entries with no such public recipe
}

// Recommendation names a variant and says why. The variant fits only when
// Condition holds, which the justfile cannot show; Otherwise is the variant
// to use when it does not.
type Recommendation struct {
	Variant   string `json:"variant"`
	Condition string `json:"condition"`
	Otherwise string `json:"otherwise"`
	Reason    string `json:"reason"`
}

// Report is the result of Assess.
type Report struct {
	Variant        string         `json:"variant"` // rich, local-rich or native: what the project uses now
	RegistryPath   string         `json:"registryPath,omitempty"`
	Recipes        Counts         `json:"recipes"`
	Modules        int            `json:"modules"` // top-level modules
	ShebangRecipes int            `json:"shebangRecipes"`
	Long           Long           `json:"long"`
	ManyOptions    int            `json:"manyOptions"` // commands with more than two documented parameters or registry args
	Registry       *Coverage      `json:"registry,omitempty"`
	Recommendation Recommendation `json:"recommendation"`
	Advice         []string       `json:"advice"` // one line each; advice, not a rule
}

// Input is everything Assess needs.
type Input struct {
	Dump         []byte
	Variant      string
	RegistryPath string
	Registry     *orderedjson.Value // nil when there is no registry or overlay
}

// Assess builds the report.
func Assess(in Input) (Report, error) {
	var d dump
	if err := json.Unmarshal(in.Dump, &d); err != nil {
		return Report{}, fmt.Errorf("could not understand the output of just --dump: %v", err)
	}
	r := Report{Variant: in.Variant, RegistryPath: textsafe.Line(in.RegistryPath), Modules: len(d.Modules)}

	defaults, helpCalls := 0, 0 // default and _default recipes, and those that run justz help
	public := map[string]bool{} // "group command" paths of public recipes, for the coverage check
	var walk func(prefix string, mod dump)
	walk = func(prefix string, mod dump) {
		for name, rc := range mod.Recipes {
			r.Recipes.Total++
			if name == "default" || name == "_default" {
				defaults++
				if callsHelp(rc.Body) {
					helpCalls++
				}
			}
			if rc.Private {
				r.Recipes.Private++
				continue
			}
			if name == "default" {
				// The dispatcher is not a command: check leaves it out too.
				r.Recipes.Dispatchers++
				continue
			}
			r.Recipes.Public++
			if rc.Doc != nil && *rc.Doc != "" {
				r.Recipes.Documented++
			} else {
				r.Recipes.Undocumented++
			}
			if rc.Shebang {
				r.ShebangRecipes++
			}
			lines := len(rc.Body)
			if lines > 40 {
				r.Long.Over40++
			}
			if lines > 100 {
				r.Long.Over100++
			}
			if lines > r.Long.Longest || (lines == r.Long.Longest && qualified(prefix, name) < r.Long.LongestName) {
				r.Long.Longest, r.Long.LongestName = lines, qualified(prefix, name)
			}
			helped := 0
			for _, p := range rc.Parameters {
				if p.Help != nil {
					helped++
				}
			}
			if helped > 2 {
				r.ManyOptions++
			}
			public[qualified(prefix, name)] = true
		}
		for name, sub := range mod.Modules {
			walk(qualified(prefix, name), sub)
		}
	}
	walk("", d)

	if in.Registry != nil {
		r.Registry = coverage(in.Registry, public)
		// Arguments documented in the registry count too: a project may keep
		// them there and not in [arg] attributes.
		if n := registryManyOptions(in.Registry); n > r.ManyOptions {
			r.ManyOptions = n
		}
	}
	r.Recommendation = recommend(r)
	r.Advice = advice(r, in.Registry != nil, defaults, helpCalls)
	return r, nil
}

// advice lists hints that are not findings. The registry carries detail that
// only justz renders, so a project that keeps one and shows its help some
// other way hides that detail.
func advice(r Report, hasRegistry bool, defaults, helpCalls int) []string {
	out := []string{}
	if r.Recipes.Dispatchers > 0 {
		out = append(out, fmt.Sprintf("%d public default recipe(s) are dispatchers, not commands: rename each _default and keep it first in the file, so it is private (justz check warns: default-public)", r.Recipes.Dispatchers))
	}
	if hasRegistry && helpCalls < defaults {
		out = append(out, "a registry exists, but not every default recipe runs justz help, so detail kept in the registry is not shown by the help people use: have each default recipe run justz help <group> (shim names such as tz are not recognized)")
	}
	return out
}

var helpCall = regexp.MustCompile(`\b(justz|jz)\s+help\b|\bjzh\b`)

// callsHelp reports whether a recipe body runs justz help (or its shortcuts).
func callsHelp(body []json.RawMessage) bool {
	for _, line := range body {
		s := string(line)
		if helpCall.MatchString(s) {
			return true
		}
	}
	return false
}

// registryManyOptions counts registry commands with more than two args.
func registryManyOptions(reg *orderedjson.Value) int {
	n := 0
	groups := reg.Get("groups")
	if groups == nil || groups.Kind != orderedjson.Object {
		return 0
	}
	for _, g := range groups.Obj {
		if g.Val == nil || g.Val.Kind != orderedjson.Object {
			continue
		}
		cmds := g.Val.Get("commands")
		if cmds == nil || cmds.Kind != orderedjson.Object {
			continue
		}
		for _, c := range cmds.Obj {
			if args := c.Val.Get("args"); args != nil && args.Kind == orderedjson.Array && len(args.Arr) > 2 {
				n++
			}
		}
	}
	return n
}

func qualified(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + " " + name
}

// coverage compares registry entries with public recipes. Registry group "core"
// stands for the root recipes, any other group for a module.
func coverage(reg *orderedjson.Value, public map[string]bool) *Coverage {
	listed := map[string]bool{}
	if groups := reg.Get("groups"); groups != nil && groups.Kind == orderedjson.Object {
		for _, g := range groups.Obj {
			if g.Val == nil || g.Val.Kind != orderedjson.Object {
				continue
			}
			if cmds := g.Val.Get("commands"); cmds != nil && cmds.Kind == orderedjson.Object {
				for _, c := range cmds.Obj {
					if g.Key == "core" {
						listed[c.Key] = true
					} else {
						listed[g.Key+" "+c.Key] = true
					}
				}
			}
		}
	}
	cov := &Coverage{Missing: []string{}, Stale: []string{}}
	for p := range public {
		if !listed[p] {
			cov.Missing = append(cov.Missing, p)
		}
	}
	for p := range listed {
		// check accepts a registry entry for a module's default as well.
		if !public[p] && p != "default" && !strings.HasSuffix(p, " default") {
			// A stale entry is a registry key, which is project text.
			cov.Stale = append(cov.Stale, textsafe.Line(p))
		}
	}
	sort.Strings(cov.Missing)
	sort.Strings(cov.Stale)
	return cov
}

// richCondition is what the rich variant needs from the team.
const richCondition = "every machine that renders the help can install justz"

// recommend applies the rule of the variant guide: rich where every machine
// that renders the help can install justz, and otherwise what the project
// has without a registry. The dump cannot show which holds, so the
// recommendation carries the condition.
func recommend(r Report) Recommendation {
	switch r.Variant {
	case "rich":
		return Recommendation{"rich", richCondition, "native", "the project has a registry: keep it while the condition holds"}
	case "local-rich":
		return Recommendation{"rich", richCondition, "local-rich", "the overlay is personal: a committed registry gives the whole team details, safety tags, exit codes, environment rows and bare-name lookup"}
	}
	return Recommendation{"rich", richCondition, "native", "a registry adds details, safety tags, exit codes, environment rows and bare-name lookup, which the justfile alone cannot hold"}
}
