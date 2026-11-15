package check

import "fmt"

const seeAlsoFix = "use the key of another command, as group.command (core.build for a top-level command), list each key once, or remove it"

// seeAlsoRules reports a seeAlso key of a command that names no entry of the
// registry, or the command itself, and a key listed twice. A key is group.command (core.build for a
// top-level command); it is compared whole with the group key, a dot and the
// command key, so nothing is parsed. A command in extra counts as an entry.
// It is an error, like registry-stale: a reference to a command that is gone.
func seeAlsoRules(entries []registryEntry) []Finding {
	known := map[string]bool{}
	for _, e := range entries {
		known[e.Group+"."+e.Key] = true
	}
	var out []Finding
	for _, e := range entries {
		self := e.Group + "." + e.Key
		seen, dup := map[string]bool{}, map[string]bool{}
		for _, key := range e.SeeAlso {
			if seen[key] {
				if !dup[key] {
					dup[key] = true
					out = append(out, Finding{Rule: RuleSeeAlso, Severity: Error, Subject: self,
						Message: fmt.Sprintf("seeAlso lists %q more than once", key),
						Fix:     seeAlsoFix})
				}
				continue
			}
			seen[key] = true
			var msg string
			switch {
			case key == self:
				msg = fmt.Sprintf("seeAlso names the command itself, %q", key)
			case !known[key]:
				msg = fmt.Sprintf("seeAlso names %q, which is not a command of the registry", key)
			default:
				continue
			}
			out = append(out, Finding{Rule: RuleSeeAlso, Severity: Error, Subject: self, Message: msg,
				Fix: seeAlsoFix})
		}
	}
	return out
}
