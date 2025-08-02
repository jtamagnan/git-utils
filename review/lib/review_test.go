package review

import (
	"testing"

	"github.com/jtamagnan/git-utils/git"
)

func TestPRTitleFromRefSummaries(t *testing.T) {
	testRepo := git.NewTestRepo(t)
	defer testRepo.Cleanup()

	testRepo.InDir(func() {
		// Create initial commit on main
		testRepo.AddCommit("README.md", "# Initial commit", "Initial commit")

		// Create a feature branch and add commits
		testRepo.CreateBranch("feature")
		testRepo.AddCommit("file1.txt", "content1", "Add authentication module")
		testRepo.AddCommit("file2.txt", "content2", "Fix login validation")
		testRepo.AddCommit("file3.txt", "content3", "Add user profile endpoint")

		testRepo.RefreshRepo()

		// Test that RefSummaries returns commits from oldest to newest
		summaries := testRepo.Repo.RefSummaries("main")

		expectedSummaries := []string{
			"Add authentication module",
			"Fix login validation",
			"Add user profile endpoint",
		}

		if len(summaries) != len(expectedSummaries) {
			t.Fatalf("Expected %d summaries, got %d", len(expectedSummaries), len(summaries))
		}

		for i, expected := range expectedSummaries {
			if summaries[i] != expected {
				t.Errorf("Expected summary[%d] = %q, got %q", i, expected, summaries[i])
			}
		}

		// The PR title should be the first (oldest) commit summary
		prTitle := summaries[0]
		expectedTitle := "Add authentication module"

		if prTitle != expectedTitle {
			t.Errorf("Expected PR title %q, got %q", expectedTitle, prTitle)
		}
	})
}

func TestRefSummariesEmptyRangeError(t *testing.T) {
	testRepo := git.NewTestRepo(t)
	defer testRepo.Cleanup()

	testRepo.InDir(func() {
		// Create initial commit on main
		testRepo.AddCommit("README.md", "# Initial commit", "Initial commit")

		// Create a feature branch but don't add any commits
		testRepo.CreateBranch("feature")

		testRepo.RefreshRepo()

		// RefSummaries should return empty slice when no commits between branches
		summaries := testRepo.Repo.RefSummaries("main")

		if len(summaries) != 0 {
			t.Errorf("Expected empty summaries for branch with no commits, got %d summaries", len(summaries))
		}
	})
}

func TestMultipleBranchesRefSummaries(t *testing.T) {
	testRepo := git.NewTestRepo(t)
	defer testRepo.Cleanup()

	testRepo.InDir(func() {
		// Create initial commit on main
		testRepo.AddCommit("README.md", "# Initial commit", "Initial commit")

		// Create first feature branch
		testRepo.CreateBranch("feature1")
		testRepo.AddCommit("file1.txt", "content1", "Feature 1 commit")

		// Switch back to main and create second feature branch
		testRepo.SwitchBranch("main")
		testRepo.CreateBranch("feature2")
		testRepo.AddCommit("file2.txt", "content2", "Feature 2 first commit")
		testRepo.AddCommit("file3.txt", "content3", "Feature 2 second commit")

		testRepo.RefreshRepo()

		// Test feature1 summaries
		feature1Summaries := testRepo.Repo.RefSummaries("main")
		expectedFeature1 := []string{"Feature 2 first commit", "Feature 2 second commit"}

		if len(feature1Summaries) != len(expectedFeature1) {
			t.Errorf("Expected %d summaries for feature2, got %d", len(expectedFeature1), len(feature1Summaries))
		}

		for i, expected := range expectedFeature1 {
			if feature1Summaries[i] != expected {
				t.Errorf("Expected feature2 summary[%d] = %q, got %q", i, expected, feature1Summaries[i])
			}
		}
	})
}
