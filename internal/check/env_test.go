package check

import (
	"strings"
	"testing"
)

func envCheck(t *testing.T, env string) Result {
	t.Helper()
	dump := dumpOf(t, map[string]any{"build": rc("B", nil)}, nil)
	reg := `{"groups":{"core":{"commands":{"build":{"desc":"B","env":` + env + `}}}}}`
	return run(t, Input{Dump: dump, Variant: "rich", Registry: parse(t, reg), RegistrySource: []byte(reg)})
}

func envFinding(res Result) *Finding {
	for i := range res.Findings {
		if res.Findings[i].Rule == RuleEnv {
			return &res.Findings[i]
		}
	}
	return nil
}

func TestEnvGoodRowsGiveNoFinding(t *testing.T) {
	res := envCheck(t, `[{"name":"STORAGE_HOST","desc":"d","required":true},{"name":"DEPLOY_X1","desc":"d","default":"unset"}]`)
	if has(res, RuleEnv, "") || has(res, RuleRegistryShape, "") {
		t.Errorf("findings = %v", rules(res))
	}
}

func TestEnvRowWithoutNameIsAnError(t *testing.T) {
	for _, row := range []string{`{"desc":"d"}`, `{"name":"","desc":"d"}`} {
		f := envFinding(envCheck(t, `[`+row+`]`))
		if f == nil || f.Severity != Error || f.Subject != "core.build" || !strings.Contains(f.Message, "no name") {
			t.Errorf("row %s: %+v", row, f)
		}
	}
}

func TestEnvDuplicateNameIsAnErrorOnce(t *testing.T) {
	res := envCheck(t, `[{"name":"A","desc":"d"},{"name":"A","desc":"e"},{"name":"A","desc":"f"}]`)
	n := 0
	for _, f := range res.Findings {
		if f.Rule == RuleEnv {
			n++
			if f.Severity != Error || !strings.Contains(f.Message, `"A"`) {
				t.Errorf("finding = %+v", f)
			}
		}
	}
	if n != 1 {
		t.Errorf("findings = %d, want 1", n)
	}
}

func TestEnvOddNameIsAWarning(t *testing.T) {
	for _, name := range []string{"lower", "Mixed_Case", "WITH-DASH", "A B", "--flag"} {
		res := envCheck(t, `[{"name":"`+name+`","desc":"d"}]`)
		f := envFinding(res)
		if f == nil || f.Severity != Warning || !strings.Contains(f.Message, `"`+name+`"`) {
			t.Errorf("name %q: %+v", name, f)
		}
		if res.Count(Error) != 0 {
			t.Errorf("name %q must not be an error: %v", name, rules(res))
		}
	}
}

func TestEnvWrongTypesAreTheShapeRule(t *testing.T) {
	for _, env := range []string{`{"A":"d"}`, `[{"name":1}]`, `[{"name":"A","required":"yes"}]`, `[{"name":"A","default":3}]`, `["A"]`} {
		res := envCheck(t, env)
		if !has(res, RuleRegistryShape, "core.build") {
			t.Errorf("env %s: %v", env, rules(res))
		}
	}
}

func TestEnvOnExtraEntry(t *testing.T) {
	dump := dumpOf(t, map[string]any{"build": rc("B", nil)}, nil)
	reg := `{"groups":{"core":{"commands":{"build":{"desc":"B"}},"extra":{"x":{"desc":"X","env":[{"desc":"d"}]}}}}}`
	res := run(t, Input{Dump: dump, Variant: "rich", Registry: parse(t, reg), RegistrySource: []byte(reg)})
	if f := envFinding(res); f == nil || f.Subject != "core.x" {
		t.Errorf("extra entry: %+v", f)
	}
}
