package git

import (
	"strconv"
	"strings"
)

type FileStatus struct {
	Path     string `json:"path"`
	OrigPath string `json:"origPath,omitempty"`
	Index    byte   `json:"index"`
	Work     byte   `json:"work"`
	Conflict bool   `json:"conflict"`
}

type Status struct {
	Head     string       `json:"head,omitempty"`
	Branch   string       `json:"branch"`
	Upstream string       `json:"upstream"`
	Ahead    int          `json:"ahead"`
	Behind   int          `json:"behind"`
	Files    []FileStatus `json:"files"`
}

func GetStatus(dir string) (Status, error) {
	out, err := RunEnv(dir, []string{"GIT_OPTIONAL_LOCKS=0"}, "status", "--porcelain=v2", "-z", "--branch", "--untracked-files=all")
	if err != nil {
		return Status{}, err
	}
	return parseStatus(out), nil
}

func parseStatus(out string) Status {
	s := Status{Files: []FileStatus{}}
	fields := strings.Split(out, "\x00")
	for i := 0; i < len(fields); i++ {
		f := fields[i]
		if f == "" {
			continue
		}
		switch f[0] {
		case '#':
			parts := strings.Fields(f)
			if len(parts) < 3 {
				continue
			}
			switch parts[1] {
			case "branch.oid":
				s.Head = parts[2]
			case "branch.head":
				s.Branch = parts[2]
			case "branch.upstream":
				s.Upstream = parts[2]
			case "branch.ab":
				if len(parts) == 4 {
					s.Ahead, _ = strconv.Atoi(strings.TrimPrefix(parts[2], "+"))
					s.Behind, _ = strconv.Atoi(strings.TrimPrefix(parts[3], "-"))
				}
			}
		case '1':
			p := strings.SplitN(f, " ", 9)
			s.Files = append(s.Files, FileStatus{Path: p[8], Index: p[1][0], Work: p[1][1]})
		case '2':
			p := strings.SplitN(f, " ", 10)
			i++
			s.Files = append(s.Files, FileStatus{Path: p[9], OrigPath: fields[i], Index: p[1][0], Work: p[1][1]})
		case 'u':
			p := strings.SplitN(f, " ", 11)
			s.Files = append(s.Files, FileStatus{Path: p[10], Index: p[1][0], Work: p[1][1], Conflict: true})
		case '?':
			s.Files = append(s.Files, FileStatus{Path: f[2:], Index: '?', Work: '?'})
		}
	}
	return s
}
