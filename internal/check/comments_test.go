package check

import "testing"

const dumpText = `set shell := ["bash", "-c"]

# First line of a block
# second line
# third line
multi:
    @echo multi

# single line
single:
    @echo single

# a block before attributes
# continues here
[group("x")]
[no-cd]
attr:
    @echo attr

# detached comment

# one more
detached:
    @echo detached

# quiet recipe
# with two lines
@quiet arg='x':
    echo {{ arg }}

# prefix block line one
# prefix block line two
build-all:
    @echo all

build:
    @echo build

assigned := "x"
`

func TestCommentLines(t *testing.T) {
	tests := []struct {
		name string
		want int
	}{
		{"multi", 3},
		{"single", 1},
		{"attr", 2},     // attributes sit between the block and the recipe
		{"detached", 1}, // a blank line ends the block
		{"quiet", 2},    // @ prefix and parameters
		{"build-all", 2},
		{"build", 0}, // not confused with build-all, and nothing above it
		{"assigned", 0},
		{"missing", 0},
	}
	for _, tt := range tests {
		if got := commentLines(dumpText, tt.name); got != tt.want {
			t.Errorf("commentLines(%q) = %d, want %d", tt.name, got, tt.want)
		}
	}
}

func TestMultilineFindings(t *testing.T) {
	explicit := rc("Explicit", []map[string]any{{"doc": "Explicit"}})
	dump := dumpOf(t, map[string]any{
		"multi":    rc("third line", nil),
		"single":   rc("single line", nil),
		"explicit": explicit,
	}, map[string]any{"tool": module(map[string]any{"cleanup": rc("last line", nil), "nosource": rc("x", nil)})})

	sources := map[string]string{
		"": `# First
# Second
multi:
    @echo

# single line
single:
    @echo

# long
# comment
[doc('Explicit')]
explicit:
    @echo
`,
		"tool": `# Two
# lines
cleanup:
    @echo
`,
	}
	res := run(t, Input{Dump: dump, Variant: "native", Sources: sources})
	if !has(res, RuleDocMultiline, "just multi") || !has(res, RuleDocMultiline, "just tool cleanup") {
		t.Errorf("multi-line comments not found: %v", rules(res))
	}
	for _, bad := range []string{"just single", "just explicit", "just tool nosource"} {
		if has(res, RuleDocMultiline, bad) {
			t.Errorf("%s must not be reported", bad)
		}
	}
	for _, f := range res.Findings {
		if f.Rule == RuleDocMultiline && f.Severity != Warning {
			t.Errorf("the rule is a warning: %+v", f)
		}
	}
}

func TestMultilineRuleNeedsSources(t *testing.T) {
	dump := dumpOf(t, map[string]any{"multi": rc("third line", grp("core"))}, nil)
	if res := run(t, Input{Dump: dump, Variant: "native"}); has(res, RuleDocMultiline, "") {
		t.Error("without the dumped text the rule must not run")
	}
}
