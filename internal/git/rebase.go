package git

import (
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type TodoItem struct {
	Action  string `json:"action"`
	Hash    string `json:"hash"`
	Subject string `json:"subject"`
	Message string `json:"message"`
}

// RebaseTodo lists base..HEAD oldest first; base "" means the whole history (--root).
func RebaseTodo(dir, base string) ([]TodoItem, error) {
	rng := "HEAD"
	if base != "" {
		rng = base + "..HEAD"
	}
	out, err := Run(dir, "log", "--no-merges", "--reverse", "-z", "--format=%H%x1f%s%x1f%B", rng)
	if err != nil {
		return nil, err
	}
	items := []TodoItem{}
	for rec := range strings.SplitSeq(out, "\x00") {
		f := strings.SplitN(strings.TrimPrefix(rec, "\n"), "\x1f", 3)
		if len(f) < 3 {
			continue
		}
		items = append(items, TodoItem{Action: "pick", Hash: f[0], Subject: f[1], Message: strings.TrimSpace(f[2])})
	}
	return items, nil
}

// Git records the original commit hash in rebase-merge/done before invoking the editor.
func writeRewords(msgDir string, items []TodoItem) error {
	for _, it := range items {
		if it.Action == "reword" {
			if strings.TrimSpace(it.Message) == "" {
				return errors.New("empty reword message")
			}
			if err := os.WriteFile(filepath.Join(msgDir, it.Hash+".msg"), []byte(it.Message), 0o600); err != nil {
				return err
			}
		}
	}
	return nil
}

// RunRebase runs git rebase -i with this binary as sequence and message editor.
// A stop for conflicts or edit is not an error; the caller checks RebaseInProgress.
func RunRebase(dir, base string, items []TodoItem) error {
	if len(items) == 0 {
		return errors.New("nothing to rebase")
	}
	havePick := false
	for _, it := range items {
		switch it.Action {
		case "squash", "fixup":
			if !havePick {
				return errors.New("first remaining commit cannot be squash or fixup")
			}
		case "pick", "edit", "reword":
			havePick = true
		case "drop":
		default:
			return errors.New("unsupported rebase action")
		}
		if _, err := hex.DecodeString(it.Hash); err != nil || (len(it.Hash) != 40 && len(it.Hash) != 64) {
			return errors.New("expected a full commit hash")
		}
		if strings.ContainsAny(it.Subject, "\r\n\x00") {
			return errors.New("invalid commit subject")
		}
	}
	if kind := OpKind(dir); kind != "" {
		return fmt.Errorf("%s already in progress", kind)
	}
	g, err := gitDir(dir)
	if err != nil {
		return err
	}
	// Kept in the git dir so a later --continue still finds the queued messages.
	tmp, err := os.MkdirTemp(g, "rift-rebase-start-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	var todo strings.Builder
	for _, it := range items {
		fmt.Fprintf(&todo, "%s %s %s\n", it.Action, it.Hash, it.Subject)
	}
	todoFile := filepath.Join(tmp, "todo")
	if err := os.WriteFile(todoFile, []byte(todo.String()), 0o600); err != nil {
		return err
	}
	msgDir := filepath.Join(tmp, "msg")
	if err := os.Mkdir(msgDir, 0o700); err != nil {
		return err
	}
	if err := writeRewords(msgDir, items); err != nil {
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	env := []string{"GIT_SEQUENCE_EDITOR=" + shq(exe) + " --rift-seq " + shq(todoFile), msgEditor(exe, msgDir)}
	args := []string{"rebase", "-i", "--no-rebase-merges", "--autostash"}
	if base == "" {
		args = append(args, "--root")
	} else {
		args = append(args, base)
	}
	_, err = RunEnv(dir, env, args...)
	if RebaseInProgress(dir) {
		// Only this sequence editor consumes our todo. A competing start cannot own this stop.
		if _, statErr := os.Stat(todoFile); os.IsNotExist(statErr) {
			saved := filepath.Join(g, "rebase-merge", "rift-msg")
			if moveErr := os.Rename(msgDir, saved); moveErr != nil {
				return errors.Join(err, moveErr)
			}
			if err != nil {
				if unmerged, statusErr := Run(dir, "ls-files", "--unmerged", "-z"); statusErr == nil && unmerged != "" {
					return nil
				}
			}
		}
	}
	return err
}

func msgEditor(exe, msgDir string) string {
	return "GIT_EDITOR=" + shq(exe) + " --rift-msg " + shq(msgDir)
}

// ContinueEditor is the GIT_EDITOR for an op --continue: rift's message queue during a
// rift-started rebase, else a no-op editor.
func ContinueEditor(dir string) string {
	if g, err := gitDir(dir); err == nil && RebaseInProgress(dir) {
		msgDir := filepath.Join(g, "rebase-merge", "rift-msg")
		if exe, err := os.Executable(); err == nil {
			for _, dir := range []string{msgDir, filepath.Join(g, "rift-rebase", "msg")} {
				if _, err := os.Stat(dir); err == nil {
					return msgEditor(exe, dir)
				}
			}
		}
	}
	return "GIT_EDITOR=true"
}

func RebaseInProgress(dir string) bool {
	out, err := Run(dir, "rev-parse", "--path-format=absolute", "--git-path", "rebase-merge")
	if err != nil {
		return false
	}
	_, err = os.Stat(strings.TrimSpace(out))
	return err == nil
}

// shq quotes for the sh that git runs editors through (git-bash on Windows too).
func shq(s string) string {
	return "'" + strings.ReplaceAll(filepath.ToSlash(s), "'", `'\''`) + "'"
}
