package check

import (
	"fmt"
	"regexp"
)

var envNamePattern = regexp.MustCompile(`^[A-Z0-9_]+$`)

// envRules reports the shape of a command's env rows: a row without a name and
// a duplicate name are errors, a name that is not upper case letters, digits
// and underscores is a warning (some tools read a lower case variable, so it
// is advice, not a broken registry). Wrong types are registry-shape. check
// cannot tell whether the command reads the variable: a script that a recipe
// runs reads its own variables, so the recipe body proves nothing.
func envRules(entries []registryEntry) []Finding {
	var out []Finding
	for _, e := range entries {
		subject := fmt.Sprintf("%s.%s", e.Group, e.Key)
		seen := map[string]bool{}
		dup := map[string]bool{}
		for _, name := range e.EnvNames {
			switch {
			case name == "":
				out = append(out, Finding{Rule: RuleEnv, Severity: Error, Subject: subject,
					Message: "an env row has no name",
					Fix:     "give the row a name, such as \"STORAGE_HOST\", or remove it"})
			case seen[name]:
				if !dup[name] {
					dup[name] = true
					out = append(out, Finding{Rule: RuleEnv, Severity: Error, Subject: subject,
						Message: fmt.Sprintf("env has the name %q more than once", name),
						Fix:     "keep one row per variable"})
				}
			case !envNamePattern.MatchString(name):
				out = append(out, Finding{Rule: RuleEnv, Severity: Warning, Subject: subject,
					Message: fmt.Sprintf("env name %q is not upper case letters, digits and underscores", name),
					Fix:     "write the variable name as the command reads it, such as \"STORAGE_HOST\"; a flag goes in args, not in env"})
			}
			if name != "" {
				seen[name] = true
			}
		}
	}
	return out
}
