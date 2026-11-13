package generate

import (
	"flag"
	"os"
	"strings"
	"testing"

	"github.com/igorlino/justz/internal/orderedjson"
)

var update = flag.Bool("update", false, "rewrite the golden file")

func marshal(t *testing.T, res Result) string {
	t.Helper()
	out, err := orderedjson.Marshal(res.Doc)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

// The dump is the real output of just --dump --dump-format json for the native
// fixture. The golden file was reviewed against the output of the old shell
// generator (testdata/variants/native/local-rich.reference.json). The
// differences are intended: a flag with a fixed value shows no placeholder
// (-w, --watch) and a positional argument is shown as <env>.
func TestBuildNativeFixture(t *testing.T) {
	dumpJSON, err := os.ReadFile("testdata/native.dump.json")
	if err != nil {
		t.Fatal(err)
	}
	res, err := Build(dumpJSON, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := marshal(t, res)
	if *update {
		if err := os.WriteFile("testdata/native.expected.json", []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile("testdata/native.expected.json")
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Errorf("output differs from the golden file:\n--- got\n%s\n--- want\n%s", got, want)
	}
	if len(res.Warnings) != 0 {
		t.Errorf("unexpected warnings: %v", res.Warnings)
	}
}

func TestPreservesHandAddedValues(t *testing.T) {
	dumpJSON := []byte(`{"recipes":{"scan":{"private":false,"doc":"Scan","parameters":[
		{"name":"level","kind":"singular","short":"l","long":"level","help":"How strict","flag":false,"value":null}]}},"modules":{}}`)
	old, err := orderedjson.Parse([]byte(`{"groups":{"core":{"commands":{"scan":{"args":[
		{"name":"-l, --level <level>","desc":"old text","values":["low","medium","high"]}]}}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	res, err := Build(dumpJSON, old)
	if err != nil {
		t.Fatal(err)
	}
	got := marshal(t, res)
	if !strings.Contains(got, `"values": [`) || !strings.Contains(got, `"medium"`) {
		t.Errorf("hand-added values were lost:\n%s", got)
	}
	if !strings.Contains(got, `"desc": "How strict"`) || strings.Contains(got, "old text") {
		t.Errorf("the description must come from the justfile, not the old file:\n%s", got)
	}

	// An argument that no longer exists does not bring its values back.
	res, _ = Build([]byte(`{"recipes":{"scan":{"private":false,"doc":"Scan","parameters":[]}},"modules":{}}`), old)
	if strings.Contains(marshal(t, res), `"values": [`) {
		t.Error("values of a removed argument must not reappear")
	}
}

func TestSkipsPrivateRecipesAndWarnsAboutNestedModules(t *testing.T) {
	dumpJSON := []byte(`{"recipes":{"_helper":{"private":true,"doc":"x","parameters":[]},"pub":{"private":false,"doc":"Public","parameters":[]}},
		"modules":{"outer":{"recipes":{"a":{"private":false,"doc":"A","parameters":[]}},"modules":{"inner":{"recipes":{},"modules":{}}}}}}`)
	res, err := Build(dumpJSON, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := marshal(t, res)
	if strings.Contains(got, "_helper") || !strings.Contains(got, `"pub"`) || !strings.Contains(got, `"outer"`) {
		t.Errorf("unexpected output:\n%s", got)
	}
	if len(res.Warnings) != 1 || !strings.Contains(res.Warnings[0], "nested modules") {
		t.Errorf("warnings = %v", res.Warnings)
	}
}

func TestArgumentKinds(t *testing.T) {
	s := func(v string) *string { return &v }
	args := arguments([]parameter{
		{Name: "files", Kind: "star", Help: s("Many")},
		{Name: "more", Kind: "plus", Help: s("At least one")},
		{Name: "env", Kind: "singular", Help: s("Target")},
		{Name: "v", Kind: "singular", Short: s("v"), Flag: true, Help: s("Verbose")},
		{Name: "n", Kind: "singular", Long: s("count"), Help: s("How many")},
		{Name: "undocumented", Kind: "singular"},
	})
	var names []string
	for _, a := range args.Arr {
		names = append(names, a.Get("name").Str)
	}
	want := []string{"*files", "+more", "<env>", "-v", "--count <n>"}
	if strings.Join(names, "|") != strings.Join(want, "|") {
		t.Errorf("names = %v, want %v (undocumented parameters are left out)", names, want)
	}
}

func TestParseDoc(t *testing.T) {
	s := func(v string) *string { return &v }
	tests := []struct {
		doc           *string
		desc, example string
	}{
		{nil, "", ""},
		{s("Just a line"), "Just a line", ""},
		{s("Run it\nexample: just run -w | Watch mode"), "Run it", "just run -w"},
		{s("Run it\nmore text\nexample: just run"), "Run it", "just run"},
	}
	for _, tt := range tests {
		desc, ex := parseDoc(tt.doc)
		if desc != tt.desc || ex != tt.example {
			t.Errorf("parseDoc(%v) = %q, %q; want %q, %q", tt.doc, desc, ex, tt.desc, tt.example)
		}
	}
}

func TestInvalidDump(t *testing.T) {
	if _, err := Build([]byte("not json"), nil); err == nil {
		t.Error("an invalid dump should be an error")
	}
}
