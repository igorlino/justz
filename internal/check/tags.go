package check

import (
	"encoding/json"
	"fmt"
	"strings"
)

// TagAsksFirst is the tag that check keeps in step with the [confirm]
// attribute of the recipe.
const TagAsksFirst = "asks-first"

// unsafeWithReadOnly are the tags that contradict read-only.
var unsafeWithReadOnly = []string{"destructive", "spends", "production"}

// tagsRules reports the claims a command's tags make against each other and,
// for asks-first, against the recipe. A tag is acted on, so a stale one is
// worse than stale prose. check does not guess any other tag from the recipe
// and does not warn on a tag outside the recommended vocabulary: the schema
// accepts any string, and a reader treats a tag it does not know as unknown.
//
// registry-tags (a contradiction or a duplicate) is a warning. tags-confirm
// (asks-first and [confirm] disagree) is a warning too, since a script can
// ask by itself, and a recipe silences it with
// [metadata('justz-ignore', 'tags-confirm')]. An extra entry has no recipe, so
// it gets no confirm rule.
func tagsRules(entries []registryEntry, items []item) []Finding {
	byPath := map[string]item{}
	for _, it := range items {
		if it.nested() {
			continue // nested modules have no registry group, and would shadow the parent recipe
		}
		byPath[pathOf(it.Group, it.Name)] = it
	}
	var out []Finding
	for _, e := range entries {
		subject := fmt.Sprintf("%s.%s", e.Group, e.Key)
		has := map[string]bool{}
		dup := map[string]bool{}
		for _, t := range e.Tags {
			if strings.TrimSpace(t) == "" {
				continue // help drops a blank tag too
			}
			if has[t] && !dup[t] {
				dup[t] = true
				out = append(out, Finding{Rule: RuleTags, Severity: Warning, Subject: subject,
					Message: fmt.Sprintf("tags has %q more than once", t),
					Fix:     "keep one of each tag"})
			}
			has[t] = true
		}
		if has["read-only"] {
			for _, bad := range unsafeWithReadOnly {
				if has[bad] {
					out = append(out, Finding{Rule: RuleTags, Severity: Warning, Subject: subject,
						Message: fmt.Sprintf("tags has read-only and %s, which contradict each other", bad),
						Fix:     fmt.Sprintf("remove read-only or %s: an agent must not run a command as safe when it is not", bad)})
				}
			}
		}

		it, ok := byPath[pathOf(e.Group, e.Key)]
		if e.Extra || !ok || ignoredCheckRules(it.R)[RuleTagsConfirm] {
			continue
		}
		confirm := hasConfirm(it.R)
		switch {
		case confirm && !has[TagAsksFirst]:
			out = append(out, Finding{Rule: RuleTagsConfirm, Severity: Warning, Subject: subject,
				Message: "the recipe has [confirm], but tags does not have " + TagAsksFirst,
				Fix:     `add "` + TagAsksFirst + `" to tags, or add [metadata('` + IgnoreKey + `', '` + RuleTagsConfirm + `')] to the recipe`})
		case !confirm && has[TagAsksFirst]:
			out = append(out, Finding{Rule: RuleTagsConfirm, Severity: Warning, Subject: subject,
				Message: "tags has " + TagAsksFirst + ", but the recipe has no [confirm]",
				Fix:     `add [confirm] to the recipe, or remove the tag, or add [metadata('` + IgnoreKey + `', '` + RuleTagsConfirm + `')] if the script asks by itself`})
		}
	}
	return out
}

// hasConfirm reports whether the recipe has the [confirm] attribute, which
// the dump writes as {"confirm": null} or {"confirm": "Really?"}.
func hasConfirm(r recipe) bool {
	if hasAttribute(r, "confirm") {
		return true
	}
	for _, raw := range r.Attributes {
		var s string
		if json.Unmarshal(raw, &s) == nil && s == "confirm" {
			return true
		}
	}
	return false
}
