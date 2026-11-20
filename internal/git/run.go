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

// commonDir is the shared git dir (refs, logs, objects) for all worktrees of a repo, cached.
func commonDir(dir string) (string, error) {
	if g, ok := commonDirs.Load(dir); ok {
		return g.(string), nil
	}
	out, err := Run(dir, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return "", err
	}
	g := filepath.Clean(strings.TrimSpace(out))
	commonDirs.Store(dir, g)
	return g, nil
}

// gitDir is the worktree's own git dir, cached: it never changes for a given worktree path.
func gitDir(dir string) (string, error) {
	if g, ok := gitDirs.Load(dir); ok {
		return g.(string), nil
	}
	out, err := Run(dir, "rev-parse", "--path-format=absolute", "--git-dir")
	if err != nil {
		return "", err
	}
	g := filepath.Clean(strings.TrimSpace(out))
	gitDirs.Store(dir, g)
	return g, nil
}
