package commit

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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

	// Remove any existing PR URL lines from the commit message
	prURLRegex := regexp.MustCompile(`(?m)^\s*PR URL:\s*https://github\.com/[^\s]+\s*$`)
	cleanedMessage := prURLRegex.ReplaceAllString(currentMessage, "")

	// Clean up any extra newlines left behind
	cleanedMessage = regexp.MustCompile(`\n\n+`).ReplaceAllString(cleanedMessage, "\n\n")
	cleanedMessage = strings.TrimSpace(cleanedMessage)

	// Add the new PR URL to the commit message
	updatedMessage := cleanedMessage + "\n\nPR URL: " + prURL

	// Check if we actually made a change
	if updatedMessage == currentMessage {
		fmt.Println("PR URL already up to date in commit message")
		return nil
	}

	if strings.Contains(currentMessage, "PR URL:") {
		fmt.Printf("Replacing existing PR URL with new one: %s\n", prURL)
	} else {
		fmt.Printf("Adding PR URL to commit message: %s\n", prURL)
	}

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

		// Multiple commits: use interactive rebase (much cleaner and preserves uncommitted changes)
	return updateCommitMessageWithRebase(repo, upstreamBranch, commitHash, newMessage)
}

// updateCommitMessageWithRebase uses git rebase -i to update a commit message
// This is much cleaner than reset+cherry-pick for the actual commit editing
func updateCommitMessageWithRebase(repo *git.Repository, upstreamBranch, commitHash, newMessage string) error {
	// Check if there are uncommitted changes that need to be preserved
	statusOutput, err := repo.GitExec("status", "--porcelain")
	if err != nil {
		return fmt.Errorf("error checking git status: %v", err)
	}

	hasUncommittedChanges := strings.TrimSpace(statusOutput) != ""

				if hasUncommittedChanges {
		// Preserve the exact state using index backup and temporary commits

		// Step 1: Save the current index state as a tree
		indexTreeHash, err := repo.GitExec("write-tree")
		if err != nil {
			return fmt.Errorf("error saving index state: %v", err)
		}
		indexTreeHash = strings.TrimSpace(indexTreeHash)

		// Step 2: Commit everything to get a clean working directory
		_, err = repo.GitExec("add", "-A")
		if err != nil {
			return fmt.Errorf("error adding all changes: %v", err)
		}

		_, err = repo.GitExec("commit", "-m", "TEMP: preserve all changes")
		if err != nil {
			return fmt.Errorf("error committing all changes: %v", err)
		}

				// Set up cleanup to restore the exact state using index restoration
		defer func() {
			// Step 1: Reset to the commit before our temporary commit, putting all changes in index
			if _, restoreErr := repo.GitExec("reset", "--soft", "HEAD~1"); restoreErr != nil {
				fmt.Printf("Warning: Failed to reset. Manual recovery: git reset --soft HEAD~1; git read-tree %s\n", indexTreeHash)
				return
			}

			// Step 2: Restore the original index state (this will automatically show correct working directory diff)
			if _, restoreErr := repo.GitExec("read-tree", indexTreeHash); restoreErr != nil {
				fmt.Printf("Warning: Failed to restore index. Manual recovery: git read-tree %s\n", indexTreeHash)
				return
			}
		}()
	}

	// Create a temporary directory for our rebase scripts
	tempDir, err := os.MkdirTemp("", "git-rebase-*")
	if err != nil {
		return fmt.Errorf("error creating temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a script that will modify the rebase todo list
	todoEditorScript := filepath.Join(tempDir, "rebase-todo-editor.sh")
	todoEditorContent := fmt.Sprintf(`#!/bin/bash
# Auto-generated script to mark specific commit for reword
sed -i.bak 's/^pick %s /reword %s /' "$1"
`, commitHash[:7], commitHash[:7]) // Use short hash (first 7 chars) as git does

	err = os.WriteFile(todoEditorScript, []byte(todoEditorContent), 0755)
	if err != nil {
		return fmt.Errorf("error creating todo editor script: %v", err)
	}

	// Create a script that will provide the commit message when git asks for it
	messageEditorScript := filepath.Join(tempDir, "commit-message-editor.sh")
	messageEditorContent := fmt.Sprintf(`#!/bin/bash
# Auto-generated script to provide the new commit message
cat > "$1" << 'EOF'
%s
EOF
`, newMessage)

	err = os.WriteFile(messageEditorScript, []byte(messageEditorContent), 0755)
	if err != nil {
		return fmt.Errorf("error creating message editor script: %v", err)
	}

	// Set up environment for the rebase
	originalGitEditor := os.Getenv("GIT_EDITOR")
	originalGitSequenceEditor := os.Getenv("GIT_SEQUENCE_EDITOR")

	// Set our custom editors
	os.Setenv("GIT_SEQUENCE_EDITOR", todoEditorScript)
	os.Setenv("GIT_EDITOR", messageEditorScript)

	// Restore original environment after rebase
	defer func() {
		if originalGitEditor != "" {
			os.Setenv("GIT_EDITOR", originalGitEditor)
		} else {
			os.Unsetenv("GIT_EDITOR")
		}
		if originalGitSequenceEditor != "" {
			os.Setenv("GIT_SEQUENCE_EDITOR", originalGitSequenceEditor)
		} else {
			os.Unsetenv("GIT_SEQUENCE_EDITOR")
		}
	}()

	// Run the interactive rebase
	_, err = repo.GitExec("rebase", "-i", upstreamBranch)
	if err != nil {
		// If rebase fails, try to abort it to leave things in a clean state
		repo.GitExec("rebase", "--abort")
		return fmt.Errorf("error during interactive rebase: %v", err)
	}

	return nil
}
