package git

import (
	"errors"
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
func Log(dir string, limit int, extra ...string) ([]Row, error) {
	commits, err := logCommits(dir, limit, 0, extra...)
	if err != nil {
		return nil, err
	}
	return Graph(commits), nil
}

func logCommits(dir string, limit, skip int, extra ...string) ([]Commit, error) {
	args := append([]string{"log", "--topo-order", "-z", "-n", strconv.Itoa(limit),
		"--format=%H%x1f%P%x1f%an%x1f%ae%x1f%at%x1f%D%x1f%s"}, extra...)
	if len(extra) == 0 {
		args = append(args, "--branches", "--remotes", "--tags", "HEAD")
	}
	if skip > 0 {
		args = append(args, "--skip="+strconv.Itoa(skip))
	}
	out, err := Run(dir, args...)
	if err != nil {
		return nil, err
	}
	return parseLog(out), nil
}

type Page struct {
	Rows    []Row    `json:"rows"`
	Lanes   []string `json:"lanes"`
	More    bool     `json:"more"`
	Restart bool     `json:"restart"`
}

// Include the previous page's last commit to detect a changed traversal between requests.
func LogPage(dir string, offset, limit int, anchor string, lanes []string) (Page, error) {
	if offset < 0 || limit <= 0 || limit > 5000 {
		return Page{}, errors.New("invalid log page")
	}
	skip, count := offset, limit+1
	if offset > 0 {
		skip--
		count++
	}
	commits, err := logCommits(dir, count, skip)
	if err != nil {
		return Page{}, err
	}
	if offset > 0 {
		if len(commits) == 0 || commits[0].Hash != anchor {
			return Page{Restart: true}, nil
		}
		commits = commits[1:]
	}
	more := len(commits) > limit
	rows, next := graphPage(commits[:min(len(commits), limit)], slices.Clone(lanes))
	return Page{Rows: rows, Lanes: next, More: more}, nil
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
	rows, _ := graphPage(commits, nil)
	return rows
}

func graphPage(commits []Commit, lanes []string) ([]Row, []string) {
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
	var bottom []int
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
			for len(bottom) <= target {
				bottom = append(bottom, 0)
			}
			bottom[target] = n + 1
			lanes[target] = p
			lines = append(lines, Line{From: col, To: target, Bottom: true})
		}
		for i, h := range lanes {
			if h != "" && (i >= len(bottom) || bottom[i] != n+1) {
				lines = append(lines, Line{From: i, To: i, Bottom: true})
			}
		}
		for len(lanes) > 0 && lanes[len(lanes)-1] == "" {
			lanes = lanes[:len(lanes)-1]
		}
		rows[n] = Row{Commit: c, Col: col, Lines: lines}
	}
	return rows, lanes
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

func CommitSides(dir, hash, path, oldPath string) Sides {
	if oldPath == "" {
		oldPath = path
	}
	b := blobs(dir, hash+"^:"+oldPath, hash+":"+path)
	return sides(b[0], b[1])
}

func CommitMessage(dir, hash string) string {
	out, _ := Run(dir, "log", "-1", "--format=%B", hash)
	return strings.TrimSpace(out)
}
