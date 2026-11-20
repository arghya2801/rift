package git

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func Fetch(dir string) (string, error) {
	return Run(dir, "fetch", "--all", "--prune")
}

// Pull with rebase=true runs pull --rebase, else a merge pull.
func Pull(dir string, rebase bool) (string, error) {
	flag := "--no-rebase"
	if rebase {
		flag = "--rebase"
	}
	return Run(dir, "pull", "--autostash", flag)
}

// Push sets upstream to origin when the branch has none.
func Push(dir string, force bool) (string, error) {
	args := []string{"push"}
	if force {
		args = append(args, "--force-with-lease")
	}
	if _, err := Run(dir, "rev-parse", "--abbrev-ref", "@{u}"); err != nil {
		branch, err := Run(dir, "symbolic-ref", "--short", "HEAD")
		if err != nil {
			return "", err
		}
		args = append(args, "-u", "origin", strings.TrimSpace(branch))
	}
	return Run(dir, args...)
}

type Stash struct {
	Ref     string `json:"ref"`
	Subject string `json:"subject"`
	Time    int64  `json:"time"`
}

// Stashes reads the stash reflog file directly (newest last on disk), so it costs no git process.
func Stashes(dir string) ([]Stash, error) {
	g, err := commonDir(dir)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(filepath.Join(g, "logs", "refs", "stash"))
	if os.IsNotExist(err) {
		if _, statErr := os.Stat(filepath.Join(g, "reftable")); os.IsNotExist(statErr) {
			return []Stash{}, nil
		}
		return stashList(dir)
	}
	if err != nil {
		return stashList(dir)
	}
	lines := strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	list := make([]Stash, 0, len(lines))
	for i := len(lines) - 1; i >= 0; i-- {
		meta, msg, ok := strings.Cut(lines[i], "\t")
		f := strings.Fields(meta)
		if !ok || len(f) < 2 {
			return stashList(dir)
		}
		t, _ := strconv.ParseInt(f[len(f)-2], 10, 64)
		list = append(list, Stash{Ref: "stash@{" + strconv.Itoa(len(list)) + "}", Subject: msg, Time: t})
	}
	return list, nil
}

func stashList(dir string) ([]Stash, error) {
	out, err := Run(dir, "stash", "list", "-z", "--format=%gd%x1f%gs%x1f%at")
	if err != nil {
		return nil, err
	}
	list := []Stash{}
	for rec := range strings.SplitSeq(out, "\x00") {
		f := strings.Split(rec, "\x1f")
		if len(f) < 3 {
			continue
		}
		var t int64
		for _, c := range f[2] {
			t = t*10 + int64(c-'0')
		}
		list = append(list, Stash{Ref: f[0], Subject: f[1], Time: t})
	}
	return list, nil
}

func StashPush(dir, message string, untracked bool) error {
	args := []string{"stash", "push"}
	if untracked {
		args = append(args, "-u")
	}
	if message != "" {
		args = append(args, "-m", message)
	}
	_, err := Run(dir, args...)
	return err
}

// StashAction runs apply, pop or drop on ref.
func StashAction(dir, action, ref string) error {
	if action != "apply" && action != "pop" && action != "drop" {
		return errors.New("invalid stash action")
	}
	_, err := Run(dir, "stash", action, ref)
	return err
}
