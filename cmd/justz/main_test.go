package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func testEnv(just bool) (env, *bytes.Buffer, *bytes.Buffer) {
	var out, errOut bytes.Buffer
	e := env{
		stdout: &out,
		stderr: &errOut,
		lookPath: func(file string) (string, error) {
			if just && file == "just" {
				return "/usr/bin/just", nil
			}
			return "", errors.New("not found")
		},
		runCmd: func(name string, args ...string) (string, error) {
			return "just 1.59.0", nil
		},
	}
	return e, &out, &errOut
}

func TestRun(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		just     bool
		wantCode int
		wantOut  string
		wantErr  string
	}{
		{"version flag", []string{"--version"}, true, 0, "justz dev\n", ""},
		{"help flag", []string{"--help"}, true, 0, "Usage:", ""},
		{"no arguments", nil, true, 2, "", "Usage:"},
		{"unknown command", []string{"nope"}, true, 2, "", `unknown command "nope"`},
		{"doctor ok", []string{"doctor"}, true, 0, "[ok] just: /usr/bin/just (just 1.59.0)", ""},
		{"doctor without just", []string{"doctor"}, false, 1, "[fail] just: not found on PATH", ""},
		{"doctor bad flag", []string{"doctor", "--wat"}, true, 2, "", `unexpected argument "--wat"`},
		{"shim list", []string{"shim", "list"}, true, 0, "No shims registered.", ""},
		{"shim without subcommand", []string{"shim"}, true, 2, "", "expected a subcommand"},
		{"shim unknown subcommand", []string{"shim", "add"}, true, 2, "", "expected a subcommand"},
		{"shim list bad flag", []string{"shim", "list", "--wat"}, true, 2, "", `unexpected argument "--wat"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, out, errOut := testEnv(tt.just)
			code := run(tt.args, e)
			if code != tt.wantCode {
				t.Errorf("exit code = %d, want %d", code, tt.wantCode)
			}
			if !strings.Contains(out.String(), tt.wantOut) {
				t.Errorf("stdout = %q, want it to contain %q", out.String(), tt.wantOut)
			}
			if !strings.Contains(errOut.String(), tt.wantErr) {
				t.Errorf("stderr = %q, want it to contain %q", errOut.String(), tt.wantErr)
			}
		})
	}
}

func TestDoctorJSON(t *testing.T) {
	e, out, _ := testEnv(true)
	if code := run([]string{"doctor", "--json"}, e); code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	var got doctorReport
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out.String())
	}
	if !got.OK || got.Version != "dev" || len(got.Checks) != 1 || got.Checks[0].Name != "just" || !got.Checks[0].OK {
		t.Errorf("unexpected report: %+v", got)
	}
}

func TestDoctorJSONFailure(t *testing.T) {
	e, out, _ := testEnv(false)
	if code := run([]string{"doctor", "--json"}, e); code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	var got doctorReport
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if got.OK || got.Checks[0].OK {
		t.Errorf("report should be failing: %+v", got)
	}
}

func TestShimListJSON(t *testing.T) {
	e, out, _ := testEnv(true)
	if code := run([]string{"shim", "list", "--json"}, e); code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(out.String(), `"shims": []`) {
		t.Errorf("want an empty shims array, got %q", out.String())
	}
}
