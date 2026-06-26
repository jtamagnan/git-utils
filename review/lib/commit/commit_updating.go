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

	fmt.Printf("DEBUG: found %d commits between %s and HEAD\n", len(commitHashes), upstreamBranch)
	for i, h := range commitHashes {
		fmt.Printf("DEBUG:   commit[%d]: %s\n", i, h)
	}

	// Get the oldest commit hash (first in the list)
	oldestCommitHash := commitHashes[0]
	fmt.Printf("DEBUG: oldest commit hash (target): %s\n", oldestCommitHash)

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

// CommitPRURL maps a commit hash to the PR URL to stamp on it
type CommitPRURL struct {
	Hash  string
	PRURL string
}

// rewordEntry holds the abbreviated hash and new message for a commit to reword
type rewordEntry struct {
	abbrevHash string
	newMessage string
}

// UpdateMultipleCommitsWithPRURLs stamps PR URLs on multiple commits in a single rebase pass
func UpdateMultipleCommitsWithPRURLs(repo *git.Repository, upstreamBranch string, updates []CommitPRURL) error {
	if len(updates) == 0 {
		return nil
	}

	// Build a map of hash -> new message for each commit that needs updating
	var commitsToUpdate []rewordEntry

	prURLRegex := regexp.MustCompile(`(?m)^\s*PR URL:\s*https://github\.com/[^\s]+\s*$`)
	cleanupNewlines := regexp.MustCompile(`\n\n+`)

	for _, u := range updates {
		currentMessage, err := repo.GitExec("log", "-1", "--pretty=format:%B", u.Hash)
		if err != nil {
			return fmt.Errorf("error getting commit message for %s: %v", u.Hash, err)
		}

		// Remove existing PR URL and add new one
		cleaned := prURLRegex.ReplaceAllString(currentMessage, "")
		cleaned = cleanupNewlines.ReplaceAllString(cleaned, "\n\n")
		cleaned = strings.TrimSpace(cleaned)
		newMessage := cleaned + "\n\nPR URL: " + u.PRURL

		if newMessage == strings.TrimSpace(currentMessage) {
			continue // already up to date
		}

		abbrev, err := repo.GitExec("rev-parse", "--short", u.Hash)
		if err != nil {
			return fmt.Errorf("error getting abbreviated hash for %s: %v", u.Hash, err)
		}
		commitsToUpdate = append(commitsToUpdate, rewordEntry{
			abbrevHash: strings.TrimSpace(abbrev),
			newMessage: newMessage,
		})
	}

	if len(commitsToUpdate) == 0 {
		fmt.Println("All PR URLs already up to date")
		return nil
	}

	// Single commit: use amend
	if len(updates) == 1 {
		countOut, err := repo.GitExec("rev-list", "--count", fmt.Sprintf("%s..HEAD", upstreamBranch))
		if err == nil && strings.TrimSpace(countOut) == "1" {
			_, err = repo.GitExec("commit", "--amend", "-m", commitsToUpdate[0].newMessage)
			if err != nil {
				return fmt.Errorf("error amending commit: %v", err)
			}
			return nil
		}
	}

	// Multiple commits: use interactive rebase
	return updateMultipleCommitMessagesWithRebase(repo, upstreamBranch, commitsToUpdate)
}

// updateMultipleCommitMessagesWithRebase rewords multiple commits in a single rebase pass
func updateMultipleCommitMessagesWithRebase(repo *git.Repository, upstreamBranch string, updates []rewordEntry) error {
	// Check for uncommitted changes
	statusOutput, err := repo.GitExec("status", "--porcelain")
	if err != nil {
		return fmt.Errorf("error checking git status: %v", err)
	}

	if strings.TrimSpace(statusOutput) != "" {
		indexTreeHash, err := repo.GitExec("write-tree")
		if err != nil {
			return fmt.Errorf("error saving index state: %v", err)
		}
		indexTreeHash = strings.TrimSpace(indexTreeHash)

		_, err = repo.GitExec("add", "-A")
		if err != nil {
			return fmt.Errorf("error adding all changes: %v", err)
		}

		_, err = repo.GitExec("commit", "-m", "TEMP: preserve all changes")
		if err != nil {
			return fmt.Errorf("error committing all changes: %v", err)
		}

		defer func() {
			if _, restoreErr := repo.GitExec("reset", "--soft", "HEAD~1"); restoreErr != nil {
				fmt.Printf("Warning: Failed to reset. Manual recovery: git reset --soft HEAD~1; git read-tree %s\n", indexTreeHash)
				return
			}
			if _, restoreErr := repo.GitExec("read-tree", indexTreeHash); restoreErr != nil {
				fmt.Printf("Warning: Failed to restore index. Manual recovery: git read-tree %s\n", indexTreeHash)
			}
		}()
	}

	tempDir, err := os.MkdirTemp("", "git-rebase-*")
	if err != nil {
		return fmt.Errorf("error creating temp directory: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	// Build sed commands to mark all target commits for reword
	var sedCommands []string
	for _, u := range updates {
		sedCommands = append(sedCommands, fmt.Sprintf("s/^pick %s /reword %s /", u.abbrevHash, u.abbrevHash))
	}

	todoEditorScript := filepath.Join(tempDir, "rebase-todo-editor.sh")
	todoEditorContent := fmt.Sprintf("#!/bin/bash\nsed -i.bak '%s' \"$1\"\n", strings.Join(sedCommands, ";"))
	if err := os.WriteFile(todoEditorScript, []byte(todoEditorContent), 0755); err != nil {
		return fmt.Errorf("error creating todo editor script: %v", err)
	}

	// Write each commit message to a numbered file, and create an editor script
	// that uses a counter to pick the right message file
	for i, u := range updates {
		msgFile := filepath.Join(tempDir, fmt.Sprintf("message-%d.txt", i))
		if err := os.WriteFile(msgFile, []byte(u.newMessage), 0644); err != nil {
			return fmt.Errorf("error creating message file: %v", err)
		}
	}

	counterFile := filepath.Join(tempDir, "counter")
	if err := os.WriteFile(counterFile, []byte("0"), 0644); err != nil {
		return fmt.Errorf("error creating counter file: %v", err)
	}

	messageEditorScript := filepath.Join(tempDir, "commit-message-editor.sh")
	messageEditorContent := fmt.Sprintf(`#!/bin/bash
COUNTER=$(cat "%s")
MSG_FILE="%s/message-${COUNTER}.txt"
if [ -f "$MSG_FILE" ]; then
    cp "$MSG_FILE" "$1"
fi
echo $((COUNTER + 1)) > "%s"
`, counterFile, tempDir, counterFile)
	if err := os.WriteFile(messageEditorScript, []byte(messageEditorContent), 0755); err != nil {
		return fmt.Errorf("error creating message editor script: %v", err)
	}

	// Set up environment
	originalGitEditor := os.Getenv("GIT_EDITOR")
	originalGitSequenceEditor := os.Getenv("GIT_SEQUENCE_EDITOR")
	_ = os.Setenv("GIT_SEQUENCE_EDITOR", todoEditorScript)
	_ = os.Setenv("GIT_EDITOR", messageEditorScript)
	defer func() {
		if originalGitEditor != "" {
			_ = os.Setenv("GIT_EDITOR", originalGitEditor)
		} else {
			_ = os.Unsetenv("GIT_EDITOR")
		}
		if originalGitSequenceEditor != "" {
			_ = os.Setenv("GIT_SEQUENCE_EDITOR", originalGitSequenceEditor)
		} else {
			_ = os.Unsetenv("GIT_SEQUENCE_EDITOR")
		}
	}()

	_, err = repo.GitExec("rebase", "-i", upstreamBranch)
	if err != nil {
		if _, abortErr := repo.GitExec("rebase", "--abort"); abortErr != nil {
			fmt.Printf("Warning: failed to abort rebase: %v\n", abortErr)
		}
		return fmt.Errorf("error during interactive rebase: %v", err)
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
	fmt.Printf("DEBUG: commit count: %s, strategy: ", commitCount)

	if commitCount == "1" {
		fmt.Println("amend")
		// Single commit: just amend it
		_, err = repo.GitExec("commit", "--amend", "-m", newMessage)
		if err != nil {
			return fmt.Errorf("error amending commit: %v", err)
		}
		return nil
	}

	// Multiple commits: use interactive rebase (much cleaner and preserves uncommitted changes)
	fmt.Println("rebase")
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
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			fmt.Printf("Warning: failed to cleanup temp directory: %v\n", err)
		}
	}()

	// Get the abbreviated hash that git will use in the rebase todo list.
	// Git's abbreviation length varies by repo size (7 in small repos, 10+ in large ones).
	abbrevHash, err := repo.GitExec("rev-parse", "--short", commitHash)
	if err != nil {
		return fmt.Errorf("error getting abbreviated hash for %s: %v", commitHash, err)
	}
	abbrevHash = strings.TrimSpace(abbrevHash)
	fmt.Printf("DEBUG: looking for abbreviated hash %q (full: %s) in rebase todo\n", abbrevHash, commitHash)

	// Create a script that will modify the rebase todo list
	todoEditorScript := filepath.Join(tempDir, "rebase-todo-editor.sh")
	todoEditorContent := fmt.Sprintf(`#!/bin/bash
# Auto-generated script to mark specific commit for reword
# Debug: show the todo list before modification
echo "DEBUG: rebase todo before:" >&2
cat "$1" >&2
sed -i.bak 's/^pick %s /reword %s /' "$1"
echo "DEBUG: rebase todo after:" >&2
cat "$1" >&2
`, abbrevHash, abbrevHash)

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
	_ = os.Setenv("GIT_SEQUENCE_EDITOR", todoEditorScript)
	_ = os.Setenv("GIT_EDITOR", messageEditorScript)

	// Restore original environment after rebase
	defer func() {
		if originalGitEditor != "" {
			_ = os.Setenv("GIT_EDITOR", originalGitEditor)
		} else {
			_ = os.Unsetenv("GIT_EDITOR")
		}
		if originalGitSequenceEditor != "" {
			_ = os.Setenv("GIT_SEQUENCE_EDITOR", originalGitSequenceEditor)
		} else {
			_ = os.Unsetenv("GIT_SEQUENCE_EDITOR")
		}
	}()

	// Run the interactive rebase
	_, err = repo.GitExec("rebase", "-i", upstreamBranch)
	if err != nil {
		// If rebase fails, try to abort it to leave things in a clean state
		if _, abortErr := repo.GitExec("rebase", "--abort"); abortErr != nil {
			fmt.Printf("Warning: failed to abort rebase: %v\n", abortErr)
		}
		return fmt.Errorf("error during interactive rebase: %v", err)
	}

	// Verify the commit message was actually updated by checking the oldest commit
	newHashes, verifyErr := repo.GitExec("log", fmt.Sprintf("%s..HEAD", upstreamBranch), "--pretty=format:%H", "--reverse")
	if verifyErr == nil {
		hashLines := strings.Split(strings.TrimSpace(newHashes), "\n")
		if len(hashLines) > 0 && hashLines[0] != "" {
			newMessage, verifyErr := repo.GitExec("log", "-1", "--pretty=format:%B", hashLines[0])
			if verifyErr == nil {
				fmt.Printf("DEBUG: oldest commit message after rebase: %q\n", strings.TrimSpace(newMessage))
			}
		}
	}

	return nil
}
