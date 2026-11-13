// Package assess reports facts about a justfile, from the JSON that
// just --dump produces. It reports what is there, not opinions, except for one
// stated recommendation of which documentation variant fits.
package assess

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/igorlino/justz/internal/orderedjson"
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

// Recommendation names a variant and says why.
type Recommendation struct {
	Variant string `json:"variant"`
	Reason  string `json:"reason"`
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
	r := Report{Variant: in.Variant, RegistryPath: in.RegistryPath, Modules: len(d.Modules)}

	public := map[string]bool{} // "group command" paths of public recipes, for the coverage check
	var walk func(prefix string, mod dump)
	walk = func(prefix string, mod dump) {
		for name, rc := range mod.Recipes {
			r.Recipes.Total++
			if rc.Private {
				r.Recipes.Private++
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
	return r, nil
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
		// A module's default recipe is its dispatcher (it shows the module's
		// help), not a command to document.
		if !listed[p] && p != "default" && !strings.HasSuffix(p, " default") {
			cov.Missing = append(cov.Missing, p)
		}
	}
	for p := range listed {
		if !public[p] {
			cov.Stale = append(cov.Stale, p)
		}
	}
	sort.Strings(cov.Missing)
	sort.Strings(cov.Stale)
	return cov
}

// recommend applies the thresholds from the variant guide: the registry only
// pays for itself in a large project with commands that take several options.
func recommend(r Report) Recommendation {
	switch {
	case r.Modules >= 3 && r.Recipes.Public >= 25 && r.ManyOptions >= 1:
		return Recommendation{"rich", fmt.Sprintf("%d namespaces, %d public recipes and %d recipe(s) with several options: enough to justify a registry", r.Modules, r.Recipes.Public, r.ManyOptions)}
	case r.Recipes.Public < 25 || r.Modules < 3:
		return Recommendation{"native", fmt.Sprintf("%d namespace(s) and %d public recipes: [doc] and [group] attributes are enough, and a registry would cost more than it gives", r.Modules, r.Recipes.Public)}
	}
	return Recommendation{"native", "no recipe takes several documented options, so a registry would mostly repeat what [doc] already says"}
}
