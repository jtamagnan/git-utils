package git

import (
	"regexp"
	"fmt"
	"os/exec"
	"strings"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
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
	return branch.repository.GitExec(
		"for-each-ref",
		"--format=%(upstream)",
		branch.Name().String(),
	)
}

func (repo *Repository) GetDefaultBranch() (string, error) {
	upstream, err := repo.Remote()
	if err != nil { return "", err }

	return repo.GitExec(
		"rev-parse",
		"--abbrev-ref",
		fmt.Sprintf("%s/HEAD", upstream),
	)
}

func (repo *Repository) WriteTree() (string, error) {
	return repo.GitExec(
		"write-tree",
	)
}

//
// Get Config
//
func (repo *Repository) GetConfig(key string) (string, error) {
	return repo.GitExec(
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
func (repo *Repository) GitExec(args ...string) (string, error) {
	workTree, err := repo.Worktree()
	cmd := exec.Command("git", args...)
	cmd.Dir = workTree.Filesystem.Root()
	out, err := cmd.CombinedOutput()
	if err != nil { return "", fmt.Errorf("Error running git command: `%s` \n %s", cmd.String(), out) }

	return strings.TrimSpace(string(out)), nil
}

func RefExec[T any](repo *Repository, inner func(*object.Commit) T, parent string) []T {
	out, err := repo.GitExec(
		"log",
		fmt.Sprintf("%s..HEAD", parent),
		"--pretty=format:%H",
		"--reverse",
	)
	if err != nil {
		fmt.Printf("Error getting ref exec: %v\n", err)
		return nil
	}
	lines := strings.Split(out, "\n")
	var results []T
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			hash := plumbing.NewHash(line)
			commit, err := repo.Repository.CommitObject(hash)
			if err != nil {
				fmt.Printf("Error getting commit: %v\n", err)
				continue
			}
			result := inner(commit)
			results = append(results, result)
		}
	}
	return results
}

func (repo *Repository) RefSummaries(parent string) []string {
	out, err := repo.GitExec(
		"log",
		fmt.Sprintf("%s..HEAD", parent),
		"--pretty=format:%s",
		"--reverse",
	)
	if err != nil {
		fmt.Printf("Error getting summary on refs: %v\n", err)
		return nil
	}
	lines := strings.Split(out, "\n")
	var summaries []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			summaries = append(summaries, line)
		}
	}
	return summaries
}
