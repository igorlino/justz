package shim

import (
	"fmt"
	"regexp"
)

var validName = regexp.MustCompile(`^[a-z][a-z0-9-]{0,31}$`)

// reserved are names a shim may never take: justz and just themselves, shell
// builtins and keywords, and everyday commands that would break scripts if
// hidden.
var reserved = map[string]bool{}

func init() {
	for _, n := range []string{
		"just", "justz",
		// shell builtins and keywords
		"alias", "bg", "cd", "command", "echo", "eval", "exec", "exit", "export", "fg", "jobs",
		"kill", "printf", "pwd", "read", "set", "source", "test", "time", "type", "ulimit",
		"umask", "unalias", "unset", "wait", "which", "if", "for", "while", "case", "do", "done",
		// everyday commands
		"awk", "bash", "cat", "chmod", "chown", "cp", "curl", "dash", "env", "fish", "git", "go",
		"grep", "gzip", "ls", "make", "mkdir", "mv", "node", "npm", "python", "python3", "rm",
		"rmdir", "sed", "sh", "ssh", "sudo", "tar", "wget", "zsh",
	} {
		reserved[n] = true
	}
}

// ValidateName checks one name against the allowed pattern and the reserved list.
func ValidateName(name string) error {
	if !validName.MatchString(name) {
		return fmt.Errorf("%q is not a valid shim name: use lower-case letters, digits and dashes, starting with a letter, at most 32 characters", name)
	}
	if reserved[name] {
		return fmt.Errorf("%q is reserved: it is a shell builtin, a core command, or justz itself", name)
	}
	return nil
}
