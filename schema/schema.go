// Package schema holds the JSON Schema of the command registry (commands.json).
package schema

import _ "embed"

// Commands is the schema of the command registry, as JSON.
//
//go:embed commands.schema.json
var Commands string
