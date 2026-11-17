package git

import (
	"os"
	"os/exec"
	"path/filepath"
)

type Op struct {
	Kind      string   `json:"kind"` // merge, rebase, cherry-pick, revert, or ""
	Conflicts []string `json:"conflicts"`
}

type Conflict struct {
	Base   string `json:"base"`
	Ours   string `json:"ours"`
	Theirs string `json:"theirs"`
	Merged string `json:"merged"`
}

// OpKind reads the in-progress operation from marker files in the git dir; no spawn once the dir is cached.
func OpKind(dir string) string {
	g, err := gitDir(dir)
	if err != nil {
		return ""
	}
	for _, c := range [][2]string{{"rebase-merge", "rebase"}, {"rebase-apply", "rebase"}, {"MERGE_HEAD", "merge"}, {"CHERRY_PICK_HEAD", "cherry-pick"}, {"REVERT_HEAD", "revert"}} {
		if _, err := os.Stat(filepath.Join(g, c[0])); err == nil {
			return c[1]
		}
	}
	return ""
}

// OpFromStatus builds the op state from an already-fetched status (conflicts are its 'u' records).
func OpFromStatus(dir string, s Status) Op {
	op := Op{Kind: OpKind(dir), Conflicts: []string{}}
	for _, f := range s.Files {
		if f.Conflict {
			op.Conflicts = append(op.Conflicts, f.Path)
		}
	}
	return op
}

func OpState(dir string) (Op, error) {
	s, err := GetStatus(dir)
	return OpFromStatus(dir, s), err
}

func ConflictSides(dir, path string) (Conflict, error) {
	b := blobs(dir, ":1:"+path, ":2:"+path, ":3:"+path)
	c := Conflict{Base: b[0], Ours: b[1], Theirs: b[2]}
	tmp, err := os.MkdirTemp("", "rift-merge")
	if err != nil {
		return c, err
	}
	defer os.RemoveAll(tmp)
	files := []string{filepath.Join(tmp, "ours"), filepath.Join(tmp, "base"), filepath.Join(tmp, "theirs")}
	for i, body := range []string{c.Ours, c.Base, c.Theirs} {
		if err := os.WriteFile(files[i], []byte(body), 0o600); err != nil {
			return c, err
		}
	}
	cmd := exec.Command(gitExe(), "merge-file", "-p", "--diff3", "-L", "ours", "-L", "base", "-L", "theirs", files[0], files[1], files[2])
	hideWindow(cmd)
	out, err := cmd.Output()
	if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() > 0 {
		err = nil // positive exit = number of conflicts
	}
	c.Merged = string(out)
	return c, err
}

func ResolveFile(dir, path, text string) error {
	if err := os.WriteFile(filepath.Join(dir, path), []byte(text), 0o644); err != nil {
		return err
	}
	_, err := Run(dir, "add", "--", path)
	return err
}

// AcceptSide resolves path wholesale with "ours" or "theirs".
func AcceptSide(dir, path, side string) error {
	if _, err := Run(dir, "checkout", "--"+side, "--", path); err != nil {
		return err
	}
	_, err := Run(dir, "add", "--", path)
	return err
}

func opRun(dir, action string) error {
	kind := OpKind(dir)
	if kind == "" {
		return nil
	}
	before, _ := Run(dir, "rev-parse", "HEAD")
	_, err := RunEnv(dir, []string{ContinueEditor(dir)}, kind, "--"+action)
	if OpKind(dir) == "" {
		if g, gerr := gitDir(dir); gerr == nil {
			os.RemoveAll(filepath.Join(g, "rift-rebase"))
		}
	} else if after, _ := Run(dir, "rev-parse", "HEAD"); err != nil && action != "abort" && after != before {
		err = nil // progressed, then stopped at the next conflict: not a failure
	}
	return err
}

func OpContinue(dir string) error { return opRun(dir, "continue") }
func OpAbort(dir string) error    { return opRun(dir, "abort") }
func OpSkip(dir string) error     { return opRun(dir, "skip") }
