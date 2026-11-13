package registry

import (
	"fmt"
	"os"

	"github.com/igorlino/justz/internal/orderedjson"
)

// CLIName reads the top-level "cli" field of the registry at path: the name
// the project wants its command shim to have.
func CLIName(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	doc, err := orderedjson.Parse(data)
	if err != nil {
		return "", fmt.Errorf("%s is not valid JSON: %v", path, err)
	}
	cli := doc.Get("cli")
	if cli == nil || cli.Kind != orderedjson.String || cli.Str == "" {
		return "", fmt.Errorf(`%s has no "cli" field, so there is no shim name to use`, path)
	}
	return cli.Str, nil
}
