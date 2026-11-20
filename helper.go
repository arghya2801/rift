package main

import (
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"rift/internal/git"
	"strings"
)

// runHelper handles the editor modes git invokes during an interactive rebase.
func runHelper() bool {
	if len(os.Args) < 4 {
		return false
	}
	target := os.Args[len(os.Args)-1]
	switch os.Args[1] {
	case "--rift-seq":
		if b, err := os.ReadFile(os.Args[2]); err == nil {
			if err := os.WriteFile(target, b, 0o600); err != nil {
				os.Exit(1)
			}
			if err := os.Remove(os.Args[2]); err != nil {
				os.Exit(1)
			}
		} else {
			os.Exit(1)
		}
		return true
	case "--rift-msg":
		if err := rewordMessage(os.Args[2], target); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return true
	}
	return false
}

func rewordMessage(dir, target string) error {
	path, err := git.Run(".", "rev-parse", "--path-format=absolute", "--git-path", "rebase-merge/done")
	if err != nil {
		return err
	}
	done, err := os.ReadFile(strings.TrimSpace(path))
	if err != nil {
		return err
	}
	lines := strings.Split(strings.TrimSpace(string(done)), "\n")
	fields := strings.Fields(lines[len(lines)-1])
	if len(fields) < 2 || (fields[0] != "reword" && fields[0] != "r") {
		return nil
	}
	hash, err := git.Run(".", "rev-parse", "--verify", "--end-of-options", fields[1]+"^{commit}")
	if err != nil {
		return err
	}
	id := strings.TrimSpace(hash)
	if _, err := hex.DecodeString(id); err != nil {
		return err
	}
	name := filepath.Join(dir, id+".msg")
	msg, err := os.ReadFile(name)
	if errors.Is(err, os.ErrNotExist) {
		// A rebase begun by an earlier Rift version queued messages by original subject.
		if filepath.Base(dir) != "msg" || filepath.Base(filepath.Dir(dir)) != "rift-rebase" {
			return nil
		}
		subject, err := git.Run(".", "show", "-s", "--format=%s", id)
		if err != nil {
			return err
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if !strings.HasSuffix(entry.Name(), ".subject") {
				continue
			}
			queued := filepath.Join(dir, entry.Name())
			old, err := os.ReadFile(queued)
			if err != nil {
				return err
			}
			if strings.TrimSpace(string(old)) != strings.TrimSpace(subject) {
				continue
			}
			name = strings.TrimSuffix(queued, ".subject") + ".msg"
			msg, err = os.ReadFile(name)
			if err != nil {
				return err
			}
			if err = os.WriteFile(target, append(msg, '\n'), 0o600); err != nil {
				return err
			}
			if err = os.Remove(queued); err != nil {
				return err
			}
			return os.Remove(name)
		}
		return nil
	}
	if err != nil {
		return err
	}
	if err := os.WriteFile(target, append(msg, '\n'), 0o600); err != nil {
		return err
	}
	return os.Remove(name)
}
