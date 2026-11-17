package main

import (
	"os"
	"path/filepath"
	"sort"
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
			_ = os.WriteFile(target, b, 0o600)
		} else {
			os.Exit(1)
		}
		return true
	case "--rift-msg":
		rewordMessage(os.Args[2], target)
		return true
	}
	return false
}

// rewordMessage replaces target with the queued message whose original subject matches the
// message git opened; anything else (conflicted pick, squash group) keeps git's text.
func rewordMessage(dir, target string) {
	cur, err := os.ReadFile(target)
	if err != nil {
		return
	}
	subject := ""
	for l := range strings.Lines(string(cur)) {
		if l = strings.TrimSpace(l); l != "" && !strings.HasPrefix(l, "#") {
			subject = l
			break
		}
	}
	subs, _ := filepath.Glob(filepath.Join(dir, "*.subject"))
	sort.Strings(subs)
	for _, s := range subs {
		b, _ := os.ReadFile(s)
		if strings.TrimSpace(string(b)) != subject {
			continue
		}
		base := strings.TrimSuffix(s, ".subject")
		if msg, err := os.ReadFile(base + ".msg"); err == nil {
			_ = os.WriteFile(target, append(msg, '\n'), 0o600)
		}
		os.Remove(s)
		os.Remove(base + ".msg")
		return
	}
}
