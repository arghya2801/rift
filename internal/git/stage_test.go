package git

import (
	"strings"
	"testing"
)

func TestSidesAndPartialStageCRLF(t *testing.T) {
	dir := newRepo(t)
	must(t, dir, "config", "core.autocrlf", "true")
	write(t, dir, "c.txt", "a\r\nb\r\nc\r\n")
	must(t, dir, "add", "c.txt")
	must(t, dir, "commit", "-qm", "c")
	write(t, dir, "c.txt", "A\r\nb\r\nC\r\n")

	s, err := FileSides(dir, "c.txt", false, "")
	if err != nil {
		t.Fatal(err)
	}
	if s.Old != "a\r\nb\r\nc\r\n" || s.New != "A\r\nb\r\nC\r\n" {
		t.Fatalf("sides %q %q", s.Old, s.New)
	}

	// Stage only the first hunk.
	if err := StageContent(dir, "", "c.txt", "A\r\nb\r\nc\r\n"); err != nil {
		t.Fatal(err)
	}
	if got := must(t, dir, "show", ":c.txt"); got != "A\nb\nc\n" {
		t.Fatalf("index %q", got)
	}
	staged, _ := FileSides(dir, "c.txt", true, "")
	if staged.New != "A\r\nb\r\nc\r\n" {
		t.Fatalf("staged new %q", staged.New)
	}
	if diff := must(t, dir, "diff", "--", "c.txt"); !strings.Contains(diff, "+C") || strings.Contains(diff, "+A") {
		t.Fatalf("unstaged diff %q", diff)
	}
}

func TestStageUnstage(t *testing.T) {
	dir := newRepo(t)
	write(t, dir, "n.txt", "n\n")
	if err := Stage(dir, []string{"n.txt"}); err != nil {
		t.Fatal(err)
	}
	if s, _ := GetStatus(dir); s.Files[0].Index != 'A' {
		t.Fatalf("%+v", s.Files)
	}
	if err := Unstage(dir, []string{"n.txt"}); err != nil {
		t.Fatal(err)
	}
	if s, _ := GetStatus(dir); s.Files[0].Index != '?' {
		t.Fatalf("%+v", s.Files)
	}
}

func TestFileActionsUseLiteralPaths(t *testing.T) {
	dir := newRepo(t)
	for _, path := range []string{"[x].txt", "x.txt"} {
		write(t, dir, path, "base\n")
	}
	must(t, dir, "add", ".")
	must(t, dir, "commit", "-qm", "base")
	for _, path := range []string{"[x].txt", "x.txt"} {
		write(t, dir, path, "edit\n")
	}
	selection := []string{"[x].txt"}
	if err := Stage(dir, selection); err != nil {
		t.Fatal(err)
	}
	if got := must(t, dir, "diff", "--cached", "--name-only"); got != "[x].txt\n" {
		t.Fatal(got)
	}
	must(t, dir, "add", "x.txt")
	if err := Unstage(dir, selection); err != nil {
		t.Fatal(err)
	}
	if err := Discard(dir, selection); err != nil {
		t.Fatal(err)
	}
	if got := must(t, dir, "diff", "--cached", "--name-only"); got != "x.txt\n" {
		t.Fatal(got)
	}
	if got := read(t, dir, "x.txt"); got != "edit\n" {
		t.Fatal(got)
	}
}

func TestRenameSidesAndUnstage(t *testing.T) {
	dir := newRepo(t)
	must(t, dir, "mv", "a.txt", "renamed.txt")
	s, err := FileSides(dir, "renamed.txt", true, "a.txt")
	if err != nil || s.Old != "one\n" || s.New != s.Old {
		t.Fatalf("%+v %v", s, err)
	}
	if err := Unstage(dir, []string{"a.txt", "renamed.txt"}); err != nil {
		t.Fatal(err)
	}
	if got := must(t, dir, "diff", "--cached", "--name-only"); got != "" {
		t.Fatal(got)
	}
}

func TestDiscardBeforeFirstCommit(t *testing.T) {
	dir := t.TempDir()
	must(t, dir, "init", "-q")
	write(t, dir, "staged", "a")
	must(t, dir, "add", "staged")
	write(t, dir, "new", "b")
	write(t, dir, "keep", "c")
	if err := Discard(dir, []string{"staged", "new"}); err != nil {
		t.Fatal(err)
	}
	s, err := GetStatus(dir)
	if err != nil || len(s.Files) != 1 || s.Files[0].Path != "keep" {
		t.Fatalf("%v %v", s, err)
	}
}

func TestIgnoreRuleIsLiteral(t *testing.T) {
	dir := newRepo(t)
	if err := AddToGitignore(dir, "[x].txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(dir, "check-ignore", "--", "[x].txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(dir, "check-ignore", "--", "x.txt"); err == nil {
		t.Fatal("ignored neighbor")
	}
	if err := AddToGitignore(dir, "a\nb"); err == nil {
		t.Fatal("accepted multiline rule")
	}
}
