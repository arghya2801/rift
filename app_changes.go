package main

import (
	"errors"

	"rift/internal/git"
	"rift/internal/store"
)

func (a *App) GetMode() string { return a.store.Get().Mode }

func (a *App) SetMode(m string) error {
	if m != "staging" && m != "changelists" {
		return errors.New("unknown mode " + m)
	}
	return a.store.Update(func(s *store.State) { s.Mode = m })
}

func (a *App) Changelists(dir string) map[string][]string {
	if l := a.store.Get().Changelists[dir]; l != nil {
		return l
	}
	return map[string][]string{}
}

func (a *App) SetChangelists(dir string, lists map[string][]string) error {
	return a.store.Update(func(s *store.State) { s.Changelists[dir] = lists })
}

func (a *App) Discard(dir string, paths []string) error { return git.Discard(dir, paths) }
func (a *App) AddToGitignore(dir, path string) error    { return git.AddToGitignore(dir, path) }

func (a *App) HeadSides(dir, path string) (git.Sides, error) { return git.HeadSides(dir, path) }

func (a *App) CommitPartial(dir, message string, amend bool, files []git.PartialFile) error {
	return git.CommitPartial(dir, message, amend, files)
}
