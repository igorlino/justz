package check

import (
	"strings"
	"testing"
)

func exitsCheck(t *testing.T, exits string) Result {
	t.Helper()
	dump := dumpOf(t, map[string]any{"build": rc("B", nil)}, nil)
	reg := `{"groups":{"core":{"commands":{"build":{"desc":"B","exits":` + exits + `}}}}}`
	return run(t, Input{Dump: dump, Variant: "rich", Registry: parse(t, reg), RegistrySource: []byte(reg)})
}

func TestExitsKeysThatAreCodesGiveNoFinding(t *testing.T) {
	res := exitsCheck(t, `{"0":"ok","1":"error","2":"usage","10":"x","255":"max","127":"not found"}`)
	if has(res, RuleExits, "") || has(res, RuleRegistryShape, "") {
		t.Errorf("findings = %v", rules(res))
	}
}

func TestExitsBadKeysAreErrorsNamingTheKey(t *testing.T) {
	for _, key := range []string{"x", "", "256", "-1", "+1", "01", "1 ", "1.5", "0x1", "1000"} {
		res := exitsCheck(t, `{"0":"ok","`+key+`":"m"}`)
		var f *Finding
		for i := range res.Findings {
			if res.Findings[i].Rule == RuleExits {
				f = &res.Findings[i]
			}
		}
		if f == nil || f.Severity != Error || f.Subject != "core.build" || !strings.Contains(f.Message, `"`+key+`"`) || !strings.Contains(f.Fix, "0 to 255") {
			t.Errorf("key %q: %+v", key, f)
		}
	}
}

func TestExitsWrongTypesAreTheShapeRule(t *testing.T) {
	res := exitsCheck(t, `{"0":3}`)
	if !has(res, RuleRegistryShape, "core.build") || has(res, RuleExits, "") {
		t.Errorf("a number as a meaning: %v", rules(res))
	}
	if res := exitsCheck(t, `["0"]`); !has(res, RuleRegistryShape, "core.build") {
		t.Errorf("an array: %v", rules(res))
	}
}

func TestExitsBlankMeaningIsAWarning(t *testing.T) {
	for _, meaning := range []string{`""`, `"   "`, `" \t "`} {
		res := exitsCheck(t, `{"0":"ok","2":`+meaning+`}`)
		var f *Finding
		for i := range res.Findings {
			if res.Findings[i].Rule == RuleExits {
				f = &res.Findings[i]
			}
		}
		if f == nil || f.Severity != Warning || f.Subject != "core.build" || !strings.Contains(f.Message, `blank meaning for "2"`) {
			t.Errorf("meaning %s: %+v", meaning, f)
		}
		if res.Count(Error) != 0 {
			t.Errorf("a blank meaning must not be an error: %v", rules(res))
		}
	}
	if has(exitsCheck(t, `{"0":"ok"}`), RuleExits, "") {
		t.Errorf("a meaning with text must not be reported")
	}
}
