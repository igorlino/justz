package resolve

import (
	"testing"

	"github.com/igorlino/justz/internal/orderedjson"
)

func TestAGroupKeyThatIsNotAPlainNameIsNeverReported(t *testing.T) {
	v, err := orderedjson.Parse([]byte(`{"groups": {
		"core": {"commands": {}},
		"good": {"commands": {"run": {}}},
		"bad group": {"commands": {"run": {}}},
		"x;y": {"commands": {"run": {}}},
		"esc\u001b[31m": {"commands": {"run": {}}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	res := Exact(v, nil, "run")
	if res.Kind != Found || len(res.Matches) != 1 || res.Matches[0].Group != "good" {
		t.Errorf("matches = %+v (kind %v), want only the group good", res.Matches, res.Kind)
	}
}

func TestACommandKeyThatIsNotAPlainNameIsNeverReported(t *testing.T) {
	v, err := orderedjson.Parse([]byte(`{"groups": {
		"core": {"commands": {}},
		"tool": {"commands": {"bad key": {}, ".dot": {}, "-dash": {}, "x;y": {}, "ok.name-1": {}}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"bad key", ".dot", "-dash", "x;y"} {
		if res := Exact(v, nil, name); res.Kind != None || res.Reason != ReasonUnknown || len(res.Matches) != 0 {
			t.Errorf("%q: %+v, want no match", name, res)
		}
	}
	if res := Exact(v, nil, "ok.name-1"); res.Kind != Found || res.Matches[0].Group != "tool" {
		t.Errorf("a plain name must still resolve: %+v", res)
	}
}
