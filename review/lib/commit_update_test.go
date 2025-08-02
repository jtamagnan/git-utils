package lint

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
		err := updateOldestCommitWithPRURL(testRepo.Repo, "main", prURL)
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
		err := updateOldestCommitWithPRURL(testRepo.Repo, "main", prURL)
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
		err := updateOldestCommitWithPRURL(testRepo.Repo, "main", prURL)
		if err != nil {
			t.Fatalf("Failed to update commit with PR URL: %v", err)
		}

		// Try to update again - should skip due to duplicate detection
		err = updateOldestCommitWithPRURL(testRepo.Repo, "main", prURL)
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

func TestDetectExistingPR(t *testing.T) {
	testRepo := git.NewTestRepo(t)
	defer testRepo.Cleanup()

	testRepo.InDir(func() {
		// Create initial commit on main
		testRepo.AddCommit("README.md", "# Initial commit", "Initial commit")

		// Create a feature branch
		testRepo.CreateBranch("feature")

		// Add commits without PR URL
		testRepo.AddCommit("file1.txt", "content1", "Add authentication module")
		testRepo.AddCommit("file2.txt", "content2", "Fix login validation")

		testRepo.RefreshRepo()

		// Should find no existing PR (expect error)
		prNumber, err := detectExistingPR(testRepo.Repo, "main")
		if err == nil {
			t.Fatalf("Expected error when no PR found, but got prNumber: %d", prNumber)
		}
		if prNumber != 0 {
			t.Errorf("Expected prNumber to be 0 when error occurs, got %d", prNumber)
		}

		// Add a commit with PR URL
		testRepo.AddCommit("file3.txt", "content3", "Add user profile endpoint\n\nPR URL: https://github.com/owner/repo/pull/123")

		testRepo.RefreshRepo()

		// Should now find the existing PR
		prNumber, err = detectExistingPR(testRepo.Repo, "main")
		if err != nil {
			t.Fatalf("Failed to detect existing PR: %v", err)
		}
		if prNumber != 123 {
			t.Errorf("Expected PR #123, got %d", prNumber)
		}
	})
}

func TestDetectExistingPRInOldestCommit(t *testing.T) {
	testRepo := git.NewTestRepo(t)
	defer testRepo.Cleanup()

	testRepo.InDir(func() {
		// Create initial commit on main
		testRepo.AddCommit("README.md", "# Initial commit", "Initial commit")

		// Create a feature branch with PR URL in oldest commit
		testRepo.CreateBranch("feature")
		testRepo.AddCommit("file1.txt", "content1", "Add authentication module\n\nPR URL: https://github.com/owner/repo/pull/456")
		testRepo.AddCommit("file2.txt", "content2", "Fix login validation")
		testRepo.AddCommit("file3.txt", "content3", "Add user profile endpoint")

		testRepo.RefreshRepo()

		// Should find the PR from the oldest commit
		prNumber, err := detectExistingPR(testRepo.Repo, "main")
		if err != nil {
			t.Fatalf("Failed to detect existing PR: %v", err)
		}
		if prNumber != 456 {
			t.Errorf("Expected PR #456, got %d", prNumber)
		}
	})
}

func TestDetectExistingPRNoCommits(t *testing.T) {
	testRepo := git.NewTestRepo(t)
	defer testRepo.Cleanup()

	testRepo.InDir(func() {
		// Create initial commit on main
		testRepo.AddCommit("README.md", "# Initial commit", "Initial commit")

		// Create a feature branch but don't add any commits
		testRepo.CreateBranch("feature")

		testRepo.RefreshRepo()

		// Should find no existing PR when there are no commits (expect error)
		prNumber, err := detectExistingPR(testRepo.Repo, "main")
		if err == nil {
			t.Fatalf("Expected error when no commits found, but got prNumber: %d", prNumber)
		}
		if prNumber != 0 {
			t.Errorf("Expected prNumber to be 0 when error occurs, got %d", prNumber)
		}
	})
}

func TestExtractPRNumber(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected int
	}{
		{
			name:     "Valid PR URL",
			message:  "Add feature\n\nPR URL: https://github.com/owner/repo/pull/123",
			expected: 123,
		},
		{
			name:     "Valid PR URL with extra spaces",
			message:  "Add feature\n\nPR URL:   https://github.com/owner/repo/pull/456   ",
			expected: 456,
		},
		{
			name:     "Valid PR URL different org/repo",
			message:  "Fix bug\n\nPR URL: https://github.com/myorg/myrepo/pull/789",
			expected: 789,
		},
		{
			name:     "No PR URL",
			message:  "Add feature\n\nSome other content",
			expected: 0,
		},
		{
			name:     "Invalid URL format",
			message:  "Add feature\n\nPR URL: https://github.com/owner/repo/issues/123",
			expected: 0,
		},
		{
			name:     "Invalid domain",
			message:  "Add feature\n\nPR URL: https://gitlab.com/owner/repo/pull/123",
			expected: 0,
		},
		{
			name:     "Multiple PR URLs (should find first)",
			message:  "Add feature\n\nPR URL: https://github.com/owner/repo/pull/111\nPR URL: https://github.com/owner/repo/pull/222",
			expected: 111,
		},
		{
			name:     "Empty message",
			message:  "",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractPRNumber(tt.message)
			if result != tt.expected {
				t.Errorf("extractPRNumber(%q) = %d, expected %d", tt.message, result, tt.expected)
			}
		})
	}
}