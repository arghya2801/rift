package git

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// withRemote returns a repo whose main branch tracks a bare origin next to it.
func withRemote(t *testing.T) (dir, bare string) {
	t.Helper()
	dir = newRepo(t)
	bare = filepath.Join(filepath.Dir(dir), "origin.git")
	must(t, dir, "init", "-q", "--bare", bare)
	must(t, dir, "remote", "add", "origin", bare)
	must(t, dir, "push", "-q", "-u", "origin", "main")
	return dir, bare
}

// clone makes a second working copy of bare, for simulating someone else's pushes.
func clone(t *testing.T, bare string) string {
	t.Helper()
	other := filepath.Join(t.TempDir(), "other")
	must(t, filepath.Dir(other), "clone", "-q", bare, other)
	for _, a := range [][]string{{"config", "user.name", "o"}, {"config", "user.email", "o@o"}, {"config", "core.autocrlf", "false"}} {
		must(t, other, a...)
	}
	return other
}

func commit(t *testing.T, dir, name, body, msg string) string {
	t.Helper()
	write(t, dir, name, body)
	must(t, dir, "add", "-A")
	must(t, dir, "commit", "-qm", msg)
	return strings.TrimSpace(must(t, dir, "rev-parse", "HEAD"))
}

func read(t *testing.T, dir, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return "<missing>"
	}
	return string(b)
}

func byPath(s Status) map[string]FileStatus {
	m := map[string]FileStatus{}
	for _, f := range s.Files {
		m[f.Path] = f
	}
	return m
}

func TestStatusAheadBehind(t *testing.T) {
	t.Parallel()
	dir, bare := withRemote(t)
	commit(t, dir, "mine.txt", "m\n", "mine")
	other := clone(t, bare)
	commit(t, other, "theirs.txt", "t\n", "theirs")
	must(t, other, "push", "-q")
	must(t, dir, "fetch", "-q")
	s, err := GetStatus(dir)
	if err != nil {
		t.Fatal(err)
	}
	if s.Upstream != "origin/main" || s.Ahead != 1 || s.Behind != 1 {
		t.Fatalf("%+v", s)
	}
}

func TestWorktreeLifecycle(t *testing.T) {
	t.Parallel()
	dir := newRepo(t)
	path, err := AddWorktree(dir, "feat/x", "", true)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(filepath.Dir(dir), "repo-feat-x"); path != want {
		t.Fatalf("path %q want %q", path, want)
	}
	must(t, dir, "branch", "other")
	if _, err := AddWorktree(dir, "other", "", false); err != nil {
		t.Fatal(err)
	}
	list, _ := Worktrees(dir)
	if len(list) != 3 || list[1].Branch != "feat/x" || list[2].Branch != "other" {
		t.Fatalf("%+v", list)
	}

	write(t, path, "dirty.txt", "x")
	if err := RemoveWorktree(dir, path, false); err == nil {
		t.Fatal("removed a dirty worktree without force")
	}
	if err := RemoveWorktree(dir, path, true); err != nil {
		t.Fatal(err)
	}

	os.RemoveAll(list[2].Path)
	if err := PruneWorktrees(dir); err != nil {
		t.Fatal(err)
	}
	if list, _ := Worktrees(dir); len(list) != 1 {
		t.Fatalf("after prune %+v", list)
	}
}

func TestRefs(t *testing.T) {
	t.Parallel()
	dir, _ := withRemote(t)
	must(t, dir, "tag", "v1")
	must(t, dir, "branch", "feat")
	commit(t, dir, "b.txt", "b\n", "ahead")
	refs, err := Refs(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]Ref{}
	for _, r := range refs {
		got[r.Name] = r
	}
	if m := got["main"]; m.Kind != "local" || !m.Current || m.Upstream != "origin/main" || m.Track != "[ahead 1]" {
		t.Fatalf("main %+v", m)
	}
	if f := got["feat"]; f.Kind != "local" || f.Current {
		t.Fatalf("feat %+v", f)
	}
	if r := got["origin/main"]; r.Kind != "remote" {
		t.Fatalf("origin/main %+v", r)
	}
	if v := got["v1"]; v.Kind != "tag" {
		t.Fatalf("v1 %+v", v)
	}
	if _, ok := got["origin/HEAD"]; ok {
		t.Fatal("origin/HEAD should be hidden")
	}
}

func head(t *testing.T, dir string) string {
	return strings.TrimSpace(must(t, dir, "rev-parse", "--abbrev-ref", "HEAD"))
}

func TestBranchOps(t *testing.T) {
	t.Parallel()
	dir, bare := withRemote(t)
	if err := CreateBranch(dir, "b1", "HEAD", true); err != nil || head(t, dir) != "b1" {
		t.Fatalf("create: %v %s", err, head(t, dir))
	}
	if err := Checkout(dir, "main"); err != nil || head(t, dir) != "main" {
		t.Fatalf("checkout: %v", err)
	}
	if err := RenameBranch(dir, "b1", "b2"); err != nil {
		t.Fatal(err)
	}
	if err := DeleteBranch(dir, "b2", false); err != nil {
		t.Fatal(err)
	}

	CreateBranch(dir, "b3", "HEAD", true)
	commit(t, dir, "u.txt", "u\n", "unmerged")
	Checkout(dir, "main")
	if err := DeleteBranch(dir, "b3", false); err == nil {
		t.Fatal("deleted unmerged branch without force")
	}
	if err := DeleteBranch(dir, "b3", true); err != nil {
		t.Fatal(err)
	}

	must(t, dir, "tag", "v1")
	if err := Checkout(dir, "v1"); err != nil || head(t, dir) != "HEAD" {
		t.Fatalf("tag checkout should detach: %v %s", err, head(t, dir))
	}
	Checkout(dir, "main")

	other := clone(t, bare)
	must(t, other, "switch", "-qc", "rb")
	commit(t, other, "rb.txt", "r\n", "rb")
	must(t, other, "push", "-q", "-u", "origin", "rb")
	must(t, dir, "fetch", "-q")
	if err := CheckoutRemote(dir, "origin/rb"); err != nil || head(t, dir) != "rb" {
		t.Fatalf("checkout remote: %v %s", err, head(t, dir))
	}
	if u := strings.TrimSpace(must(t, dir, "rev-parse", "--abbrev-ref", "@{u}")); u != "origin/rb" {
		t.Fatalf("upstream %q", u)
	}
}

// conflicted leaves dir mid-merge with a.txt conflicting (base "one", ours "ours", theirs "theirs").
func conflicted(t *testing.T) string {
	t.Helper()
	dir := newRepo(t)
	must(t, dir, "checkout", "-qb", "other")
	commit(t, dir, "a.txt", "theirs\n", "t")
	must(t, dir, "checkout", "-q", "main")
	commit(t, dir, "a.txt", "ours\n", "o")
	Run(dir, "merge", "other")
	return dir
}

func TestConflictResolveContinue(t *testing.T) {
	t.Parallel()
	dir := conflicted(t)
	if k := OpKind(dir); k != "merge" {
		t.Fatalf("kind %q", k)
	}
	s, _ := GetStatus(dir)
	if op := OpFromStatus(dir, s); !slices.Equal(op.Conflicts, []string{"a.txt"}) {
		t.Fatalf("%+v", op)
	}
	c, err := ConflictSides(dir, "a.txt")
	if err != nil {
		t.Fatal(err)
	}
	if c.Base != "one\n" || c.Ours != "ours\n" || c.Theirs != "theirs\n" {
		t.Fatalf("%+v", c)
	}
	for _, m := range []string{"<<<<<<< ours", "||||||| base", "=======", ">>>>>>> theirs"} {
		if !strings.Contains(c.Merged, m) {
			t.Fatalf("merged lacks %q:\n%s", m, c.Merged)
		}
	}
	if err := ResolveFile(dir, "a.txt", "resolved\n"); err != nil {
		t.Fatal(err)
	}
	if s, _ := GetStatus(dir); len(OpFromStatus(dir, s).Conflicts) != 0 {
		t.Fatal("still conflicted")
	}
	if err := OpContinue(dir); err != nil {
		t.Fatal(err)
	}
	if k := OpKind(dir); k != "" {
		t.Fatalf("op still %q", k)
	}
	if got := must(t, dir, "show", "HEAD:a.txt"); got != "resolved\n" {
		t.Fatalf("committed %q", got)
	}
}

func TestConflictAcceptAndAbort(t *testing.T) {
	t.Parallel()
	for _, side := range []string{"ours", "theirs"} {
		t.Run(side, func(t *testing.T) {
			t.Parallel()
			dir := conflicted(t)
			if err := AcceptSide(dir, "a.txt", side); err != nil {
				t.Fatal(err)
			}
			if got := read(t, dir, "a.txt"); got != side+"\n" {
				t.Fatalf("file %q", got)
			}
		})
	}
	t.Run("abort", func(t *testing.T) {
		t.Parallel()
		dir := conflicted(t)
		if err := OpAbort(dir); err != nil {
			t.Fatal(err)
		}
		if OpKind(dir) != "" || read(t, dir, "a.txt") != "ours\n" {
			t.Fatalf("abort left %q %q", OpKind(dir), read(t, dir, "a.txt"))
		}
	})
}

func TestCherryPickRevert(t *testing.T) {
	t.Parallel()
	dir := newRepo(t)
	must(t, dir, "checkout", "-qb", "feat")
	h := commit(t, dir, "f.txt", "f\n", "feat")
	must(t, dir, "checkout", "-q", "main")
	if err := CherryPick(dir, h); err != nil || read(t, dir, "f.txt") != "f\n" {
		t.Fatalf("cherry-pick: %v", err)
	}
	if err := Revert(dir, "HEAD"); err != nil || read(t, dir, "f.txt") != "<missing>" {
		t.Fatalf("revert: %v", err)
	}
}

func TestResetModes(t *testing.T) {
	t.Parallel()
	for mode, want := range map[string]byte{"soft": 'A', "mixed": '?', "hard": 0} {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()
			dir := newRepo(t)
			commit(t, dir, "b.txt", "b\n", "b")
			if err := Reset(dir, mode, "HEAD~1"); err != nil {
				t.Fatal(err)
			}
			f, ok := byPath(must2(GetStatus(dir)))["b.txt"]
			if want == 0 {
				if ok {
					t.Fatalf("hard reset left %+v", f)
				}
				return
			}
			if f.Index != want {
				t.Fatalf("index %c want %c", f.Index, want)
			}
		})
	}
}

func must2[T any](v T, _ error) T { return v }

func TestStashes(t *testing.T) {
	t.Parallel()
	dir := newRepo(t)
	if s, err := Stashes(dir); err != nil || len(s) != 0 {
		t.Fatalf("empty: %v %v", s, err)
	}
	write(t, dir, "a.txt", "first\n")
	if err := StashPush(dir, "first", false); err != nil {
		t.Fatal(err)
	}
	write(t, dir, "new.txt", "u\n")
	if err := StashPush(dir, "second", true); err != nil {
		t.Fatal(err)
	}
	fast, _ := Stashes(dir)
	slow, _ := stashList(dir)
	if len(fast) != 2 || len(slow) != 2 {
		t.Fatalf("fast %+v slow %+v", fast, slow)
	}
	for i := range fast {
		if fast[i].Ref != slow[i].Ref || fast[i].Subject != slow[i].Subject || fast[i].Time != slow[i].Time {
			t.Fatalf("reflog %+v != stash list %+v", fast[i], slow[i])
		}
	}
	if !strings.Contains(fast[0].Subject, "second") {
		t.Fatalf("newest first: %+v", fast)
	}
	if err := StashAction(dir, "pop", "stash@{0}"); err != nil {
		t.Fatal(err)
	}
	if s, _ := Stashes(dir); len(s) != 1 || read(t, dir, "new.txt") != "u\n" {
		t.Fatalf("after pop %+v", s)
	}
}

func TestFetchPullPush(t *testing.T) {
	t.Parallel()
	dir, bare := withRemote(t)
	other := clone(t, bare)
	commit(t, other, "o.txt", "o\n", "from other")
	must(t, other, "push", "-q")
	if _, err := Fetch(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := Pull(dir, false); err != nil || read(t, dir, "o.txt") != "o\n" {
		t.Fatalf("pull: %v", err)
	}

	must(t, dir, "switch", "-qc", "nb")
	commit(t, dir, "n.txt", "n\n", "new branch")
	if _, err := Push(dir, false); err != nil {
		t.Fatal(err)
	}
	if u := strings.TrimSpace(must(t, dir, "rev-parse", "--abbrev-ref", "@{u}")); u != "origin/nb" {
		t.Fatalf("upstream %q", u)
	}
	must(t, dir, "commit", "-q", "--amend", "-m", "rewritten")
	if _, err := Push(dir, false); err == nil {
		t.Fatal("non-fast-forward push should fail")
	}
	if _, err := Push(dir, true); err != nil {
		t.Fatalf("force-with-lease: %v", err)
	}
}

func TestFileSides(t *testing.T) {
	t.Parallel()
	dir := newRepo(t)
	write(t, dir, "a.txt", "two\n")
	must(t, dir, "add", "a.txt")
	write(t, dir, "a.txt", "three\n")
	write(t, dir, "new.txt", "n\n")
	cases := []struct {
		path     string
		staged   bool
		old, cur string
	}{
		{"a.txt", false, "two\n", "three\n"},
		{"a.txt", true, "one\n", "two\n"},
		{"new.txt", false, "", "n\n"},
	}
	for _, c := range cases {
		s, err := FileSides(dir, c.path, c.staged)
		if err != nil || s.Old != c.old || s.New != c.cur || s.Binary {
			t.Fatalf("%+v: got %+v %v", c, s, err)
		}
	}
	os.Remove(filepath.Join(dir, "a.txt"))
	if s, _ := FileSides(dir, "a.txt", false); s.Old != "two\n" || s.New != "" {
		t.Fatalf("deleted %+v", s)
	}
}

func TestBlobsMissingAndLarge(t *testing.T) {
	t.Parallel()
	dir := newRepo(t)
	big := strings.Repeat("x", MaxBlob+1)
	commit(t, dir, "big.txt", big, "big")
	got := blobs(dir, "HEAD:nope", "HEAD:big.txt", "HEAD:a.txt")
	if got[0] != "" || got[1] != "\x00" || got[2] != "one\n" {
		t.Fatalf("lens %d %d %q", len(got[0]), len(got[1]), got[2])
	}
	if s, _ := FileSides(dir, "big.txt", false); !s.Binary || s.Old != "" || s.New != "" {
		t.Fatal("oversized file should be binary")
	}
	commit(t, dir, "bin.dat", "a\x00b", "bin")
	if s := CommitSides(dir, strings.TrimSpace(must(t, dir, "rev-parse", "HEAD")), "bin.dat", ""); !s.Binary {
		t.Fatal("NUL content should be binary")
	}
}

func TestCommitSidesRoot(t *testing.T) {
	t.Parallel()
	dir := newRepo(t)
	root := strings.TrimSpace(must(t, dir, "rev-parse", "HEAD"))
	if s := CommitSides(dir, root, "a.txt", ""); s.Old != "" || s.New != "one\n" {
		t.Fatalf("%+v", s)
	}
}

func TestCommitPartial(t *testing.T) {
	t.Parallel()
	dir := newRepo(t)
	write(t, dir, "b.txt", "b\n")
	commit(t, dir, "c.txt", "x\ny\nz\n", "base")
	write(t, dir, "b.txt", "b2\n")
	must(t, dir, "add", "b.txt")
	write(t, dir, "c.txt", "X\ny\nZ\n")
	os.Remove(filepath.Join(dir, "a.txt"))

	part := "X\ny\nz\n"
	err := CommitPartial(dir, "partial", false, []PartialFile{{Path: "c.txt", Text: &part}, {Path: "a.txt", Delete: true}})
	if err != nil {
		t.Fatal(err)
	}
	if got := must(t, dir, "show", "HEAD:c.txt"); got != part {
		t.Fatalf("committed c %q", got)
	}
	if _, err := Run(dir, "cat-file", "-e", "HEAD:a.txt"); err == nil {
		t.Fatal("a.txt should be deleted in HEAD")
	}
	if got := must(t, dir, "show", "HEAD:b.txt"); got != "b\n" {
		t.Fatalf("b.txt leaked into commit: %q", got)
	}
	st := byPath(must2(GetStatus(dir)))
	if st["b.txt"].Index != 'M' {
		t.Fatalf("b.txt should stay staged: %+v", st["b.txt"])
	}
	if st["c.txt"].Index != '.' || st["c.txt"].Work != 'M' || read(t, dir, "c.txt") != "X\ny\nZ\n" {
		t.Fatalf("c.txt rest should stay unstaged: %+v", st["c.txt"])
	}
	if _, ok := st["a.txt"]; ok {
		t.Fatalf("a.txt deletion not committed: %+v", st["a.txt"])
	}
}

func TestDiscard(t *testing.T) {
	t.Parallel()
	dir := newRepo(t)
	commit(t, dir, "b.txt", "b\n", "b")
	write(t, dir, "a.txt", "changed\n")
	write(t, dir, "u.txt", "u\n")
	must(t, dir, "mv", "b.txt", "r.txt")
	if err := Discard(dir, []string{"a.txt", "u.txt", "r.txt"}); err != nil {
		t.Fatal(err)
	}
	if s := must2(GetStatus(dir)); len(s.Files) != 0 {
		t.Fatalf("left %+v", s.Files)
	}
	if read(t, dir, "a.txt") != "one\n" || read(t, dir, "b.txt") != "b\n" || read(t, dir, "r.txt") != "<missing>" {
		t.Fatal("files not restored")
	}
}

func TestAddToGitignore(t *testing.T) {
	t.Parallel()
	dir := newRepo(t)
	write(t, dir, "build", "x")
	if err := AddToGitignore(dir, "build"); err != nil {
		t.Fatal(err)
	}
	write(t, dir, ".gitignore", read(t, dir, ".gitignore")+"keep")
	if err := AddToGitignore(dir, "/y.txt"); err != nil {
		t.Fatal(err)
	}
	if got := read(t, dir, ".gitignore"); got != "/build\nkeep\n/y.txt\n" {
		t.Fatalf("%q", got)
	}
	if _, ok := byPath(must2(GetStatus(dir)))["build"]; ok {
		t.Fatal("build still untracked")
	}
}

func TestCommitStagedAmend(t *testing.T) {
	t.Parallel()
	dir := newRepo(t)
	write(t, dir, "m.txt", "m\n")
	Stage(dir, []string{"m.txt"})
	if err := CommitStaged(dir, "m1", false); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(LastMessage(dir)) != "m1" {
		t.Fatalf("%q", LastMessage(dir))
	}
	if err := CommitStaged(dir, "m2", true); err != nil {
		t.Fatal(err)
	}
	if n := strings.TrimSpace(must(t, dir, "rev-list", "--count", "HEAD")); n != "2" || strings.TrimSpace(LastMessage(dir)) != "m2" {
		t.Fatalf("count %s msg %q", n, LastMessage(dir))
	}
}

func TestLogSpecialChars(t *testing.T) {
	t.Parallel()
	dir := newRepo(t)
	subject := "fix: a, b; c | d \"q\" ünïcødé 日本"
	commit(t, dir, "s.txt", "s\n", subject)
	must(t, dir, "branch", "x,y")
	must(t, dir, "tag", "v1")
	rows, err := Log(dir, 10)
	if err != nil {
		t.Fatal(err)
	}
	if rows[0].Subject != subject {
		t.Fatalf("subject %q", rows[0].Subject)
	}
	for _, want := range []string{"HEAD -> main", "x,y", "tag: v1"} {
		if !slices.Contains(rows[0].Refs, want) {
			t.Fatalf("refs %q lack %q", rows[0].Refs, want)
		}
	}
	if rows[0].Author != "t" || rows[0].Email != "t@t" || rows[0].Time == 0 || len(rows[0].Parents) != 1 {
		t.Fatalf("%+v", rows[0].Commit)
	}
}

func TestGraphInvariants(t *testing.T) {
	t.Parallel()
	c := func(h string, p ...string) Commit { return Commit{Hash: h, Parents: p} }
	histories := map[string][]Commit{
		"criss-cross": {c("T", "X", "Y"), c("X", "A", "B"), c("Y", "B", "A"), c("A", "R"), c("B", "R"), c("R")},
		"octopus":     {c("O", "P1", "P2", "P3"), c("P1", "R"), c("P2", "R"), c("P3", "R"), c("R")},
		"two roots":   {c("M", "A", "B"), c("A"), c("B")},
		"parallel":    {c("F1", "F0"), c("G1", "G0"), c("F0"), c("G0")},
	}
	for name, commits := range histories {
		rows := Graph(commits)
		seen := map[string]bool{}
		for _, r := range rows {
			bottoms := 0
			for _, l := range r.Lines {
				if l.From < 0 || l.To < 0 {
					t.Fatalf("%s %s: negative lane %+v", name, r.Hash, l)
				}
				if l.Bottom && l.From == r.Col {
					bottoms++
				}
			}
			if bottoms != len(r.Parents) {
				t.Fatalf("%s %s: %d parent edges, want %d (%+v)", name, r.Hash, bottoms, len(r.Parents), r.Lines)
			}
			if seen[r.Hash] && !slices.ContainsFunc(r.Lines, func(l Line) bool { return !l.Bottom && l.To == r.Col }) {
				t.Fatalf("%s %s: no incoming edge to col %d (%+v)", name, r.Hash, r.Col, r.Lines)
			}
			for _, p := range r.Parents {
				seen[p] = true
			}
		}
	}
}
