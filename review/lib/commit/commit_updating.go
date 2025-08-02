package commit

import (
	"fmt"
	"strings"

	"github.com/jtamagnan/git-utils/git"
)

// UpdateOldestCommitWithPRURL updates the oldest commit message to include the PR URL
func UpdateOldestCommitWithPRURL(repo *git.Repository, upstreamBranch, prURL string) error {
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
