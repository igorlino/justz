package lint

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/igorlino/justz/internal/check"
)

var camelBoundary = regexp.MustCompile(`([a-z0-9])([A-Z])`)

// Parts of a name that make it secret-like when a part ends in one of them.
var secretSuffixes = []string{"token", "tokens", "password", "passwd", "passphrase", "secret", "secrets", "credential", "credentials", "apikey"}

// Words that, as the last part of a name, say it is something about a secret
// and not the secret itself: SSH_KEY_FILE, TOKEN_PATH, OP_TOKEN_REF.
var notSecretLast = map[string]bool{
	"file": true, "path": true, "dir": true, "name": true, "id": true, "user": true, "url": true,
	"host": true, "ref": true, "arn": true, "ttl": true, "endpoint": true, "length": true,
	"count": true, "type": true, "args": true,
}

// Words before KEY that make API_KEY, PRIVATE_KEY and the like secret-like. A
// bare KEY is not enough: it matches too much.
var keyPrefixes = map[string]bool{"api": true, "private": true, "access": true, "secret": true, "signing": true}

// SecretLike reports whether a variable or parameter name looks like it holds a
// secret. It looks at the name only, never at a value. The name is split on
// underscores, dashes and camel-case boundaries, so API_TOKEN, apiToken and
// api-token are alike.
func SecretLike(name string) bool {
	parts := nameParts(name)
	if len(parts) == 0 || notSecretLast[parts[len(parts)-1]] {
		return false
	}
	for _, p := range parts {
		for _, s := range secretSuffixes {
			if strings.HasSuffix(p, s) {
				return true
			}
		}
	}
	if last := parts[len(parts)-1]; last == "pass" || last == "pw" {
		return true
	}
	for i := 1; i < len(parts); i++ {
		if parts[i] == "key" && keyPrefixes[parts[i-1]] {
			return true
		}
	}
	return false
}

func nameParts(name string) []string {
	name = camelBoundary.ReplaceAllString(name, "${1}_${2}")
	var parts []string
	for _, p := range strings.FieldsFunc(strings.ToLower(name), func(r rune) bool { return r == '_' || r == '-' }) {
		parts = append(parts, p)
	}
	return parts
}

// exprScope resolves a variable the way the recipe would: its own parameters
// (with their defaults) first, then the assignments of the justfile.
type exprScope struct {
	assignments map[string]any // name to its expression
	params      map[string]any // parameter name to its default expression, nil when none
}

// secretNames returns the secret-like names an expression reads, in the order
// found and without repeats. skip lists names that another rule already
// reports (the recipe's own secret-like parameters).
func (s exprScope) secretNames(expr any, skip map[string]bool) []string {
	var out []string
	seen := map[string]bool{}
	add := func(n string) {
		if !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	visiting := map[string]bool{}

	var walk func(e any)
	walk = func(e any) {
		arr, ok := e.([]any)
		if !ok || len(arr) == 0 {
			return // a literal string
		}
		head, _ := arr[0].(string)
		switch head {
		case "variable":
			if len(arr) < 2 {
				return
			}
			name, _ := arr[1].(string)
			if skip[name] {
				return
			}
			if SecretLike(name) {
				add(name)
				return
			}
			if visiting[name] {
				return
			}
			visiting[name] = true
			defer delete(visiting, name)
			if def, isParam := s.params[name]; isParam {
				walk(def)
			} else if val, isAssign := s.assignments[name]; isAssign {
				walk(val)
			}
		case "call":
			if len(arr) >= 3 {
				fn, _ := arr[1].(string)
				if fn == "env" || fn == "env_var" || fn == "env_var_or_default" {
					if name, ok := arr[2].(string); ok && SecretLike(name) {
						add(name)
					}
				}
			}
			for _, a := range arr[2:] {
				walk(a)
			}
		default:
			// concatenate, if, ==, !=, evaluate, join, and the like: look inside.
			for _, a := range arr[1:] {
				walk(a)
			}
		}
	}
	walk(expr)
	sort.Strings(out)
	return out
}

// secretRules reports secrets that reach a command line. It reads names only,
// never values, and no message carries the text of a line.
//
// A parameter is typed on the command line, so just itself shows its value in
// the process list. An interpolation {{ ... }} in a linewise recipe is replaced
// before just starts the shell, so the value is in the shell's argument list
// whatever the command is. Shebang and [script] recipes write the text to a
// file instead, and are left out.
func secretRules(d dump, r recipe, subject string, ignored map[string]bool, res *Result) {
	own := map[string]bool{} // the recipe's secret-like parameters, reported once below
	params := map[string]any{}
	for _, p := range r.Parameters {
		params[p.Name] = p.Default
		if SecretLike(p.Name) {
			own[p.Name] = true
			if !ignored[RuleSecretParameter] {
				res.Findings = append(res.Findings, Finding{
					Rule: RuleSecretParameter, Severity: check.Warning, Subject: subject,
					Message: fmt.Sprintf("parameter %q looks like a secret: its value is typed on the command line, so it is in the process list and the shell history", p.Name),
					Fix:     "take no secret as a parameter: read it inside the recipe from the environment, a mode 0600 file or the password manager (see credentials-in-recipes.md in the justfile-authoring skill)",
				})
			}
		}
	}
	if r.Shebang || ignored[RuleSecretInterpol] {
		return
	}
	assignments := map[string]any{}
	for name, a := range d.Assignments {
		assignments[name] = a.Value
	}
	scope := exprScope{assignments: assignments, params: params}
	for i, line := range r.Body {
		items, ok := line.([]any)
		if !ok {
			continue
		}
		var names []string
		for _, item := range items {
			// An interpolation is a list holding one expression; text is a string.
			if interp, ok := item.([]any); ok && len(interp) > 0 {
				names = append(names, scope.secretNames(interp[0], own)...)
			}
		}
		if len(names) == 0 {
			continue
		}
		names = uniqueSorted(names)
		res.Findings = append(res.Findings, Finding{
			Rule: RuleSecretInterpol, Severity: check.Warning, Subject: subject, Line: i + 1,
			Message: fmt.Sprintf("interpolates %s into a command line: just substitutes the value before the shell starts, so it is in the process list", strings.Join(names, ", ")),
			Fix:     "use the shell's own variable, with the value exported from the environment ($NAME), so the value never appears in the command line",
		})
	}
}

func uniqueSorted(in []string) []string {
	sort.Strings(in)
	out := in[:0]
	for i, s := range in {
		if i == 0 || s != in[i-1] {
			out = append(out, s)
		}
	}
	return out
}
