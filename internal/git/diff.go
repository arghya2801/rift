package git

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// MaxBlob caps how much of a file a diff loads; bigger files are shown as binary.
const MaxBlob = 4 << 20

// Sides is the before/after text of one file, both in working-tree form (smudged),
// so autocrlf repos don't show every line as changed.
type Sides struct {
	Old    string `json:"old"`
	New    string `json:"new"`
	Binary bool   `json:"binary"`
}

// FileSides compares index to worktree, or HEAD to index when staged.
func FileSides(dir, path string, staged bool) (Sides, error) {
	if staged {
		b := blobs(dir, "HEAD:"+path, ":"+path)
		return sides(b[0], b[1]), nil
	}
	old := blobs(dir, ":"+path)[0]
	cur, err := readCapped(filepath.Join(dir, path))
	return sides(old, cur), err
}

func sides(old, cur string) Sides {
	if old == "\x00" || cur == "\x00" || strings.IndexByte(old, 0) >= 0 || strings.IndexByte(cur, 0) >= 0 {
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

func blob(dir, spec string) string {
	return blobs(dir, spec)[0]
}

// blobs reads several <rev>:<path> specs, smudged, concurrently (wall time of one spawn).
// Missing objects are "" (added or deleted file); oversized ones are "\x00" (shown as binary).
// ponytail: `cat-file --batch --filters` fails with "missing path" on git 2.53, so one process per spec.
func blobs(dir string, specs ...string) []string {
	res := make([]string, len(specs))
	var wg sync.WaitGroup
	for i, spec := range specs {
		wg.Go(func() {
			cmd := command(dir, nil, "cat-file", "--filters", spec)
			out, err := cmd.StdoutPipe()
			if err != nil || cmd.Start() != nil {
				return
			}
			b, _ := io.ReadAll(io.LimitReader(out, MaxBlob+1))
			if len(b) > MaxBlob {
				_ = out.Close()
				_ = cmd.Process.Kill()
				_ = cmd.Wait()
				res[i] = "\x00"
				return
			}
			if cmd.Wait() == nil {
				res[i] = string(b)
			}
		})
	}
	wg.Wait()
	return res
}
