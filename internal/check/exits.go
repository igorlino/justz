package check

import (
	"fmt"

	"github.com/igorlino/justz/internal/registry"
)

// exitsRules reports a key of a command's exits map that is not an exit code:
// an integer from 0 to 255 in decimal. The schema cannot say this (it never
// uses a pattern on keys), so the rule lives here. check cannot tell whether a
// recipe really exits with the code, so the meaning is not verified. It is an
// error, like registry-key-signature: the key is not what the field needs. A
// blank meaning is the same rule at warning severity: the code is listed and
// says nothing, which is advice to fix, not a broken registry.
func exitsRules(entries []registryEntry) []Finding {
	var out []Finding
	for _, e := range entries {
		for _, key := range e.ExitKeys {
			if _, ok := registry.ExitCode(key); !ok {
				out = append(out, Finding{Rule: RuleExits, Severity: Error, Subject: fmt.Sprintf("%s.%s", e.Group, e.Key),
					Message: fmt.Sprintf("exits has the key %q, which is not an exit code", key),
					Fix:     "use an integer from 0 to 255 written in decimal (such as \"0\" or \"2\"), with no sign, no leading zero and no spaces, or remove the entry"})
			}
		}
		for _, key := range e.ExitBlank {
			out = append(out, Finding{Rule: RuleExits, Severity: Warning, Subject: fmt.Sprintf("%s.%s", e.Group, e.Key),
				Message: fmt.Sprintf("exits has a blank meaning for %q", key),
				Fix:     "say what the exit code means in one line, or remove the entry"})
		}
	}
	return out
}
