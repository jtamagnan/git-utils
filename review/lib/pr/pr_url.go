package pr

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

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

// StackCommitPR holds the PR info extracted from a single commit
type StackCommitPR struct {
	Hash    string
	Summary string
	PRURL   string // empty if no PR URL found
	PRNum   int    // 0 if no PR URL found
}

// DetectAllPRs returns per-commit PR info for all commits between parent and HEAD (oldest first)
func DetectAllPRs(repo *git.Repository, upstreamBranch string) ([]StackCommitPR, error) {
	results := git.RefExec(repo, func(commit *object.Commit) StackCommitPR {
		url, num := extractPRURLAndNumber(commit.Message)
		// Use first line of message as summary
		summary := commit.Message
		if idx := strings.Index(summary, "\n"); idx != -1 {
			summary = summary[:idx]
		}
		return StackCommitPR{
			Hash:    commit.Hash.String(),
			Summary: summary,
			PRURL:   url,
			PRNum:   num,
		}
	}, upstreamBranch)

	if len(results) == 0 {
		return nil, fmt.Errorf("no commits found between %s and HEAD", upstreamBranch)
	}

	return results, nil
}

// extractPRURLAndNumber extracts both the full PR URL and number from a commit message
func extractPRURLAndNumber(message string) (string, int) {
	re := regexp.MustCompile(`PR URL:\s*(https://github\.com/[^/]+/[^/]+/pull/(\d+))`)
	matches := re.FindStringSubmatch(message)

	if len(matches) >= 3 {
		if prNumber, err := strconv.Atoi(matches[2]); err == nil {
			return matches[1], prNumber
		}
	}

	return "", 0
}

// extractPRNumber extracts the PR number from a commit message containing "PR URL: ..."
// Returns 0 if no valid PR URL is found
func extractPRNumber(message string) int {
	_, num := extractPRURLAndNumber(message)
	return num
}
