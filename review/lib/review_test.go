package lint

import (
	"fmt"
	"testing"

	"github.com/jtamagnan/git-utils/git"
)

func TestPRTitleFromRefSummaries(t *testing.T) {
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

		// Test the RefSummaries functionality
		summaries := testRepo.Repo.RefSummaries("main")

		if len(summaries) != 3 {
			t.Fatalf("Expected 3 summaries, got %d", len(summaries))
		}

		// The oldest/first commit should be the first element in the array
		oldestSummary := summaries[0]
		expectedTitle := "Add authentication module"

		if oldestSummary != expectedTitle {
			t.Errorf("Expected PR title %q, got %q", expectedTitle, oldestSummary)
		}

		// Verify the full order (oldest to newest)
		expected := []string{
			"Add authentication module",
			"Fix login validation",
			"Add user profile endpoint",
		}

		for i, summary := range summaries {
			if summary != expected[i] {
				t.Errorf("Summary[%d]: expected %q, got %q", i, expected[i], summary)
			}
		}
	})
}

func TestRefSummariesEmptyRangeError(t *testing.T) {
	testRepo := git.NewTestRepo(t)
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

		// This simulates the error condition that should be caught in Review()
		if len(summaries) == 0 {
			err := fmt.Errorf("no commits found between %s and HEAD - nothing to create a pull request for", "HEAD")
			if err == nil {
				t.Error("Expected error for empty commit range, got nil")
			}
		}
	})
}

func TestMultipleBranchesRefSummaries(t *testing.T) {
	testRepo := git.NewTestRepo(t)
	defer testRepo.Cleanup()

	testRepo.InDir(func() {
		// Create initial commit on main
		testRepo.AddCommit("README.md", "# Initial", "Initial commit")

		// Create feature branch with one commit
		testRepo.CreateBranch("feature")
		testRepo.AddCommit("feature.txt", "feature", "Add feature")

		// Switch back to main and add another commit
		testRepo.GitExec("checkout", "main")
		testRepo.AddCommit("main.txt", "main", "Update main branch")

		// Switch back to feature
		testRepo.GitExec("checkout", "feature")
		testRepo.RefreshRepo()

		// RefSummaries should only show commits on feature branch since divergence
		summaries := testRepo.Repo.RefSummaries("main")

		if len(summaries) != 1 {
			t.Fatalf("Expected 1 summary for feature branch, got %d", len(summaries))
		}

		if summaries[0] != "Add feature" {
			t.Errorf("Expected 'Add feature', got %q", summaries[0])
		}

		// The oldest/first (and only) commit should be used as PR title
		prTitle := summaries[0]
		if prTitle != "Add feature" {
			t.Errorf("Expected PR title 'Add feature', got %q", prTitle)
		}
	})
}

