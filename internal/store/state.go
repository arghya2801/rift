package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"sync"
)

type State struct {
	Repos       []string                       `json:"repos"`
	Mode        string                         `json:"mode"`
	Changelists map[string]map[string][]string `json:"changelists"`
}

type Store struct {
	path string
	mu   sync.Mutex
	s    State
}

func ConfigDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return filepath.Join(dir, "rift")
}

func Open(path string) *Store {
	st := &Store{path: path}
	if b, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(b, &st.s)
	}
	if st.s.Repos == nil {
		st.s.Repos = []string{}
	}
	if st.s.Mode == "" {
		st.s.Mode = "staging"
	}
	if st.s.Changelists == nil {
		st.s.Changelists = map[string]map[string][]string{}
	}
	return st
}

func (st *Store) Get() State {
	st.mu.Lock()
	defer st.mu.Unlock()
	cl := make(map[string]map[string][]string, len(st.s.Changelists))
	for dir, lists := range st.s.Changelists {
		m := make(map[string][]string, len(lists))
		for name, paths := range lists {
			m[name] = slices.Clone(paths)
		}
		cl[dir] = m
	}
	return State{Repos: slices.Clone(st.s.Repos), Mode: st.s.Mode, Changelists: cl}
}

func (st *Store) Update(f func(*State)) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	f(&st.s)
	b, err := json.MarshalIndent(st.s, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(st.path), 0o700); err != nil {
		return err
	}
	tmp := st.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, st.path)
}
