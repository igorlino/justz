package resolve

import (
	"testing"

	"github.com/igorlino/justz/internal/orderedjson"
)

const registry = `{
  "groups": {
    "core": { "commands": { "ci": {}, "build": {} } },
    "demo": { "commands": { "greet": {}, "scan": {}, "only-demo": {} }, "extra": { "hidden": {} } },
    "other": { "commands": { "greet": {} } },
    "tool": { "commands": { "git": {}, "doctor": {} } },
    "sbx": { "commands": { "doctor": {} } }
  }
}`

func doc(t *testing.T) *orderedjson.Value {
	t.Helper()
	v, err := orderedjson.Parse([]byte(registry))
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestExact(t *testing.T) {
	roots := map[string]bool{"rootrecipe": true, "scan": true}
	tests := []struct {
		name       string
		want       Kind
		wantReason string
		wantGroups []string
	}{
		{"git", Found, "", []string{"tool"}},
		{"only-demo", Found, "", []string{"demo"}},
		{"greet", Ambiguous, "", []string{"demo", "other"}},
		{"doctor", Ambiguous, "", []string{"sbx", "tool"}},
		{"demo", None, ReasonGroup, nil}, // a group word is a namespace
		{"ci", None, ReasonCore, nil},    // core commands run without a group
		{"build", None, ReasonCore, nil},
		{"hidden", None, ReasonExtra, nil},    // documented extras are not run by bare name
		{"rootrecipe", None, ReasonRoot, nil}, // a real root recipe wins
		{"scan", None, ReasonRoot, nil},       // even when a group also has it
		{"gi", None, ReasonUnknown, nil},      // a prefix never resolves
		{"nope", None, ReasonUnknown, nil},
		{"", None, ReasonUnknown, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Exact(doc(t), roots, tt.name)
			if got.Kind != tt.want || got.Reason != tt.wantReason {
				t.Fatalf("Exact(%q) = kind %v reason %q, want kind %v reason %q", tt.name, got.Kind, got.Reason, tt.want, tt.wantReason)
			}
			var groups []string
			for _, m := range got.Matches {
				groups = append(groups, m.Group)
				if m.Command != tt.name {
					t.Errorf("match command = %q, want %q", m.Command, tt.name)
				}
			}
			if len(groups) != len(tt.wantGroups) {
				t.Fatalf("groups = %v, want %v", groups, tt.wantGroups)
			}
			for i := range groups {
				if groups[i] != tt.wantGroups[i] {
					t.Errorf("groups = %v, want %v (sorted)", groups, tt.wantGroups)
				}
			}
		})
	}
}

func TestExactWithoutGroups(t *testing.T) {
	for _, in := range []string{`{}`, `{"groups":[]}`, `[]`, `{"groups":{"a":"not an object"}}`} {
		v, err := orderedjson.Parse([]byte(in))
		if err != nil {
			t.Fatal(err)
		}
		if got := Exact(v, nil, "x"); got.Kind != None || got.Reason != ReasonUnknown {
			t.Errorf("Exact on %s = %+v, want an unknown None", in, got)
		}
	}
}
