// Package atomicfile writes files so that a crash never leaves a half-written
// one: the data goes to a temporary file in the same directory and is then
// renamed over the target.
package atomicfile

import (
	"fmt"
	"os"
	"path/filepath"
)

// Write replaces path with data. If path already exists it keeps that file's
// permission bits, otherwise it uses perm. It refuses to write through a
// symbolic link, so a link planted in a project can never redirect the write.
func Write(path string, data []byte, perm os.FileMode) error {
	mode := perm
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%s is a symbolic link; refusing to write through it", path)
		}
		mode = info.Mode().Perm()
	} else if !os.IsNotExist(err) {
		return err
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), ".justz-tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := func() { os.Remove(tmpName) }

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return err
	}
	if err := os.Chmod(tmpName, mode); err != nil {
		cleanup()
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		cleanup()
		return err
	}
	return nil
}
