package main

import (
	"runtime/debug"
	"strings"
)

// currentVersion returns the stamped version, or for a plain `go install
// github.com/igorlino/justz/cmd/justz@vX.Y.Z` build, the module version that
// Go recorded in the binary.
func currentVersion() string {
	if version != "dev" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		if v := info.Main.Version; v != "" && v != "(devel)" {
			return strings.TrimPrefix(v, "v")
		}
	}
	return version
}
