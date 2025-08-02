package commit

import (
	"os"
	"path/filepath"
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

func TestUncommittedChangesPreservation(t *testing.T) {
	testRepo := git.NewTestRepo(t)
	defer testRepo.Cleanup()

	testRepo.InDir(func() {
		// Create initial commit on main
		testRepo.AddCommit("README.md", "# Initial commit", "Initial commit")

		// Create a feature branch
		testRepo.CreateBranch("feature")

		// Add multiple commits on feature branch to trigger the reset+cherry-pick path
		testRepo.AddCommit("file1.txt", "content1", "Add authentication module")
		testRepo.AddCommit("file2.txt", "content2", "Fix login validation")
		testRepo.AddCommit("file3.txt", "content3", "Add user profile endpoint")

		testRepo.RefreshRepo()

				// Create uncommitted changes of different types

		// 1. Modified but unstaged file
		testRepo.CreateFile("file1.txt", "modified content1")

		// 2. New untracked file
		testRepo.CreateFile("untracked.txt", "untracked content")

		// 3. Staged but uncommitted changes
		testRepo.CreateFile("staged.txt", "staged content")
		_, err := testRepo.Repo.GitExec("add", "staged.txt")
		if err != nil {
			t.Fatalf("Failed to stage file: %v", err)
		}

		// Verify the uncommitted changes exist before the operation

				// Check modified file
		modifiedContentBytes, err := os.ReadFile(filepath.Join(testRepo.Dir, "file1.txt"))
		if err != nil {
			t.Fatalf("Failed to read modified file: %v", err)
		}
		modifiedContent := string(modifiedContentBytes)
		if modifiedContent != "modified content1" {
			t.Fatalf("Expected modified content, got: %s", modifiedContent)
		}

		// Check untracked file
		untrackedContentBytes, err := os.ReadFile(filepath.Join(testRepo.Dir, "untracked.txt"))
		if err != nil {
			t.Fatalf("Failed to read untracked file: %v", err)
		}
		untrackedContent := string(untrackedContentBytes)
		if untrackedContent != "untracked content" {
			t.Fatalf("Expected untracked content, got: %s", untrackedContent)
		}

		// Check staged file
		stagedContentBytes, err := os.ReadFile(filepath.Join(testRepo.Dir, "staged.txt"))
		if err != nil {
			t.Fatalf("Failed to read staged file: %v", err)
		}
		stagedContent := string(stagedContentBytes)
		if stagedContent != "staged content" {
			t.Fatalf("Expected staged content, got: %s", stagedContent)
		}

		// Verify git status shows our uncommitted changes
		statusOutput, err := testRepo.Repo.GitExec("status", "--porcelain")
		if err != nil {
			t.Fatalf("Failed to get git status: %v", err)
		}
		t.Logf("Git status before update:\n%s", statusOutput)

		// Now test updating the commit message (this should preserve uncommitted changes)
		prURL := "https://github.com/owner/repo/pull/123"
		err = UpdateOldestCommitWithPRURL(testRepo.Repo, "main", prURL)
		if err != nil {
			t.Fatalf("Failed to update commit with PR URL: %v", err)
		}

		// Verify all uncommitted changes are still there

				// 1. Check modified file is still modified
		modifiedContentAfterBytes, err := os.ReadFile(filepath.Join(testRepo.Dir, "file1.txt"))
		if err != nil {
			t.Fatalf("Failed to read modified file after update: %v", err)
		}
		modifiedContentAfter := string(modifiedContentAfterBytes)
		if modifiedContentAfter != "modified content1" {
			t.Errorf("Modified file lost changes! Expected 'modified content1', got: %s", modifiedContentAfter)
		}

		// 2. Check untracked file still exists
		untrackedContentAfterBytes, err := os.ReadFile(filepath.Join(testRepo.Dir, "untracked.txt"))
		if err != nil {
			t.Errorf("Untracked file was lost after commit update: %v", err)
		} else {
			untrackedContentAfter := string(untrackedContentAfterBytes)
			if untrackedContentAfter != "untracked content" {
				t.Errorf("Untracked file content changed! Expected 'untracked content', got: %s", untrackedContentAfter)
			}
		}

		// 3. Check staged file is still staged
		stagedContentAfterBytes, err := os.ReadFile(filepath.Join(testRepo.Dir, "staged.txt"))
		if err != nil {
			t.Errorf("Staged file was lost after commit update: %v", err)
		} else {
			stagedContentAfter := string(stagedContentAfterBytes)
			if stagedContentAfter != "staged content" {
				t.Errorf("Staged file content changed! Expected 'staged content', got: %s", stagedContentAfter)
			}
		}

		// Verify git status still shows our uncommitted changes
		statusOutputAfter, err := testRepo.Repo.GitExec("status", "--porcelain")
		if err != nil {
			t.Fatalf("Failed to get git status after update: %v", err)
		}
		t.Logf("Git status after update:\n%s", statusOutputAfter)

		// The status should still show our changes
		if !strings.Contains(statusOutputAfter, "file1.txt") {
			t.Errorf("Modified file1.txt no longer shows as modified in git status")
		}
		if !strings.Contains(statusOutputAfter, "untracked.txt") {
			t.Errorf("Untracked file untracked.txt no longer shows in git status")
		}
		if !strings.Contains(statusOutputAfter, "staged.txt") {
			t.Errorf("Staged file staged.txt no longer shows as staged in git status")
		}

		t.Log("✅ All uncommitted changes preserved after commit message update")
	})
}

func TestStagedChangesPreservation(t *testing.T) {
	testRepo := git.NewTestRepo(t)
	defer testRepo.Cleanup()

	testRepo.InDir(func() {
		// Create initial commit on main
		testRepo.AddCommit("README.md", "# Initial commit", "Initial commit")

		// Create a feature branch
		testRepo.CreateBranch("feature")

		// Add multiple commits on feature branch to trigger the rebase path
		testRepo.AddCommit("file1.txt", "original content", "Add file1")
		testRepo.AddCommit("file2.txt", "original content", "Add file2")
		testRepo.AddCommit("file3.txt", "original content", "Add file3")

		testRepo.RefreshRepo()

		// Create complex staging scenarios

		// 1. New file, fully staged
		testRepo.CreateFile("new-file.txt", "new file content")
		_, err := testRepo.Repo.GitExec("add", "new-file.txt")
		if err != nil {
			t.Fatalf("Failed to stage new file: %v", err)
		}

		// 2. Modified file, fully staged
		testRepo.CreateFile("file1.txt", "modified content")
		_, err = testRepo.Repo.GitExec("add", "file1.txt")
		if err != nil {
			t.Fatalf("Failed to stage modified file: %v", err)
		}

		// 3. Modified file with staged and unstaged changes (this is tricky!)
		testRepo.CreateFile("file2.txt", "partially staged content")
		_, err = testRepo.Repo.GitExec("add", "file2.txt")
		if err != nil {
			t.Fatalf("Failed to stage partial changes: %v", err)
		}
		// Now make additional unstaged changes
		testRepo.CreateFile("file2.txt", "partially staged content\nplus unstaged changes")

		// 4. Deleted file, staged for deletion
		_, err = testRepo.Repo.GitExec("rm", "file3.txt")
		if err != nil {
			t.Fatalf("Failed to stage file deletion: %v", err)
		}

		// 5. Untracked file (for completeness)
		testRepo.CreateFile("untracked.txt", "untracked content")

		// Get detailed git status before the operation
		statusBefore, err := testRepo.Repo.GitExec("status", "--porcelain")
		if err != nil {
			t.Fatalf("Failed to get git status: %v", err)
		}
		t.Logf("Git status before update:\n%s", statusBefore)

		// Parse the status to understand what we expect
		expectedStates := make(map[string]string)
		for _, line := range strings.Split(strings.TrimSpace(statusBefore), "\n") {
			if line == "" {
				continue
			}
			if len(line) >= 3 {
				statusCode := line[:2]
				filename := line[3:]
				expectedStates[filename] = statusCode
			}
		}

		// Verify we have the complex staging states we expect
		if expectedStates["new-file.txt"] != "A " {
			t.Fatalf("Expected new-file.txt to be 'A ', got '%s'", expectedStates["new-file.txt"])
		}
		if expectedStates["file1.txt"] != "M " {
			t.Fatalf("Expected file1.txt to be 'M ', got '%s'", expectedStates["file1.txt"])
		}
		if expectedStates["file2.txt"] != "MM" {
			t.Fatalf("Expected file2.txt to be 'MM', got '%s'", expectedStates["file2.txt"])
		}
		if expectedStates["file3.txt"] != "D " {
			t.Fatalf("Expected file3.txt to be 'D ', got '%s'", expectedStates["file3.txt"])
		}
		if expectedStates["untracked.txt"] != "??" {
			t.Fatalf("Expected untracked.txt to be '??', got '%s'", expectedStates["untracked.txt"])
		}

		// Now test updating the commit message (this should preserve all staging states)
		prURL := "https://github.com/owner/repo/pull/123"
		err = UpdateOldestCommitWithPRURL(testRepo.Repo, "main", prURL)
		if err != nil {
			t.Fatalf("Failed to update commit with PR URL: %v", err)
		}

		// Get detailed git status after the operation
		statusAfter, err := testRepo.Repo.GitExec("status", "--porcelain")
		if err != nil {
			t.Fatalf("Failed to get git status after update: %v", err)
		}
		t.Logf("Git status after update:\n%s", statusAfter)

		// Parse the status after operation
		actualStates := make(map[string]string)
		for _, line := range strings.Split(strings.TrimSpace(statusAfter), "\n") {
			if line == "" {
				continue
			}
			if len(line) >= 3 {
				statusCode := line[:2]
				filename := line[3:]
				actualStates[filename] = statusCode
			}
		}

		// Verify every staging state is exactly preserved
		for filename, expectedState := range expectedStates {
			actualState, exists := actualStates[filename]
			if !exists {
				t.Errorf("File %s missing from git status after update", filename)
				continue
			}
			if actualState != expectedState {
				t.Errorf("File %s: expected state '%s', got '%s'", filename, expectedState, actualState)
			}
		}

		// Verify no extra files appeared
		for filename := range actualStates {
			if _, expected := expectedStates[filename]; !expected {
				t.Errorf("Unexpected file %s appeared in git status after update", filename)
			}
		}

		// Verify the content of the partially staged file is correct
		file2ContentBytes, err := os.ReadFile(filepath.Join(testRepo.Dir, "file2.txt"))
		if err != nil {
			t.Errorf("Failed to read file2.txt: %v", err)
		} else {
			file2Content := string(file2ContentBytes)
			expectedContent := "partially staged content\nplus unstaged changes"
			if file2Content != expectedContent {
				t.Errorf("File2.txt content mismatch. Expected:\n%s\nGot:\n%s", expectedContent, file2Content)
			}
		}

		// Verify the staged content vs working directory content for file2.txt
		stagedContent, err := testRepo.Repo.GitExec("show", ":file2.txt")
		if err != nil {
			t.Errorf("Failed to get staged content of file2.txt: %v", err)
		} else {
			expectedStagedContent := "partially staged content"
			if strings.TrimSpace(stagedContent) != expectedStagedContent {
				t.Errorf("Staged content of file2.txt mismatch. Expected:\n%s\nGot:\n%s", expectedStagedContent, strings.TrimSpace(stagedContent))
			}
		}

		t.Log("✅ All staging states perfectly preserved after commit message update")
	})
}
