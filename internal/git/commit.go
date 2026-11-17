package git

import (
	"os"
	"path/filepath"
)

func CommitStaged(dir, message string, amend bool) error {
	args := []string{"commit", "-q", "-m", message}
	if amend {
		args = append(args, "--amend")
	}
	_, err := Run(dir, args...)
	return err
}

func LastMessage(dir string) string {
	out, _ := Run(dir, "log", "-1", "--format=%B")
	return out
}

type PartialFile struct {
	Path   string  `json:"path"`
	Text   *string `json:"text"`
	Delete bool    `json:"delete"`
}

// CommitPartial commits files through a temp index seeded from HEAD, leaving the real index alone
// except for resetting the committed paths to the new HEAD.
func CommitPartial(dir, message string, amend bool, files []PartialFile) error {
	f, err := os.CreateTemp("", "rift-index-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	f.Close()
	os.Remove(tmp)
	defer os.Remove(tmp)
	env := []string{"GIT_INDEX_FILE=" + tmp}

	if _, err := Run(dir, "rev-parse", "--verify", "-q", "HEAD"); err == nil {
		_, err = RunEnv(dir, env, "read-tree", "HEAD")
		if err != nil {
			return err
		}
	} else if _, err := RunEnv(dir, env, "read-tree", "--empty"); err != nil {
		return err
	}

	paths := make([]string, 0, len(files))
	for _, pf := range files {
		paths = append(paths, pf.Path)
		switch {
		case pf.Delete:
			_, err = RunEnv(dir, env, "update-index", "--force-remove", "--", pf.Path)
		case pf.Text != nil:
			err = StageContent(dir, tmp, pf.Path, *pf.Text)
		default:
			if _, statErr := os.Stat(filepath.Join(dir, pf.Path)); os.IsNotExist(statErr) {
				_, err = RunEnv(dir, env, "update-index", "--force-remove", "--", pf.Path)
			} else {
				_, err = RunEnv(dir, env, "update-index", "--add", "--", pf.Path)
			}
		}
		if err != nil {
			return err
		}
	}

	args := []string{"commit", "-q", "-m", message}
	if amend {
		args = append(args, "--amend")
	}
	if _, err := RunEnv(dir, env, args...); err != nil {
		return err
	}
	if len(paths) > 0 {
		_, err = Run(dir, append([]string{"reset", "-q", "--"}, paths...)...)
	}
	return err
}

// HeadSides is HEAD vs the working tree file, both in worktree form.
func HeadSides(dir, path string) (Sides, error) {
	cur, err := readCapped(filepath.Join(dir, path))
	return sides(blob(dir, "HEAD:"+path), cur), err
}
