package check

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// IgnorableRules lists the rules of check that a recipe may silence with
// [metadata('justz-ignore', '<rule>')], the attribute justz lint uses.
var IgnorableRules = []string{RuleVolatileText, RuleDetailFlagNoRow, RuleDetailEnvNoRow, RuleDetailLong, RuleDetailNoParagraphs, RuleDetailRestatesDesc, RuleTagsConfirm}

// IgnoreKey is the first value of that attribute.
const IgnoreKey = "justz-ignore"

// volatile patterns are few on purpose: each is a kind of text that goes stale
// when the recipe changes, and none matches ordinary prose.
var volatilePatterns = []*regexp.Regexp{
	// A status word.
	regexp.MustCompile(`\b(TODO|FIXME|WIP)\b`),
	regexp.MustCompile(`(?i)\b(placeholder|not implemented|coming soon)\b`),
	// A count of checks, tests, steps, jobs, stages or phases, singular or
	// plural, with a range as one hit: "11 checks", "six steps", "3-4 steps".
	regexp.MustCompile(`(?i)\b((?:\d+(?:\s?[-\x{2013}]\s?\d+)?|one|two|three|four|five|six|seven|eight|nine|ten|eleven|twelve|thirteen|fourteen|fifteen|sixteen|seventeen|eighteen|nineteen|twenty)\s+(?:checks?|tests?|steps?|jobs?|stages?|phases?))\b`),
	// A price: $4.99, $10 or more, or an amount with a unit ($5k, $2M, $5 USD).
	// A lone $1 is a shell positional, not a price.
	regexp.MustCompile(`(\$\d+[.,]\d{2})\b`),
	regexp.MustCompile(`(\$\d{2,})\b`),
	regexp.MustCompile(`(\$\d+(?:\.\d+)?\s?(?:k|M|USD|EUR|GBP))\b`),
	// A date: 2026-05 or 2026-05-17 (not inside a version such as 1.2026-10),
	// or a month name with a year.
	regexp.MustCompile(`(?:^|[^\d.\-])(\d{4}-\d{2}(?:-\d{2})?)(?:[^\d.]|\.?$)`),
	regexp.MustCompile(`\b((?:Jan(?:uary)?|Feb(?:ruary)?|Mar(?:ch)?|Apr(?:il)?|May|Jun(?:e)?|Jul(?:y)?|Aug(?:ust)?|Sep(?:t(?:ember)?)?|Oct(?:ober)?|Nov(?:ember)?|Dec(?:ember)?)\.?(?: \d{1,2},?)? \d{4})\b`),
}

// detailFlag finds a flag written as one in prose: at the start of the text or
// after a space, an opening bracket or a quote, so a word such as non-stop, a
// path or a URL is not read as a flag. A short flag is one letter.
var detailFlag = regexp.MustCompile("(?:^|[\\s(\\[`'\"])(--[A-Za-z][A-Za-z0-9-]*|-[A-Za-z])(?:[=\\s,.;:)\\]`'\"]|$)")

// envToken is a candidate environment variable name in prose: upper case
// letters, digits and underscores, three characters or more.
var envToken = regexp.MustCompile(`[A-Z][A-Z0-9_]+[A-Z0-9]`)

// detailEnvNames lists the environment variables a detail names, in order. A
// name counts when it has an underscore (NO_COLOR, API_TOKEN), or when it is
// written as a variable: after $ or ${, after export, or before = (HOME=/x).
// An acronym (HTTP, JSON, TLS) therefore never counts, nor a word such as
// README, unless it is written as a variable. A token inside another word, after
// a hyphen, slash, dot, ? or & (a flag, a path, a dotted name, a URL query key)
// or before a dot and a letter or a slash (a file name) is skipped, unless it is
// written as $NAME or ${NAME}, so $HOME/bin is reported. Prose constants with an
// underscore (UTF_8, SHA_256, READ_ONLY) can fire; justz-ignore is the opt-out.
// A name of fewer than three characters ($CI) is not caught.
func detailEnvNames(s string) []string {
	var out []string
	seen := map[string]bool{}
	isWord := func(r rune) bool { return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) }
	for _, m := range envToken.FindAllStringIndex(s, -1) {
		start, end := m[0], m[1]
		name := s[start:end]
		if seen[name] {
			continue
		}
		before, after := s[:start], s[end:]
		if r, _ := utf8.DecodeLastRuneInString(before); start > 0 && isWord(r) {
			continue
		}
		if r, _ := utf8.DecodeRuneInString(after); end < len(s) && isWord(r) {
			continue
		}
		dollar := strings.HasSuffix(before, "$") || strings.HasSuffix(before, "${")
		asVar := dollar || strings.HasSuffix(before, "export ") || strings.HasPrefix(after, "=") && !strings.HasPrefix(after, "==")
		if !dollar {
			if strings.HasSuffix(before, "-") || strings.HasSuffix(before, "/") || strings.HasSuffix(before, ".") || strings.HasSuffix(before, "?") || strings.HasSuffix(before, "&") {
				continue
			}
			if strings.HasPrefix(after, "/") {
				continue
			}
			if r, _ := utf8.DecodeRuneInString(after[min(1, len(after)):]); strings.HasPrefix(after, ".") && isWord(r) {
				continue
			}
		}
		if !asVar && !strings.Contains(name, "_") {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	return out
}

// adviceRules reports volatile text in desc and detail, and flags that detail
// names without an argument row. Both are warnings. A recipe silences either
// with [metadata('justz-ignore', '<rule>')].
func adviceRules(entries []registryEntry, items []item) []Finding {
	byPath := map[string]item{}
	for _, it := range items {
		if it.nested() {
			continue // nested modules have no registry group, and would shadow the parent recipe
		}
		byPath[pathOf(it.Group, it.Name)] = it
	}
	var out []Finding
	for _, e := range entries {
		ignored := map[string]bool{}
		if it, ok := byPath[pathOf(e.Group, e.Key)]; ok && !e.Extra {
			ignored = ignoredCheckRules(it.R)
		}
		subject := fmt.Sprintf("%s.%s", e.Group, e.Key)

		if !ignored[RuleVolatileText] {
			for _, f := range []struct{ field, text string }{{"desc", e.Desc}, {"detail", e.Detail}} {
				if hits := volatileMatches(f.text); len(hits) > 0 {
					out = append(out, Finding{Rule: RuleVolatileText, Severity: Warning, Subject: subject, Message: fmt.Sprintf("%s has text that goes stale: %s", f.field, strings.Join(hits, ", ")), Fix: "say what the command does, not a status, count, price or date, or add [metadata('" + IgnoreKey + "', '" + RuleVolatileText + "')] to the recipe if the text is stable"})
				}
			}
		}
		out = append(out, detailShapeFindings(e, subject, ignored)...)
		if !ignored[RuleDetailEnvNoRow] && e.Detail != "" {
			rows := map[string]bool{}
			for _, n := range e.EnvNames {
				rows[n] = true
			}
			for _, n := range detailEnvNames(e.Detail) {
				if !rows[n] {
					out = append(out, Finding{Rule: RuleDetailEnvNoRow, Severity: Warning, Subject: subject, Message: fmt.Sprintf("detail names the environment variable %s, but no env row declares it", n), Fix: "add an env row for the variable, or reword the detail (a prose constant such as UTF_8 is a false hit), or add [metadata('" + IgnoreKey + "', '" + RuleDetailEnvNoRow + "')] to the recipe"})
				}
			}
		}
		if !ignored[RuleDetailFlagNoRow] && e.Detail != "" {
			declared, variadic := declaredFlags(e)
			if variadic {
				continue
			}
			seen := map[string]bool{}
			for _, m := range detailFlag.FindAllStringSubmatch(e.Detail, -1) {
				if f := m[1]; !declared[f] && !seen[f] {
					seen[f] = true
					out = append(out, Finding{Rule: RuleDetailFlagNoRow, Severity: Warning, Subject: subject, Message: fmt.Sprintf("detail names the flag %s, but no argument row declares it", f), Fix: "add an argument row for the flag, or reword the detail, or add [metadata('" + IgnoreKey + "', '" + RuleDetailFlagNoRow + "')] to the recipe"})
				}
			}
		}
	}
	return out
}

// volatileMatches lists every distinct hit of the patterns, quoted, in the
// order they appear, so that fixing one does not reveal the next.
func volatileMatches(s string) []string {
	type hit struct {
		at   int
		text string
	}
	var found []hit
	seen := map[string]bool{}
	for _, re := range volatilePatterns {
		for _, m := range re.FindAllStringSubmatchIndex(s, -1) {
			text := strings.TrimSpace(s[m[2]:m[3]])
			if !seen[text] {
				seen[text] = true
				found = append(found, hit{m[2], text})
			}
		}
	}
	sort.SliceStable(found, func(i, j int) bool { return found[i].at < found[j].at })
	var out []string
	for _, h := range found {
		out = append(out, fmt.Sprintf("%q", h.text))
	}
	return out
}

// ignoredCheckRules reads the [metadata('justz-ignore', rule...)] attributes of
// a recipe and keeps the rules of check. Unknown ids are the business of lint.
func ignoredCheckRules(r recipe) map[string]bool {
	out := map[string]bool{}
	for _, raw := range r.Attributes {
		var obj struct {
			Metadata []string `json:"metadata"`
		}
		if json.Unmarshal(raw, &obj) != nil || len(obj.Metadata) < 2 || obj.Metadata[0] != IgnoreKey {
			continue
		}
		for _, id := range obj.Metadata[1:] {
			out[id] = true
		}
	}
	return out
}
