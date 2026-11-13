package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/igorlino/justz/internal/orderedjson"
	"github.com/igorlino/justz/internal/registry"
)

// projectInfo is what justz knows about one justfile: the registry that groups its
// commands (if there is one) and the recipes it defines at the top level.
type projectInfo struct {
	Justfile    string
	Registry    *orderedjson.Value // nil when the project has no registry
	RootRecipes map[string]bool
}

// loadProject reads the project around a justfile. A missing registry is not an
// error: the project is then "native" and has nothing to resolve. If just
// cannot dump the justfile, rootRecipes is empty and err says why.
func loadProject(e env, justfile string) (*projectInfo, error) {
	p := &projectInfo{Justfile: justfile, RootRecipes: map[string]bool{}}

	if path, err := registry.Discover(filepath.Dir(justfile)); err == nil {
		data, err := os.ReadFile(path)
		if err != nil {
			return p, err
		}
		doc, err := orderedjson.Parse(data)
		if err != nil {
			return p, fmt.Errorf("%s is not valid JSON: %v", path, err)
		}
		p.Registry = doc
	}

	out, err := e.runCmd("just", "--justfile", justfile, "--dump", "--dump-format", "json")
	if err != nil {
		return p, fmt.Errorf("could not read the justfile with just: %v", err)
	}
	var dump struct {
		Recipes map[string]struct {
			Private bool `json:"private"`
		} `json:"recipes"`
	}
	if err := json.Unmarshal([]byte(out), &dump); err != nil {
		return p, fmt.Errorf("could not understand the output of just --dump: %v", err)
	}
	for name, r := range dump.Recipes {
		if !r.Private {
			p.RootRecipes[name] = true
		}
	}
	return p, nil
}
