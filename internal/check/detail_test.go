package check

import (
	"strings"
	"testing"
)

func detailRun(t *testing.T, desc, detail string, attrs []map[string]any) Result {
	t.Helper()
	dump := dumpOf(t, map[string]any{"build": rc(desc, attrs)}, nil)
	reg := `{"groups":{"core":{"commands":{"build":{"desc":` + jsonString(desc) + `,"detail":` + jsonString(detail) + `}}}}}`
	return run(t, Input{Dump: dump, Variant: "rich", Registry: parse(t, reg), RegistrySource: []byte(reg)})
}

func jsonString(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`)
	return `"` + r.Replace(s) + `"`
}

func hasRule(r Result, rule string) *Finding {
	for i := range r.Findings {
		if r.Findings[i].Rule == rule {
			return &r.Findings[i]
		}
	}
	return nil
}

// words makes a text of exactly n characters with no blank line.
func chars(n int) string { return strings.Repeat("a", n) }

func TestDetailLongBoundary(t *testing.T) {
	// Paragraphs, so that only the length rule can fire.
	para := func(n int) string { return chars(n/2) + "\n\n" + chars(n-n/2-2) }
	if hasRule(detailRun(t, "Build", para(600), nil), RuleDetailLong) != nil {
		t.Errorf("exactly 600 characters is not over the limit")
	}
	f := hasRule(detailRun(t, "Build", para(601), nil), RuleDetailLong)
	if f == nil || f.Severity != Warning || f.Subject != "core.build" || !strings.Contains(f.Message, "601 characters") || !strings.Contains(f.Fix, "600") {
		t.Errorf("finding = %+v", f)
	}
	if hasRule(detailRun(t, "Build", para(601), ignoreAttr(RuleDetailLong)), RuleDetailLong) != nil {
		t.Errorf("ignored rule reported")
	}
	if hasRule(detailRun(t, "Build", para(601), ignoreAttr(RuleDetailNoParagraphs)), RuleDetailLong) == nil {
		t.Errorf("ignoring another rule must not silence this one")
	}
	// Characters, not bytes.
	if hasRule(detailRun(t, "Build", strings.Repeat("é", 250)+"\n\n"+strings.Repeat("é", 250), nil), RuleDetailLong) != nil {
		t.Errorf("502 characters counted as bytes")
	}
}

func TestDetailNoParagraphsBoundary(t *testing.T) {
	if hasRule(detailRun(t, "Build", chars(300), nil), RuleDetailNoParagraphs) != nil {
		t.Errorf("exactly 300 characters is not over the limit")
	}
	f := hasRule(detailRun(t, "Build", chars(301), nil), RuleDetailNoParagraphs)
	if f == nil || f.Severity != Warning || !strings.Contains(f.Message, "301 characters") || !strings.Contains(f.Fix, "300") {
		t.Errorf("finding = %+v", f)
	}
	// A blank line, with or without spaces on it, is a paragraph break; one
	// newline is not.
	for _, ok := range []string{chars(200) + "\n\n" + chars(200), chars(200) + "\n  \n" + chars(200)} {
		if hasRule(detailRun(t, "Build", ok, nil), RuleDetailNoParagraphs) != nil {
			t.Errorf("a paragraph break was missed in %q", ok[:5])
		}
	}
	if hasRule(detailRun(t, "Build", chars(200)+"\n"+chars(200), nil), RuleDetailNoParagraphs) == nil {
		t.Errorf("a single line break is not a paragraph break")
	}
	if hasRule(detailRun(t, "Build", chars(301), ignoreAttr(RuleDetailNoParagraphs)), RuleDetailNoParagraphs) != nil {
		t.Errorf("ignored rule reported")
	}
	// Leading and trailing blank lines are not a break between paragraphs.
	if hasRule(detailRun(t, "Build", "\n\n"+chars(301)+"\n\n", nil), RuleDetailNoParagraphs) == nil {
		t.Errorf("edge blank lines must not count")
	}
}

func TestDetailRestatesDesc(t *testing.T) {
	hits := [][2]string{
		{"Deploy the report to the cloud", "Deploy the report to the cloud and reindex it, then send the roll-up."},
		{"Deploy the report to the cloud.", "deploy the REPORT to the cloud, and more"},
		{"Run the tests (fast)", "Run the tests fast: all of them."},
		{"Deploy the report to the cloud", "Deploy the report to the cloud"},
		{"Run all the tests", "Runs all the tests in parallel on four workers."}, // third person
		{"Build the release images", "Builds the release image, then tags it."},  // plural and singular
	}
	for _, h := range hits {
		f := hasRule(detailRun(t, h[0], h[1], nil), RuleDetailRestatesDesc)
		if f == nil || f.Severity != Warning || !strings.Contains(f.Fix, "four words") {
			t.Errorf("%q / %q: %+v", h[0], h[1], f)
		}
	}
	misses := [][2]string{
		{"Run the linter", "Run the tests first, then the linter."},                // the same verb only
		{"Deploy the report", "Builds the day directory, then deploy the report."}, // later, not an opening
		{"Build it", "Build it with the flags below."},                             // a desc of two words is not compared
		{"Deploy the report to the cloud", "Deploy the report to"},                 // the detail is shorter than the desc
		{"Deploy the report to the cloud", "Uploads and reindexes."},
		{"Deploy the report to the cloud", ""},
	}
	for _, m := range misses {
		if hasRule(detailRun(t, m[0], m[1], nil), RuleDetailRestatesDesc) != nil {
			t.Errorf("%q / %q must not fire", m[0], m[1])
		}
	}
	if hasRule(detailRun(t, "Deploy the report to the cloud", "Deploy the report to the cloud and more", ignoreAttr(RuleDetailRestatesDesc)), RuleDetailRestatesDesc) != nil {
		t.Errorf("ignored rule reported")
	}
}

func TestDetailShapeCleanDetailHasNoFinding(t *testing.T) {
	d := "Builds the day directory.\n\n" + strings.Repeat("Uploads and reindexes it. ", 10)
	res := detailRun(t, "Deploy the report", d, nil)
	for _, r := range []string{RuleDetailLong, RuleDetailNoParagraphs, RuleDetailRestatesDesc} {
		if hasRule(res, r) != nil {
			t.Errorf("%s fired on a good detail: %v", r, rules(res))
		}
	}
}

func TestDetailRulesAreIgnorableAndListed(t *testing.T) {
	for _, r := range []string{RuleDetailLong, RuleDetailNoParagraphs, RuleDetailRestatesDesc} {
		found := false
		for _, i := range IgnorableRules {
			found = found || i == r
		}
		if !found {
			t.Errorf("%s is not in IgnorableRules", r)
		}
	}
}
