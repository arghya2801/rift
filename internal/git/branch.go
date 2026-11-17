package git

import "strings"

type Ref struct {
	Name     string `json:"name"`
	Full     string `json:"full"`
	Kind     string `json:"kind"` // local, remote, tag
	Hash     string `json:"hash"`
	Upstream string `json:"upstream"`
	Track    string `json:"track"`
	Current  bool   `json:"current"`
}

func Refs(dir string) ([]Ref, error) {
	out, err := Run(dir, "for-each-ref", "--format=%(refname)%1f%(refname:short)%1f%(objectname)%1f%(upstream:short)%1f%(upstream:track)%1f%(HEAD)",
		"refs/heads", "refs/remotes", "refs/tags")
	if err != nil {
		return nil, err
	}
	refs := []Ref{}
	for line := range strings.Lines(out) {
		f := strings.Split(strings.TrimRight(line, "\n"), "\x1f")
		if len(f) < 6 || strings.HasSuffix(f[0], "/HEAD") {
			continue
		}
		r := Ref{Full: f[0], Name: f[1], Hash: f[2], Upstream: f[3], Track: f[4], Current: f[5] == "*"}
		switch {
		case strings.HasPrefix(f[0], "refs/heads/"):
			r.Kind = "local"
		case strings.HasPrefix(f[0], "refs/remotes/"):
			r.Kind = "remote"
		default:
			r.Kind = "tag"
		}
		refs = append(refs, r)
	}
	return refs, nil
}

func Checkout(dir, ref string) error {
	_, err := Run(dir, "switch", "--no-guess", ref)
	if err != nil && strings.Contains(err.Error(), "a branch is expected") {
		_, err = Run(dir, "switch", "--detach", ref)
	}
	return err
}

// CheckoutRemote creates a local tracking branch for a remote branch like origin/feat.
func CheckoutRemote(dir, remoteRef string) error {
	_, err := Run(dir, "switch", "--track", remoteRef)
	return err
}

func CreateBranch(dir, name, at string, checkout bool) error {
	if checkout {
		_, err := Run(dir, "switch", "-c", name, at)
		return err
	}
	_, err := Run(dir, "branch", name, at)
	return err
}

func DeleteBranch(dir, name string, force bool) error {
	flag := "-d"
	if force {
		flag = "-D"
	}
	_, err := Run(dir, "branch", flag, name)
	return err
}

func RenameBranch(dir, old, name string) error {
	_, err := Run(dir, "branch", "-m", old, name)
	return err
}

func Merge(dir, ref string) error {
	_, err := Run(dir, "merge", "--no-edit", ref)
	return err
}

func Rebase(dir, onto string) error {
	_, err := Run(dir, "rebase", "--autostash", onto)
	return err
}

func CherryPick(dir, hash string) error {
	_, err := Run(dir, "cherry-pick", hash)
	return err
}

func Revert(dir, hash string) error {
	_, err := Run(dir, "revert", "--no-edit", hash)
	return err
}

// Reset mode is soft, mixed or hard.
func Reset(dir, mode, hash string) error {
	_, err := Run(dir, "reset", "--"+mode, hash)
	return err
}

func CreateTag(dir, name, hash string) error {
	_, err := Run(dir, "tag", name, hash)
	return err
}
