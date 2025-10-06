package parent

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/jtamagnan/git-utils/git"
	githubapi "github.com/jtamagnan/git-utils/review/lib/github"
)

// ResolvedParent contains the resolved parent branch information
type ResolvedParent struct {
	// The git reference to use for comparisons (e.g., "origin/main", "origin/feature-branch")
	GitRef string
	// The GitHub base branch name to use when creating the PR (e.g., "main", "feature-branch")
	GitHubBase string
}

// ResolveParent resolves a parent specification into a ResolvedParent
// The parent can be:
// - Empty string: uses the upstream default branch
// - A PR number (e.g., "123"): resolves to the PR's head branch
// - A branch name (e.g., "feature/base"): resolves to remote/branch
// - A git reference (e.g., "origin/main", "HEAD~3"): uses as-is
func ResolveParent(repo *git.Repository, parentSpec, owner, repoName string) (*ResolvedParent, error) {
	upstream, err := repo.Remote()
	if err != nil {
		return nil, fmt.Errorf("failed to get remote: %w", err)
	}

	// If no parent specified, use upstream default branch
	if parentSpec == "" {
		defaultBranch, err := repo.GetDefaultBranch()
		if err != nil {
			return nil, fmt.Errorf("failed to get default branch: %w", err)
		}

		baseBranch := stripRemotePrefix(defaultBranch, upstream)
		return &ResolvedParent{
			GitRef:     defaultBranch,
			GitHubBase: baseBranch,
		}, nil
	}

	// Check if it's a PR number (pure digits)
	if isPRNumber(parentSpec) {
		prNumber, _ := strconv.Atoi(parentSpec)
		return resolveFromPR(owner, repoName, prNumber, upstream)
	}

	// Check if it's already a full git reference (contains a slash or special chars)
	if isGitReference(parentSpec) {
		return resolveFromGitRef(repo, parentSpec, upstream, owner, repoName)
	}

	// Assume it's a branch name, resolve to remote/branch
	return resolveFromBranchName(repo, parentSpec, upstream, owner, repoName)
}

// isPRNumber checks if the string is a valid PR number (positive integer)
func isPRNumber(s string) bool {
	matched, _ := regexp.MatchString(`^\d+$`, s)
	if !matched {
		return false
	}
	num, err := strconv.Atoi(s)
	return err == nil && num > 0
}

// isGitReference checks if the string looks like a git reference
// (contains slashes, tildes, carets, or other git ref syntax)
func isGitReference(s string) bool {
	return strings.Contains(s, "/") || strings.Contains(s, "~") || strings.Contains(s, "^")
}

// resolveFromPR resolves a parent from a PR number
func resolveFromPR(owner, repoName string, prNumber int, upstream string) (*ResolvedParent, error) {
	// Get the PR details from GitHub
	pr, err := githubapi.GetExistingPR(owner, repoName, prNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get PR #%d: %w", prNumber, err)
	}

	if pr.Head == nil || pr.Head.Ref == nil {
		return nil, fmt.Errorf("PR #%d has no head branch", prNumber)
	}

	branchName := *pr.Head.Ref
	gitRef := fmt.Sprintf("%s/%s", upstream, branchName)

	// The GitHub base for this PR should be the PR's base branch
	var githubBase string
	if pr.Base != nil && pr.Base.Ref != nil {
		githubBase = *pr.Base.Ref
	} else {
		// Fallback to the branch name if base is not available
		githubBase = branchName
	}

	return &ResolvedParent{
		GitRef:     gitRef,
		GitHubBase: githubBase,
	}, nil
}

// resolveFromBranchName resolves a parent from a branch name
func resolveFromBranchName(repo *git.Repository, branchName, upstream, owner, repoName string) (*ResolvedParent, error) {
	gitRef := fmt.Sprintf("%s/%s", upstream, branchName)

	// Verify the reference exists
	_, err := repo.GitExec("rev-parse", "--verify", gitRef)
	if err != nil {
		return nil, fmt.Errorf("branch %s does not exist on remote %s", branchName, upstream)
	}

	// Detect the GitHub base by finding where this branch diverges from the default branch
	githubBase, err := detectGitHubBase(repo, gitRef, upstream, owner, repoName)
	if err != nil {
		return nil, fmt.Errorf("failed to detect GitHub base for branch %s: %w", branchName, err)
	}

	return &ResolvedParent{
		GitRef:     gitRef,
		GitHubBase: githubBase,
	}, nil
}

// resolveFromGitRef resolves a parent from a git reference
func resolveFromGitRef(repo *git.Repository, gitRef, upstream, owner, repoName string) (*ResolvedParent, error) {
	// Verify the reference exists
	_, err := repo.GitExec("rev-parse", "--verify", gitRef)
	if err != nil {
		return nil, fmt.Errorf("git reference %s does not exist", gitRef)
	}

	// Detect the GitHub base by finding where this ref diverges from the default branch
	githubBase, err := detectGitHubBase(repo, gitRef, upstream, owner, repoName)
	if err != nil {
		return nil, fmt.Errorf("failed to detect GitHub base for ref %s: %w", gitRef, err)
	}

	return &ResolvedParent{
		GitRef:     gitRef,
		GitHubBase: githubBase,
	}, nil
}

// detectGitHubBase detects the GitHub base branch by finding where the parent branch
// diverges from the default branch
func detectGitHubBase(repo *git.Repository, parentRef, upstream, owner, repoName string) (string, error) {
	// Get the default branch
	defaultBranch, err := repo.GetDefaultBranch()
	if err != nil {
		return "", fmt.Errorf("failed to get default branch: %w", err)
	}

	// Check if the parent is the default branch
	parentCommit, err := repo.GitExec("rev-parse", parentRef)
	if err != nil {
		return "", err
	}

	defaultCommit, err := repo.GitExec("rev-parse", defaultBranch)
	if err != nil {
		return "", err
	}

	// If they point to the same commit or parent is ahead of default, base is default
	if parentCommit == defaultCommit {
		return stripRemotePrefix(defaultBranch, upstream), nil
	}

	// Check if parent is ahead of default (i.e., default is an ancestor of parent)
	_, err = repo.GitExec("merge-base", "--is-ancestor", defaultBranch, parentRef)
	if err == nil {
		// Default branch is an ancestor of parent, so base should be default
		return stripRemotePrefix(defaultBranch, upstream), nil
	}

	// Parent diverges from default, need to find the actual base
	// Try to find if the parent branch exists on GitHub
	if strings.HasPrefix(parentRef, upstream+"/") {
		branchName := strings.TrimPrefix(parentRef, upstream+"/")
		// Verify this is a real branch on GitHub by trying to fetch it
		_, err := repo.GitExec("ls-remote", "--heads", upstream, branchName)
		if err == nil {
			// It's a real remote branch, so we can use it as the base
			return branchName, nil
		}
	}

	// Fallback: if we can't determine a better base, use the default branch
	return stripRemotePrefix(defaultBranch, upstream), nil
}

// stripRemotePrefix removes the remote prefix from a branch name
func stripRemotePrefix(branch, remote string) string {
	prefix := remote + "/"
	return strings.TrimPrefix(branch, prefix)
}
