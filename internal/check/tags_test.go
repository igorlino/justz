package check

import (
	"strings"
	"testing"
)

func tagsRun(t *testing.T, tags string, attrs []map[string]any) Result {
	t.Helper()
	return adviceRun(t, `{"build":{"desc":"Build","tags":`+tags+`}}`, attrs)
}

func confirmAttr() []map[string]any { return []map[string]any{{"confirm": nil}} }

func finding(res Result, rule string) *Finding {
	for i := range res.Findings {
		if res.Findings[i].Rule == rule {
			return &res.Findings[i]
		}
	}
	return nil
}

func TestTagsCleanGiveNoFinding(t *testing.T) {
	res := tagsRun(t, `["idempotent","read-only","custom-word"]`, nil)
	if has(res, RuleTags, "") || has(res, RuleTagsConfirm, "") || has(res, RuleRegistryShape, "") {
		t.Errorf("findings = %v", rules(res))
	}
}

func TestTagsContradictionIsAWarning(t *testing.T) {
	for _, bad := range []string{"destructive", "spends", "production"} {
		f := finding(tagsRun(t, `["read-only","`+bad+`"]`, nil), RuleTags)
		if f == nil || f.Severity != Warning || f.Subject != "core.build" || !strings.Contains(f.Message, bad) {
			t.Errorf("%s: %+v", bad, f)
		}
	}
	if has(tagsRun(t, `["destructive","spends","production"]`, nil), RuleTags, "") {
		t.Errorf("without read-only there is no contradiction")
	}
}

func TestTagsDuplicateIsAWarningOnce(t *testing.T) {
	res := tagsRun(t, `["spends","spends","spends"]`, nil)
	n := 0
	for _, f := range res.Findings {
		if f.Rule == RuleTags {
			n++
			if f.Severity != Warning || !strings.Contains(f.Message, `"spends"`) {
				t.Errorf("finding = %+v", f)
			}
		}
	}
	if n != 1 {
		t.Errorf("findings = %d, want 1", n)
	}
}

func TestTagsOutsideTheVocabularyAreNotReported(t *testing.T) {
	if res := tagsRun(t, `["Whatever","x y"]`, nil); len(res.Findings) != 0 {
		t.Errorf("findings = %v", rules(res))
	}
}

func TestTagsWrongTypeIsRegistryShape(t *testing.T) {
	if !has(tagsRun(t, `["a", 1]`, nil), RuleRegistryShape, "core.build") || !has(adviceRun(t, `{"build":{"desc":"B","tags":"read-only"}}`, nil), RuleRegistryShape, "core.build") {
		t.Errorf("wrong tags types must be registry-shape")
	}
}

func TestTagsConfirmBothDirections(t *testing.T) {
	// [confirm] without the tag.
	f := finding(tagsRun(t, `["spends"]`, confirmAttr()), RuleTagsConfirm)
	if f == nil || f.Severity != Warning || !strings.Contains(f.Message, "[confirm]") || !strings.Contains(f.Fix, "justz-ignore") {
		t.Errorf("confirm without tag: %+v", f)
	}
	// No tags field at all still reports it.
	if res := adviceRun(t, `{"build":{"desc":"Build"}}`, confirmAttr()); !has(res, RuleTagsConfirm, "core.build") {
		t.Errorf("no tags: %v", rules(res))
	}
	// The tag without [confirm].
	f = finding(tagsRun(t, `["asks-first"]`, nil), RuleTagsConfirm)
	if f == nil || f.Severity != Warning || !strings.Contains(f.Message, "no [confirm]") {
		t.Errorf("tag without confirm: %+v", f)
	}
	// Misses: both, or neither. A confirm with a prompt counts.
	for _, c := range []struct {
		tags  string
		attrs []map[string]any
	}{
		{`["asks-first"]`, confirmAttr()},
		{`["asks-first"]`, []map[string]any{{"confirm": "Really?"}}},
		{`["asks-first"]`, []map[string]any{{"confirm": nil}, {"group": "core"}}},
		{`[]`, nil},
	} {
		if res := tagsRun(t, c.tags, c.attrs); has(res, RuleTagsConfirm, "") {
			t.Errorf("%s %v: %v", c.tags, c.attrs, rules(res))
		}
	}
}

func TestTagsConfirmIgnore(t *testing.T) {
	for name, c := range map[string]struct {
		tags  string
		attrs []map[string]any
	}{
		"confirm without tag": {`[]`, append(confirmAttr(), ignoreAttr(RuleTagsConfirm)...)},
		"tag without confirm": {`["asks-first"]`, ignoreAttr(RuleTagsConfirm)},
	} {
		if has(tagsRun(t, c.tags, c.attrs), RuleTagsConfirm, "") {
			t.Errorf("%s: ignored rule reported", name)
		}
	}
	if !has(tagsRun(t, `["asks-first"]`, ignoreAttr(RuleVolatileText)), RuleTagsConfirm, "") {
		t.Errorf("ignoring another rule must not silence this one")
	}
}

func TestTagsOnExtraEntryHaveNoConfirmRule(t *testing.T) {
	dump := dumpOf(t, map[string]any{"build": rc("Build", nil)}, nil)
	s := `{"groups":{"core":{"commands":{"build":{"desc":"Build"}},"extra":{"alias":{"desc":"A","tags":["asks-first","read-only","spends","read-only"]}}}}}`
	res := run(t, Input{Dump: dump, Variant: "rich", Registry: parse(t, s), RegistrySource: []byte(s)})
	if has(res, RuleTagsConfirm, "") {
		t.Errorf("extra entry has no recipe: %v", rules(res))
	}
	if !has(res, RuleTags, "core.alias") {
		t.Errorf("contradiction and duplicate still apply to extra: %v", rules(res))
	}
}

func TestTagsTextIsCleaned(t *testing.T) {
	res := tagsRun(t, `["read-only","read-only\u001b[31m","spends\u001b]0;x\u0007","spends\u001b]0;x\u0007"]`, nil)
	for _, f := range res.Findings {
		if strings.ContainsAny(f.Message+f.Subject+f.Fix, "\x1b\x07") {
			t.Errorf("control character in %+v", f)
		}
	}
}

func nestedTagsRun(t *testing.T, parent, nested []map[string]any) Result {
	t.Helper()
	dump := dumpOf(t, map[string]any{},
		map[string]any{"tool": map[string]any{
			"recipes": map[string]any{"x": rc("X", parent)},
			"modules": map[string]any{"sub": module(map[string]any{"x": rc("X", nested)})},
		}})
	s := `{"groups":{"tool":{"commands":{"x":{"desc":"X","tags":["asks-first"]}}}}}`
	return run(t, Input{Dump: dump, Variant: "rich", Registry: parse(t, s), RegistrySource: []byte(s)})
}

func TestTagsConfirmIgnoresNestedModuleRecipeWithTheSameName(t *testing.T) {
	// The parent has [confirm] and the tag; the nested recipe has neither and must not be read as the parent.
	if res := nestedTagsRun(t, confirmAttr(), nil); has(res, RuleTagsConfirm, "") {
		t.Errorf("nested recipe shadowed the parent: %v", rules(res))
	}
	// An ignore on the nested recipe must not silence the parent.
	res := nestedTagsRun(t, nil, ignoreAttr(RuleTagsConfirm))
	if !has(res, RuleTagsConfirm, "tool.x") {
		t.Errorf("ignore on the nested recipe silenced the parent: %v", rules(res))
	}
	// An ignore on the parent still applies.
	if res := nestedTagsRun(t, ignoreAttr(RuleTagsConfirm), nil); has(res, RuleTagsConfirm, "") {
		t.Errorf("ignore on the parent not applied: %v", rules(res))
	}
}

func TestTagsBlankTagsAreNotDuplicates(t *testing.T) {
	if res := tagsRun(t, `["", " ", "", "\t"]`, nil); has(res, RuleTags, "") {
		t.Errorf("blank tags reported: %v", rules(res))
	}
}
