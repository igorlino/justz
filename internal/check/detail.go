package check

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Thresholds of the detail rules, in characters of the trimmed text. They are
// named in the fix text.
const (
	MaxDetailLength      = 600 // detail-long: over this is long
	MaxDetailUnbroken    = 300 // detail-no-paragraphs: over this needs a blank line
	minRestatementTokens = 4   // detail-restates-desc: a shorter desc is not compared
)

const (
	detailLongFix     = "keep a detail to 600 characters or fewer: cut it, or move what stays to the skill or a doc, or add [metadata('" + IgnoreKey + "', '" + RuleDetailLong + "')] to the recipe if the length is needed"
	detailParasFix    = "split a detail of more than 300 characters into paragraphs with a blank line, or add [metadata('" + IgnoreKey + "', '" + RuleDetailNoParagraphs + "')] to the recipe"
	detailRestatesFix = "start the detail with what the desc does not say, not with the desc again (a desc of four words or more, followed by the same words), or add [metadata('" + IgnoreKey + "', '" + RuleDetailRestatesDesc + "')] to the recipe"
)

// detailShapeFindings applies the three advice rules about the shape of a
// detail: too long, long and in one block, and opening with the desc again.
// All are warnings. A detail is the longer text for one command, and a long one
// is as bad as a long desc.
func detailShapeFindings(e registryEntry, subject string, ignored map[string]bool) []Finding {
	text := e.DetailText
	if text == "" {
		return nil
	}
	n := utf8.RuneCountInString(text)
	var out []Finding
	if n > MaxDetailLength && !ignored[RuleDetailLong] {
		out = append(out, Finding{Rule: RuleDetailLong, Severity: Warning, Subject: subject,
			Message: fmt.Sprintf("detail is %d characters long (over %d)", n, MaxDetailLength), Fix: detailLongFix})
	}
	if n > MaxDetailUnbroken && !hasParagraphBreak(text) && !ignored[RuleDetailNoParagraphs] {
		out = append(out, Finding{Rule: RuleDetailNoParagraphs, Severity: Warning, Subject: subject,
			Message: fmt.Sprintf("detail is %d characters long (over %d) with no blank line between paragraphs", n, MaxDetailUnbroken), Fix: detailParasFix})
	}
	if restatesDesc(e.Desc, text) && !ignored[RuleDetailRestatesDesc] {
		out = append(out, Finding{Rule: RuleDetailRestatesDesc, Severity: Warning, Subject: subject,
			Message: "detail opens by repeating the desc", Fix: detailRestatesFix})
	}
	return out
}

// hasParagraphBreak reports whether the text has a blank line: two line breaks
// with only white space between them.
func hasParagraphBreak(s string) bool {
	lines := strings.Split(s, "\n")
	for i := 1; i < len(lines)-1; i++ {
		if strings.TrimSpace(lines[i]) == "" {
			return true
		}
	}
	return false
}

// restatesDesc reports whether the words of the desc, lowercased, without
// punctuation and with a trailing s removed, are the first words of the detail.
// It is a prefix test and not a similarity score on purpose: it has no
// threshold to tune, an author can see why it fired, and a detail that only
// starts with the same verb ("Run the tests ..." after "Run the linter") does
// not match. The stem makes "Run the tests" and "Runs the tests in parallel"
// the same words. It misses a paraphrase, which is accepted. A desc of fewer
// than four words is not compared: a short desc is a prefix of too many honest
// details ("Deploy to production" before the new information).
func restatesDesc(desc, detail string) bool {
	d := tokens(desc)
	if len(d) < minRestatementTokens {
		return false
	}
	t := tokens(detail)
	if len(t) < len(d) {
		return false
	}
	for i := range d {
		if d[i] != t[i] {
			return false
		}
	}
	return true
}

// tokens lowercases s and splits it into runs of letters and digits. A word of
// more than three letters loses a trailing s, so a plural and a third-person
// verb match their base. Nothing else is stemmed.
func tokens(s string) []string {
	words := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	for i, w := range words {
		if utf8.RuneCountInString(w) > 3 && strings.HasSuffix(w, "s") {
			words[i] = strings.TrimSuffix(w, "s")
		}
	}
	return words
}
