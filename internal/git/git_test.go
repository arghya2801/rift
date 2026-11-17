package git

import (
	"os"
	"path/filepath"
	"testing"
)

func newRepo(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "repo")
	os.MkdirAll(dir, 0o755)
	for _, a := range [][]string{
		{"init", "-q", "-b", "main"},
		{"config", "user.name", "t"},
		{"config", "user.email", "t@t"},
		{"config", "commit.gpgsign", "false"},
		{"config", "core.autocrlf", "false"},
	} {
		must(t, dir, a...)
	}
	write(t, dir, "a.txt", "one\n")
	must(t, dir, "add", ".")
	must(t, dir, "commit", "-qm", "init")
	return dir
}

func must(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := Run(dir, args...)
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return out
}

func write(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestWorktrees(t *testing.T) {
	dir := newRepo(t)
	wt := filepath.Join(filepath.Dir(dir), "repo-feat")
	must(t, dir, "worktree", "add", "-q", "-b", "feat", wt)
	list, err := Worktrees(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || !list[0].Main || list[0].Branch != "main" || list[1].Branch != "feat" || list[1].Main {
		t.Fatalf("got %+v", list)
	}
}

func TestStatus(t *testing.T) {
	dir := newRepo(t)
	write(t, dir, "a.txt", "two\n")
	write(t, dir, "new file.txt", "x\n")
	write(t, dir, "b.txt", "b\n")
	must(t, dir, "add", "b.txt")
	must(t, dir, "mv", "a.txt", "c.txt")
	s, err := GetStatus(dir)
	if err != nil {
		t.Fatal(err)
	}
	if s.Branch != "main" {
		t.Fatalf("branch %q", s.Branch)
	}
	got := map[string]FileStatus{}
	for _, f := range s.Files {
		got[f.Path] = f
	}
	if f := got["c.txt"]; f.Index != 'R' || f.OrigPath != "a.txt" || f.Work != 'M' {
		t.Fatalf("rename %+v", f)
	}
	if f := got["b.txt"]; f.Index != 'A' {
		t.Fatalf("added %+v", f)
	}
	if f := got["new file.txt"]; f.Index != '?' {
		t.Fatalf("untracked %+v", f)
	}
}

func TestConflictStatus(t *testing.T) {
	dir := newRepo(t)
	must(t, dir, "checkout", "-qb", "other")
	write(t, dir, "a.txt", "theirs\n")
	must(t, dir, "commit", "-qam", "t")
	must(t, dir, "checkout", "-q", "main")
	write(t, dir, "a.txt", "ours\n")
	must(t, dir, "commit", "-qam", "o")
	Run(dir, "merge", "other")
	s, _ := GetStatus(dir)
	if len(s.Files) != 1 || !s.Files[0].Conflict {
		t.Fatalf("got %+v", s.Files)
	}
}
