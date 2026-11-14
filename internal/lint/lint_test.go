package lint

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func body(n int) string {
	return "[" + strings.TrimSuffix(strings.Repeat(`["echo"],`, n), ",") + "]"
}

func recipeJSON(n int, attrs string) string {
	return fmt.Sprintf(`{"attributes":[%s],"body":%s}`, attrs, body(n))
}

func run(t *testing.T, d string) Result {
	t.Helper()
	res, err := Run([]byte(d))
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func rules(res Result) []string {
	var out []string
	for _, f := range res.Findings {
		out = append(out, f.Rule+" "+f.Subject)
	}
	return out
}

func TestRecipeLongThreshold(t *testing.T) {
	d := fmt.Sprintf(`{"recipes":{"ok":%s,"long":%s},"modules":{}}`, recipeJSON(40, ""), recipeJSON(41, ""))
	res := run(t, d)
	if got := strings.Join(rules(res), ","); got != "recipe-long just long" {
		t.Errorf("findings = %q", got)
	}
	if res.Checked != 2 {
		t.Errorf("checked = %d", res.Checked)
	}
	f := res.Findings[0]
	if f.Severity != "warning" || !strings.Contains(f.Message, "41 lines (limit 40)") {
		t.Errorf("finding = %+v", f)
	}
}

func TestRecipeLongCountsPrivateDefaultAndNestedModules(t *testing.T) {
	d := fmt.Sprintf(`{"recipes":{"_helper":%s,"default":%s},"modules":{"ops":{"recipes":{"deploy":%s},"modules":{"db":{"recipes":{"restore":%s},"modules":{}}}},"modules":{}}}`,
		recipeJSON(50, ""), recipeJSON(50, ""), recipeJSON(50, ""), recipeJSON(50, ""))
	res := run(t, d)
	want := "recipe-long just _helper,recipe-long just default,recipe-long just ops db restore,recipe-long just ops deploy"
	if got := strings.Join(rules(res), ","); got != want {
		t.Errorf("findings = %q\nwant       %q", got, want)
	}
}

func TestIgnoreSilencesTheRule(t *testing.T) {
	for _, attrs := range []string{
		`{"metadata":["justz-ignore","recipe-long"]}`,
		`{"group":"core"},{"metadata":["other","x"]},{"metadata":["justz-ignore","recipe-long"]}`,
	} {
		res := run(t, fmt.Sprintf(`{"recipes":{"long":%s},"modules":{}}`, recipeJSON(60, attrs)))
		if len(res.Findings) != 0 {
			t.Errorf("attrs %s: findings = %v", attrs, rules(res))
		}
	}
}

func TestUnknownIgnoreValueIsAFindingAndSilencesNothing(t *testing.T) {
	res := run(t, fmt.Sprintf(`{"recipes":{"long":%s},"modules":{}}`, recipeJSON(60, `{"metadata":["justz-ignore","nope"]}`)))
	if got := strings.Join(rules(res), ","); got != "ignore-unknown-rule just long,recipe-long just long" {
		t.Errorf("findings = %q", got)
	}
}

func TestIgnoreWithoutARuleIsAFinding(t *testing.T) {
	res := run(t, fmt.Sprintf(`{"recipes":{"a":%s},"modules":{}}`, recipeJSON(1, `{"metadata":["justz-ignore"]}`)))
	if len(res.Findings) != 1 || res.Findings[0].Rule != RuleIgnoreUnknown || !strings.Contains(res.Findings[0].Message, "names no rule") {
		t.Errorf("findings = %+v", res.Findings)
	}
}

func TestIgnoreKeyIsExact(t *testing.T) {
	// JUSTZ-IGNORE is not the key: the attribute is plain metadata and is left alone.
	res := run(t, fmt.Sprintf(`{"recipes":{"long":%s},"modules":{}}`, recipeJSON(60, `{"metadata":["JUSTZ-IGNORE","recipe-long"]}`)))
	if got := strings.Join(rules(res), ","); got != "recipe-long just long" {
		t.Errorf("findings = %q", got)
	}
}

func TestRunRejectsBadJSON(t *testing.T) {
	if _, err := Run([]byte("nope")); err == nil {
		t.Error("want an error")
	}
}

func TestNoFindingsIsAnEmptyListNotNull(t *testing.T) {
	res := run(t, `{"recipes":{},"modules":{}}`)
	if res.Findings == nil {
		t.Error("Findings must be an empty slice so JSON prints []")
	}
}

func TestIgnoreWorksInNestedModulesAndAddsUpAcrossAttributes(t *testing.T) {
	both := `{"metadata":["justz-ignore","recipe-long"]},{"metadata":["justz-ignore","recipe-long"]}`
	d := fmt.Sprintf(`{"recipes":{},"modules":{"a":{"recipes":{"x":%s},"modules":{"b":{"recipes":{"y":%s,"z":%s},"modules":{}}}}}}`,
		recipeJSON(60, `{"metadata":["justz-ignore","recipe-long"]}`), recipeJSON(60, both), recipeJSON(60, ""))
	res := run(t, d)
	if got := strings.Join(rules(res), ","); got != "recipe-long just a b z" {
		t.Errorf("findings = %q", got)
	}
}

func TestLineIsOmittedFromJSONWhenARuleHasNone(t *testing.T) {
	res := run(t, fmt.Sprintf(`{"recipes":{"long":%s},"modules":{}}`, recipeJSON(60, "")))
	b, err := json.Marshal(res.Findings[0])
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), `"line"`) {
		t.Errorf("line should be omitted: %s", b)
	}
}
