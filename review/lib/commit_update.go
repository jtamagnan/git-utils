package lint

import (
	"context"
	"crypto/rand"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/google/go-github/v71/github"
	"github.com/jtamagnan/git-utils/git"
)

// generateUUIDBranchName creates a UUID-based branch name for new PRs
func generateUUIDBranchName() string {
	// Generate a simple UUID-like string for branch naming
	bytes := make([]byte, 16)
	rand.Read(bytes)

	// Format as UUID: 8-4-4-4-12 characters
	return fmt.Sprintf("pr-%x-%x-%x-%x-%x",
		bytes[0:4], bytes[4:6], bytes[6:8], bytes[8:10], bytes[10:16])
}

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

// updateOldestCommitWithPRURL updates the oldest commit message to include the PR URL
func updateOldestCommitWithPRURL(repo *git.Repository, upstreamBranch, prURL string) error {
	// Get all commit summaries to find the oldest one
	summaries := repo.RefSummaries(upstreamBranch)
	if len(summaries) == 0 {
		return fmt.Errorf("no commits found to update")
	}

	// Get the commit hashes in oldest-to-newest order
	out, err := repo.GitExec(
		"log",
		fmt.Sprintf("%s..HEAD", upstreamBranch),
		"--pretty=format:%H",
		"--reverse",
	)
	if err != nil {
		return fmt.Errorf("error getting commit hashes: %v", err)
	}

	lines := strings.Split(out, "\n")
	var commitHashes []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			commitHashes = append(commitHashes, strings.TrimSpace(line))
		}
	}

	if len(commitHashes) == 0 {
		return fmt.Errorf("no commit hashes found")
	}

	// Get the oldest commit hash (first in the list)
	oldestCommitHash := commitHashes[0]

	// Get the current commit message
	currentMessage, err := repo.GitExec("log", "-1", "--pretty=format:%B", oldestCommitHash)
	if err != nil {
		return fmt.Errorf("error getting current commit message: %v", err)
	}

	// Check if PR URL is already in the message to avoid duplicates
	if strings.Contains(currentMessage, "PR URL:") {
		fmt.Println("PR URL already present in commit message, skipping update")
		return nil
	}

	// Add PR URL to the commit message
	updatedMessage := strings.TrimSpace(currentMessage) + "\n\nPR URL: " + prURL

	// Update the commit message
	err = updateCommitMessage(repo, upstreamBranch, oldestCommitHash, updatedMessage)
	if err != nil {
		return fmt.Errorf("error updating commit message: %v", err)
	}

	return nil
}

// updateCommitMessage updates a specific commit's message using the simplest reliable approach
func updateCommitMessage(repo *git.Repository, upstreamBranch, commitHash, newMessage string) error {
	// Count commits to determine strategy
	countOut, err := repo.GitExec("rev-list", "--count", fmt.Sprintf("%s..HEAD", upstreamBranch))
	if err != nil {
		return fmt.Errorf("error counting commits: %v", err)
	}

	commitCount := strings.TrimSpace(countOut)

	if commitCount == "1" {
		// Single commit: just amend it
		_, err = repo.GitExec("commit", "--amend", "-m", newMessage)
		if err != nil {
			return fmt.Errorf("error amending commit: %v", err)
		}
		return nil
	}

	// Multiple commits: use reset and recommit approach
	// Get all commit info we need to replay
	commitsInfo, err := repo.GitExec(
		"log",
		fmt.Sprintf("%s..HEAD", upstreamBranch),
		"--pretty=format:%H|%s|%B",
		"--reverse",
	)
	if err != nil {
		return fmt.Errorf("error getting commit info: %v", err)
	}

	// Parse commit information
	type commitInfo struct {
		hash    string
		subject string
		body    string
	}

	var commits []commitInfo
	for _, line := range strings.Split(strings.TrimSpace(commitsInfo), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 3)
		if len(parts) >= 3 {
			commits = append(commits, commitInfo{
				hash:    parts[0],
				subject: parts[1],
				body:    parts[2],
			})
		}
	}

	if len(commits) == 0 {
		return fmt.Errorf("no commits found to process")
	}

	// Reset to the base branch
	_, err = repo.GitExec("reset", "--hard", upstreamBranch)
	if err != nil {
		return fmt.Errorf("error resetting to base: %v", err)
	}

	// Replay each commit
	for i, commit := range commits {
		if commit.hash == commitHash {
			// This is the commit we want to update - use new message
			_, err = repo.GitExec("cherry-pick", "--no-commit", commit.hash)
			if err != nil {
				return fmt.Errorf("error cherry-picking target commit: %v", err)
			}
			_, err = repo.GitExec("commit", "-m", newMessage)
			if err != nil {
				return fmt.Errorf("error committing with new message: %v", err)
			}
		} else {
			// Regular commit - preserve original message
			_, err = repo.GitExec("cherry-pick", commit.hash)
			if err != nil {
				return fmt.Errorf("error cherry-picking commit %d: %v", i, err)
			}
		}
	}

	return nil
}
