package git

import (
	"regexp"
	"fmt"
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

	repository *Repository
}

//
// Repository functions
//
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

func (repo *Repository) Remote() (string, error) {
	branch, err := repo.Head()
	if err != nil { return "", err }

	trackingBranch, err := branch.TrackingBranch()
	if err != nil { return "", err }

	return regexp.MustCompile(`refs/remotes/(.*)/.*`).FindStringSubmatch(trackingBranch)[1], nil
}

//
// Get branch information
//
func (repo *Repository) Head() (*Reference, error) {
	head, err := repo.Repository.Head()
	if err != nil { return nil, err }
	return &Reference{head, repo}, nil
}

// This will be of the form refs/heads/<remove>/<branch>
func (branch *Reference) TrackingBranch() (string, error) {
	return branch.repository.ExecGit(
		"for-each-ref",
		"--format=%(upstream)",
		branch.Name().String(),
	)
}

func (repo *Repository) GetDefaultBranch() (string, error) {
	upstream, err := repo.Remote()
	if err != nil { return "", err }

	return repo.ExecGit(
		"rev-parse",
		"--abbrev-ref",
		fmt.Sprintf("%s/HEAD", upstream),
	)
}

func (repo *Repository) WriteTree() (string, error) {
	return repo.ExecGit(
		"write-tree",
	)
}

//
// Get Config
//
func (repo *Repository) GetConfig(key string) (string, error) {
	return repo.ExecGit(
		"config",
		"--get",
		key,
	)
}

func GetConfig(key string) (string, error) {
	repo, err := GetRepository()
	if err != nil { return "", err }
	return repo.GetConfig(key)
}

//
// Exec on a repository
//
func (repo *Repository) ExecGit(args ...string) (string, error) {
	workTree, err := repo.Worktree()
	cmd := exec.Command("git", args...)
	cmd.Dir = workTree.Filesystem.Root()
	out, err := cmd.CombinedOutput()
	if err != nil { return "", fmt.Errorf("Error running git command: `%s` \n %s", cmd.String(), out) }

	return strings.TrimSpace(string(out)), nil
}


// TODO(jat): Function to get all of "x" from a list of commits
// between HEAD and the upstream. We'll want to use this when getting
// the PR url as well as the "global" PR description
