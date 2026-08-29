package commit

import (
	"fmt"
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

	// Remove any existing PR URL lines from the commit message (with or without a URL)
	prURLRegex := regexp.MustCompile(`(?m)^\s*PR URL:\s*(https://github\.com/[^\s]+)?\s*$`)
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

	prURLRegex := regexp.MustCompile(`(?m)^\s*PR URL:\s*(https://github\.com/[^\s]+)?\s*$`)
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

	// Multiple commits: rewrite directly with commit-tree (avoids slow rebase on large repos)
	return rewriteCommitMessages(repo, upstreamBranch, commitsToUpdate)
}

// rewriteCommitMessages rewrites commit messages using git commit-tree.
// This is much faster than interactive rebase on large repos because it
// only touches the commits in the range, not the entire repo history.
func rewriteCommitMessages(repo *git.Repository, upstreamBranch string, updates []rewordEntry) error {
	// Build a map of abbreviated hash -> new message
	newMessages := make(map[string]string)
	for _, u := range updates {
		newMessages[u.abbrevHash] = u.newMessage
	}

	// Get all commits in oldest-to-newest order
	out, err := repo.GitExec("log", fmt.Sprintf("%s..HEAD", upstreamBranch), "--pretty=format:%H", "--reverse")
	if err != nil {
		return fmt.Errorf("error listing commits: %v", err)
	}

	var commitHashes []string
	for _, line := range strings.Split(out, "\n") {
		if h := strings.TrimSpace(line); h != "" {
			commitHashes = append(commitHashes, h)
		}
	}

	if len(commitHashes) == 0 {
		return fmt.Errorf("no commits found between %s and HEAD", upstreamBranch)
	}

	// Walk commits oldest-to-newest, rebuilding the chain with commit-tree
	// For each commit, if it needs a new message, use that; otherwise keep the original.
	// The parent of the first commit is the upstream branch tip.
	parentHash := ""
	upstreamHash, err := repo.GitExec("rev-parse", upstreamBranch)
	if err != nil {
		return fmt.Errorf("error resolving %s: %v", upstreamBranch, err)
	}
	parentHash = strings.TrimSpace(upstreamHash)

	var lastNewHash string
	for _, hash := range commitHashes {
		// Get the tree for this commit
		tree, err := repo.GitExec("rev-parse", hash+"^{tree}")
		if err != nil {
			return fmt.Errorf("error getting tree for %s: %v", hash, err)
		}
		tree = strings.TrimSpace(tree)

		// Check if this commit needs a new message
		abbrev, err := repo.GitExec("rev-parse", "--short", hash)
		if err != nil {
			return fmt.Errorf("error getting short hash for %s: %v", hash, err)
		}
		abbrev = strings.TrimSpace(abbrev)

		var message string
		if newMsg, ok := newMessages[abbrev]; ok {
			message = newMsg
		} else {
			// Keep original message
			message, err = repo.GitExec("log", "-1", "--pretty=format:%B", hash)
			if err != nil {
				return fmt.Errorf("error getting message for %s: %v", hash, err)
			}
		}

		// Create new commit with commit-tree
		newHash, err := repo.GitExec("commit-tree", tree, "-p", parentHash, "-m", message)
		if err != nil {
			return fmt.Errorf("error creating commit for %s: %v", hash, err)
		}
		lastNewHash = strings.TrimSpace(newHash)
		parentHash = lastNewHash
	}

	// Update HEAD to point at the new chain
	_, err = repo.GitExec("update-ref", "HEAD", lastNewHash)
	if err != nil {
		return fmt.Errorf("error updating HEAD: %v", err)
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

	// Multiple commits: rewrite with commit-tree
	fmt.Println("commit-tree")
	abbrev, err := repo.GitExec("rev-parse", "--short", commitHash)
	if err != nil {
		return fmt.Errorf("error getting abbreviated hash: %v", err)
	}
	return rewriteCommitMessages(repo, upstreamBranch, []rewordEntry{
		{abbrevHash: strings.TrimSpace(abbrev), newMessage: newMessage},
	})
}
