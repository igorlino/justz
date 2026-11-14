package complete

import (
	"strings"
	"testing"

	"github.com/igorlino/justz/internal/help"
	"github.com/igorlino/justz/internal/orderedjson"
)

// dirtyRegistry has group and command keys that are not plain names, and
// descriptions with an escape sequence. Completion inserts a candidate on the
// command line, so such a key must never be offered.
const dirtyRegistry = `{
  "groups": {
    "core": { "commands": { "fine": {"desc": "ok"}, "manifest [n]": {"desc": "signature"}, "x;curl": {"desc": "semi"} } },
    "g;h": { "label": "bad group", "commands": { "k": {"desc": "in a bad group"} } },
    "good": { "label": "Good\u001b]0;t\u0007", "commands": {
        "ok": {"desc": "fine\u001b[31m text"}, "a b": {"desc": "space"}, "$(x)": {"desc": "subst"}, "-flag": {"desc": "dash"} } }
  }
}`

func TestCompletionOffersOnlyPlainNamesAndCleanDescriptions(t *testing.T) {
	doc, err := orderedjson.Parse([]byte(dirtyRegistry))
	if err != nil {
		t.Fatal(err)
	}
	proj, err := help.FromRegistry(doc, "p")
	if err != nil {
		t.Fatal(err)
	}
	e := Env{Project: func() *help.Project { return proj }, RootRecipes: func() map[string]bool { return nil }, Shims: func() []string { return nil }}

	top := Shim(false, []string{""}, e)
	if got := values(top); got != "fine good" {
		t.Errorf("top-level candidates = %q, want only the plain names", got)
	}
	inGroup := Shim(false, []string{"good", ""}, e)
	if got := values(inGroup); got != "ok" {
		t.Errorf("candidates in a group = %q, want only ok", got)
	}
	bare := Shim(false, []string{"o"}, e)
	if v := values(bare); strings.ContainsAny(v, " ;$[") {
		t.Errorf("bare candidates hold a name that is not plain: %q", v)
	}
	for _, r := range []Result{top, inGroup, bare} {
		for _, c := range r.Candidates {
			for _, ch := range c.Value + c.Desc {
				if ch < 0x20 || ch == 0x7f || (ch >= 0x80 && ch <= 0x9f) {
					t.Errorf("a control character is in candidate %q %q", c.Value, c.Desc)
				}
			}
		}
	}
	if d := desc(inGroup, "ok"); d != "fine�[31m text" {
		t.Errorf("description = %q", d)
	}
}
