package shim

import (
	"os"
	"path/filepath"
	"strings"
)

// marker identifies a stub that justz wrote. justz only ever replaces or
// removes files that contain it.
const marker = "managed by justz"

// stubPath is where the stub for a command name goes. On Windows it is a
// .cmd file.
func (d Dirs) stubPath(name string) string {
	if d.GOOS == "windows" {
		return filepath.Join(d.Shim, name+".cmd")
	}
	return filepath.Join(d.Shim, name)
}

// stubContent is the text of a stub. The stub carries no settings: it only
// asks justz to behave as the named shim, so the registry stays the single
// source of truth and a moved repository needs one edit.
func (d Dirs) stubContent(name string) string {
	if d.GOOS == "windows" {
		return "@echo off\r\n" +
			"rem " + marker + "; remove it with: justz shim remove " + name + "\r\n" +
			"justz --as " + name + " %*\r\n"
	}
	return "#!/bin/sh\n" +
		"# " + marker + "; remove it with: justz shim remove " + name + "\n" +
		"exec justz --as " + name + " \"$@\"\n"
}

// isOurs reports whether the file at path is a stub that justz wrote.
func isOurs(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	buf := make([]byte, 4096)
	n, _ := f.Read(buf)
	return strings.Contains(string(buf[:n]), marker)
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}
