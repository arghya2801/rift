package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"rift/internal/git"
)

func repoDir(t *testing.T) (string, func(...string) string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "r")
	os.MkdirAll(dir, 0o755)
	run := func(args ...string) string {
		t.Helper()
		out, err := git.Run(dir, args...)
		if err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
		return out
	}
	run("init", "-q", "-b", "main")
	run("config", "user.name", "t")
	run("config", "user.email", "t@t")
	run("config", "core.autocrlf", "false")
	return dir, run
}

func commitFile(t *testing.T, dir string, run func(...string) string, name, body, msg string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "-A")
	run("commit", "-qm", msg)
}

func TestRebaseSquashKeepsCombinedMessage(t *testing.T) {
	t.Parallel()
	dir, run := repoDir(t)
	for _, n := range []string{"base", "a", "b"} {
		commitFile(t, dir, run, n, n, n)
	}
	base := strings.TrimSpace(run("rev-parse", "HEAD~2"))
	items, _ := git.RebaseTodo(dir, base)
	items[1].Action = "squash"
	if err := git.RunRebase(dir, base, items); err != nil {
		t.Fatal(err)
	}
	if n := strings.TrimSpace(run("rev-list", "--count", base+"..HEAD")); n != "1" {
		t.Fatalf("commits %s", n)
	}
	if msg := strings.TrimSpace(run("log", "-1", "--format=%B")); msg != "a\n\nb" {
		t.Fatalf("message %q", msg)
	}
}

// conflictingRebase reorders c2 before c1 so the first pick conflicts, with c1 queued for reword.
func conflictingRebase(t *testing.T) (string, func(...string) string, string) {
	t.Helper()
	dir, run := repoDir(t)
	commitFile(t, dir, run, "f", "0\n", "base")
	commitFile(t, dir, run, "f", "1\n", "c1")
	commitFile(t, dir, run, "f", "2\n", "c2")
	base := strings.TrimSpace(run("rev-parse", "HEAD~2"))
	items, _ := git.RebaseTodo(dir, base)
	c1, c2 := items[0], items[1]
	c1.Action, c1.Message = "reword", "R1"
	if err := git.RunRebase(dir, base, []git.TodoItem{c2, c1}); err != nil {
		t.Fatal(err)
	}
	if !git.RebaseInProgress(dir) || git.OpKind(dir) != "rebase" {
		t.Fatal("rebase should stop on the conflict")
	}
	return dir, run, base
}

func TestRebaseConflictContinueUsesQueue(t *testing.T) {
	t.Parallel()
	dir, run, base := conflictingRebase(t)
	if e := git.ContinueEditor(dir); !strings.Contains(e, "--rift-msg") {
		t.Fatalf("continue editor %q", e)
	}
	if err := git.ResolveFile(dir, "f", "2\n"); err != nil {
		t.Fatal(err)
	}
	if err := git.OpContinue(dir); err != nil {
		t.Fatal(err)
	}
	if git.RebaseInProgress(dir) {
		// c1 now conflicts against c2's content too.
		git.ResolveFile(dir, "f", "1\n")
		if err := git.OpContinue(dir); err != nil {
			t.Fatal(err)
		}
	}
	if got := run("log", "--format=%s", base+"..HEAD"); got != "R1\nc2\n" {
		t.Fatalf("log %q", got)
	}
}

func TestRebaseAbortClearsQueue(t *testing.T) {
	t.Parallel()
	dir, run, _ := conflictingRebase(t)
	if err := git.OpAbort(dir); err != nil {
		t.Fatal(err)
	}
	g := strings.TrimSpace(run("rev-parse", "--path-format=absolute", "--git-dir"))
	if _, err := os.Stat(filepath.Join(g, "rift-rebase")); !os.IsNotExist(err) {
		t.Fatal("stale rift-rebase queue left behind; a later plain rebase would reuse it")
	}
	if e := git.ContinueEditor(dir); e != "GIT_EDITOR=true" {
		t.Fatalf("editor %q", e)
	}
}

func TestSnapshot(t *testing.T) {
	t.Parallel()
	dir, run := repoDir(t)
	commitFile(t, dir, run, "a", "a\n", "a")
	run("tag", "v1")
	os.WriteFile(filepath.Join(dir, "a"), []byte("b\n"), 0o644)
	run("stash", "push", "-q", "-m", "s")
	os.WriteFile(filepath.Join(dir, "u"), []byte("u"), 0o644)
	a := &App{}

	light := a.Snapshot(dir, false)
	if light.Error != "" || len(light.Refs) != 0 || len(light.Stashes) != 0 || light.Full {
		t.Fatalf("light %+v", light)
	}
	if light.Status.Branch != "main" || len(light.Status.Files) != 1 || light.Op.Kind != "" || light.Op.Conflicts == nil {
		t.Fatalf("light status %+v op %+v", light.Status, light.Op)
	}
	full := a.Snapshot(dir, true)
	if len(full.Refs) != 2 || len(full.Stashes) != 1 || !full.Full {
		t.Fatalf("full refs %+v stashes %+v", full.Refs, full.Stashes)
	}
}

func TestInsideGuard(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	escapes := []string{"..", "../x", "a/../../x", "a/../../../etc"}
	if filepath.Separator == '\\' {
		escapes = append(escapes, `..\x`)
	}
	for _, rel := range escapes {
		if _, err := inside(dir, rel); err == nil {
			t.Errorf("%q escaped", rel)
		}
	}
	for _, rel := range []string{".", "a/b", "a/../b", "/etc/passwd", "..x"} {
		p, err := inside(dir, rel)
		if err != nil || !strings.HasPrefix(p, dir) {
			t.Errorf("%q: %q %v", rel, p, err)
		}
	}
}

func TestReadWriteFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	a := &App{}
	if err := a.WriteFile(dir, "f.txt", "héllo\r\n"); err != nil {
		t.Fatal(err)
	}
	if got, err := a.ReadFile(dir, "f.txt"); err != nil || got != "héllo\r\n" {
		t.Fatalf("%q %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "f.txt.rift-tmp")); !os.IsNotExist(err) {
		t.Fatal("temp file left behind")
	}
	os.WriteFile(filepath.Join(dir, "bin"), []byte("a\x00b"), 0o644)
	if _, err := a.ReadFile(dir, "bin"); err == nil {
		t.Fatal("binary read")
	}
	os.WriteFile(filepath.Join(dir, "big"), make([]byte, 5<<20+1), 0o644)
	if _, err := a.ReadFile(dir, "big"); err == nil {
		t.Fatal("large read")
	}
	if err := a.WriteFile(dir, "../escape", "x"); err == nil {
		t.Fatal("write escaped")
	}
	ents, _ := a.ListDir(dir, ".")
	os.Mkdir(filepath.Join(dir, ".git"), 0o755)
	os.Mkdir(filepath.Join(dir, "Zdir"), 0o755)
	ents, _ = a.ListDir(dir, ".")
	if len(ents) != 4 || ents[0].Name != "Zdir" || !ents[0].IsDir {
		t.Fatalf("dirs first, .git hidden: %+v", ents)
	}
}

func TestInitialResolvesWorktreeRoot(t *testing.T) {
	dir, run := repoDir(t)
	commitFile(t, dir, run, "a", "a", "a")
	sub := filepath.Join(dir, "sub")
	os.Mkdir(sub, 0o755)
	want, _ := git.Root(dir)

	saved := os.Args
	defer func() { os.Args = saved }()
	a := NewApp(t.TempDir())
	os.Args = []string{"rift", sub}
	if got := a.Initial(); !strings.EqualFold(got, want) {
		t.Fatalf("got %q want %q", got, want)
	}
	if repos := a.Repos(); len(repos) != 1 || !strings.EqualFold(repos[0].Path, want) {
		t.Fatalf("not pinned: %+v", repos)
	}
	for _, args := range [][]string{{"rift"}, {"rift", "--flag"}, {"rift", t.TempDir()}} {
		os.Args = args
		if got := a.Initial(); got != "" {
			t.Fatalf("%v: %q", args, got)
		}
	}
}

func TestDashboardSendsCountsWithoutPaths(t *testing.T) {
	dir, run := repoDir(t)
	commitFile(t, dir, run, "tracked", "original", "initial")
	for i := range 1000 {
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("changed-%04d", i)), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	run("add", "changed-0000")
	cards := (&App{}).Overview([]Repo{{Name: "dashboard-test", Worktrees: []git.Worktree{{Path: dir}}}})
	if len(cards) != 1 || cards[0].Changed != 1000 || cards[0].Staged != 1 || cards[0].Error != "" {
		t.Fatalf("unexpected summary: %+v", cards)
	}
	s, err := git.GetStatus(dir)
	if err != nil {
		t.Fatal(err)
	}
	newBytes, _ := json.Marshal(cards[0])
	oldBytes, _ := json.Marshal(s)
	t.Logf("1000 changed paths: old status alone %d bytes, new complete card %d bytes", len(oldBytes), len(newBytes))
	if strings.Contains(string(newBytes), "changed-0000") {
		t.Fatal("dashboard sent path data")
	}
}
