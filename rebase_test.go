package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"rift/internal/git"
)

func TestMain(m *testing.M) {
	if runHelper() {
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestRebaseReorderSquashReword(t *testing.T) {
	dir := t.TempDir()
	run := func(args ...string) string {
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
	for _, n := range []string{"base", "a", "b", "c"} {
		os.WriteFile(filepath.Join(dir, n), []byte(n), 0o644)
		run("add", ".")
		run("commit", "-qm", n)
	}
	base := strings.TrimSpace(run("rev-parse", "HEAD~3"))
	items, err := git.RebaseTodo(dir, base)
	if err != nil || len(items) != 3 {
		t.Fatalf("%v %v", items, err)
	}
	// c first (reworded), then a with b squashed into it.
	c, a, b := items[2], items[0], items[1]
	c.Action, c.Message = "reword", "C renamed"
	b.Action = "fixup"
	if err := git.RunRebase(dir, base, []git.TodoItem{c, a, b}); err != nil {
		t.Fatal(err)
	}
	if got := run("log", "--format=%s", base+"..HEAD"); got != "a\nC renamed\n" {
		t.Fatalf("log %q", got)
	}
}
