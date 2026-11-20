package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"rift/internal/git"
	"rift/internal/store"
	"rift/internal/watch"

	wr "github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx     context.Context
	store   *store.Store
	watcher *watch.Watcher
}

type Repo struct {
	Path      string         `json:"path"`
	Name      string         `json:"name"`
	Worktrees []git.Worktree `json:"worktrees"`
	Error     string         `json:"error,omitempty"`
}

func NewApp(config string) *App {
	return &App{store: store.Open(filepath.Join(config, "state.json"))}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.watcher, _ = watch.New(func(p string, refs, index bool, paths []string) {
		wr.EventsEmit(ctx, "repo://changed", p, refs, index, paths)
	})
}

// Initial pins the repo given as the first CLI argument and returns its worktree path ("" if none).
func (a *App) Initial() string {
	if len(os.Args) < 2 || strings.HasPrefix(os.Args[1], "-") {
		return ""
	}
	dir, err := git.Root(os.Args[1])
	if err != nil {
		return ""
	}
	if _, err := a.AddRepo(dir); err != nil {
		return ""
	}
	return dir
}

// Watch switches the live-refresh watcher to dir ("" stops it).
func (a *App) Watch(dir string) {
	if a.watcher != nil {
		a.watcher.Switch(dir)
	}
}

// Inventory and dashboard requests share a process limit, including overlapping refreshes.
var dashboardSlots = make(chan struct{}, 4)

func parallel(n int, fn func(int)) {
	var wg sync.WaitGroup
	for i := range n {
		dashboardSlots <- struct{}{}
		wg.Go(func() {
			defer func() { <-dashboardSlots }()
			fn(i)
		})
	}
	wg.Wait()
}

func (a *App) Repos() []Repo {
	paths := a.store.Repos()
	out := make([]Repo, len(paths))
	parallel(len(paths), func(i int) {
		p := paths[i]
		r := Repo{Path: p, Name: filepath.Base(p)}
		if wts, err := git.Worktrees(p); err != nil {
			r.Error = err.Error()
		} else {
			r.Worktrees = wts
		}
		out[i] = r
	})
	return out
}

// AddRepo pins the main worktree of the repo containing dir; a linked worktree resolves to its main one.
func (a *App) AddRepo(dir string) (string, error) {
	wts, err := git.Worktrees(dir)
	if err != nil {
		return "", err
	}
	main := wts[0].Path
	return main, a.store.Update(func(s *store.State) {
		if !slices.Contains(s.Repos, main) {
			s.Repos = append(s.Repos, main)
		}
	})
}

func (a *App) RemoveRepo(path string) error {
	return a.store.Update(func(s *store.State) {
		s.Repos = slices.DeleteFunc(s.Repos, func(p string) bool { return p == path })
	})
}

func (a *App) AddWorktree(mainDir, branch, base string, newBranch bool) (string, error) {
	return git.AddWorktree(mainDir, branch, base, newBranch)
}
func (a *App) RemoveWorktree(mainDir, path string, force bool) error {
	return git.RemoveWorktree(mainDir, path, force)
}
func (a *App) PruneWorktrees(mainDir string) error { return git.PruneWorktrees(mainDir) }

// Overview is one dashboard card per worktree of every pinned repo; status runs concurrently.
type Overview struct {
	Repo      string       `json:"repo"`
	Wt        git.Worktree `json:"wt"`
	Changed   int          `json:"changed"`
	Conflicts int          `json:"conflicts"`
	Staged    int          `json:"staged"`
	Status    git.Status   `json:"status"`
	Last      string       `json:"last"`
	Op        string       `json:"op"`
	Error     string       `json:"error,omitempty"`
}

func (a *App) Overview(repos []Repo) []Overview {
	var out []Overview
	for _, r := range repos {
		for _, w := range r.Worktrees {
			if !w.Bare {
				out = append(out, Overview{Repo: r.Name, Wt: w})
			}
		}
	}
	parallel(len(out), func(i int) {
		o := &out[i]
		s, err := git.GetStatus(o.Wt.Path)
		if err != nil {
			o.Error = err.Error()
			return
		}
		o.Changed = len(s.Files)
		for _, f := range s.Files {
			if f.Conflict {
				o.Conflicts++
			} else if f.Index != '.' && f.Index != '?' {
				o.Staged++
			}
		}
		s.Files = nil
		o.Status = s
		last, _ := git.Run(o.Wt.Path, "log", "-1", "--format=%s (%ar)")
		o.Last = strings.TrimSpace(last)
		o.Op = git.OpName(o.Wt.Path)
	})
	return out
}

// OpenInATC runs `atc --open <dir>` (needs ATC issue #278).
func (a *App) OpenInATC(dir string) error {
	exe, err := exec.LookPath("atc")
	if err != nil {
		return errors.New("atc not found on PATH")
	}
	cmd := exec.Command(exe, "--open", dir)
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait()
	return nil
}

func (a *App) PickFolder() (string, error) {
	return wr.OpenDirectoryDialog(a.ctx, wr.OpenDialogOptions{Title: "Open repository"})
}

func (a *App) Status(path string) (git.Status, error) {
	return git.GetStatus(path)
}

func (a *App) FileSides(dir, path string, staged bool, oldPath string) (git.Sides, error) {
	return git.FileSides(dir, path, staged, oldPath)
}

func (a *App) Stage(dir string, paths []string) error   { return git.Stage(dir, paths) }
func (a *App) Unstage(dir string, paths []string) error { return git.Unstage(dir, paths) }

func (a *App) StageContent(dir, path, text string) error {
	return git.StageContent(dir, "", path, text)
}

func (a *App) Commit(dir, message string, amend bool) error {
	return git.CommitStaged(dir, message, amend)
}
func (a *App) LastMessage(dir string) string { return git.LastMessage(dir) }

func (a *App) Log(dir string, limit int) ([]git.Row, error) { return git.Log(dir, limit) }
func (a *App) LogPage(dir string, offset, limit int, anchor string, lanes []string) (git.Page, error) {
	return git.LogPage(dir, offset, limit, anchor, lanes)
}
func (a *App) CommitFiles(dir, hash string) ([]git.ChangedFile, error) {
	return git.CommitFiles(dir, hash)
}
func (a *App) CommitSides(dir, hash, path, oldPath string) (git.Sides, error) {
	return git.CommitSides(dir, hash, path, oldPath)
}
func (a *App) CommitMessage(dir, hash string) string { return git.CommitMessage(dir, hash) }

func (a *App) Refs(dir string) ([]git.Ref, error)   { return git.Refs(dir) }
func (a *App) Checkout(dir, ref string) error       { return git.Checkout(dir, ref) }
func (a *App) CheckoutRemote(dir, ref string) error { return git.CheckoutRemote(dir, ref) }
func (a *App) DeleteBranch(dir, name string, force bool) error {
	return git.DeleteBranch(dir, name, force)
}
func (a *App) RenameBranch(dir, old, name string) error { return git.RenameBranch(dir, old, name) }
func (a *App) CreateBranch(dir, name, at string, checkout bool) error {
	return git.CreateBranch(dir, name, at, checkout)
}
func (a *App) Merge(dir, ref string) error                  { return git.Merge(dir, ref) }
func (a *App) Rebase(dir, onto string) error                { return git.Rebase(dir, onto) }
func (a *App) CherryPick(dir, hash string) error            { return git.CherryPick(dir, hash) }
func (a *App) Revert(dir, hash string) error                { return git.Revert(dir, hash) }
func (a *App) Reset(dir, mode, hash string) error           { return git.Reset(dir, mode, hash) }
func (a *App) CreateTag(dir, name, hash string) error       { return git.CreateTag(dir, name, hash) }
func (a *App) Fetch(dir string) (string, error)             { return git.Fetch(dir) }
func (a *App) Pull(dir string, rebase bool) (string, error) { return git.Pull(dir, rebase) }
func (a *App) Push(dir string, force bool) (string, error)  { return git.Push(dir, force) }
func (a *App) Stashes(dir string) ([]git.Stash, error)      { return git.Stashes(dir) }
func (a *App) StashPush(dir, message string, untracked bool) error {
	return git.StashPush(dir, message, untracked)
}
func (a *App) StashAction(dir, action, ref string) error { return git.StashAction(dir, action, ref) }
