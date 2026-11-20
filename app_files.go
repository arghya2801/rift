package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"rift/internal/workfile"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Entry struct {
	Name  string `json:"name"`
	IsDir bool   `json:"isDir"`
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
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	listing, err := root.Open(filepath.Clean(filepath.FromSlash(rel)))
	if err != nil {
		return nil, err
	}
	defer listing.Close()
	des, err := listing.ReadDir(-1)
	if err != nil {
		return nil, err
	}
	out := make([]Entry, 0, len(des))
	for _, d := range des {
		if d.Name() == ".git" {
			continue
		}
		out = append(out, Entry{Name: d.Name(), IsDir: d.IsDir()})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].IsDir != out[j].IsDir {
			return out[i].IsDir
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out, nil
}

func (a *App) ReadFile(dir, rel string) (string, error) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return "", err
	}
	defer root.Close()
	p := filepath.FromSlash(rel)
	if st, err := root.Lstat(p); err != nil {
		return "", err
	} else if !st.Mode().IsRegular() {
		return "", errors.New("only regular files can be edited")
	} else if st.Size() > 5<<20 {
		return "", errors.New("file is larger than 5 MB")
	}
	b, err := root.ReadFile(p)
	if err != nil {
		return "", err
	}
	if bytes.IndexByte(b, 0) >= 0 {
		return "", errors.New("binary file")
	}
	if !utf8.Valid(b) {
		return "", errors.New("file is not UTF-8 text")
	}
	return string(b), nil
}

func (a *App) WriteFile(dir, rel, text string) error {
	return workfile.Write(dir, rel, text, nil)
}

func (a *App) SaveFile(dir, rel, text, expected string) error {
	return workfile.Write(dir, rel, text, &expected)
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
	args, err := editorArgs(tpl)
	if err != nil {
		return err
	}
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

// Quotes group an argument; backslashes stay literal for Windows executable paths.
func editorArgs(template string) ([]string, error) {
	var args []string
	var word strings.Builder
	var quote rune
	started := false
	for _, r := range template {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				word.WriteRune(r)
			}
		case r == '"' || r == '\'':
			quote = r
			started = true
		case unicode.IsSpace(r):
			if started {
				args = append(args, word.String())
				word.Reset()
				started = false
			}
		default:
			word.WriteRune(r)
			started = true
		}
	}
	if quote != 0 {
		return nil, errors.New("unclosed quote in RIFT_EDITOR")
	}
	if started {
		args = append(args, word.String())
	}
	if len(args) == 0 || args[0] == "" {
		return nil, errors.New("missing executable in RIFT_EDITOR")
	}
	return args, nil
}
