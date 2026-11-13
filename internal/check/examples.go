package check

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/igorlino/justz/internal/orderedjson"
)

var flagScan = regexp.MustCompile(`-{1,2}[A-Za-z0-9][A-Za-z0-9-]*`)
var shortLong = regexp.MustCompile(`(-[A-Za-z0-9]),\s*(--[A-Za-z0-9-]+)`)

// exampleRules checks the group examples of a registry. They are rules from
// the old checker: an example should teach something the argument rows do not.
func exampleRules(reg *orderedjson.Value) []Finding {
	var out []Finding
	groups := reg.Get("groups")
	if groups == nil || groups.Kind != orderedjson.Object {
		return nil
	}
	for _, g := range groups.Obj {
		if g.Val == nil || g.Val.Kind != orderedjson.Object {
			continue
		}
		examples := exampleCommands(g.Val)
		var cmds []registryEntry
		for _, e := range registryEntries(reg) {
			if e.Group == g.Key && !e.Extra {
				cmds = append(cmds, e)
			}
		}
		byKey := map[string]registryEntry{}
		for _, c := range cmds {
			byKey[c.Key] = c
		}

		// The old checker never matched the core group's examples (it looked for
		// a literal "just core ..."), so projects that pass it have such
		// examples. Report them, but as warnings.
		sev := Error
		if g.Key == "core" {
			sev = Warning
		}

		prefix := `^just +`
		if g.Key != "core" {
			prefix += regexp.QuoteMeta(g.Key) + ` +`
		}
		bare := regexp.MustCompile(prefix + `([A-Za-z0-9_-]+) *$`)
		single := regexp.MustCompile(prefix + `([A-Za-z0-9_-]+) +(-[A-Za-z0-9-]+) *$`)

		for _, ex := range examples {
			subject := g.Key + ".examples"
			if bare.MatchString(ex) {
				out = append(out, Finding{RuleExampleBare, sev, subject,
					fmt.Sprintf("%q is a bare command with no flags or arguments", ex),
					"remove it, or show multi-flag, value or environment usage"})
				continue
			}
			if m := single.FindStringSubmatch(ex); m != nil {
				if c, ok := byKey[m[1]]; ok && isBooleanFlag(c, m[2]) {
					out = append(out, Finding{RuleExampleSingle, sev, subject,
						fmt.Sprintf("%q only passes the single boolean flag %s, which the argument rows already show", ex, m[2]),
						"remove it, or combine flags, values or environment"})
				}
			}
		}

		for _, c := range cmds {
			path := c.Key
			if g.Key != "core" {
				path = g.Key + " " + c.Key
			}
			onCmd := regexp.MustCompile(`^just +` + regexp.QuoteMeta(path) + `\b`)
			for _, a := range c.Args {
				// Long flag used in an example although a short alias exists.
				if m := shortLong.FindStringSubmatch(a.Name); m != nil {
					long := regexp.MustCompile(`(^| )` + regexp.QuoteMeta(m[2]) + `([ =]|$)`)
					for _, ex := range examples {
						if onCmd.MatchString(ex) && long.MatchString(ex) {
							out = append(out, Finding{RuleExampleLongFlag, sev, g.Key + ".examples",
								fmt.Sprintf("%q uses the long flag %s", ex, m[2]),
								"prefer the short alias " + m[1]})
						}
					}
				}
				// A flag with enum values must be shown at least once.
				if strings.HasPrefix(a.Name, "-") && strings.Contains(a.Name, "<") && len(a.Values) > 0 {
					flags := flagScan.FindAllString(a.Name, -1)
					shown := false
					for _, ex := range examples {
						if !onCmd.MatchString(ex) {
							continue
						}
						for _, f := range flags {
							if regexp.MustCompile(`(^| )` + regexp.QuoteMeta(f) + `([ =]|$)`).MatchString(ex) {
								shown = true
							}
						}
					}
					if !shown {
						out = append(out, Finding{RuleExampleEnumUnset, sev, fmt.Sprintf("%s.%s", g.Key, c.Key),
							fmt.Sprintf("%s declares the values %s, but no example in %s.examples uses it", a.Name, strings.Join(a.Values, "|"), g.Key),
							"add an example that passes the flag"})
					}
				}
			}
		}
	}
	return out
}

func exampleCommands(group *orderedjson.Value) []string {
	var out []string
	if ex := group.Get("examples"); ex != nil && ex.Kind == orderedjson.Array {
		for _, e := range ex.Arr {
			if c := e.Get("cmd"); c != nil && c.Kind == orderedjson.String {
				out = append(out, c.Str)
			}
		}
	}
	return out
}

// isBooleanFlag reports whether flag is one of the names of an argument that
// takes no value (its name has no <placeholder>).
func isBooleanFlag(c registryEntry, flag string) bool {
	for _, a := range c.Args {
		if strings.Contains(a.Name, "<") {
			continue
		}
		for _, f := range flagScan.FindAllString(a.Name, -1) {
			if f == flag {
				return true
			}
		}
	}
	return false
}
