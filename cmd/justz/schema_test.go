package main

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/igorlino/justz/schema"
)

func TestSchemaPrintsTheEmbeddedSchema(t *testing.T) {
	r := checkEnv("{}")
	if code := run([]string{"schema"}, r.env); code != 0 {
		t.Fatalf("exit %d: %s", code, r.errOut.String())
	}
	if r.out.String() != schema.Commands {
		t.Error("the output is not the embedded schema, byte for byte")
	}
	var v map[string]any
	if err := json.Unmarshal(r.out.Bytes(), &v); err != nil {
		t.Errorf("the output is not JSON: %v", err)
	}
	if r.errOut.Len() != 0 {
		t.Errorf("stderr: %s", r.errOut.String())
	}
}

func TestSchemaNeedsNoProjectAndNoJust(t *testing.T) {
	r := newRuntimeEnv()
	r.env.lookPath = func(string) (string, error) { return "", errors.New("not found") }
	if code := run([]string{"schema"}, r.env); code != 0 {
		t.Fatalf("exit %d: %s", code, r.errOut.String())
	}
}

func TestSchemaUsage(t *testing.T) {
	r := checkEnv("{}")
	if code := run([]string{"schema", "--help"}, r.env); code != 0 || !strings.HasPrefix(r.out.String(), "Usage: justz schema") {
		t.Errorf("--help: exit %d, out %q", code, r.out.String())
	}
	r = checkEnv("{}")
	if code := run([]string{"schema", "--json"}, r.env); code != 2 || !strings.Contains(r.errOut.String(), `unexpected argument "--json"`) {
		t.Errorf("--json: exit %d, err %q", code, r.errOut.String())
	}
	if r.out.Len() != 0 {
		t.Errorf("stdout is not empty on a usage error: %q", r.out.String())
	}
}
