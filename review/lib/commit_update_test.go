package lint

import (
	"os"
	"regexp"
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

func TestGetUserIdentifier(t *testing.T) {
	// Save original USER env var
	originalUser := os.Getenv("USER")
	defer os.Setenv("USER", originalUser)

	// Test fallback to USER environment variable
	os.Setenv("USER", "testuser")
	userID, err := getUserIdentifier()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if userID != "testuser" {
		t.Errorf("Expected 'testuser', got '%s'", userID)
	}

	// Test with different user name
	os.Setenv("USER", "alice")
	userID, err = getUserIdentifier()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if userID != "alice" {
		t.Errorf("Expected 'alice', got '%s'", userID)
	}

	// Test with empty USER (should return error)
	os.Setenv("USER", "")
	userID, err = getUserIdentifier()
	if err == nil {
		t.Errorf("Expected error for empty USER, but got userID: '%s'", userID)
	}
	if userID != "" {
		t.Errorf("Expected empty userID on error, got '%s'", userID)
	}
}

func TestGenerateUUIDBranchName(t *testing.T) {
	// Save original USER env var
	originalUser := os.Getenv("USER")
	defer os.Setenv("USER", originalUser)

	// Set a known user for consistent testing
	os.Setenv("USER", "testuser")

	branch1, err := generateUUIDBranchName()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	// Verify it starts with the expected prefix
	if !strings.HasPrefix(branch1, "testuser/pr/") {
		t.Errorf("Expected branch name to start with 'testuser/pr/', got: %s", branch1)
	}

	// Verify it matches the expected UUID pattern
	expectedPattern := `^testuser/pr/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`
	matched, _ := regexp.MatchString(expectedPattern, branch1)
	if !matched {
		t.Errorf("Branch name doesn't match UUID pattern. Expected format: testuser/pr/XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX, got: %s", branch1)
	}
}

func TestGenerateUUIDBranchNameWithDifferentUsers(t *testing.T) {
	// Save original USER env var
	originalUser := os.Getenv("USER")
	defer os.Setenv("USER", originalUser)

	// Test with different users
	users := []string{"alice", "bob", "charlie"}

	for _, user := range users {
		os.Setenv("USER", user)
		branch, err := generateUUIDBranchName()
		if err != nil {
			t.Fatalf("Expected no error for user %s, got: %v", user, err)
		}

		expectedPrefix := user + "/pr/"
		if !strings.HasPrefix(branch, expectedPrefix) {
			t.Errorf("Expected branch to start with '%s', got: %s", expectedPrefix, branch)
		}

		// Verify the UUID part after the user prefix
		uuidPart := strings.TrimPrefix(branch, expectedPrefix)
		expectedPattern := `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`
		matched, err := regexp.MatchString(expectedPattern, uuidPart)
		if err != nil {
			t.Fatalf("Failed to compile regex: %v", err)
		}
		if !matched {
			t.Errorf("UUID part doesn't match expected pattern for user %s. Got: %s", user, uuidPart)
		}
	}
}

func TestGenerateUUIDBranchNameUniqueness(t *testing.T) {
	// Save original USER env var
	originalUser := os.Getenv("USER")
	defer os.Setenv("USER", originalUser)

	// Set a known user for testing
	os.Setenv("USER", "testuser")

	// Generate multiple UUIDs and verify they're unique
	generatedUUIDs := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		uuid, err := generateUUIDBranchName()
		if err != nil {
			t.Fatalf("Expected no error on iteration %d, got: %v", i, err)
		}
		if generatedUUIDs[uuid] {
			t.Fatalf("Generated duplicate UUID: %s", uuid)
		}
		generatedUUIDs[uuid] = true
	}
}

func TestGenerateUUIDBranchNameError(t *testing.T) {
	// Save original USER env var
	originalUser := os.Getenv("USER")
	defer os.Setenv("USER", originalUser)

	// Clear USER environment variable
	os.Setenv("USER", "")

	// Should return error when no user identifier is available
	branch, err := generateUUIDBranchName()
	if err == nil {
		t.Errorf("Expected error when no user identifier available, but got branch: %s", branch)
	}
	if branch != "" {
		t.Errorf("Expected empty branch name on error, got: %s", branch)
	}

	// Error message should be helpful
	if !strings.Contains(err.Error(), "no user identifier found") {
		t.Errorf("Expected helpful error message, got: %v", err)
	}
}