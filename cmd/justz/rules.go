package main

import (
	"fmt"
	"sort"
	"strings"
)

// ruleFlag reads --rule NAME or --rule=NAME at args[*i], advancing *i past a
// separate value. The flag repeats and a value may hold several names
// separated by commas. ok is false when the argument is not --rule, and err
// is set when it has no value.
func ruleFlag(args []string, i *int, rules *[]string) (ok bool, err string) {
	a := args[*i]
	var v string
	switch {
	case strings.HasPrefix(a, "--rule="):
		v = strings.TrimPrefix(a, "--rule=")
	case a == "--rule":
		if *i+1 >= len(args) {
			return true, "--rule needs a value"
		}
		*i++
		v = args[*i]
	default:
		return false, ""
	}
	for _, name := range strings.Split(v, ",") {
		name = strings.TrimSpace(name)
		if name == "" {
			return true, "--rule needs a rule name"
		}
		dup := false
		for _, r := range *rules {
			dup = dup || r == name
		}
		if !dup {
			*rules = append(*rules, name)
		}
	}
	return true, ""
}

// unknownRule returns the first name that is not a valid rule, with the usage
// message that lists the valid names.
func unknownRule(cmd string, asked, valid []string) string {
	ok := map[string]bool{}
	for _, r := range valid {
		ok[r] = true
	}
	for _, name := range asked {
		if !ok[name] {
			sorted := append([]string{}, valid...)
			sort.Strings(sorted)
			return fmt.Sprintf("justz %s: unknown rule %q; valid rules: %s", cmd, name, strings.Join(sorted, ", "))
		}
	}
	return ""
}

func ruleSet(rules []string) map[string]bool {
	if len(rules) == 0 {
		return nil
	}
	m := map[string]bool{}
	for _, r := range rules {
		m[r] = true
	}
	return m
}

// fixRuns marks where a fix is printed: under the first finding of each rule
// and fix text, wherever the findings fall in the order (one rule can span the
// error and the warning block). show[i] is true at that first finding, and
// count[i] is the number of findings in the whole list that share the rule and
// the fix.
func fixRuns(rule, fix []string) (show []bool, count []int) {
	show, count = make([]bool, len(rule)), make([]int, len(rule))
	first := map[string]int{}
	for i := range rule {
		key := rule[i] + "\x00" + fix[i]
		at, ok := first[key]
		if !ok {
			first[key], at = i, i
			show[i] = true
		}
		count[at]++
	}
	return show, count
}

// fixLine is the text under a finding that starts a run.
func fixLine(fix string, n int) string {
	line := "fix: " + strings.TrimSpace(fix)
	if n > 1 {
		line += fmt.Sprintf("  (the same fix applies to all %d findings of this rule)", n)
	}
	return line
}

func ruleScope(rules []string) string {
	return strings.Join(rules, ", ")
}

// hiddenNote is the line printed to stderr when --rule hides findings of error
// severity: a filter is for reading, and it can turn a failing run into an
// exit 0.
func hiddenNote(hidden, hiddenErrors int) string {
	if hiddenErrors == 0 {
		return ""
	}
	return fmt.Sprintf("note: --rule hides %d other findings (%s); the exit code follows the shown findings only, so do not gate CI on --rule", hidden, plural(hiddenErrors, "error"))
}
