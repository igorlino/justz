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

// longRules is rules without the module-shell findings, for tests of a rule
// that builds modules with no settings.
func longRules(res Result) []string {
	var out []string
	for _, r := range rules(res) {
		if !strings.HasPrefix(r, RuleModuleShellUnset) {
			out = append(out, r)
		}
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
	if got := strings.Join(longRules(res), ","); got != want {
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
	if got := strings.Join(longRules(res), ","); got != "recipe-long just a b z" {
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

func moduleDump(settings, recipes string) string {
	return fmt.Sprintf(`{"recipes":{},"settings":{"shell":null},"modules":{"m":{"recipes":{%s},"settings":{%s},"modules":{}}}}`, recipes, settings)
}

func TestModuleShellUnset(t *testing.T) {
	linewise := `"a":{"attributes":[],"body":[["echo"]],"shebang":false}`
	shebang := `"s":{"attributes":[],"body":[["#!/bin/sh"],["echo"]],"shebang":true}`
	tests := []struct {
		name, settings, recipes, want string
	}{
		{"no shell, linewise recipe", `"shell":null,"windows_shell":null`, linewise, "module-shell-unset module m"},
		{"mixed recipes", `"shell":null`, linewise + "," + shebang, "module-shell-unset module m"},
		{"settings missing", ``, linewise, "module-shell-unset module m"},
		{"shell set", `"shell":{"command":"bash","arguments":["-eu","-c"]}`, linewise, ""},
		{"zsh is a choice", `"shell":{"command":"zsh","arguments":["-c"]}`, linewise, ""},
		{"windows-shell does not change sh on Linux", `"shell":null,"windows_shell":{"command":"pwsh","arguments":["-c"]}`, linewise, "module-shell-unset module m"},
		{"empty body runs nothing", `"shell":null`, `"e":{"attributes":[],"body":[],"shebang":false}`, ""},
		{"dependency-only recipe runs nothing", `"shell":null`, shebang + `,"b":{"attributes":[],"body":[],"shebang":false}`, ""},
		{"only shebang recipes", `"shell":null`, shebang, ""},
		{"no recipes", `"shell":null`, ``, ""},
	}
	for _, tt := range tests {
		got := strings.Join(rules(run(t, moduleDump(tt.settings, tt.recipes))), ",")
		if got != tt.want {
			t.Errorf("%s: findings = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestModuleShellUnsetNestedAndRootIsLeftAlone(t *testing.T) {
	linewise := `"a":{"attributes":[],"body":[["echo"]],"shebang":false}`
	d := fmt.Sprintf(`{"recipes":{%s},"settings":{"shell":null},"modules":{"o":{"recipes":{},"settings":{"shell":{"command":"bash","arguments":["-c"]}},"modules":{"i":{"recipes":{%s},"settings":{"shell":null},"modules":{}}}}}}`, linewise, linewise)
	res := run(t, d)
	if got := strings.Join(rules(res), ","); got != "module-shell-unset module o i" {
		t.Errorf("findings = %q", got)
	}
	if f := res.Findings[0]; f.Severity != "warning" || !strings.Contains(f.Fix, "preamble.just") {
		t.Errorf("finding = %+v", f)
	}
}

func TestModuleRuleCannotBeIgnoredPerRecipe(t *testing.T) {
	res := run(t, fmt.Sprintf(`{"recipes":{"a":%s},"modules":{}}`, recipeJSON(1, `{"metadata":["justz-ignore","module-shell-unset"]}`)))
	if got := strings.Join(rules(res), ","); got != "ignore-unknown-rule just a" {
		t.Errorf("findings = %q", got)
	}
}

func dispatchDump(first string, recipes string) string {
	return fmt.Sprintf(`{"recipes":{},"modules":{"m":{"first":%s,"recipes":{%s},"settings":{"shell":{"command":"bash","arguments":["-c"]}},"modules":{}}}}`, first, recipes)
}

func TestModuleDispatcher(t *testing.T) {
	help := `["just help m"]`
	listRoot := `["just --list --justfile ",[["call","justfile"]]]`
	listOwn := `["just --list --justfile ",[["call","justfile"]]," m"]`
	rec := func(private bool, line string) string {
		return fmt.Sprintf(`{"private":%v,"attributes":[],"body":[%s],"shebang":false}`, private, line)
	}
	tests := []struct {
		name, first, recipes, want string
	}{
		{"public default runs justz help", `"default"`, `"default":` + rec(false, help), ""},
		{"private _default runs justz help", `"_default"`, `"_default":` + rec(true, `["justz help m"]`) + `,"up":` + rec(false, `["echo"]`), ""},
		{"no default: first recipe runs", `"deploy"`, `"deploy":` + rec(false, `["echo"]`) + `,"default":` + rec(false, help), "module-dispatcher-missing module m"},
		{"only a command", `"up"`, `"up":` + rec(false, `["echo"]`), "module-dispatcher-missing module m"},
		{"default lists the root", `"default"`, `"default":` + rec(false, listRoot), "module-dispatcher-lists-root module m"},
		{"private default lists the root", `"_default"`, `"_default":` + rec(true, listRoot), "module-dispatcher-lists-root module m"},
		{"list without justfile() is not the root rule", `"default"`, `"default":` + rec(false, `["just --list"]`), ""},
		{"justfile() without --list is fine", `"default"`, `"default":` + rec(false, `["echo ",[["call","justfile"]]]`), ""},
		{"list with justfile() nested in an expression", `"default"`, `"default":` + rec(false, listOwn), "module-dispatcher-lists-root module m"},
		{"no recipes", `null`, ``, ""},
		{"private non-default first recipe is a command", `"_helper"`, `"_helper":` + rec(true, `["rm -rf x"]`) + `,"up":` + rec(false, `["echo"]`), "module-dispatcher-missing module m"},
		{"[default] on a command", `"run"`, `"run":{"private":false,"attributes":["default"],"body":[["echo"]],"shebang":false},"default":` + rec(false, help), "module-dispatcher-missing module m"},
		{"justfile_directory lists the root", `"default"`, `"default":` + rec(false, `["just --list --justfile ",[["call","justfile_directory"]]]`), "module-dispatcher-lists-root module m"},
	}
	for _, tt := range tests {
		res := run(t, dispatchDump(tt.first, tt.recipes))
		if got := strings.Join(rules(res), ","); got != tt.want {
			t.Errorf("%s: findings = %q, want %q", tt.name, got, tt.want)
		}
		for _, f := range res.Findings {
			if f.Severity != "warning" || !strings.Contains(f.Fix, "justz help m") {
				t.Errorf("%s: finding = %+v", tt.name, f)
			}
		}
	}
}

func TestDispatcherRootIsLeftAlone(t *testing.T) {
	d := `{"first":"build","recipes":{"build":{"attributes":[],"body":[["echo"]],"shebang":false}},"modules":{}}`
	if res := run(t, d); len(res.Findings) != 0 {
		t.Errorf("findings = %v", rules(res))
	}
}

func TestDispatcherNestedModule(t *testing.T) {
	d := `{"recipes":{},"modules":{"o":{"first":"default","recipes":{"default":{"attributes":[],"body":[["justz help o"]],"shebang":false}},"settings":{"shell":{"command":"bash","arguments":["-c"]}},"modules":{"i":{"first":"up","recipes":{"up":{"attributes":[],"body":[["echo"]],"shebang":false}},"settings":{"shell":{"command":"bash","arguments":["-c"]}},"modules":{}}}}}}`
	res := run(t, d)
	if got := strings.Join(rules(res), ","); got != "module-dispatcher-missing module o i" {
		t.Errorf("findings = %q", got)
	}
	if m := res.Findings[0].Message; !strings.Contains(m, "just o i runs up, which is not a dispatcher") {
		t.Errorf("message = %q", m)
	}
}

func TestIgnoreAcceptsTheCheckAdviceRules(t *testing.T) {
	for _, id := range []string{"volatile-text", "detail-flag-no-row"} {
		res := run(t, fmt.Sprintf(`{"recipes":{"a":%s},"modules":{}}`, recipeJSON(1, `{"metadata":["justz-ignore","`+id+`"]}`)))
		if len(res.Findings) != 0 {
			t.Errorf("%s: findings = %v", id, rules(res))
		}
	}
}

// longWith is a recipe of 41 lines: padding plus the given body lines (JSON).
func longWith(shebang bool, lines ...string) string {
	all := strings.Split(strings.TrimSuffix(strings.Repeat(`["echo"],`, 41-len(lines)), ","), ",")
	all = append(all, lines...)
	return fmt.Sprintf(`{"attributes":[],"shebang":%v,"body":[%s]}`, shebang, strings.Join(all, ","))
}

func longFinding(t *testing.T, recipe string) Finding {
	t.Helper()
	res := run(t, `{"recipes":{"r":`+recipe+`},"modules":{}}`)
	for _, f := range res.Findings {
		if f.Rule == RuleRecipeLong {
			return f
		}
	}
	t.Fatalf("no recipe-long finding: %v", rules(res))
	return Finding{}
}

func TestRecipeLongInterpolationCounts(t *testing.T) {
	v := `[["variable","x"]]`
	tests := []struct {
		name, recipe           string
		total, quoted, escaped int
		message                string
	}{
		{"none", longWith(false), 0, 0, 0, ", no interpolations"},
		{"some", longWith(false, `["echo ",`+v+`," and ",`+v+`]`), 2, 0, 0, ", 2 interpolations"},
		{"one", longWith(false, `["echo ",`+v+`]`), 1, 0, 0, ", 1 interpolation"},
		{"in double quotes is fine", longWith(false, `["echo \"",`+v+`,"\""]`), 1, 0, 0, ", 1 interpolation"},
		{"in single quotes", longWith(false, `["echo '",`+v+`,"' ",`+v+`]`), 2, 1, 0, "(1 in single quotes or a quoted here-document: not mechanical to move)"},
		{"quote closed before", longWith(false, `["echo 'a' ",`+v+`]`), 1, 0, 0, ", 1 interpolation"},
		{"a new line starts a new shell", longWith(false, `["echo 'open"]`, `["echo ",`+v+`]`), 1, 0, 0, ", 1 interpolation"},
		{"single quote carries across lines in a script", longWith(true, `["echo 'open"]`, `[`+v+`]`), 1, 1, 0, "1 in single quotes"},
		{"escaped braces", longWith(false, `["echo {{{{ lit }}}} ",["{{"]," ",`+v+`]`), 1, 0, 2, ", 2 escaped"},
		{"quoted heredoc", longWith(true, `["cat <<'EOF'"]`, `[`+v+`]`, `["EOF"]`, `[`+v+`]`), 2, 1, 0, "1 in single quotes or a quoted here-document"},
		{"double-quoted dash heredoc", longWith(true, `["cat <<-\"EOF\""]`, `[`+v+`]`, `["\tEOF"]`, `[`+v+`]`), 2, 1, 0, "not mechanical to move"},
		{"unquoted heredoc expands", longWith(true, `["cat <<EOF"]`, `[`+v+`]`, `["EOF"]`), 1, 0, 0, ", 1 interpolation"},
		{"trailing comment apostrophe", longWith(false, `["echo x # it's"]`, `["echo ",`+v+`]`), 1, 0, 0, ", 1 interpolation"},
		{"trailing comment apostrophe in a script", longWith(true, `["echo x # it's"]`, `["echo ",`+v+`]`), 1, 0, 0, ", 1 interpolation"},
		{"interpolation inside a comment is not quoted", longWith(false, `["echo 'a' # '",`+v+`]`), 1, 0, 0, ", 1 interpolation"},
		{"hash inside a word is not a comment", longWith(false, `["echo a#b '",`+v+`,"'"]`), 1, 1, 0, "1 in single quotes"},
		{"hash inside quotes is not a comment", longWith(false, `["echo '# ",`+v+`,"'"]`), 1, 1, 0, "1 in single quotes"},
		{"here-string is not a here-document", longWith(true, `["cat <<< \"hi\""]`, `[`+v+`]`, `["EOF"]`), 1, 0, 0, ", 1 interpolation"},
		{"here-string with a quoted word", longWith(true, `["cat <<<'hi'"]`, `["echo ",`+v+`]`), 1, 0, 0, ", 1 interpolation"},
		{"arithmetic shift is not a here-document", longWith(true, `["n=$((a<<b))"]`, `["echo ",`+v+`]`), 1, 0, 0, ", 1 interpolation"},
		{"here-document after a shift", longWith(true, `["n=$((1<<2)); cat <<'EOF'"]`, `[`+v+`]`, `["EOF"]`), 1, 1, 0, "1 in single quotes"},
		{"ansi-c string with an escaped quote", longWith(true, `["x=$'a\\'b ",`+v+`," c'"]`), 1, 1, 0, "1 in single quotes"},
		{"ansi-c string closed", longWith(true, `["x=$'a\\'b' ",`+v+`]`), 1, 0, 0, ", 1 interpolation"},
		{"backslash does not escape in plain single quotes", longWith(true, `["x='a\\' ",`+v+`]`), 1, 0, 0, ", 1 interpolation"},
		{"comment apostrophe", longWith(true, `["# don't"]`, `["echo ",`+v+`]`), 1, 0, 0, ", 1 interpolation"},
	}
	for _, tt := range tests {
		f := longFinding(t, tt.recipe)
		if f.Interpolations == nil || *f.Interpolations != tt.total || *f.QuotedInterpolations != tt.quoted || *f.EscapedBraces != tt.escaped {
			t.Errorf("%s: counts = %v %v %v, want %d %d %d", tt.name, deref(f.Interpolations), deref(f.QuotedInterpolations), deref(f.EscapedBraces), tt.total, tt.quoted, tt.escaped)
		}
		if !strings.Contains(f.Message, tt.message) || !strings.HasPrefix(f.Message, "body has 41 lines (limit 40)") {
			t.Errorf("%s: message = %q, want it to contain %q", tt.name, f.Message, tt.message)
		}
	}
}

func deref(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}

func TestRecipeLongFixNamesTheIgnoreCaseOnceAndHasNoNumbers(t *testing.T) {
	f := longFinding(t, longWith(false, `["echo ",[["variable","x"]]]`))
	for _, want := range []string{"shebang wrapper that execs", "on line N", "flat list", "justz-ignore"} {
		if !strings.Contains(f.Fix, want) {
			t.Errorf("fix lacks %q: %s", want, f.Fix)
		}
	}
	g := longFinding(t, longWith(false))
	if f.Fix != g.Fix {
		t.Errorf("the fix must not carry per-finding numbers:\n%s\n%s", f.Fix, g.Fix)
	}
	b, _ := json.Marshal(f)
	if !strings.Contains(string(b), `"interpolations":1`) || !strings.Contains(string(b), `"quotedInterpolations":0`) {
		t.Errorf("json = %s", b)
	}
}
