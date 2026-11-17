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

	s, err := FileSides(dir, "c.txt", false)
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
	staged, _ := FileSides(dir, "c.txt", true)
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
