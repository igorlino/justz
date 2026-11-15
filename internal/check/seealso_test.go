package check

import (
	"strings"
	"testing"
)

func seeAlsoCheck(t *testing.T, reg string) Result {
	t.Helper()
	dump := dumpOf(t, map[string]any{"build": rc("B", nil)}, map[string]any{"tool": module(map[string]any{"clean": rc("C", nil), "run": rc("R", nil)})})
	return run(t, Input{Dump: dump, Variant: "rich", Registry: parse(t, reg), RegistrySource: []byte(reg)})
}

func seeAlsoFindings(r Result) []Finding {
	var out []Finding
	for _, f := range r.Findings {
		if f.Rule == RuleSeeAlso {
			out = append(out, f)
		}
	}
	return out
}

func TestSeeAlsoKeysThatExistGiveNoFinding(t *testing.T) {
	reg := `{"groups":{"core":{"commands":{"build":{"desc":"B","seeAlso":["tool.clean","tool.old"]}}},
	  "tool":{"commands":{"clean":{"desc":"C","seeAlso":["core.build"]},"run":{"desc":"R"}},"extra":{"old":{"desc":"O","seeAlso":["tool.clean"]}}}}}`
	if got := seeAlsoFindings(seeAlsoCheck(t, reg)); len(got) != 0 {
		t.Errorf("findings = %+v", got)
	}
}

func TestSeeAlsoUnknownAndSelfAreErrors(t *testing.T) {
	reg := `{"groups":{"core":{"commands":{"build":{"desc":"B","seeAlso":["tool.gone","build","core.build","tool.clean","tool.gone"]}}},
	  "tool":{"commands":{"clean":{"desc":"C"},"run":{"desc":"R","seeAlso":["tool.run","clean",""]}}}}}`
	got := seeAlsoFindings(seeAlsoCheck(t, reg))
	var lines []string
	for _, f := range got {
		if f.Severity != Error || f.Fix == "" {
			t.Errorf("finding = %+v", f)
		}
		lines = append(lines, f.Subject+" "+f.Message)
	}
	want := []string{
		`core.build seeAlso names "tool.gone", which is not a command of the registry`,
		`core.build seeAlso names "build", which is not a command of the registry`,
		`core.build seeAlso names the command itself, "core.build"`,
		`core.build seeAlso lists "tool.gone" more than once`,
		`tool.run seeAlso names the command itself, "tool.run"`,
		`tool.run seeAlso names "clean", which is not a command of the registry`,
		`tool.run seeAlso names "", which is not a command of the registry`,
	}
	if strings.Join(lines, "\n") != strings.Join(want, "\n") {
		t.Errorf("findings:\n%s\nwant:\n%s", strings.Join(lines, "\n"), strings.Join(want, "\n"))
	}
}

func TestSeeAlsoWrongTypeIsTheShapeRule(t *testing.T) {
	reg := `{"groups":{"core":{"commands":{"build":{"desc":"B","seeAlso":"tool.clean"}}}}}`
	res := seeAlsoCheck(t, reg)
	if !has(res, RuleRegistryShape, "core.build") || len(seeAlsoFindings(res)) != 0 {
		t.Errorf("findings = %v", rules(res))
	}
}
