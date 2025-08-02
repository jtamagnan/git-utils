package pr

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/jtamagnan/git-utils/git"
)



// DetectExistingPR checks all commit messages in the current branch for PR URLs
// Returns the PR number if found, or an error if no PR URL is detected
func DetectExistingPR(repo *git.Repository, upstreamBranch string) (int, error) {
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
