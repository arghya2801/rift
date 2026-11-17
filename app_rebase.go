package main

import "rift/internal/git"

func (a *App) RebaseTodo(dir, base string) ([]git.TodoItem, error) { return git.RebaseTodo(dir, base) }
func (a *App) RunRebase(dir, base string, items []git.TodoItem) error {
	return git.RunRebase(dir, base, items)
}
