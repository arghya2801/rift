package watch

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"rift/internal/git"

	"github.com/fsnotify/fsnotify"
)

// Watcher follows one worktree at a time and calls onChange (debounced) when it changes.
type Watcher struct {
	mu       sync.Mutex
	w        *fsnotify.Watcher
	path     string
	skip     map[string]bool
	gitDirs  []string
	gitHit   bool
	gen      int
	onChange func(path string, git bool)
}

// New calls onChange with git=true when the change touched the git dir (HEAD, index, refs).
func New(onChange func(path string, git bool)) (*Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	wt := &Watcher{w: w, onChange: onChange}
	go wt.loop()
	return wt, nil
}

// Set switches to dir. Ignored dirs are skipped.
// ponytail: one fsnotify watch per dir; huge trees can hit OS limits (inotify), focus refresh still works.
func (wt *Watcher) Set(dir string) {
	wt.mu.Lock()
	wt.gen++
	gen := wt.gen
	wt.path = dir
	wt.mu.Unlock()

	// Slow work (git spawns, tree walk) happens unlocked; a newer Set makes this one bail out.
	var dirs, gitDirs []string
	var skip map[string]bool
	if dir != "" {
		skip = ignoredDirs(dir)
		_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
			if err != nil || !d.IsDir() {
				return nil
			}
			if d.Name() == ".git" || skip[p] {
				return filepath.SkipDir
			}
			dirs = append(dirs, p)
			return nil
		})
		if out, err := git.Run(dir, "rev-parse", "--path-format=absolute", "--git-dir", "--git-common-dir"); err == nil {
			for l := range strings.Lines(out) {
				g := filepath.Clean(strings.TrimSpace(l))
				gitDirs = append(gitDirs, g)
				dirs = append(dirs, g, filepath.Join(g, "logs"))
				_ = filepath.WalkDir(filepath.Join(g, "refs"), func(p string, d fs.DirEntry, err error) error {
					if err == nil && d.IsDir() {
						dirs = append(dirs, p)
					}
					return nil
				})
			}
		}
	}

	wt.mu.Lock()
	defer wt.mu.Unlock()
	if gen != wt.gen {
		return
	}
	for _, p := range wt.w.WatchList() {
		_ = wt.w.Remove(p)
	}
	wt.skip, wt.gitDirs = skip, gitDirs
	for _, p := range dirs {
		// fsnotify allocates 64 KB per watched dir on Windows by default; 4 KB is plenty for a debounced refresh.
		_ = wt.w.AddWith(p, fsnotify.WithBufferSize(4096))
	}
}

func ignoredDirs(dir string) map[string]bool {
	out, _ := git.Run(dir, "ls-files", "-o", "-i", "--exclude-standard", "--directory", "-z")
	m := map[string]bool{}
	for p := range strings.SplitSeq(out, "\x00") {
		if d, ok := strings.CutSuffix(p, "/"); ok {
			m[filepath.Join(dir, filepath.FromSlash(d))] = true
		}
	}
	return m
}

func (wt *Watcher) loop() {
	var timer *time.Timer
	for {
		select {
		case ev, ok := <-wt.w.Events:
			if !ok {
				return
			}
			wt.mu.Lock()
			ignored := wt.skip[filepath.Clean(ev.Name)]
			wt.mu.Unlock()
			if ignored || strings.HasSuffix(ev.Name, ".lock") {
				continue
			}
			// Windows reports a dir as written when a child changes, sometimes late; the child's own event is enough.
			if ev.Op == fsnotify.Write {
				if fi, err := os.Stat(ev.Name); err == nil && fi.IsDir() {
					continue
				}
			}
			if ev.Has(fsnotify.Create) {
				wt.mu.Lock()
				if fi, err := os.Stat(ev.Name); err == nil && fi.IsDir() {
					_ = wt.w.AddWith(ev.Name, fsnotify.WithBufferSize(4096))
				}
				wt.mu.Unlock()
			}
			wt.mu.Lock()
			name := filepath.Clean(ev.Name)
			for _, g := range wt.gitDirs {
				if name == g || strings.HasPrefix(name, g+string(filepath.Separator)) {
					wt.gitHit = true
				}
			}
			wt.mu.Unlock()
			if timer != nil {
				timer.Stop()
			}
			timer = time.AfterFunc(250*time.Millisecond, func() {
				wt.mu.Lock()
				p, g := wt.path, wt.gitHit
				wt.gitHit = false
				wt.mu.Unlock()
				if p != "" {
					wt.onChange(p, g)
				}
			})
		case _, ok := <-wt.w.Errors:
			if !ok {
				return
			}
		}
	}
}
