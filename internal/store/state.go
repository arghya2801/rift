package store

import (
	"encoding/json"
	"errors"
	"fmt"
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
	state, _ := read(path)
	return &Store{path: path, s: state}
}

func read(path string) (State, error) {
	state := State{Repos: []string{}, Mode: "staging", Changelists: map[string]map[string][]string{}}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return state, nil
	}
	if err != nil {
		return state, err
	}
	var decoded State
	if err := json.Unmarshal(b, &decoded); err != nil {
		return state, fmt.Errorf("read settings: %w", err)
	}
	if decoded.Repos != nil {
		state.Repos = decoded.Repos
	}
	if decoded.Mode != "" {
		state.Mode = decoded.Mode
	}
	if decoded.Changelists != nil {
		state.Changelists = decoded.Changelists
	}
	return state, nil
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
	if err := os.MkdirAll(filepath.Dir(st.path), 0o700); err != nil {
		return err
	}
	lock, err := os.OpenFile(st.path+".lock", os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	defer lock.Close()
	if err := lockFile(lock); err != nil {
		return err
	}
	next, err := read(st.path)
	if err != nil {
		return err
	}
	f(&next)
	b, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(st.path), ".rift-state-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	_, err = tmp.Write(b)
	closeErr := tmp.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if err := os.Rename(tmp.Name(), st.path); err != nil {
		return err
	}
	st.s = next
	return nil
}
