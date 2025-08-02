package git

import (
	"strings"
	"testing"
)

func TestGetRepository(t *testing.T) {
	testRepo := NewTestRepo(t)
	defer testRepo.Cleanup()

	testRepo.InDir(func() {
		repo, err := GetRepository()
		if err != nil {
			t.Fatalf("GetRepository failed: %v", err)
		}
		if repo == nil {
			t.Fatal("GetRepository returned nil repository")
		}
	})
}

func TestRefSummaries(t *testing.T) {
	testRepo := NewTestRepo(t)
	defer testRepo.Cleanup()

	testRepo.InDir(func() {
		// Create initial commit on main
		testRepo.AddCommit("README.md", "# Initial commit", "Initial commit")
		
		// Create a feature branch
		testRepo.CreateBranch("feature")
		
		// Add multiple commits on feature branch
		testRepo.AddCommit("file1.txt", "content1", "First feature commit")
		testRepo.AddCommit("file2.txt", "content2", "Second feature commit")
		testRepo.AddCommit("file3.txt", "content3", "Third feature commit")
		
		testRepo.RefreshRepo()
		
		// Test RefSummaries from main to HEAD (feature branch)
		summaries := testRepo.Repo.RefSummaries("main")
		
		if len(summaries) != 3 {
			t.Fatalf("Expected 3 summaries, got %d", len(summaries))
		}
		
		// Verify the order (oldest to newest)
		expected := []string{
			"First feature commit",
			"Second feature commit", 
			"Third feature commit",
		}
		
		for i, summary := range summaries {
			if summary != expected[i] {
				t.Errorf("Summary[%d]: expected %q, got %q", i, expected[i], summary)
			}
		}
		
		// Test the oldest/first commit (first in array)
		oldestSummary := summaries[0]
		if oldestSummary != "First feature commit" {
			t.Errorf("Oldest summary: expected %q, got %q", "First feature commit", oldestSummary)
		}
	})
}

func TestRefSummariesEmptyRange(t *testing.T) {
	testRepo := NewTestRepo(t)
	defer testRepo.Cleanup()

	testRepo.InDir(func() {
		// Create initial commit
		testRepo.AddCommit("README.md", "# Initial commit", "Initial commit")
		testRepo.RefreshRepo()
		
		// Test RefSummaries when HEAD equals the parent (no commits between)
		summaries := testRepo.Repo.RefSummaries("HEAD")
		
		if len(summaries) != 0 {
			t.Fatalf("Expected 0 summaries for equal refs, got %d", len(summaries))
		}
	})
}

func TestGitExec(t *testing.T) {
	testRepo := NewTestRepo(t)
	defer testRepo.Cleanup()

	testRepo.InDir(func() {
		// Test a simple git command
		output, err := testRepo.Repo.GitExec("status", "--porcelain")
		if err != nil {
			t.Fatalf("GitExec failed: %v", err)
		}
		
		// Should be empty for a clean repo
		if strings.TrimSpace(output) != "" {
			t.Errorf("Expected clean status, got: %q", output)
		}
		
		// Create a file and test status again
		testRepo.CreateFile("test.txt", "test content")
		output, err = testRepo.Repo.GitExec("status", "--porcelain")
		if err != nil {
			t.Fatalf("GitExec failed: %v", err)
		}
		
		if !strings.Contains(output, "test.txt") {
			t.Errorf("Expected test.txt in status output, got: %q", output)
		}
	})
}

func TestGetConfig(t *testing.T) {
	testRepo := NewTestRepo(t)
	defer testRepo.Cleanup()

	testRepo.InDir(func() {
		// Test getting a config value that was set during repo creation
		email, err := testRepo.Repo.GetConfig("user.email")
		if err != nil {
			t.Fatalf("GetConfig failed: %v", err)
		}
		
		if email != "test@example.com" {
			t.Errorf("Expected user.email to be test@example.com, got: %q", email)
		}
		
		// Test global GetConfig function
		globalEmail, err := GetConfig("user.email")
		if err != nil {
			t.Fatalf("Global GetConfig failed: %v", err)
		}
		
		if globalEmail != email {
			t.Errorf("Global GetConfig returned different value: %q vs %q", globalEmail, email)
		}
	})
}

func TestWriteTree(t *testing.T) {
	testRepo := NewTestRepo(t)
	defer testRepo.Cleanup()

	testRepo.InDir(func() {
		// Create and stage a file
		testRepo.CreateFile("test.txt", "test content")
		testRepo.GitExec("add", "test.txt")
		
		// Test WriteTree
		treeHash, err := testRepo.Repo.WriteTree()
		if err != nil {
			t.Fatalf("WriteTree failed: %v", err)
		}
		
		// Should be a valid git hash (40 characters)
		if len(treeHash) != 40 {
			t.Errorf("Expected 40-character hash, got %d characters: %q", len(treeHash), treeHash)
		}
	})
}

func TestHeadAndReference(t *testing.T) {
	testRepo := NewTestRepo(t)
	defer testRepo.Cleanup()

	testRepo.InDir(func() {
		// Create initial commit so HEAD points to something
		testRepo.AddCommit("README.md", "# Test", "Initial commit")
		testRepo.RefreshRepo()
		
		// Test Head()
		head, err := testRepo.Repo.Head()
		if err != nil {
			t.Fatalf("Head failed: %v", err)
		}
		
		if head == nil {
			t.Fatal("Head returned nil reference")
		}
		
		// Test the reference name
		headName := head.Name().String()
		if headName != "refs/heads/main" {
			t.Errorf("Expected refs/heads/main, got: %q", headName)
		}
	})
}

func TestRemoteAndTracking(t *testing.T) {
	testRepo := NewTestRepo(t)
	defer testRepo.Cleanup()

	testRepo.InDir(func() {
		// Create initial commit
		testRepo.AddCommit("README.md", "# Test", "Initial commit")
		
		// Add remote and set up tracking
		testRepo.AddRemote("origin", "https://github.com/test/repo.git")
		testRepo.CreateRemoteTrackingBranch("origin", "main")
		testRepo.SetUpstream("origin", "main")
		
		testRepo.RefreshRepo()
		
		// Test Remote()
		remote, err := testRepo.Repo.Remote()
		if err != nil {
			t.Fatalf("Remote failed: %v", err)
		}
		
		if remote != "origin" {
			t.Errorf("Expected remote 'origin', got: %q", remote)
		}
		
		// Test TrackingBranch()
		head, err := testRepo.Repo.Head()
		if err != nil {
			t.Fatalf("Head failed: %v", err)
		}
		
		tracking, err := head.TrackingBranch()
		if err != nil {
			t.Fatalf("TrackingBranch failed: %v", err)
		}
		
		if tracking != "refs/remotes/origin/main" {
			t.Errorf("Expected refs/remotes/origin/main, got: %q", tracking)
		}
	})
}

func TestGetDefaultBranch(t *testing.T) {
	testRepo := NewTestRepo(t)
	defer testRepo.Cleanup()

	testRepo.InDir(func() {
		// Create initial commit
		testRepo.AddCommit("README.md", "# Test", "Initial commit")
		
		// Add remote and set up tracking
		testRepo.AddRemote("origin", "https://github.com/test/repo.git")
		testRepo.CreateRemoteTrackingBranch("origin", "main")
		testRepo.SetUpstream("origin", "main")
		
		// Set origin/HEAD to point to main
		testRepo.GitExec("remote", "set-head", "origin", "main")
		
		testRepo.RefreshRepo()
		
		// Test GetDefaultBranch()
		defaultBranch, err := testRepo.Repo.GetDefaultBranch()
		if err != nil {
			t.Fatalf("GetDefaultBranch failed: %v", err)
		}
		
		// Should return just the branch name, not the full ref
		if !strings.Contains(defaultBranch, "main") {
			t.Errorf("Expected default branch to contain 'main', got: %q", defaultBranch)
		}
	})
}

func TestRefExec(t *testing.T) {
	testRepo := NewTestRepo(t)
	defer testRepo.Cleanup()

	testRepo.InDir(func() {
		// Create initial commit on main
		testRepo.AddCommit("README.md", "# Initial commit", "Initial commit")
		
		// Create a feature branch with commits
		testRepo.CreateBranch("feature")
		testRepo.AddCommit("file1.txt", "content1", "First commit")
		testRepo.AddCommit("file2.txt", "content2", "Second commit")
		
		testRepo.RefreshRepo()
		
		// Test RefExec
		callCount := 0
		results := testRepo.Repo.RefExec(func() {
			callCount++
		}, "main")
		
		// Should have 2 results for 2 commits
		if len(results) != 2 {
			t.Fatalf("Expected 2 results, got %d", len(results))
		}
		
		// The inner function should be called once per commit
		if callCount != 2 {
			t.Errorf("Expected inner function called 2 times, got %d", callCount)
		}
		
		// Results should be commit objects
		for i, result := range results {
			if result == nil {
				t.Errorf("Result[%d] is nil", i)
			}
		}
	})
} 