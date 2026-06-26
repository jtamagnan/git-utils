package pr

import (
	"testing"

	"github.com/jtamagnan/git-utils/git"
)

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
		prNumber, err := DetectExistingPR(testRepo.Repo, "main")
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
		prNumber, err = DetectExistingPR(testRepo.Repo, "main")
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
		prNumber, err := DetectExistingPR(testRepo.Repo, "main")
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
		prNumber, err := DetectExistingPR(testRepo.Repo, "main")
		if err == nil {
			t.Fatalf("Expected error when no commits found, but got prNumber: %d", prNumber)
		}
		if prNumber != 0 {
			t.Errorf("Expected prNumber to be 0 when error occurs, got %d", prNumber)
		}
	})
}

func TestDetectAllPRs(t *testing.T) {
	testRepo := git.NewTestRepo(t)
	defer testRepo.Cleanup()

	testRepo.InDir(func() {
		testRepo.AddCommit("README.md", "# Init", "Initial commit")
		testRepo.CreateBranch("feature")
		testRepo.AddCommit("a.txt", "a", "First commit\n\nPR URL: https://github.com/owner/repo/pull/10")
		testRepo.AddCommit("b.txt", "b", "Second commit")
		testRepo.AddCommit("c.txt", "c", "Third commit\n\nPR URL: https://github.com/owner/repo/pull/20")
		testRepo.RefreshRepo()

		results, err := DetectAllPRs(testRepo.Repo, "main")
		if err != nil {
			t.Fatalf("DetectAllPRs failed: %v", err)
		}

		if len(results) != 3 {
			t.Fatalf("Expected 3 commits, got %d", len(results))
		}

		// First commit has PR #10
		if results[0].PRNum != 10 {
			t.Errorf("Expected commit 0 PRNum=10, got %d", results[0].PRNum)
		}
		if results[0].PRURL != "https://github.com/owner/repo/pull/10" {
			t.Errorf("Expected commit 0 PRURL, got %q", results[0].PRURL)
		}
		if results[0].Summary != "First commit" {
			t.Errorf("Expected summary 'First commit', got %q", results[0].Summary)
		}

		// Second commit has no PR
		if results[1].PRNum != 0 {
			t.Errorf("Expected commit 1 PRNum=0, got %d", results[1].PRNum)
		}
		if results[1].PRURL != "" {
			t.Errorf("Expected commit 1 empty PRURL, got %q", results[1].PRURL)
		}

		// Third commit has PR #20
		if results[2].PRNum != 20 {
			t.Errorf("Expected commit 2 PRNum=20, got %d", results[2].PRNum)
		}
	})
}

func TestExtractPRURLAndNumber(t *testing.T) {
	tests := []struct {
		name        string
		message     string
		expectedURL string
		expectedNum int
	}{
		{
			name:        "Valid PR URL",
			message:     "Fix bug\n\nPR URL: https://github.com/owner/repo/pull/42",
			expectedURL: "https://github.com/owner/repo/pull/42",
			expectedNum: 42,
		},
		{
			name:        "No PR URL",
			message:     "Just a commit",
			expectedURL: "",
			expectedNum: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, num := extractPRURLAndNumber(tt.message)
			if url != tt.expectedURL {
				t.Errorf("URL: got %q, want %q", url, tt.expectedURL)
			}
			if num != tt.expectedNum {
				t.Errorf("Num: got %d, want %d", num, tt.expectedNum)
			}
		})
	}
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
