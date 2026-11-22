package check

import (
	"fmt"

	"github.com/igorlino/justz/internal/orderedjson"
	"github.com/igorlino/justz/internal/textsafe"
)

// tiersRules reports the tiers that do not match the groups. When a registry
// has tiers, the summary lists only the groups a tier names, so a group left
// out of every tier disappears from it, and help drops a name that is not a
// group without a word. The core group is listed on its own, so a tier that
// names it is left alone: help ignores the name.
//
// All of it is advice, so warnings: a project upgrading justz keeps a passing
// check, and check --strict fails on them.
func tiersRules(reg *orderedjson.Value) []Finding {
	tiers := reg.Get("tiers")
	groups := reg.Get("groups")
	if tiers == nil || tiers.Kind != orderedjson.Array || len(tiers.Arr) == 0 || groups == nil || groups.Kind != orderedjson.Object {
		return nil
	}
	var keys []string
	known := map[string]bool{}
	for _, g := range groups.Obj {
		if g.Key != "core" {
			keys = append(keys, g.Key)
			known[g.Key] = true
		}
	}
	var out []Finding
	placed := map[string]string{} // group -> label of the first tier that names it
	for i, t := range tiers.Arr {
		if t == nil || t.Kind != orderedjson.Object {
			continue
		}
		label := fmt.Sprintf("tiers[%d]", i)
		if l := t.Get("label"); l != nil && l.Kind == orderedjson.String && l.Str != "" {
			label = textsafe.Line(l.Str)
		}
		gs := t.Get("groups")
		if gs == nil || gs.Kind != orderedjson.Array {
			continue
		}
		for _, g := range gs.Arr {
			if g == nil || g.Kind != orderedjson.String {
				continue
			}
			name := textsafe.Line(g.Str)
			switch {
			case g.Str == "core":
				continue
			case !known[g.Str]:
				out = append(out, Finding{Rule: RuleTiers, Severity: Warning, Subject: "tiers",
					Message: fmt.Sprintf("tier %q names %q, which is not a group of the registry, so help drops it", label, name),
					Fix:     "use the key of a group, or remove the name"})
			case placed[g.Str] != "":
				out = append(out, Finding{Rule: RuleTiers, Severity: Warning, Subject: "tiers",
					Message: fmt.Sprintf("group %q is in tier %q and again in tier %q, so the summary lists it twice", name, placed[g.Str], label),
					Fix:     "keep the group in one tier"})
			default:
				placed[g.Str] = label
			}
		}
	}
	for _, k := range keys {
		if placed[k] == "" {
			out = append(out, Finding{Rule: RuleTiers, Severity: Warning, Subject: "tiers",
				Message: fmt.Sprintf("group %q is in no tier, so the summary does not list it", textsafe.Line(k)),
				Fix:     "add the group to a tier"})
		}
	}
	return out
}
