package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenDefaults(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	corrupt := filepath.Join(dir, "corrupt.json")
	os.WriteFile(corrupt, []byte("{not json"), 0o600)
	for _, p := range []string{filepath.Join(dir, "missing.json"), corrupt} {
		s := Open(p).Get()
		if s.Repos == nil || len(s.Repos) != 0 || s.Mode != "staging" || s.Changelists == nil {
			t.Fatalf("%s: %+v", p, s)
		}
	}
}

func TestUpdatePersists(t *testing.T) {
	t.Parallel()
	p := filepath.Join(t.TempDir(), "sub", "state.json")
	st := Open(p)
	err := st.Update(func(s *State) {
		s.Repos = append(s.Repos, "C:/r")
		s.Mode = "changelists"
		s.Changelists["C:/r"] = map[string][]string{"Fix": {"a.go"}}
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p + ".tmp"); !os.IsNotExist(err) {
		t.Fatal("temp file left behind")
	}
	got := Open(p).Get()
	if len(got.Repos) != 1 || got.Mode != "changelists" || got.Changelists["C:/r"]["Fix"][0] != "a.go" {
		t.Fatalf("%+v", got)
	}
}

func TestGetIsDeepCopy(t *testing.T) {
	t.Parallel()
	st := Open(filepath.Join(t.TempDir(), "s.json"))
	st.Update(func(s *State) {
		s.Repos = []string{"a"}
		s.Changelists["d"] = map[string][]string{"L": {"x"}}
	})
	g := st.Get()
	g.Repos[0] = "mutated"
	g.Changelists["d"]["L"][0] = "mutated"
	g.Changelists["d"]["New"] = nil
	g.Changelists["e"] = nil
	again := st.Get()
	if again.Repos[0] != "a" || again.Changelists["d"]["L"][0] != "x" || len(again.Changelists["d"]) != 1 || len(again.Changelists) != 1 {
		t.Fatalf("store changed through Get: %+v", again)
	}
}
