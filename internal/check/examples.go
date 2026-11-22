package check

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/igorlino/justz/internal/orderedjson"
	"github.com/igorlino/justz/internal/textsafe"
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
		var cmds []registryEntry
		for _, e := range registryEntries(reg) {
			if e.Group == g.Key && !e.Extra {
				cmds = append(cmds, e)
			}
		}
		// The group examples and the per-command example fields go through the
		// same rules. A finding on the first names the group, on the second the
		// command.
		var found []example
		for _, ex := range exampleCommands(g.Val) {
			found = append(found, example{Text: ex, Raw: ex, Subject: g.Key + ".examples"})
		}
		for _, c := range cmds {
			if c.Example != "" {
				path := c.Key
				if g.Key != "core" {
					path = g.Key + " " + c.Key
				}
				found = append(found, example{Text: expandExample(c.Example, path), Raw: c.Example, Subject: g.Key + "." + c.Key})
			}
		}
		for i := range found {
			_, found[i].HasEnv = envPrefix(strings.TrimSpace(found[i].Raw))
			found[i].Text = stripEnv(found[i].Text)
		}
		var examples []string
		for _, ex := range found {
			examples = append(examples, ex.Text)
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

		for _, e := range found {
			ex, subject := e.Text, e.Subject
			if e.HasEnv {
				continue // the environment is what the argument rows do not show
			}
			if bare.MatchString(ex) {
				out = append(out, Finding{Rule: RuleExampleBare, Severity: sev, Subject: subject, Message: fmt.Sprintf("%q is a bare command with no flags or arguments", e.Raw), Fix: "remove it, or show multi-flag, value or environment usage"})
				continue
			}
			if m := single.FindStringSubmatch(ex); m != nil {
				if c, ok := byKey[m[1]]; ok && isBooleanFlag(c, m[2]) {
					out = append(out, Finding{Rule: RuleExampleSingle, Severity: sev, Subject: subject, Message: fmt.Sprintf("%q only passes the single boolean flag %s, which the argument rows already show", e.Raw, m[2]), Fix: "remove it, or combine flags, values or environment"})
				}
			}
		}

		for _, c := range cmds {
			path := c.Key
			if g.Key != "core" {
				path = g.Key + " " + c.Key
			}
			onCmd := regexp.MustCompile(`^just +` + regexp.QuoteMeta(path) + `\b`)
			out = append(out, unknownFlags(c, path, found)...)
			out = append(out, booleanValues(c, path, found)...)
			for _, a := range c.Args {
				// Long flag used in an example although a short alias exists.
				if m := shortLong.FindStringSubmatch(a.Name); m != nil {
					long := regexp.MustCompile(`(^| )` + regexp.QuoteMeta(m[2]) + `([ =]|$)`)
					for _, e := range found {
						if ex := e.Text; onCmd.MatchString(ex) && long.MatchString(ex) {
							out = append(out, Finding{Rule: RuleExampleLongFlag, Severity: sev, Subject: e.Subject, Message: fmt.Sprintf("%q uses the long flag %s", e.Raw, m[2]), Fix: "prefer the short alias " + m[1]})
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
						out = append(out, Finding{Rule: RuleExampleEnumUnset, Severity: sev, Subject: fmt.Sprintf("%s.%s", g.Key, c.Key), Message: fmt.Sprintf("%s declares the values %s, but no example in %s.examples uses it", a.Name, strings.Join(a.Values, "|"), textsafe.Line(g.Key)), Fix: "add an example that passes the flag"})
					}
				}
			}
		}
	}
	return out
}

// example is one example line: Text is what the rules read (an environment
// prefix removed, a leading ellipsis expanded), Raw is what the project wrote,
// HasEnv is true when Raw leads with a NAME=value assignment (after env or not).
type example struct {
	Text, Raw, Subject string
	HasEnv             bool
}

// stripEnv removes the environment words that lead a command line: NAME=value
// (the value may be quoted, with escaped quotes), env and its options (-i, -u
// NAME, -C DIR, --flag). PORT=9000 just x start is then read as just x start.
func stripEnv(s string) string {
	s = strings.TrimSpace(s)
	return strings.TrimSpace(s[envPrefixLen(s):])
}

// envPrefixLen is the length of the leading environment words of s.
func envPrefixLen(s string) int {
	n, _ := envPrefix(s)
	return n
}

// envPrefix is envPrefixLen, plus whether a NAME=value assignment was in the
// prefix (a bare env or env options alone set no variable).
func envPrefix(s string) (length int, assigned bool) {
	pos := 0
	next := func() (word string, end int) { // the word at pos, and where it ends
		i := pos
		for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
			i++
		}
		start, j := i, i
		var quote byte
		for j < len(s) {
			c := s[j]
			switch {
			case c == '\\' && quote != '\'' && j+1 < len(s):
				j++
			case quote != 0:
				if c == quote {
					quote = 0
				}
			case c == '"' || c == '\'':
				quote = c
			case c == ' ' || c == '\t':
				return s[start:j], j
			}
			j++
		}
		return s[start:], len(s)
	}
	assign := regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)
	sawEnv := false
	for {
		w, end := next()
		switch {
		case w == "":
			return pos, assigned
		case !sawEnv && w == "env":
			sawEnv = true
		case sawEnv && (w == "-u" || w == "-C" || w == "--unset" || w == "--chdir"):
			pos = end
			_, end = next() // the option's argument
		case sawEnv && strings.HasPrefix(w, "-") && w != "-" && w != "--":
		case assign.MatchString(w):
			assigned = true
		default:
			return pos, assigned
		}
		pos = end
	}
}

// expandExample turns a command's example field into a command line. The field
// is written with a leading ellipsis (… or ...) for "just <path>", after any
// environment words; one without it is read as written.
func expandExample(ex, path string) string {
	ex = strings.TrimSpace(ex)
	n := envPrefixLen(ex)
	env, rest := strings.TrimSpace(ex[:n]), strings.TrimSpace(ex[n:])
	for _, mark := range []string{"…", "..."} {
		if r, ok := strings.CutPrefix(rest, mark); ok {
			out := "just " + path + " " + strings.TrimSpace(r)
			return strings.TrimSpace(env + " " + out)
		}
	}
	return ex
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

var noFlag = regexp.MustCompile(`--\[no-\]([A-Za-z0-9][A-Za-z0-9-]*)`)

// declaredFlags lists the flags the argument rows of a command declare, the
// aliases of a row and both forms of a --[no-]x row included. variadic is true
// when a row ends in ..., which passes the rest of the line on.
func declaredFlags(c registryEntry) (declared map[string]bool, variadic bool) {
	declared = map[string]bool{}
	for _, a := range c.Args {
		if strings.Contains(a.Name, "...") {
			variadic = true
		}
		for _, m := range noFlag.FindAllStringSubmatch(a.Name, -1) {
			declared["--"+m[1]], declared["--no-"+m[1]] = true, true // --[no-]color
		}
		for _, f := range flagScan.FindAllString(a.Name, -1) {
			declared[f] = true
		}
	}
	return declared, variadic
}

// unknownFlags reports the flags of an example that no argument row of the
// command declares. It is advice, so always a warning. It stays silent for a
// command with no rows (nothing says the flags are wrong), for one with a
// variadic row (...), which passes the rest on, and after a -- terminator. A
// --flag=value is read as --flag, and bundled short flags (-sv) are read one by
// one.
func unknownFlags(c registryEntry, path string, found []example) []Finding {
	if len(c.Args) == 0 {
		return nil
	}
	declared, variadic := declaredFlags(c)
	if variadic {
		return nil
	}
	on := regexp.MustCompile(`^just +` + regexp.QuoteMeta(path) + `(\s|$)`)
	var out []Finding
	for _, e := range found {
		if !on.MatchString(e.Text) {
			continue
		}
		var unknown []string
		for _, tok := range shellWords(strings.TrimPrefix(strings.TrimSpace(e.Text[len("just"):]), path)) {
			if tok == "--" {
				break
			}
			if !strings.HasPrefix(tok, "-") || len(tok) == 1 || (tok[1] >= '0' && tok[1] <= '9') {
				continue
			}
			name, _, _ := strings.Cut(tok, "=")
			if declared[name] {
				continue
			}
			if !strings.HasPrefix(name, "--") && !strings.Contains(tok, "=") {
				all := true
				for _, r := range name[1:] {
					if !declared["-"+string(r)] {
						all = false
					}
				}
				if all {
					continue
				}
			}
			unknown = append(unknown, textsafe.Line(name))
		}
		for _, f := range unknown {
			out = append(out, Finding{Rule: RuleExampleFlagUnknown, Severity: Warning, Subject: e.Subject, Message: fmt.Sprintf("%q passes %s, which no argument row of %s declares", e.Raw, f, c.Key), Fix: "add an argument row for the flag, or fix the example"})
		}
	}
	return out
}

// shellWords splits a line on white space, keeping a quoted string whole. It
// stops at an unquoted shell operator (| ; &) or a comment (# at the start of a
// word), since what follows is another command.
func shellWords(s string) []string {
	var out []string
	var cur strings.Builder
	var quote rune
	started, esc := false, false
	for _, r := range s {
		switch {
		case esc:
			esc = false
			cur.WriteRune(r)
		case r == '\\' && quote != '\'':
			esc = true
			cur.WriteRune(r)
		case quote != 0:
			if r == quote {
				quote = 0
			}
			cur.WriteRune(r) // quotes stay, so a quoted "-x" is not a flag
		case r == '"' || r == '\'':
			quote, started = r, true
			cur.WriteRune(r)
		case r == '|' || r == ';' || r == '&' || (r == '#' && cur.Len() == 0 && !started):
			if started || cur.Len() > 0 {
				out = append(out, cur.String())
			}
			return out
		case r == ' ' || r == '\t':
			if started || cur.Len() > 0 {
				out = append(out, cur.String())
				cur.Reset()
				started = false
			}
		default:
			cur.WriteRune(r)
		}
	}
	if started || cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out
}

// takesValue reports whether an argument row that names flag also declares a
// value for it: a <placeholder> anywhere in the row, or a value written right
// after the flag without brackets, as --date=YYYY-MM-DD or --only KIND. A comma
// (-v, --verbose), another flag or the end of the row is no value.
func takesValue(c registryEntry, flag string) bool {
	for _, a := range c.Args {
		for _, loc := range flagScan.FindAllStringIndex(a.Name, -1) {
			if a.Name[loc[0]:loc[1]] != flag {
				continue
			}
			if strings.Contains(a.Name, "<") {
				return true
			}
			rest := a.Name[loc[1]:]
			if strings.HasPrefix(rest, "=") {
				return true
			}
			next := strings.TrimLeft(strings.TrimSpace(rest), "[")
			if strings.HasPrefix(rest, " ") && next != "" && !strings.HasPrefix(next, "-") && !strings.HasPrefix(next, "]") && !strings.HasPrefix(next, "|") {
				return true
			}
		}
	}
	return false
}

// booleanValues reports an example that gives a value with = to a flag whose
// argument rows declare it with no value, such as -tail=200 for a [-tail] row.
// Many CLIs reject that form (Go's flag package: invalid boolean value). Only
// the = form is read: in -tail 200 the 200 can be a positional argument. It is
// advice, so a warning, and it stops at a -- terminator.
func booleanValues(c registryEntry, path string, found []example) []Finding {
	on := regexp.MustCompile(`^just +` + regexp.QuoteMeta(path) + `(\s|$)`)
	var out []Finding
	for _, e := range found {
		if !on.MatchString(e.Text) {
			continue
		}
		for _, tok := range shellWords(strings.TrimPrefix(strings.TrimSpace(e.Text[len("just"):]), path)) {
			if tok == "--" {
				break
			}
			name, value, ok := strings.Cut(tok, "=")
			if !ok || !strings.HasPrefix(name, "-") || !isBooleanFlag(c, name) || takesValue(c, name) {
				continue
			}
			out = append(out, Finding{Rule: RuleExampleBoolValue, Severity: Warning, Subject: e.Subject,
				Message: fmt.Sprintf("%q gives %s the value %q, but no argument row of %s declares a value for it", e.Raw, textsafe.Line(name), textsafe.Line(value), c.Key),
				Fix:     "drop the value, or write the row as " + textsafe.Line(name) + " <value> if the flag takes one"})
		}
	}
	return out
}
