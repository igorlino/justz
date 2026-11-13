package registry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/igorlino/justz/internal/orderedjson"
)

func canonical(t *testing.T, in []byte) []byte {
	t.Helper()
	v, err := orderedjson.Parse(in)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	Canonicalize(v)
	out, err := orderedjson.Marshal(v)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	return out
}

// The golden files were produced by the jq filter of the original help tooling,
// so these tests check that the canonical order did not change.
func TestMatchesReferenceOutput(t *testing.T) {
	inputs, err := filepath.Glob("testdata/*.in.json")
	if err != nil || len(inputs) == 0 {
		t.Fatalf("no fixtures found: %v", err)
	}
	for _, in := range inputs {
		name := strings.TrimSuffix(filepath.Base(in), ".in.json")
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(in)
			if err != nil {
				t.Fatal(err)
			}
			want, err := os.ReadFile(strings.TrimSuffix(in, ".in.json") + ".golden.json")
			if err != nil {
				t.Fatal(err)
			}
			if got := canonical(t, data); string(got) != string(want) {
				t.Errorf("output differs from the reference:\n--- got\n%s\n--- want\n%s", got, want)
			}
		})
	}
}

func TestIsIdempotent(t *testing.T) {
	data, err := os.ReadFile("testdata/full.in.json")
	if err != nil {
		t.Fatal(err)
	}
	once := canonical(t, data)
	twice := canonical(t, once)
	if string(once) != string(twice) {
		t.Errorf("a second pass changed the output:\n%s\n---\n%s", once, twice)
	}
}

func TestKeepsValuesOrder(t *testing.T) {
	out := string(canonical(t, []byte(`{"groups":{"g":{"commands":{"c":{"args":[{"name":"--level","values":["low","medium","high"]}]}}}}}`)))
	if !strings.Contains(out, "\"low\",\n") || strings.Index(out, "\"low\"") > strings.Index(out, "\"medium\"") || strings.Index(out, "\"medium\"") > strings.Index(out, "\"high\"") {
		t.Errorf("values should keep the author's order:\n%s", out)
	}
}

func TestIgnoresOtherShapes(t *testing.T) {
	for _, in := range []string{`[]`, `"x"`, `null`, `{"groups":[1,2]}`, `{"groups":{"a":"not an object"}}`} {
		v, err := orderedjson.Parse([]byte(in))
		if err != nil {
			t.Fatal(err)
		}
		Canonicalize(v) // must not panic
	}
}
