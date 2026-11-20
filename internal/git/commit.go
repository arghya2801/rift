package git

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	ExpectedBase *string `json:"expectedBase,omitempty"`
	Path         string  `json:"path"`
	Text         *string `json:"text"`
	Delete       bool    `json:"delete"`
}

type IndexRepair struct {
	Head    string   `json:"head"`
	Paths   []string `json:"paths"`
	Index   string   `json:"index"`
	Message string   `json:"message"`
}

func (r *IndexRepair) Error() string { return r.Message }

func indexEntries(dir string, paths []string) (string, error) {
	return Run(dir, append([]string{"--literal-pathspecs", "ls-files", "--stage", "-z", "--"}, paths...)...)
}

func RepairPartialIndex(dir string, repair IndexRepair) error {
	if len(repair.Paths) == 0 || repair.Head == "" {
		return errors.New("missing index repair details")
	}
	head, err := Run(dir, "rev-parse", "--verify", "HEAD")
	if err != nil {
		return err
	}
	if strings.TrimSpace(head) != repair.Head {
		return errors.New("HEAD changed: review staging manually")
	}
	entries, err := indexEntries(dir, repair.Paths)
	if err != nil {
		return err
	}
	if entries != repair.Index {
		return errors.New("staging changed: review the selected paths manually")
	}
	_, err = Run(dir, append([]string{"--literal-pathspecs", "reset", "-q", repair.Head, "--"}, repair.Paths...)...)
	return err
}

// CommitPartial commits files through a temp index seeded from HEAD, leaving the real index alone
// except for resetting the committed paths to the new HEAD.
func CommitPartial(dir, message string, amend bool, files []PartialFile) error {
	unmerged, err := Run(dir, "ls-files", "--unmerged", "-z")
	if err != nil {
		return err
	}
	if unmerged != "" {
		return errors.New("resolve index conflicts before committing a changelist")
	}
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
			if pf.ExpectedBase != nil {
				base, err := blob(dir, "HEAD:"+pf.Path)
				if err != nil {
					return err
				}
				if base != *pf.ExpectedBase {
					return errors.New("base file changed: select partial changes again")
				}
			}
			err = StageContent(dir, tmp, pf.Path, *pf.Text)
		default:
			if _, statErr := os.Lstat(filepath.Join(dir, pf.Path)); os.IsNotExist(statErr) {
				_, err = RunEnv(dir, env, "update-index", "--force-remove", "--", pf.Path)
			} else if statErr != nil {
				return statErr
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
	before := ""
	if len(paths) > 0 {
		before, err = indexEntries(dir, paths)
		if err != nil {
			return err
		}
		st, err := GetStatus(dir)
		if err != nil {
			return err
		}
		selected := make(map[string]bool, len(paths))
		for _, p := range paths {
			selected[p] = true
		}
		for _, file := range st.Files {
			if selected[file.Path] && file.Index != '?' && file.Index != '.' && file.Work != '.' {
				return fmt.Errorf("%s has staged and unstaged changes: use staging mode or unstage it before committing", file.Path)
			}
		}
	}
	if _, err := RunEnv(dir, env, args...); err != nil {
		return err
	}
	if len(paths) == 0 {
		return nil
	}
	head, err := Run(dir, "rev-parse", "--verify", "HEAD")
	if err != nil {
		return fmt.Errorf("commit created; could not read new HEAD: %w", err)
	}
	repair := &IndexRepair{Head: strings.TrimSpace(head), Paths: paths, Index: before}
	if err := RepairPartialIndex(dir, *repair); err != nil {
		repair.Message = fmt.Sprintf("Commit %s was created. Its staging area still needs repair: %v", repair.Head, err)
		return repair
	}
	return nil
}

// HeadSides is HEAD vs the working tree file, both in worktree form.
func HeadSides(dir, path string) (Sides, error) {
	return workingSides(dir, path, "HEAD:"+path)
}
