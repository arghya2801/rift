package git

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"unicode/utf8"
)

// MaxBlob caps how much of a file a diff loads; bigger files are shown as binary.
const MaxBlob = 4 << 20

// Sides is the before/after text of one file, both in working-tree form (smudged),
// so autocrlf repos don't show every line as changed.
type Sides struct {
	ReadOnly bool   `json:"readOnly,omitempty"`
	Old      string `json:"old"`
	New      string `json:"new"`
	Binary   bool   `json:"binary"`
}

// FileSides compares index to worktree, or HEAD to index when staged.
func FileSides(dir, path string, staged bool, oldPath string) (Sides, error) {
	if oldPath == "" {
		oldPath = path
	}
	if staged {
		return blobSides(dir, "HEAD:"+oldPath, ":"+path)
	}
	return workingSides(dir, path, ":"+path)
}

func workingSides(dir, path, spec string) (Sides, error) {
	p := filepath.Join(dir, path)
	if fi, err := os.Lstat(p); err == nil {
		if fi.Mode()&os.ModeSymlink != 0 {
			return Sides{Binary: true}, nil
		}
		if fi.IsDir() {
			return submoduleSides(dir, path, spec)
		}
	}
	old, err := blob(dir, spec)
	if err != nil {
		return Sides{}, err
	}
	cur, err := readCapped(filepath.Join(dir, path))
	return sides(old, cur), err
}

func submoduleSides(dir, path, spec string) (Sides, error) {
	s := Sides{ReadOnly: true}
	record, err := Run(dir, "--literal-pathspecs", "ls-files", "--stage", "--", path)
	if err != nil {
		return s, err
	}
	if !strings.HasPrefix(record, "160000 ") {
		return s, errors.New("directory is not a tracked submodule")
	}
	index := strings.Fields(record)[1]
	old, err := Run(dir, "rev-parse", "--verify", spec)
	if err == nil {
		s.Old = "Subproject commit " + strings.TrimSpace(old) + "\n"
	} else if !strings.HasPrefix(spec, "HEAD:") {
		return s, err
	}
	p := filepath.Join(dir, path)
	if _, err := os.Lstat(filepath.Join(p, ".git")); os.IsNotExist(err) {
		s.New = "Subproject commit " + index + "\n"
		return s, nil
	}
	head, err := Run(p, "rev-parse", "--verify", "HEAD")
	if err != nil {
		return s, err
	}
	dirty, err := Run(p, "status", "--porcelain", "--untracked-files=normal")
	if err != nil {
		return s, err
	}
	head = strings.TrimSpace(head)
	if dirty != "" {
		head += "-dirty"
	}
	s.New = "Subproject commit " + head + "\n"
	return s, nil
}

func isText(s string) bool { return utf8.ValidString(s) && strings.IndexByte(s, 0) < 0 }

func sides(old, cur string) Sides {
	if !isText(old) || !isText(cur) {
		return Sides{Binary: true}
	}
	return Sides{Old: old, New: cur}
}

func readCapped(p string) (string, error) {
	fi, err := os.Stat(p)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if fi.Size() > MaxBlob {
		return "\x00", nil
	}
	b, err := os.ReadFile(p)
	return string(b), err
}

func blob(dir, spec string) (string, error) {
	cmd := command(dir, nil, "cat-file", "--filters", spec)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	if err = cmd.Start(); err != nil {
		return "", err
	}
	b, readErr := io.ReadAll(io.LimitReader(out, MaxBlob+1))
	if len(b) > MaxBlob {
		_ = out.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return "\x00", nil
	}
	waitErr := cmd.Wait()
	if err := readErr; err != nil {
		return "", err
	}
	if waitErr != nil {
		// Added and deleted files legitimately have no object on one side.
		probe := command(dir, nil, "rev-parse", "--verify", "--quiet", spec)
		var missing *exec.ExitError
		if errors.As(probe.Run(), &missing) && missing.ExitCode() == 1 {
			return "", nil
		}
		return "", fmt.Errorf("read %s: %w: %s", spec, waitErr, strings.TrimSpace(stderr.String()))
	}
	return string(b), nil
}

func blobs(dir string, specs ...string) ([]string, error) {
	results := make([]string, len(specs))
	errs := make([]error, len(specs))
	var wg sync.WaitGroup
	for i, spec := range specs {
		wg.Go(func() { results[i], errs[i] = blob(dir, spec) })
	}
	wg.Wait()
	return results, errors.Join(errs...)
}

func blobSides(dir, before, after string) (Sides, error) {
	b, err := blobs(dir, before, after)
	if err != nil {
		return Sides{}, err
	}
	return sides(b[0], b[1]), nil
}
