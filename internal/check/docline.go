package check

import "strings"

// DocLine returns the [doc(...)] attribute that gives a recipe the
// description. A string in just has no escape in single quotes, so a
// description without a single quote is written that way. One with a single
// quote is written in double quotes, where just reads \\ and \" as a backslash
// and a double quote and keeps everything else, including ' and {{, as it is.
// The text is trimmed, as the comparison with the registry trims it.
func DocLine(desc string) string {
	desc = strings.TrimSpace(desc)
	if !strings.Contains(desc, "'") {
		return "[doc('" + desc + "')]"
	}
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`)
	return `[doc("` + r.Replace(desc) + `")]`
}
