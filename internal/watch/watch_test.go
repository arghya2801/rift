package watch

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"rift/internal/git"
)

type ev struct {
	path string
	git  bool
}

func repo(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "r")
	for _, d := range []string{"src", "out", ".github"} {
		os.MkdirAll(filepath.Join(dir, d), 0o755)
	}
	git.Run(dir, "init", "-q", "-b", "main")
	git.Run(dir, "config", "user.name", "t")
	git.Run(dir, "config", "user.email", "t@t")
	// Detached auto-maintenance after a commit writes into .git later and would look like a git event.
	git.Run(dir, "config", "maintenance.auto", "false")
	git.Run(dir, "config", "gc.auto", "0")
	os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("out/\n"), 0o644)
	git.Run(dir, "add", ".")
	git.Run(dir, "commit", "-qm", "init")
	return dir
}

func start(t *testing.T, dir string) (*Watcher, chan ev) {
	t.Helper()
	ch := make(chan ev, 64)
	w, err := New(func(p string, g bool) { ch <- ev{p, g} })
	if err != nil {
		t.Fatal(err)
	}
	w.Set(dir)
	quiet(ch)
	return w, ch
}

// quiet drains events until 600 ms pass without one.
func quiet(ch chan ev) {
	for {
		select {
		case <-ch:
		case <-time.After(600 * time.Millisecond):
			return
		}
	}
}

// next waits for one debounced event; ok=false on timeout.
func next(ch chan ev) (ev, bool) {
	select {
	case e := <-ch:
		return e, true
	case <-time.After(3 * time.Second):
		return ev{}, false
	}
}

// noWorktreeEvent fails on a worktree (git=false) event within 800 ms. Git-dir events are
// tolerated: git's background maintenance after a commit can touch .git at any time.
func noWorktreeEvent(t *testing.T, ch chan ev, why string) {
	t.Helper()
	deadline := time.After(800 * time.Millisecond)
	for {
		select {
		case e := <-ch:
			if !e.git {
				t.Fatalf("%s: %+v", why, e)
			}
		case <-deadline:
			return
		}
	}
}

func touch(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(time.Now().String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestEventKinds(t *testing.T) {
	t.Parallel()
	dir := repo(t)
	_, ch := start(t, dir)
	cases := []struct {
		name string
		act  func()
		git  bool
	}{
		{"worktree file", func() { touch(t, filepath.Join(dir, "src", "a.go")) }, false},
		{".github is not .git", func() { touch(t, filepath.Join(dir, ".github", "ci.yml")) }, false},
		{"new branch ref", func() { git.Run(dir, "branch", "feat") }, true},
		{"HEAD move", func() { git.Run(dir, "switch", "-q", "feat") }, true},
	}
	for _, c := range cases {
		c.act()
		e, ok := next(ch)
		if !ok {
			t.Fatalf("%s: no event", c.name)
		}
		if e.path != dir || e.git != c.git {
			t.Fatalf("%s: got %+v want git=%v", c.name, e, c.git)
		}
		quiet(ch)
	}
}

func TestIgnoredDirSilent(t *testing.T) {
	t.Parallel()
	dir := repo(t)
	_, ch := start(t, dir)
	touch(t, filepath.Join(dir, "out", "x"))
	noWorktreeEvent(t, ch, "event for ignored dir")
}

func TestNewSubdirWatched(t *testing.T) {
	t.Parallel()
	dir := repo(t)
	_, ch := start(t, dir)
	sub := filepath.Join(dir, "src", "deep")
	os.Mkdir(sub, 0o755)
	quiet(ch)
	touch(t, filepath.Join(sub, "f"))
	if e, ok := next(ch); !ok || e.git {
		t.Fatalf("no worktree event from new subdir: %+v %v", e, ok)
	}
}

func TestSwitchLeavesOnlyNewWatched(t *testing.T) {
	t.Parallel()
	a, b := repo(t), repo(t)
	w, ch := start(t, a)
	w.Set(b)
	quiet(ch)
	touch(t, filepath.Join(a, "src", "old"))
	noWorktreeEvent(t, ch, "old worktree still watched")
	touch(t, filepath.Join(b, "src", "new"))
	e, ok := next(ch)
	for ok && e.git {
		e, ok = next(ch)
	}
	if !ok || e.path != b {
		t.Fatalf("new worktree not watched: %+v %v", e, ok)
	}
	w.Set("")
	quiet(ch)
	touch(t, filepath.Join(b, "src", "after"))
	select {
	case e := <-ch:
		t.Fatalf("event after Set(\"\"): %+v", e)
	case <-time.After(800 * time.Millisecond):
	}
}

func TestProjectLockfileRefreshesStatus(t *testing.T) {
	dir := repo(t)
	_, ch := start(t, dir)
	touch(t, filepath.Join(dir, "yarn.lock"))
	if e, ok := next(ch); !ok || e.git || e.path != dir {
		t.Fatalf("lockfile refresh: %+v %v", e, ok)
	}
}
