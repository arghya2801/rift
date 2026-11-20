package git

import (
	"slices"
	"strconv"
	"strings"
)

type Commit struct {
	Hash    string   `json:"hash"`
	Parents []string `json:"parents"`
	Author  string   `json:"author"`
	Email   string   `json:"email"`
	Time    int64    `json:"time"`
	Refs    []string `json:"refs"`
	Subject string   `json:"subject"`
}

// Line is one graph segment in a row: lane From to lane To, in the top or bottom half.
type Line struct {
	From   int  `json:"from"`
	To     int  `json:"to"`
	Bottom bool `json:"bottom"`
}

type Row struct {
	Commit
	Col   int    `json:"col"`
	Lines []Line `json:"lines"`
}

// Log returns up to limit commits from all branches, tags and HEAD with graph lanes.
// ponytail: whole graph computed per call; page/cache it when repos with 100k+ commits feel slow.
func Log(dir string, limit int, extra ...string) ([]Row, error) {
	args := append([]string{"log", "--topo-order", "-z", "-n", strconv.Itoa(limit),
		"--format=%H%x1f%P%x1f%an%x1f%ae%x1f%at%x1f%D%x1f%s"}, extra...)
	if len(extra) == 0 {
		args = append(args, "--branches", "--remotes", "--tags", "HEAD")
	}
	out, err := Run(dir, args...)
	if err != nil && len(extra) == 0 {
		if _, headErr := Run(dir, "rev-parse", "--verify", "--quiet", "HEAD"); headErr != nil {
			out, err = Run(dir, args[:len(args)-1]...)
		}
	}
	if err != nil {
		return nil, err
	}
	return Graph(parseLog(out)), nil
}

func parseLog(out string) []Commit {
	var list []Commit
	for rec := range strings.SplitSeq(out, "\x00") {
		f := strings.Split(strings.TrimPrefix(rec, "\n"), "\x1f")
		if len(f) < 7 {
			continue
		}
		t, _ := strconv.ParseInt(f[4], 10, 64)
		c := Commit{Hash: f[0], Parents: strings.Fields(f[1]), Author: f[2], Email: f[3], Time: t, Subject: f[6], Refs: []string{}}
		for r := range strings.SplitSeq(f[5], ", ") {
			if r != "" {
				c.Refs = append(c.Refs, r)
			}
		}
		list = append(list, c)
	}
	return list
}

// Graph assigns each commit a lane. lanes[i] holds the hash lane i is waiting for.
func Graph(commits []Commit) []Row {
	var lanes []string
	rows := make([]Row, len(commits))
	free := func() int {
		for i, h := range lanes {
			if h == "" {
				return i
			}
		}
		lanes = append(lanes, "")
		return len(lanes) - 1
	}
	for n, c := range commits {
		col := -1
		lines := []Line{}
		for i, h := range lanes {
			if h == "" {
				continue
			}
			if h == c.Hash {
				if col < 0 {
					col = i
				}
				lines = append(lines, Line{From: i, To: col})
			} else {
				lines = append(lines, Line{From: i, To: i})
			}
		}
		for i := range lanes {
			if lanes[i] == c.Hash {
				lanes[i] = ""
			}
		}
		if col < 0 {
			col = free()
		}
		for k, p := range c.Parents {
			target := -1
			if k == 0 {
				target = col
			} else {
				target = slices.Index(lanes, p)
			}
			if target < 0 {
				target = free()
			}
			lanes[target] = p
			lines = append(lines, Line{From: col, To: target, Bottom: true})
		}
		for i, h := range lanes {
			if h != "" && !hasBottom(lines, i) {
				lines = append(lines, Line{From: i, To: i, Bottom: true})
			}
		}
		for len(lanes) > 0 && lanes[len(lanes)-1] == "" {
			lanes = lanes[:len(lanes)-1]
		}
		rows[n] = Row{Commit: c, Col: col, Lines: lines}
	}
	return rows
}

type ChangedFile struct {
	Status  string `json:"status"`
	Path    string `json:"path"`
	OldPath string `json:"oldPath,omitempty"`
}

// CommitFiles lists files changed by hash against its first parent.
func CommitFiles(dir, hash string) ([]ChangedFile, error) {
	out, err := Run(dir, "diff-tree", "-r", "-z", "-M", "--name-status", "--root", "--no-commit-id", "--diff-merges=first-parent", hash)
	if err != nil {
		return nil, err
	}
	f := strings.Split(strings.TrimSuffix(out, "\x00"), "\x00")
	files := []ChangedFile{}
	for i := 0; i+1 < len(f); i += 2 {
		c := ChangedFile{Status: f[i][:1], Path: f[i+1]}
		if c.Status == "R" || c.Status == "C" {
			c.OldPath, c.Path = f[i+1], f[i+2]
			i++
		}
		files = append(files, c)
	}
	return files, nil
}

func CommitSides(dir, hash, path, oldPath string) (Sides, error) {
	if oldPath == "" {
		oldPath = path
	}
	return blobSides(dir, hash+"^:"+oldPath, hash+":"+path)
}

func CommitMessage(dir, hash string) string {
	out, _ := Run(dir, "log", "-1", "--format=%B", hash)
	return strings.TrimSpace(out)
}

func hasBottom(lines []Line, to int) bool {
	for _, l := range lines {
		if l.Bottom && l.To == to {
			return true
		}
	}
	return false
}
