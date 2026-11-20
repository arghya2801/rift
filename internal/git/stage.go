package git

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func Stage(dir string, paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	_, err := Run(dir, append([]string{"--literal-pathspecs", "add", "-A", "--"}, paths...)...)
	return err
}

func Unstage(dir string, paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	if _, err := Run(dir, "rev-parse", "--verify", "-q", "HEAD"); err != nil {
		_, err = Run(dir, append([]string{"--literal-pathspecs", "rm", "-q", "--cached", "--"}, paths...)...)
		return err
	}
	_, err := Run(dir, append([]string{"--literal-pathspecs", "reset", "-q", "--"}, paths...)...)
	return err
}

// Discard drops all changes to paths: untracked files are deleted, tracked ones restored to HEAD.
func Discard(dir string, paths []string) error {
	s, err := GetStatus(dir)
	if err != nil {
		return err
	}
	want := map[string]bool{}
	for _, p := range paths {
		want[p] = true
	}
	var tracked, untracked []string
	for _, f := range s.Files {
		if !want[f.Path] {
			continue
		}
		if f.Index == '?' {
			untracked = append(untracked, f.Path)
			continue
		}
		tracked = append(tracked, f.Path)
		if f.OrigPath != "" && (f.Index == 'R' || f.Work == 'R') {
			tracked = append(tracked, f.OrigPath)
		}
	}
	if len(tracked) != 0 {
		source := "HEAD"
		if s.unborn {
			hash, err := Run(dir, "hash-object", "-w", "-t", "tree", "--stdin")
			if err != nil {
				return err
			}
			source = strings.TrimSpace(hash)
		}
		if _, err := Run(dir, append([]string{"--literal-pathspecs", "restore", "--source=" + source, "--staged", "--worktree", "--"}, tracked...)...); err != nil {
			return err
		}
	}
	for _, path := range untracked {
		if err := os.Remove(filepath.Join(dir, filepath.FromSlash(path))); err != nil {
			return err
		}
	}
	return nil
}

// AddToGitignore appends path (repo-relative, slash form) to the top-level .gitignore.
func AddToGitignore(dir, path string) error {
	name := filepath.Join(dir, ".gitignore")
	if strings.ContainsAny(path, "\r\n") {
		return errors.New("filename cannot be represented by one ignore rule")
	}
	f, err := os.OpenFile(name, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	escape := strings.NewReplacer("\\", "\\\\", "*", "\\*", "?", "\\?", "[", "\\[", " ", "\\ ")
	line := "/" + escape.Replace(strings.TrimPrefix(path, "/"))
	if n := info.Size(); n > 0 {
		var tail [1]byte
		if _, err := f.ReadAt(tail[:], n-1); err != nil {
			return err
		}
		if tail[0] != '\n' {
			line = "\n" + line
		}
	}
	_, err = f.WriteString(line + "\n")
	return err
}

// StageContent writes text (worktree form) as the index entry for path, keeping the
// existing file mode. indexFile overrides GIT_INDEX_FILE when set.
func StageContent(dir, indexFile, path, text string) error {
	var env []string
	if indexFile != "" {
		env = []string{"GIT_INDEX_FILE=" + indexFile}
	}
	sha, err := hashObject(dir, path, text)
	if err != nil {
		return err
	}
	_, err = RunEnv(dir, env, "update-index", "--add", "--cacheinfo", fileMode(dir, env, path)+","+sha+","+path)
	return err
}

func hashObject(dir, path, text string) (string, error) {
	cmd := command(dir, nil, "hash-object", "-w", "--stdin", "--path="+path)
	cmd.Stdin = strings.NewReader(text)
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

func fileMode(dir string, env []string, path string) string {
	if out, _ := RunEnv(dir, env, "--literal-pathspecs", "ls-files", "-s", "--", path); len(out) > 6 {
		return out[:6]
	}
	if out, _ := Run(dir, "--literal-pathspecs", "ls-tree", "HEAD", "--", path); len(out) > 6 {
		return out[:6]
	}
	if st, err := os.Lstat(filepath.Join(dir, path)); err == nil && st.Mode().IsRegular() && st.Mode().Perm()&0o111 != 0 {
		if mode, _ := Run(dir, "config", "--bool", "core.filemode"); strings.TrimSpace(mode) == "true" {
			return "100755"
		}
	}
	return "100644"
}
