package main

import (
	"sync"

	"rift/internal/git"
)

// Snapshot is everything the views need about one worktree. A file edit costs one git process
// (status); refs are re-read only when full is set (the git dir changed).
type Snapshot struct {
	Status  git.Status  `json:"status"`
	Refs    []git.Ref   `json:"refs"`
	Stashes []git.Stash `json:"stashes"`
	Op      git.Op      `json:"op"`
	Full    bool        `json:"full"`
	Error   string      `json:"error,omitempty"`
}

func (a *App) Snapshot(dir string, full bool) Snapshot {
	s := Snapshot{Full: full, Refs: []git.Ref{}, Stashes: []git.Stash{}}
	var wg sync.WaitGroup
	var refErr error
	if full {
		wg.Go(func() {
			if r, err := git.Refs(dir); err == nil {
				s.Refs = r
			} else {
				refErr = err
			}
			if st, err := git.Stashes(dir); err == nil {
				s.Stashes = st
			}
		})
	}
	st, err := git.GetStatus(dir)
	wg.Wait()
	s.Status = st
	s.Op = git.OpFromStatus(dir, st)
	if err == nil {
		err = refErr
	}
	if err != nil {
		s.Error = err.Error()
	}
	return s
}
