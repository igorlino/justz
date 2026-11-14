package main

import (
	"fmt"

	"github.com/igorlino/justz/schema"
)

const schemaUsage = `Usage: justz schema

Print the JSON Schema of the command registry (commands.json) to standard
output. Point an editor at it for completion and type checks, or save it:

  justz schema > commands.schema.json

The schema checks the shape of the file only: types and nesting. Unknown fields
are allowed. Whether the registry matches the justfile is the job of justz
check. The schema is part of this binary; justz never fetches it.

Exit codes: 0 printed, 2 usage error.
`

func runSchema(args []string, e env) int {
	for _, a := range args {
		if a == "--help" || a == "-h" {
			fmt.Fprint(e.stdout, schemaUsage)
			return 0
		}
		fmt.Fprintf(e.stderr, "justz schema: unexpected argument %q\n\n%s", a, schemaUsage)
		return 2
	}
	fmt.Fprint(e.stdout, schema.Commands)
	return 0
}
