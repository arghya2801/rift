package workfile

import (
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
)

// Write replaces a regular file. expected is the exact version last read by the editor.
func Write(dir, rel, text string, expected *string) error {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return err
	}
	defer root.Close()
	rel = filepath.FromSlash(rel)
	parent, err := root.OpenRoot(filepath.Dir(rel))
	if err != nil {
		return err
	}
	defer parent.Close()
	name := filepath.Base(rel)
	info, err := parent.Lstat(name)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if info != nil && !info.Mode().IsRegular() {
		return errors.New("only regular files can be edited")
	}
	mode := os.FileMode(0o644)
	if info != nil {
		mode = info.Mode().Perm()
	}
	tmp := ".rift-save-" + rand.Text()
	f, err := parent.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	defer parent.Remove(tmp)
	if _, err = f.WriteString(text); err == nil && info != nil {
		err = f.Chmod(mode)
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if expected != nil {
		current, err := parent.ReadFile(name)
		if err != nil {
			return err
		}
		if string(current) != *expected {
			return errors.New("file changed on disk since it was opened")
		}
	}
	return parent.Rename(tmp, name)
}
