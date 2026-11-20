package watch

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
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
	cancel   chan struct{}
	gen      int
	onChange func(path string, refs, index bool, paths []string)
}

// New calls onChange with git=true when the change touched the git dir (HEAD, index, refs).
func New(onChange func(path string, refs, index bool, paths []string)) (*Watcher, error) {
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
	wt.setup(dir, wt.reserve(dir))
}

func (wt *Watcher) Switch(dir string) {
	cancel := wt.reserve(dir)
	go wt.setup(dir, cancel)
}

func (wt *Watcher) reserve(dir string) chan struct{} {
	wt.mu.Lock()
	defer wt.mu.Unlock()
	if wt.cancel != nil {
		close(wt.cancel)
	}
	wt.gen++
	wt.cancel = make(chan struct{})
	wt.path = dir
	return wt.cancel
}

func (wt *Watcher) setup(dir string, cancel chan struct{}) {
	cancelled := func() bool {
		select {
		case <-cancel:
			return true
		default:
			return false
		}
	}
	if cancelled() {
		return
	}
	var dirs, gitDirs []string
	var skip map[string]bool
	if dir != "" {
		skip = ignoredDirs(dir)
		_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
			if cancelled() {
				return fs.SkipAll
			}
			if err != nil || !d.IsDir() {
				return nil
			}
			if d.Name() == ".git" || skip[p] {
				return filepath.SkipDir
			}
			dirs = append(dirs, p)
			return nil
		})
		if cancelled() {
			return
		}
		if out, err := git.Run(dir, "rev-parse", "--path-format=absolute", "--git-dir", "--git-common-dir"); err == nil {
			for l := range strings.Lines(out) {
				g := filepath.Clean(strings.TrimSpace(l))
				if slices.Contains(gitDirs, g) {
					continue
				}
				gitDirs = append(gitDirs, g)
				dirs = append(dirs, g, filepath.Join(g, "logs"))
				_ = filepath.WalkDir(filepath.Join(g, "refs"), func(p string, d fs.DirEntry, err error) error {
					if cancelled() {
						return fs.SkipAll
					}
					if err == nil && d.IsDir() {
						dirs = append(dirs, p)
					}
					return nil
				})
			}
		}
	}

	wt.mu.Lock()
	if cancelled() {
		wt.mu.Unlock()
		return
	}
	for _, p := range wt.w.WatchList() {
		_ = wt.w.Remove(p)
	}
	wt.skip, wt.gitDirs = skip, gitDirs
	wt.mu.Unlock()
	for _, p := range dirs {
		wt.mu.Lock()
		if cancelled() {
			wt.mu.Unlock()
			return
		}
		// fsnotify allocates 64 KB per watched dir on Windows by default; 4 KB is plenty for a debounced refresh.
		_ = wt.w.AddWith(p, fsnotify.WithBufferSize(4096))
		wt.mu.Unlock()
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
	timer := time.NewTimer(time.Hour)
	timer.Stop()
	defer timer.Stop()
	var generation int
	var refs, index, all bool
	paths := map[string]bool{}
	for {
		select {
		case <-timer.C:
			wt.mu.Lock()
			dir, current := wt.path, wt.gen
			wt.mu.Unlock()
			if generation == current && dir != "" {
				var changed []string
				if !all {
					changed = make([]string, 0, len(paths))
					for p := range paths {
						changed = append(changed, p)
					}
				}
				wt.onChange(dir, refs, index, changed)
			}
			clear(paths)
			refs, index, all = false, false, false
		case ev, ok := <-wt.w.Events:
			if !ok {
				return
			}
			wt.mu.Lock()
			name := filepath.Clean(ev.Name)
			gen := wt.gen
			inGit := false
			for _, g := range wt.gitDirs {
				if name == g || strings.HasPrefix(name, g+string(filepath.Separator)) {
					inGit = true
					break
				}
			}
			ignored := wt.skip[name] || wt.path == "" || (!inGit && name != wt.path && !strings.HasPrefix(name, wt.path+string(filepath.Separator)))
			wt.mu.Unlock()
			if ignored || (inGit && strings.HasSuffix(name, ".lock")) {
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
					if gen == wt.gen {
						_ = filepath.WalkDir(name, func(p string, d fs.DirEntry, err error) error {
							if err != nil || !d.IsDir() {
								return nil
							}
							if wt.skip[p] || d.Name() == ".git" {
								return filepath.SkipDir
							}
							_ = wt.w.AddWith(p, fsnotify.WithBufferSize(4096))
							return nil
						})
					}
				}
				wt.mu.Unlock()
			}
			wt.mu.Lock()
			if gen != wt.gen {
				wt.mu.Unlock()
				continue
			}
			if generation != wt.gen {
				clear(paths)
				refs, index, all, generation = false, false, false, wt.gen
			}
			meta := false
			for _, g := range wt.gitDirs {
				if name == g || strings.HasPrefix(name, g+string(filepath.Separator)) {
					meta = true
					if !strings.HasSuffix(name, ".lock") {
						if name == filepath.Join(g, "index") {
							index = true
						} else {
							refs = true
						}
					}
					break
				}
			}
			if !meta && !all {
				if rel, err := filepath.Rel(wt.path, name); err == nil && filepath.IsLocal(rel) {
					paths[filepath.ToSlash(rel)] = true
					all = len(paths) >= 256
				}
			}
			wt.mu.Unlock()
			if refs || index || all || len(paths) > 0 {
				timer.Reset(250 * time.Millisecond)
			}
		case _, ok := <-wt.w.Errors:
			if !ok {
				return
			}
			wt.mu.Lock()
			generation = wt.gen
			dir, cancel := wt.path, wt.cancel
			wt.mu.Unlock()
			if dir != "" {
				wt.setup(dir, cancel)
			}
			refs, index, all = true, true, true
			timer.Reset(250 * time.Millisecond)
		}
	}
}
