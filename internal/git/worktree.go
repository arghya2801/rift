package git

import (
	"path/filepath"
	"strings"
)

type Worktree struct {
	Path     string `json:"path"`
	Head     string `json:"head"`
	Branch   string `json:"branch"`
	Detached bool   `json:"detached"`
	Bare     bool   `json:"bare"`
	Locked   bool   `json:"locked"`
	Main     bool   `json:"main"`
}

func Worktrees(dir string) ([]Worktree, error) {
	out, err := Run(dir, "worktree", "list", "--porcelain", "-z")
	if err != nil {
		return nil, err
	}
	return parseWorktrees(out), nil
}

func parseWorktrees(out string) []Worktree {
	var list []Worktree
	var w *Worktree
	for f := range strings.SplitSeq(out, "\x00") {
		key, val, _ := strings.Cut(f, " ")
		switch key {
		case "worktree":
			list = append(list, Worktree{Path: filepath.Clean(val), Main: len(list) == 0})
			w = &list[len(list)-1]
		case "HEAD":
			w.Head = val
		case "branch":
			w.Branch = strings.TrimPrefix(val, "refs/heads/")
		case "detached":
			w.Detached = true
		case "bare":
			w.Bare = true
		case "locked":
			w.Locked = true
		}
	}
	return list
}

// AddWorktree creates ../<repo>-<branch> next to the main worktree. newBranch creates branch from base.
func AddWorktree(mainDir, branch, base string, newBranch bool) (string, error) {
	path := filepath.Join(filepath.Dir(mainDir), filepath.Base(mainDir)+"-"+strings.ReplaceAll(branch, "/", "-"))
	args := []string{"worktree", "add", path, branch}
	if newBranch {
		args = []string{"worktree", "add", "-b", branch, path}
		if base != "" {
			args = append(args, base)
		}
	}
	_, err := Run(mainDir, args...)
	if err == nil {
		gitDirs.Delete(path)
		commonDirs.Delete(path)
	}
	return path, err
}

func RemoveWorktree(mainDir, path string, force bool) error {
	args := []string{"worktree", "remove", path}
	if force {
		args = append(args, "--force")
	}
	_, err := Run(mainDir, args...)
	if err == nil {
		gitDirs.Delete(path)
		commonDirs.Delete(path)
	}
	return err
}

func PruneWorktrees(mainDir string) error {
	_, err := Run(mainDir, "worktree", "prune")
	return err
}

// OpName names the operation in progress in dir ("" when none), for dashboard badges.
func OpName(dir string) string { return OpKind(dir) }

// Root returns the top-level dir of the worktree containing dir.
func Root(dir string) (string, error) {
	out, err := Run(dir, "rev-parse", "--show-toplevel")
	return filepath.Clean(strings.TrimSpace(out)), err
}
