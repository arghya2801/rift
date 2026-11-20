package git

import (
	"reflect"
	"slices"
	"testing"
)

func TestGraphMerge(t *testing.T) {
	// M merges C into B; both come from A.
	rows := Graph([]Commit{
		{Hash: "M", Parents: []string{"B", "C"}},
		{Hash: "C", Parents: []string{"A"}},
		{Hash: "B", Parents: []string{"A"}},
		{Hash: "A"},
	})
	cols := []int{rows[0].Col, rows[1].Col, rows[2].Col, rows[3].Col}
	if !slices.Equal(cols, []int{0, 1, 0, 0}) {
		t.Fatalf("cols %v", cols)
	}
	if !slices.Contains(rows[0].Lines, Line{From: 0, To: 1, Bottom: true}) {
		t.Fatalf("merge edge missing: %+v", rows[0].Lines)
	}
	if !slices.Contains(rows[3].Lines, Line{From: 1, To: 0}) {
		t.Fatalf("fork edge missing: %+v", rows[3].Lines)
	}
	if !slices.Contains(rows[2].Lines, Line{From: 1, To: 1, Bottom: true}) {
		t.Fatalf("pass-through missing: %+v", rows[2].Lines)
	}
}

func TestLogReal(t *testing.T) {
	dir := newRepo(t)
	must(t, dir, "checkout", "-qb", "feat")
	write(t, dir, "f.txt", "f\n")
	must(t, dir, "add", ".")
	must(t, dir, "commit", "-qm", "feat: subject, with comma")
	must(t, dir, "checkout", "-q", "main")
	rows, err := Log(dir, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].Subject != "feat: subject, with comma" || !slices.Contains(rows[0].Refs, "feat") {
		t.Fatalf("%+v", rows)
	}
	if !slices.Contains(rows[1].Refs, "HEAD -> main") {
		t.Fatalf("refs %v", rows[1].Refs)
	}
	must(t, dir, "mv", "a.txt", "renamed.txt")
	must(t, dir, "commit", "-qm", "mv")
	head := must(t, dir, "rev-parse", "HEAD")[:40]
	files, err := CommitFiles(dir, head)
	if err != nil || len(files) != 1 || files[0].Status != "R" || files[0].OldPath != "a.txt" || files[0].Path != "renamed.txt" {
		t.Fatalf("%+v %v", files, err)
	}
	if s := CommitSides(dir, head, "renamed.txt", "a.txt"); s.Old != "one\n" || s.New != "one\n" {
		t.Fatalf("%+v", s)
	}
	if files, _ := CommitFiles(dir, rows[1].Hash); len(files) != 1 || files[0].Status != "A" {
		t.Fatalf("root %+v", files)
	}
}

func TestHistoryPagesPreserveRowsAndRejectMovedAnchor(t *testing.T) {
	dir := newRepo(t)
	for i := 0; i < 5; i++ {
		must(t, dir, "commit", "--allow-empty", "-qm", "another")
	}
	want, err := Log(dir, 20)
	if err != nil {
		t.Fatal(err)
	}
	first, err := LogPage(dir, 0, 3, "", nil)
	if err != nil || !first.More {
		t.Fatal(first, err)
	}
	next, err := LogPage(dir, 3, 3, first.Rows[2].Hash, first.Lanes)
	if err != nil || next.More {
		t.Fatal(next, err)
	}
	if !reflect.DeepEqual(append(first.Rows, next.Rows...), want) {
		t.Fatal("paged rows differ")
	}
	must(t, dir, "commit", "--allow-empty", "-qm", "move HEAD")
	changed, err := LogPage(dir, 3, 3, first.Rows[2].Hash, first.Lanes)
	if err != nil || !changed.Restart {
		t.Fatal(changed, err)
	}
}
