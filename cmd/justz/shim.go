package main

import (
	"encoding/json"
	"fmt"
)

type shim struct {
	Name     string `json:"name"`
	Justfile string `json:"justfile"`
}

type shimList struct {
	Shims []shim `json:"shims"`
}

func runShim(args []string, e env) int {
	if len(args) == 0 || args[0] != "list" {
		fmt.Fprint(e.stderr, "justz shim: expected a subcommand, one of: list\n")
		return 2
	}
	asJSON, err := parseJSONFlag(args[1:])
	if err != nil {
		fmt.Fprintf(e.stderr, "justz shim list: %v\n", err)
		return 2
	}

	// The shim registry does not exist yet, so there is never anything to list.
	list := shimList{Shims: []shim{}}

	if asJSON {
		enc := json.NewEncoder(e.stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(list); err != nil {
			fmt.Fprintf(e.stderr, "justz shim list: %v\n", err)
			return 1
		}
		return 0
	}
	if len(list.Shims) == 0 {
		fmt.Fprintln(e.stdout, "No shims registered.")
		return 0
	}
	for _, s := range list.Shims {
		fmt.Fprintf(e.stdout, "%s\t%s\n", s.Name, s.Justfile)
	}
	return 0
}
