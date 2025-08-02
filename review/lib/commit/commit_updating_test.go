package commit

import (
	"strings"
	"testing"

	"github.com/jtamagnan/git-utils/git"
)

func TestUpdateCommitMessageWithPRURL(t *testing.T) {
	testRepo := git.NewTestRepo(t)
	defer testRepo.Cleanup()

	testRepo.InDir(func() {
		// Create initial commit on main
		testRepo.AddCommit("README.md", "# Initial commit", "Initial commit")

		// Create a feature branch
		testRepo.CreateBranch("feature")

		// Add multiple commits on feature branch
		testRepo.AddCommit("file1.txt", "content1", "Add authentication module")
		testRepo.AddCommit("file2.txt", "content2", "Fix login validation")
		testRepo.AddCommit("file3.txt", "content3", "Add user profile endpoint")

		testRepo.RefreshRepo()

		// Test updating the oldest commit with PR URL
		prURL := "https://github.com/owner/repo/pull/123"
		err := UpdateOldestCommitWithPRURL(testRepo.Repo, "main", prURL)
		if err != nil {
			t.Fatalf("Failed to update commit with PR URL: %v", err)
		}

		// Verify the oldest commit message was updated
		summaries := testRepo.Repo.RefSummaries("main")
		if len(summaries) != 3 {
			t.Fatalf("Expected 3 commits, got %d", len(summaries))
		}

		// Get the detailed commit message of the oldest commit
		commitHashes, err := testRepo.Repo.GitExec(
			"log",
			"main..HEAD",
			"--pretty=format:%H",
			"--reverse",
		)
		if err != nil {
			t.Fatalf("Failed to get commit hashes: %v", err)
		}

		oldestHash := strings.Split(strings.TrimSpace(commitHashes), "\n")[0]

		fullMessage, err := testRepo.Repo.GitExec("log", "-1", "--pretty=format:%B", oldestHash)
		if err != nil {
			t.Fatalf("Failed to get commit message: %v", err)
		}

		// Check that PR URL was added
		if !strings.Contains(fullMessage, "PR URL: "+prURL) {
			t.Errorf("PR URL not found in commit message. Got: %q", fullMessage)
		}

		// Check that original message is still there
		if !strings.Contains(fullMessage, "Add authentication module") {
			t.Errorf("Original commit message lost. Got: %q", fullMessage)
		}

		// Verify other commits weren't affected
		if summaries[1] != "Fix login validation" {
			t.Errorf("Second commit was affected: %q", summaries[1])
		}
		if summaries[2] != "Add user profile endpoint" {
			t.Errorf("Third commit was affected: %q", summaries[2])
		}
	})
}

func TestUpdateCommitMessageSingleCommit(t *testing.T) {
	testRepo := git.NewTestRepo(t)
	defer testRepo.Cleanup()

	testRepo.InDir(func() {
		// Create initial commit on main
		testRepo.AddCommit("README.md", "# Initial commit", "Initial commit")

		// Create a feature branch with single commit
		testRepo.CreateBranch("feature")
		testRepo.AddCommit("file1.txt", "content1", "Add authentication module")

		testRepo.RefreshRepo()

		// Test updating the single commit with PR URL
		prURL := "https://github.com/owner/repo/pull/456"
		err := UpdateOldestCommitWithPRURL(testRepo.Repo, "main", prURL)
		if err != nil {
			t.Fatalf("Failed to update single commit with PR URL: %v", err)
		}

		// Verify the commit message was updated
		summaries := testRepo.Repo.RefSummaries("main")
		if len(summaries) != 1 {
			t.Fatalf("Expected 1 commit, got %d", len(summaries))
		}

		// Get the detailed commit message
		fullMessage, err := testRepo.Repo.GitExec("log", "-1", "--pretty=format:%B")
		if err != nil {
			t.Fatalf("Failed to get commit message: %v", err)
		}

		// Check that PR URL was added
		if !strings.Contains(fullMessage, "PR URL: "+prURL) {
			t.Errorf("PR URL not found in commit message. Got: %q", fullMessage)
		}

		// Check that original message is still there
		if !strings.Contains(fullMessage, "Add authentication module") {
			t.Errorf("Original commit message lost. Got: %q", fullMessage)
		}
	})
}

func TestUpdateCommitMessageNoDuplicates(t *testing.T) {
	testRepo := git.NewTestRepo(t)
	defer testRepo.Cleanup()

	testRepo.InDir(func() {
		// Create initial commit on main
		testRepo.AddCommit("README.md", "# Initial commit", "Initial commit")

		// Create a feature branch
		testRepo.CreateBranch("feature")
		testRepo.AddCommit("file1.txt", "content1", "Add authentication module")

		testRepo.RefreshRepo()

		prURL := "https://github.com/owner/repo/pull/789"

		// Update once
		err := UpdateOldestCommitWithPRURL(testRepo.Repo, "main", prURL)
		if err != nil {
			t.Fatalf("Failed to update commit with PR URL: %v", err)
		}

		// Try to update again - should skip due to duplicate detection
		err = UpdateOldestCommitWithPRURL(testRepo.Repo, "main", prURL)
		if err != nil {
			t.Fatalf("Failed on second update attempt: %v", err)
		}

		// Get the commit message
		fullMessage, err := testRepo.Repo.GitExec("log", "-1", "--pretty=format:%B")
		if err != nil {
			t.Fatalf("Failed to get commit message: %v", err)
		}

		// Count occurrences of PR URL - should only be one
		count := strings.Count(fullMessage, "PR URL: "+prURL)
		if count != 1 {
			t.Errorf("Expected PR URL to appear once, found %d times in: %q", count, fullMessage)
		}
	})
}
