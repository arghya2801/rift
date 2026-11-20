package git

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"rift/internal/workfile"
	"strconv"
	"strings"
)

type Op struct {
	Kind      string   `json:"kind"` // merge, rebase, cherry-pick, revert, or ""
	Conflicts []string `json:"conflicts"`
}

type Conflict struct {
	MarkerSize int    `json:"markerSize"`
	Stages     string `json:"stages"`
	WholeOnly  string `json:"wholeOnly,omitempty"`
	Base       string `json:"base"`
	Ours       string `json:"ours"`
	Theirs     string `json:"theirs"`
	Merged     string `json:"merged"`
}

// OpKind reads the in-progress operation from marker files in the git dir; no spawn once the dir is cached.
func OpKind(dir string) string {
	g, err := gitDir(dir)
	if err != nil {
		return ""
	}
	for _, c := range [][2]string{{"rebase-merge", "rebase"}, {"rebase-apply", "rebase"}, {"MERGE_HEAD", "merge"}, {"CHERRY_PICK_HEAD", "cherry-pick"}, {"REVERT_HEAD", "revert"}} {
		if _, err := os.Stat(filepath.Join(g, c[0])); err == nil {
			if c[0] == "rebase-apply" {
				if _, err := os.Stat(filepath.Join(g, c[0], "applying")); err == nil {
					return "am"
				}
			}
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

func conflictStages(dir, path string) (string, error) {
	if !filepath.IsLocal(path) {
		return "", errors.New("invalid conflict path")
	}
	stages, err := Run(dir, "--literal-pathspecs", "ls-files", "--unmerged", "-z", "--", path)
	if err != nil {
		return "", err
	}
	if stages == "" {
		return "", errors.New("file is no longer conflicted")
	}
	for entry := range strings.SplitSeq(strings.TrimSuffix(stages, "\x00"), "\x00") {
		_, name, ok := strings.Cut(entry, "\t")
		if !ok || name != path {
			return "", errors.New("expected one conflicted file")
		}
	}
	return stages, nil
}

func ConflictSides(dir, path string) (Conflict, error) {
	stages, err := conflictStages(dir, path)
	if err != nil {
		return Conflict{}, err
	}
	c := Conflict{Stages: stages}
	for entry := range strings.SplitSeq(strings.TrimSuffix(stages, "\x00"), "\x00") {
		if !strings.HasPrefix(entry, "100") {
			c.WholeOnly = "Choose an entire side for this file type."
			return c, nil
		}
	}
	b, err := blobs(dir, ":1:"+path, ":2:"+path, ":3:"+path)
	if err != nil {
		return Conflict{}, err
	}
	for _, side := range b {
		if !isText(side) {
			c.WholeOnly = "Binary, oversized or non-UTF-8 conflict. Choose an entire side."
			return c, nil
		}
	}
	c.Base, c.Ours, c.Theirs = b[0], b[1], b[2]
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
	c.MarkerSize = markerSize(b)
	cmd := exec.Command(gitExe(), "merge-file", "-p", "--diff3", "--marker-size="+strconv.Itoa(c.MarkerSize), "-L", "ours", "-L", "base", "-L", "theirs", files[0], files[1], files[2])
	hideWindow(cmd)
	out, err := cmd.Output()
	if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() > 0 && ee.ExitCode() <= 127 {
		err = nil // positive exit = number of conflicts
	}
	c.Merged = string(out)
	return c, err
}

func markerSize(sides []string) int {
	used := make(map[int]bool)
	for _, body := range sides {
		for line := range strings.Lines(body) {
			if len(line) < 7 || !strings.ContainsRune("<|=>", rune(line[0])) {
				continue
			}
			n := 1
			for n < len(line) && line[n] == line[0] {
				n++
			}
			used[n] = true
		}
	}
	n := 7
	for used[n] {
		n++
	}
	return n
}

func verifyStages(dir, path, expected string) error {
	stages, err := conflictStages(dir, path)
	if err != nil {
		return err
	}
	if stages != expected {
		return errors.New("conflict changed: reopen the resolver")
	}
	return nil
}

func ResolveFile(dir, path, text, stages string) error {
	if err := verifyStages(dir, path, stages); err != nil {
		return err
	}
	if err := workfile.Write(dir, path, text, nil); err != nil {
		return err
	}
	_, err := Run(dir, "--literal-pathspecs", "add", "--", path)
	return err
}

// AcceptSide resolves path wholesale with "ours" or "theirs".
func AcceptSide(dir, path, side, stages string) error {
	stage := "2"
	switch side {
	case "ours":
	case "theirs":
		stage = "3"
	default:
		return errors.New("invalid conflict side")
	}
	if err := verifyStages(dir, path, stages); err != nil {
		return err
	}
	present := false
	for entry := range strings.SplitSeq(strings.TrimSuffix(stages, "\x00"), "\x00") {
		meta, _, _ := strings.Cut(entry, "\t")
		if strings.HasSuffix(meta, " "+stage) {
			present = true
			break
		}
	}
	if !present {
		_, err := Run(dir, "--literal-pathspecs", "rm", "--", path)
		return err
	}
	if _, err := Run(dir, "--literal-pathspecs", "checkout", "--"+side, "--", path); err != nil {
		return err
	}
	_, err := Run(dir, "--literal-pathspecs", "add", "--", path)
	return err
}

func opRun(dir, action string) error {
	kind := OpKind(dir)
	if kind == "" {
		return nil
	}
	before, _ := Run(dir, "rev-parse", "HEAD")
	_, err := RunEnv(dir, []string{ContinueEditor(dir)}, kind, "--"+action)
	if after, _ := Run(dir, "rev-parse", "HEAD"); err != nil && action != "abort" && OpKind(dir) != "" && after != before {
		err = nil // progressed, then stopped at the next conflict: not a failure
	}
	return err
}

func OpContinue(dir string) error { return opRun(dir, "continue") }
func OpAbort(dir string) error    { return opRun(dir, "abort") }
func OpSkip(dir string) error     { return opRun(dir, "skip") }
