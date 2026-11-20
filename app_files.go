package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type Entry struct {
	Name     string `json:"name"`
	IsDir    bool   `json:"isDir"`
	sortName string
}

func inside(dir, rel string) (string, error) {
	p := filepath.Join(dir, filepath.FromSlash(rel))
	r, err := filepath.Rel(dir, p)
	if err != nil || r == ".." || strings.HasPrefix(r, ".."+string(filepath.Separator)) {
		return "", errors.New("path outside worktree")
	}
	return p, nil
}

func (a *App) ListDir(dir, rel string) ([]Entry, error) {
	p, err := inside(dir, rel)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	des, err := f.ReadDir(-1)
	if err != nil {
		return nil, err
	}
	out := make([]Entry, 0, len(des))
	for _, d := range des {
		if d.Name() == ".git" {
			continue
		}
		out = append(out, Entry{Name: d.Name(), IsDir: d.IsDir(), sortName: strings.ToLower(d.Name())})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].IsDir != out[j].IsDir {
			return out[i].IsDir
		}
		if out[i].sortName == out[j].sortName {
			return out[i].Name < out[j].Name
		}
		return out[i].sortName < out[j].sortName
	})
	return out, nil
}

func (a *App) ReadFile(dir, rel string) (string, error) {
	p, err := inside(dir, rel)
	if err != nil {
		return "", err
	}
	if st, err := os.Stat(p); err != nil {
		return "", err
	} else if st.Size() > 5<<20 {
		return "", errors.New("file is larger than 5 MB")
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	if bytes.IndexByte(b, 0) >= 0 {
		return "", errors.New("binary file")
	}
	return string(b), nil
}

func (a *App) WriteFile(dir, rel, text string) error {
	p, err := inside(dir, rel)
	if err != nil {
		return err
	}
	mode := os.FileMode(0o644)
	if st, err := os.Stat(p); err == nil {
		mode = st.Mode().Perm()
	}
	tmp := p + ".rift-tmp"
	if err := os.WriteFile(tmp, []byte(text), mode); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

// OpenExternal runs $RIFT_EDITOR (default "code -g {path}:{line}") without waiting.
func (a *App) OpenExternal(dir, rel string, line int) error {
	p, err := inside(dir, rel)
	if err != nil {
		return err
	}
	tpl := os.Getenv("RIFT_EDITOR")
	if tpl == "" {
		tpl = "code -g {path}:{line}"
	}
	if line < 1 {
		line = 1
	}
	if fi, err := os.Stat(p); err == nil && fi.IsDir() {
		tpl = strings.ReplaceAll(tpl, ":{line}", "")
	}
	args := strings.Fields(tpl)
	for i, s := range args {
		args[i] = strings.NewReplacer("{path}", p, "{line}", strconv.Itoa(line), "{dir}", dir).Replace(s)
	}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait()
	return nil
}
