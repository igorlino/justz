package main

import (
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/igorlino/justz/internal/textsafe"
)

// minJust is the oldest just that justz supports: from 1.37.0 the JSON dump
// carries the doc of a [doc] recipe, and everything before it fails or reports
// a false doc-missing on a justfile that `justz init` wrote. Verified by
// running check, help and lint against 1.27.0 to 1.46.0 (see the README).
// Two features need a newer just than this baseline: [metadata] (the
// justz-ignore opt-out) needs 1.42.0, and [arg] needs 1.46.0.
const minJust = "1.37.0"

// minJustMetadata is the first just that accepts the [metadata] attribute.
const minJustMetadata = "1.42.0"

const justUpgradeHint = "upgrade with `brew install just`, the install script (https://just.systems/install.sh) or `cargo install just`"

// stderrLines and stderrBytes bound the just stderr that goes into an error.
const (
	stderrLines = 5
	stderrBytes = 600
)

// cmdError is a failed command together with what it wrote to stderr.
type cmdError struct {
	err    error
	stderr string
}

func (c *cmdError) Error() string { return c.err.Error() }
func (c *cmdError) Unwrap() error { return c.err }

// runOutput runs a command and returns its trimmed stdout. A command that
// exits non-zero returns a *cmdError that keeps its stderr.
func runOutput(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).Output()
	var ee *exec.ExitError
	if errors.As(err, &ee) && len(ee.Stderr) > 0 {
		err = &cmdError{err: err, stderr: string(ee.Stderr)}
	}
	return strings.TrimSpace(string(out)), err
}

// parseVersion reads "1.21.0" from "just 1.21.0" (or a bare version) as
// numbers. It returns false when there is no version in the text.
func parseVersion(s string) ([3]int, bool) {
	f := strings.Fields(s)
	if len(f) == 0 {
		return [3]int{}, false
	}
	parts := strings.SplitN(f[len(f)-1], ".", 3)
	if len(parts) < 2 {
		return [3]int{}, false
	}
	var v [3]int
	for i, p := range parts {
		end := 0
		for end < len(p) && p[end] >= '0' && p[end] <= '9' {
			end++
		}
		n, err := strconv.Atoi(p[:end])
		if err != nil {
			return [3]int{}, false
		}
		v[i] = n
	}
	return v, true
}

// versionBelow reports whether the version in have is older than min.
// A text with no readable version is never below.
func versionBelow(have, min string) bool {
	h, ok := parseVersion(have)
	if !ok {
		return false
	}
	m, _ := parseVersion(min)
	for i := range h {
		if h[i] != m[i] {
			return h[i] < m[i]
		}
	}
	return false
}

// justReadError describes a failed read of the justfile: just's own reason
// (its first lines, cleaned for the terminal) and, when the reason names an
// attribute or a setting that just does not know, a hint about an old just.
func justReadError(e env, err error) error {
	msg := "could not read the justfile with just: " + err.Error()
	var ce *cmdError
	if !errors.As(err, &ce) {
		return errors.New(msg)
	}
	text := boundedStderr(ce.stderr)
	if text == "" {
		return errors.New(msg)
	}
	msg += "\n" + indent(text)
	low := strings.ToLower(text)
	ver, verr := e.runCmd("just", "--version")
	if verr != nil {
		ver = ""
	}
	named := strings.Contains(low, "unknown attribute") || strings.Contains(low, "unknown setting") ||
		strings.Contains(low, "invalid attribute")
	// A current just that rejects an attribute is a typo, not an old just; only
	// when the version cannot be read does the message text decide.
	old := versionBelow(ver, minJust)
	if ver == "" {
		old = named
	}
	if old {
		hint := "an old just is a likely cause; justz needs just " + minJust + " or later"
		if ver != "" {
			hint += ", this one is " + textsafe.Line(ver)
		}
		msg += "\n" + hint + "; " + justUpgradeHint
	}
	return errors.New(msg)
}

// boundedStderr cleans the text for the terminal and keeps its first lines
// and its first bytes.
func boundedStderr(s string) string {
	var out []string
	for _, l := range strings.Split(textsafe.Text(s), "\n") {
		if l = strings.TrimRight(l, " \t\r"); l != "" {
			out = append(out, l)
		}
		if len(out) == stderrLines {
			break
		}
	}
	text := strings.Join(out, "\n")
	if len(text) > stderrBytes {
		text = strings.ToValidUTF8(text[:stderrBytes], "") + "..."
	}
	return text
}

func indent(s string) string {
	return "  " + strings.ReplaceAll(s, "\n", "\n  ")
}

// justVersionNote is the doctor text for a just older than the baseline.
func justVersionNote(path, ver string) string {
	return fmt.Sprintf("%s (%s) is older than %s, the oldest justz supports; %s. [metadata] needs %s",
		path, ver, minJust, justUpgradeHint, minJustMetadata)
}
