package git

import (
	"os/exec"
	"strings"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

type Repository struct {
    *gogit.Repository
}

type Reference struct {
	*plumbing.Reference
}

func GetRepository() (*Repository, error) {
	repo, err := gogit.PlainOpenWithOptions(
		".",
		&gogit.PlainOpenOptions{
			DetectDotGit: true,
		},
	)
	if err != nil { return nil, err }

	return &Repository{repo}, nil
}

func (repo *Repository) Head() (*Reference, error) {
	head, err := repo.Repository.Head()
	if err != nil { return nil, err }
	return &Reference{head}, nil
}

func (branch *Reference) TrackingBranch() (string, error) {
	// Return a tracking branch by running git for-each-ref --format='%(upstream)' main
	cmd := exec.Command(
		"git",
		"for-each-ref",
		"--format=%(upstream)",
		branch.Name().String(),
	)
	out, err := cmd.CombinedOutput()
	if err != nil { return "", err }

	return strings.TrimSpace(string(out)), nil


}

func (repo *Repository) WriteTree() (string, error) {
	// return the hash by running git write-tree
	cmd := exec.Command(
		"git",
		"write-tree",
	)
	out, err := cmd.CombinedOutput()
	if err != nil { return "", err }

	return strings.TrimSpace(string(out)), nil
}
