package check

import (
	"fmt"
	"regexp"
	"strings"
)

// RuleDocMultiline is the rule about comment blocks used as a recipe doc.
const RuleDocMultiline = "doc-multiline-comment"

// docMultilineFindings looks for recipes whose doc is a comment block of two or
// more lines. just uses only the last line of such a block as the doc, so
// "just --list" shows a fragment of the text. The dump in JSON keeps only that
// last line, so the full block is read from the formatted text that just
// --dump prints for the file the recipe is in.
//
// sources maps a module path ("" for the root justfile, a module name for a
// top-level module) to that text. A recipe whose file has no text is skipped.
// A recipe with an explicit [doc(...)] attribute is fine: it overrides the
// comment.
func docMultilineFindings(items []item, sources map[string]string) []Finding {
	var out []Finding
	for _, it := range items {
		module := ""
		if !it.TopLevel {
			module = it.Group
		}
		if strings.Count(it.Path, " ") > 1 {
			continue // a nested module: its text is not read
		}
		text, ok := sources[module]
		if !ok || hasExplicitDoc(it.R) {
			continue
		}
		if n := commentLines(text, it.Name); n >= 2 {
			out = append(out, Finding{RuleDocMultiline, Warning, "just " + it.Path,
				fmt.Sprintf("the doc is a comment block of %d lines, and just uses only the last line as the recipe doc", n),
				"add [doc('one-line description')] above the recipe, and keep the longer text in a runbook"})
		}
	}
	return out
}

func hasExplicitDoc(r recipe) bool {
	return hasAttribute(r, "doc")
}

// commentLines counts the consecutive comment lines directly above a recipe,
// looking past its attribute lines. The text is the formatted output of
// just --dump, where each attribute is on its own line and a recipe header
// starts in the first column.
func commentLines(text, name string) int {
	header := regexp.MustCompile(`^@?` + regexp.QuoteMeta(name) + `(\s[^:]*)?:([^=]|$)`)
	lines := strings.Split(text, "\n")
	for i, l := range lines {
		if !header.MatchString(l) {
			continue
		}
		j := i - 1
		for j >= 0 && strings.HasPrefix(strings.TrimSpace(lines[j]), "[") {
			j-- // attributes sit between the comment and the recipe
		}
		n := 0
		for j >= 0 && strings.HasPrefix(lines[j], "#") {
			n++
			j--
		}
		return n
	}
	return 0
}
