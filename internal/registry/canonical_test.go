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

func TestCanonicalSortsCommandSeeAlso(t *testing.T) {
	out := string(canonical(t, []byte(`{"groups":{"core":{"commands":{"b":{"desc":"B","seeAlso":["core.z","core.a","core.m"]}}}}}`)))
	a, m, z := strings.Index(out, "core.a"), strings.Index(out, "core.m"), strings.Index(out, "core.z")
	if a < 0 || a > m || m > z {
		t.Errorf("seeAlso not sorted: %s", out)
	}
}

func TestCanonicalSortsExitsByNumericCode(t *testing.T) {
	out := string(canonical(t, []byte(`{"groups":{"core":{"commands":{"b":{"desc":"B","exits":{"10":"j","zed":"z","2":"b","alpha":"a","0":"a"}}}}}}`)))
	var pos []int
	for _, k := range []string{`"0"`, `"2"`, `"10"`, `"alpha"`, `"zed"`} {
		pos = append(pos, strings.Index(out, k))
	}
	for i := 1; i < len(pos); i++ {
		if pos[i-1] < 0 || pos[i] < pos[i-1] {
			t.Fatalf("exits not in order 0, 2, 10, alpha, zed: %s", out)
		}
	}
}

func TestExitCodeKeys(t *testing.T) {
	for key, want := range map[string]int{"0": 0, "7": 7, "10": 10, "255": 255} {
		if got, ok := ExitCode(key); !ok || got != want {
			t.Errorf("ExitCode(%q) = %d, %v", key, got, ok)
		}
	}
	for _, key := range []string{"", "00", "01", "256", "-1", "+1", "1 ", "a", "1000", "0x1", "١"} {
		if _, ok := ExitCode(key); ok {
			t.Errorf("ExitCode(%q) should be false", key)
		}
	}
}
