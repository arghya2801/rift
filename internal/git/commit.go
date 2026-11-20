package git

import (
	"errors"
	"fmt"
	"os"
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
	var entries strings.Builder
	for len(paths) > 0 {
		n, bytes := 0, 0
		for n < len(paths) && bytes < 8000 {
			bytes += len(paths[n]) + 3
			n++
		}
		out, err := Run(dir, append([]string{"--literal-pathspecs", "ls-files", "--stage", "-z", "--"}, paths[:n]...)...)
		if err != nil {
			return "", err
		}
		entries.WriteString(out)
		paths = paths[n:]
	}
	return entries.String(), nil
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
	_, err = runInput(dir, nil, strings.Join(repair.Paths, "\x00")+"\x00", "--literal-pathspecs", "reset", "-q", "--pathspec-from-file=-", "--pathspec-file-nul", repair.Head)
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

	var paths []string
	var input strings.Builder
	mode := ""
	flush := func() error {
		if input.Len() == 0 {
			return nil
		}
		args := []string{"update-index", "--add", "--remove", "-z", "--stdin"}
		if mode == "delete" {
			args = []string{"update-index", "--force-remove", "-z", "--stdin"}
		}
		if mode == "partial" {
			args = []string{"update-index", "-z", "--index-info"}
		}
		_, err := runInput(dir, env, input.String(), args...)
		input.Reset()
		return err
	}
	var modes map[string]string
	for i, pf := range files {
		if pf.Path == "" || strings.ContainsRune(pf.Path, 0) {
			return errors.New("invalid index path")
		}
		next := "whole"
		if pf.Delete {
			next = "delete"
		} else if pf.Text != nil {
			next = "partial"
		}
		if next != mode {
			if err := flush(); err != nil {
				return err
			}
			mode = next
			if mode == "partial" {
				modes, err = partialModes(dir, env, files[i:])
				if err != nil {
					return err
				}
			}
		}
		if next == "partial" {
			if pf.ExpectedBase != nil {
				base, err := blob(dir, "HEAD:"+pf.Path)
				if err != nil {
					return err
				}
				if base != *pf.ExpectedBase {
					return errors.New("base file changed: select partial changes again")
				}
			}
			sha, err := hashObject(dir, pf.Path, *pf.Text)
			if err != nil {
				return err
			}
			filemode := modes[pf.Path]
			if filemode == "" {
				filemode = fileMode(dir, env, pf.Path)
			}
			input.WriteString(filemode + " " + sha + "\t")
		}
		paths = append(paths, pf.Path)
		input.WriteString(pf.Path)
		input.WriteByte(0)
	}
	if err := flush(); err != nil {
		return err
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

// Read modes only for this run of partial files. Bound argv for Git on Windows.
func partialModes(dir string, env []string, files []PartialFile) (map[string]string, error) {
	modes := make(map[string]string)
	for len(files) > 0 && files[0].Text != nil && !files[0].Delete {
		args := []string{"--literal-pathspecs", "ls-files", "--stage", "-z", "--"}
		size := 0
		for len(files) > 0 && files[0].Text != nil && !files[0].Delete && size < 8000 {
			args = append(args, files[0].Path)
			size += len(files[0].Path) + 3
			files = files[1:]
		}
		out, err := RunEnv(dir, env, args...)
		if err != nil {
			return nil, err
		}
		for record := range strings.SplitSeq(out, "\x00") {
			header, path, ok := strings.Cut(record, "\t")
			if ok && len(header) >= 6 {
				modes[path] = header[:6]
			}
		}
	}
	return modes, nil
}

// HeadSides is HEAD vs the working tree file, both in worktree form.
func HeadSides(dir, path string) (Sides, error) {
	return workingSides(dir, path, "HEAD:"+path)
}
