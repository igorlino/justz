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
