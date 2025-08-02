package lint

import (
	"context"
	"fmt"
	"regexp"
	"strconv"

	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/google/go-github/v71/github"
	"github.com/jtamagnan/git-utils/git"
)

// getRemoteBranchFromPR gets the remote branch name from an existing PR
func getRemoteBranchFromPR(prNumber int) (string, error) {
	client := github.NewClient(nil)

	// Get the PR details
	pr, _, err := client.PullRequests.Get(context.Background(), "owner", "repo", prNumber)
	if err != nil {
		return "", fmt.Errorf("failed to get PR #%d: %v", prNumber, err)
	}

	// Return the head branch name (the branch the PR is coming from)
	if pr.Head != nil && pr.Head.Ref != nil {
		return *pr.Head.Ref, nil
	}

	return "", fmt.Errorf("PR #%d has no head branch information", prNumber)
}

// detectExistingPR checks all commit messages in the current branch for PR URLs
// Returns the PR number if found, or an error if no PR URL is detected
func detectExistingPR(repo *git.Repository, upstreamBranch string) (int, error) {
	// Use RefExec to collect PR numbers from all commits (0 if no PR found in that commit)
	prNumbers := git.RefExec(repo, func(commit *object.Commit) int {
		// Extract PR number from this commit's message
		return extractPRNumber(commit.Message)
	}, upstreamBranch)

	// Find the first non-zero PR number (oldest commit with PR)
	for _, prNumber := range prNumbers {
		if prNumber > 0 {
			return prNumber, nil
		}
	}

	return 0, fmt.Errorf("no existing PR URL found in any commit")
}

// extractPRNumber extracts the PR number from a commit message containing "PR URL: ..."
// Returns 0 if no valid PR URL is found
func extractPRNumber(message string) int {
	// Look for pattern: PR URL: https://github.com/owner/repo/pull/123
	re := regexp.MustCompile(`PR URL:\s*https://github\.com/[^/]+/[^/]+/pull/(\d+)`)
	matches := re.FindStringSubmatch(message)

	if len(matches) >= 2 {
		if prNumber, err := strconv.Atoi(matches[1]); err == nil {
			return prNumber
		}
	}

	return 0
}

// getExistingPR fetches an existing pull request by number
func getExistingPR(prNumber int) (*github.PullRequest, error) {
	client := github.NewClient(nil)

	// Get the PR
	pr, _, err := client.PullRequests.Get(context.Background(), "owner", "repo", prNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get PR #%d: %v", prNumber, err)
	}

	return pr, nil
}