package check

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

// tiersRun checks a registry with the groups a, b and c (and core), and the
// given tiers JSON.
func tiersRun(t *testing.T, tiers string) Result {
	t.Helper()
	mods := map[string]any{}
	groups := []string{`"core":{"commands":{"build":{"desc":"B"}}}`}
	for _, g := range []string{"a", "b", "c"} {
		mods[g] = module(map[string]any{"run": rc("R", nil)})
		groups = append(groups, fmt.Sprintf(`"%s":{"label":"%s","commands":{"run":{"desc":"R"}}}`, g, strings.ToUpper(g)))
	}
	dump := dumpOf(t, map[string]any{"build": rc("B", nil)}, mods)
	reg := `{"groups":{` + strings.Join(groups, ",") + `},"tiers":` + tiers + `}`
	return run(t, Input{Dump: dump, Variant: "rich", Registry: parse(t, reg), RegistrySource: []byte(reg)})
}

func tierMessages(r Result) []string {
	var out []string
	for _, f := range r.Findings {
		if f.Rule == RuleTiers {
			out = append(out, f.Message)
		}
	}
	return out
}

func TestTiersThatCoverEveryGroupOnceGiveNoFinding(t *testing.T) {
	res := tiersRun(t, `[{"label":"One","groups":["a","b"]},{"label":"Two","groups":["c"]}]`)
	if got := tierMessages(res); len(got) != 0 {
		t.Errorf("findings = %v", got)
	}
}

func TestTiersReportsHiddenUnknownAndDuplicate(t *testing.T) {
	// core is listed on its own, so naming it in a tier is not reported.
	res := tiersRun(t, `[{"label":"One","groups":["a","gone","core"]},{"label":"Two","groups":["a"]}]`)
	want := []string{
		`group "a" is in tier "One" and again in tier "Two", so the summary lists it twice`,
		`group "b" is in no tier, so the summary does not list it`,
		`group "c" is in no tier, so the summary does not list it`,
		`tier "One" names "gone", which is not a group of the registry, so help drops it`,
	}
	got := tierMessages(res)
	if strings.Join(sorted(got), "\n") != strings.Join(sorted(want), "\n") {
		t.Errorf("findings:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	for _, f := range res.Findings {
		if f.Rule == RuleTiers && (f.Severity != Warning || f.Fix == "") {
			t.Errorf("finding = %+v", f)
		}
	}
}

func TestTiersWithoutTiersGiveNoFinding(t *testing.T) {
	if got := tierMessages(tiersRun(t, `[]`)); len(got) != 0 {
		t.Errorf("findings = %v", got)
	}
}

func sorted(s []string) []string {
	out := append([]string(nil), s...)
	sort.Strings(out)
	return out
}
