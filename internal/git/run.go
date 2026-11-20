package git

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// gitExe resolves the real git binary once. On Windows, Git\cmd\git.exe is a launcher that spawns a
// second process, so calling git-core\git.exe directly halves process creations per call.
var gitExe = sync.OnceValue(func() string {
	if runtime.GOOS == "windows" {
		if out, err := exec.Command("git", "--exec-path").Output(); err == nil {
			p := filepath.Join(filepath.FromSlash(strings.TrimSpace(string(out))), "git.exe")
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	}
	return "git"
})

func command(dir string, env []string, args ...string) *exec.Cmd {
	cmd := exec.Command(gitExe(), append([]string{"-c", "core.quotepath=off"}, args...)...)
	cmd.Dir = dir
	// Optional locks off: read commands must not rewrite .git/index, or the watcher sees our own refreshes.
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_OPTIONAL_LOCKS=0", "LC_ALL=C")
	cmd.Env = append(cmd.Env, env...)
	hideWindow(cmd)
	return cmd
}

func Run(dir string, args ...string) (string, error) {
	return RunEnv(dir, nil, args...)
}

func RunEnv(dir string, env []string, args ...string) (string, error) {
	return runInput(dir, env, "", args...)
}

func runInput(dir string, env []string, input string, args ...string) (string, error) {
	cmd := command(dir, env, args...)
	if input != "" {
		cmd.Stdin = strings.NewReader(input)
	}
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		if msg := strings.TrimSpace(errb.String()); msg != "" {
			return out.String(), errors.New(msg)
		}
		return out.String(), err
	}
	return out.String(), nil
}

var gitDirs, commonDirs sync.Map

type gitLocation struct{ marker, path string }

func cachedDir(dir, option string, cache *sync.Map) (string, error) {
	marker := ""
	entry := filepath.Join(dir, ".git")
	if st, err := os.Lstat(entry); err == nil && st.Mode().IsRegular() {
		b, err := os.ReadFile(entry)
		if err != nil {
			return "", err
		}
		marker = string(b)
	}
	if saved, ok := cache.Load(dir); ok {
		loc := saved.(gitLocation)
		if loc.marker == marker {
			return loc.path, nil
		}
	}
	out, err := Run(dir, "rev-parse", "--path-format=absolute", option)
	if err != nil {
		return "", err
	}
	path := filepath.Clean(strings.TrimSpace(out))
	cache.Store(dir, gitLocation{marker, path})
	return path, nil
}

func commonDir(dir string) (string, error) { return cachedDir(dir, "--git-common-dir", &commonDirs) }
func gitDir(dir string) (string, error)    { return cachedDir(dir, "--git-dir", &gitDirs) }
