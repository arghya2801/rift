package main

import "rift/internal/git"

func (a *App) OpState(dir string) (git.Op, error) { return git.OpState(dir) }
func (a *App) ConflictSides(dir, path string) (git.Conflict, error) {
	return git.ConflictSides(dir, path)
}
func (a *App) ResolveFile(dir, path, text string) error { return git.ResolveFile(dir, path, text) }
func (a *App) AcceptSide(dir, path, side string) error  { return git.AcceptSide(dir, path, side) }
func (a *App) OpContinue(dir string) error              { return git.OpContinue(dir) }
func (a *App) OpAbort(dir string) error                 { return git.OpAbort(dir) }
func (a *App) OpSkip(dir string) error                  { return git.OpSkip(dir) }
