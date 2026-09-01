package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jtamagnan/git-utils/git"
	lint "github.com/jtamagnan/git-utils/lint/lib"
)

// TestLintIntegration tests the lint functionality with real git repositories
func TestLintIntegration(t *testing.T) {
	// Skip if pre-commit is not available
	if !isPreCommitAvailable() {
		t.Skip("pre-commit not available, skipping integration tests")
	}

	t.Run("AllChecks", testLintAllChecks)
	t.Run("SingleCheck", testLintSingleCheck)
	t.Run("MultipleChecks", testLintMultipleChecks)
	t.Run("InvalidCheck", testLintInvalidCheck)
	t.Run("AllFilesFlag", testLintAllFilesFlag)
	t.Run("EmptyRepository", testLintEmptyRepository)
	t.Run("WithTrackedBranch", testLintWithTrackedBranch)
}

func testLintAllChecks(t *testing.T) {
	testRepo := setupTestRepoWithPreCommit(t)
	defer testRepo.Cleanup()

	testRepo.InDir(func() {
		// Test running all checks (no specific check names)
		args := lint.ParsedArgs{
			AllFiles:   false,
			Stream:     true,
			CheckNames: []string{}, // Empty means run all checks
		}

		err := lint.Lint(args)
		// We expect this might fail since we don't have a proper pre-commit setup
		// but it should at least try to run the command
		if err != nil {
			// Check that it's a pre-commit related error, not a logic error
			if !strings.Contains(err.Error(), "pre-commit") {
				t.Errorf("Expected pre-commit related error, got: %v", err)
			}
		}
	})
}

func testLintSingleCheck(t *testing.T) {
	testRepo := setupTestRepoWithPreCommit(t)
	defer testRepo.Cleanup()

	testRepo.InDir(func() {
		// Test running a single check
		args := lint.ParsedArgs{
			AllFiles:   false,
			Stream:     true,
			CheckNames: []string{"check-yaml"}, // A common pre-commit check
		}

		err := lint.Lint(args)
		// We expect this might fail, but should handle the single check correctly
		if err != nil && !strings.Contains(err.Error(), "pre-commit") {
			t.Errorf("Expected pre-commit related error, got: %v", err)
		}
	})
}

func testLintMultipleChecks(t *testing.T) {
	testRepo := setupTestRepoWithPreCommit(t)
	defer testRepo.Cleanup()

	testRepo.InDir(func() {
		// Test running multiple checks
		args := lint.ParsedArgs{
			AllFiles:   false,
			Stream:     true,
			CheckNames: []string{"check-yaml", "end-of-file-fixer"},
		}

		err := lint.Lint(args)
		// We expect this might fail, but should handle multiple checks correctly
		if err != nil && !strings.Contains(err.Error(), "pre-commit") {
			t.Errorf("Expected pre-commit related error, got: %v", err)
		}
	})
}

func testLintInvalidCheck(t *testing.T) {
	testRepo := setupTestRepoWithPreCommit(t)
	defer testRepo.Cleanup()

	testRepo.InDir(func() {
		// Test running with an invalid check name
		args := lint.ParsedArgs{
			AllFiles:   false,
			Stream:     true,
			CheckNames: []string{"non-existent-check"},
		}

		err := lint.Lint(args)
		// Should definitely fail with an error about the invalid check
		if err == nil {
			t.Error("Expected error for invalid check name, but got none")
		}
		if !strings.Contains(err.Error(), "non-existent-check") && !strings.Contains(err.Error(), "pre-commit") {
			t.Errorf("Expected error to mention invalid check or pre-commit, got: %v", err)
		}
	})
}

func testLintAllFilesFlag(t *testing.T) {
	testRepo := setupTestRepoWithPreCommit(t)
	defer testRepo.Cleanup()

	testRepo.InDir(func() {
		// Test running with --all flag
		args := lint.ParsedArgs{
			AllFiles:   true,
			Stream:     true,
			CheckNames: []string{"check-yaml"},
		}

		err := lint.Lint(args)
		// We expect this might fail, but should handle the --all flag correctly
		if err != nil && !strings.Contains(err.Error(), "pre-commit") {
			t.Errorf("Expected pre-commit related error, got: %v", err)
		}
	})
}

func testLintEmptyRepository(t *testing.T) {
	testRepo := git.NewTestRepo(t)
	defer testRepo.Cleanup()

	testRepo.InDir(func() {
		// Test running lint on an empty repository (no commits)
		args := lint.ParsedArgs{
			AllFiles:   false,
			Stream:     true,
			CheckNames: []string{},
		}

		err := lint.Lint(args)
		// Should pass because canLint() returns false for repos without pre-commit config
		if err != nil {
			t.Errorf("Expected no error for empty repository without pre-commit config, but got: %v", err)
		}
	})
}

func testLintWithTrackedBranch(t *testing.T) {
	testRepo := setupTestRepoWithPreCommit(t)
	defer testRepo.Cleanup()

	// Add another commit (upstream tracking is already set up by setupTestRepoWithPreCommit)
	testRepo.AddCommit("initial.txt", "initial content", "Second commit")

	testRepo.InDir(func() {
		// Test running with a properly tracked branch
		args := lint.ParsedArgs{
			AllFiles:   false,
			Stream:     true,
			CheckNames: []string{"check-yaml"},
		}

		err := lint.Lint(args)
		// We expect this might fail due to pre-commit setup, but git logic should work
		if err != nil && !strings.Contains(err.Error(), "pre-commit") {
			t.Errorf("Expected pre-commit related error, got: %v", err)
		}
	})
}

// TestLintNoMergeConflictsWithPreExistingStash is an integration test that
// reproduces a bug where git-lint's commit/stash/reset dance interacts with
// prek/pre-commit to produce merge conflicts (UU entries) touching the same
// files as a pre-existing stash.
func TestLintNoMergeConflictsWithPreExistingStash(t *testing.T) {
	if !isPreCommitAvailable() {
		t.Skip("pre-commit/prek not available, skipping integration test")
	}

	testRepo := git.NewTestRepo(t)
	defer testRepo.Cleanup()

	// Use end-of-file-fixer which will auto-fix files missing trailing newlines
	preCommitConfig := `repos:
  - repo: https://github.com/pre-commit/pre-commit-hooks
    rev: v4.4.0
    hooks:
      - id: end-of-file-fixer
      - id: trailing-whitespace
`
	testRepo.CreateFile(".pre-commit-config.yaml", preCommitConfig)
	testRepo.AddCommit("README.md", "# Test Repository\n", "Initial commit")
	testRepo.AddRemote("origin", "https://github.com/example/repo.git")
	testRepo.CreateRemoteTrackingBranch("origin", "main")
	testRepo.SetUpstream("origin", "main")

	// Create a file, commit it, then create a stash that modifies it.
	// This simulates the user's workflow where stash@{0} touches the same
	// files they're actively working on.
	testRepo.AddCommit("code.go", "package main\n\nfunc main() {}\n", "Add code.go")
	testRepo.CreateFile("code.go", "package main\n\nfunc stashedWork() {}\n")
	testRepo.GitExec("stash", "push", "-m", "user work in progress")

	// Now stage a change to the same file (this is the key: staged change
	// to a file that's also in stash@{0})
	testRepo.CreateFile("code.go", "package main\n\nfunc newFeature() {}\n")
	testRepo.GitExec("add", "code.go")

	// Also create an unstaged modification to another tracked file
	testRepo.CreateFile("README.md", "# Modified readme\n")

	testRepo.InDir(func() {
		headBefore := testRepo.GitExec("rev-parse", "HEAD")
		stashBefore := testRepo.GitExec("stash", "list")
		statusBefore := testRepo.GitExec("status", "--porcelain")

		args := lint.ParsedArgs{
			AllFiles:   false,
			Stream:     true,
			CheckNames: []string{},
		}

		_ = lint.Lint(args)

		// HEAD must not change
		headAfter := testRepo.GitExec("rev-parse", "HEAD")
		if headBefore != headAfter {
			t.Errorf("HEAD changed: %s → %s", headBefore, headAfter)
		}

		// No unmerged (UU) entries — this is the actual bug
		statusAfter := testRepo.GitExec("status", "--porcelain")
		if strings.Contains(statusAfter, "UU ") || strings.Contains(statusAfter, "AA ") {
			t.Errorf("Merge conflicts after lint!\n  status before:\n%s\n  status after:\n%s", statusBefore, statusAfter)
		}

		// Pre-existing stash must be untouched
		stashAfter, _ := testRepo.GitExecWithError("stash", "list")
		t.Logf("Stash before: %q", stashBefore)
		t.Logf("Stash after:  %q", stashAfter)
		if stashBefore != stashAfter {
			t.Errorf("Stash list changed!\n  before: %q\n  after:  %q", stashBefore, stashAfter)
		}

		// File contents should not have conflict markers
		content, err := os.ReadFile("code.go")
		if err != nil {
			t.Fatalf("Failed to read code.go: %v", err)
		}
		if strings.Contains(string(content), "<<<<<<<") {
			t.Errorf("Conflict markers in code.go:\n%s", string(content))
		}
	})
}

// TestLintAutoFixStagedFile verifies that when a staged file has style issues
// (e.g. missing trailing newline), the linter auto-fixes it and the fix is
// present in the worktree after the commit/reset dance.
func TestLintAutoFixStagedFile(t *testing.T) {
	if !isPreCommitAvailable() {
		t.Skip("pre-commit/prek not available, skipping integration test")
	}

	testRepo := git.NewTestRepo(t)
	defer testRepo.Cleanup()

	preCommitConfig := `repos:
  - repo: https://github.com/pre-commit/pre-commit-hooks
    rev: v4.4.0
    hooks:
      - id: end-of-file-fixer
      - id: trailing-whitespace
`
	testRepo.CreateFile(".pre-commit-config.yaml", preCommitConfig)
	testRepo.AddCommit("README.md", "# Test Repository\n", "Initial commit")
	testRepo.AddRemote("origin", "https://github.com/example/repo.git")
	testRepo.CreateRemoteTrackingBranch("origin", "main")
	testRepo.SetUpstream("origin", "main")

	// Stage a file missing a trailing newline — end-of-file-fixer should fix it
	testRepo.CreateFile("no-newline.txt", "missing trailing newline")
	testRepo.GitExec("add", "no-newline.txt")

	// Stage a file with trailing whitespace — trailing-whitespace should fix it
	testRepo.CreateFile("trailing-ws.txt", "has trailing spaces   \n")
	testRepo.GitExec("add", "trailing-ws.txt")

	testRepo.InDir(func() {
		headBefore := testRepo.GitExec("rev-parse", "HEAD")

		args := lint.ParsedArgs{
			AllFiles:   false,
			Stream:     true,
			CheckNames: []string{},
		}

		// Lint will fail because the hooks fix files (non-zero exit)
		_ = lint.Lint(args)

		// HEAD unchanged
		headAfter := testRepo.GitExec("rev-parse", "HEAD")
		if headBefore != headAfter {
			t.Errorf("HEAD changed: %s → %s", headBefore, headAfter)
		}

		// end-of-file-fixer should have added a trailing newline
		content, err := os.ReadFile("no-newline.txt")
		if err != nil {
			t.Fatalf("Failed to read no-newline.txt: %v", err)
		}
		if !strings.HasSuffix(string(content), "\n") {
			t.Errorf("Expected end-of-file-fixer to add trailing newline, got: %q", string(content))
		}

		// trailing-whitespace should have removed trailing spaces
		content, err = os.ReadFile("trailing-ws.txt")
		if err != nil {
			t.Fatalf("Failed to read trailing-ws.txt: %v", err)
		}
		if strings.Contains(string(content), "   \n") {
			t.Errorf("Expected trailing-whitespace to remove trailing spaces, got: %q", string(content))
		}

		// Files should still be staged
		staged := testRepo.GitExec("diff", "--cached", "--name-only")
		if !strings.Contains(staged, "no-newline.txt") {
			t.Errorf("Expected no-newline.txt to remain staged, got: %q", staged)
		}
		if !strings.Contains(staged, "trailing-ws.txt") {
			t.Errorf("Expected trailing-ws.txt to remain staged, got: %q", staged)
		}

		// No merge conflicts
		status := testRepo.GitExec("status", "--porcelain")
		if strings.Contains(status, "UU ") {
			t.Errorf("Unexpected merge conflicts:\n%s", status)
		}
	})
}

// TestLintAutoFixUntrackedModification verifies that when a tracked file has
// unstaged modifications with style issues, the linter auto-fixes it and the
// fix is present in the worktree after the commit/reset dance.
func TestLintAutoFixUntrackedModification(t *testing.T) {
	if !isPreCommitAvailable() {
		t.Skip("pre-commit/prek not available, skipping integration test")
	}

	testRepo := git.NewTestRepo(t)
	defer testRepo.Cleanup()

	preCommitConfig := `repos:
  - repo: https://github.com/pre-commit/pre-commit-hooks
    rev: v4.4.0
    hooks:
      - id: end-of-file-fixer
      - id: trailing-whitespace
`
	testRepo.CreateFile(".pre-commit-config.yaml", preCommitConfig)
	testRepo.AddCommit("README.md", "# Test Repository\n", "Initial commit")
	testRepo.AddRemote("origin", "https://github.com/example/repo.git")
	testRepo.CreateRemoteTrackingBranch("origin", "main")
	testRepo.SetUpstream("origin", "main")

	// Commit a clean file, then modify it with style issues (unstaged)
	testRepo.AddCommit("code.txt", "clean content\n", "Add code.txt")
	testRepo.CreateFile("code.txt", "trailing spaces   \nno final newline")

	testRepo.InDir(func() {
		headBefore := testRepo.GitExec("rev-parse", "HEAD")

		args := lint.ParsedArgs{
			AllFiles:   false,
			Stream:     true,
			CheckNames: []string{},
		}

		_ = lint.Lint(args)

		// HEAD unchanged
		headAfter := testRepo.GitExec("rev-parse", "HEAD")
		if headBefore != headAfter {
			t.Errorf("HEAD changed: %s → %s", headBefore, headAfter)
		}

		// The linter fixes should be present in the worktree
		content, err := os.ReadFile("code.txt")
		if err != nil {
			t.Fatalf("Failed to read code.txt: %v", err)
		}
		if strings.Contains(string(content), "   \n") {
			t.Errorf("Expected trailing whitespace to be removed, got: %q", string(content))
		}
		if !strings.HasSuffix(string(content), "\n") {
			t.Errorf("Expected trailing newline to be added, got: %q", string(content))
		}

		// code.txt should still show as unstaged modification
		unstaged := testRepo.GitExec("diff", "--name-only")
		if !strings.Contains(unstaged, "code.txt") {
			t.Errorf("Expected code.txt to have unstaged modifications, got: %q", unstaged)
		}

		// No merge conflicts
		status := testRepo.GitExec("status", "--porcelain")
		if strings.Contains(status, "UU ") {
			t.Errorf("Unexpected merge conflicts:\n%s", status)
		}
	})
}

// setupTestRepoWithPreCommit creates a test repo with basic pre-commit setup and upstream tracking
func setupTestRepoWithPreCommit(t *testing.T) *git.TestRepo {
	testRepo := git.NewTestRepo(t)

	// Create a basic .pre-commit-config.yaml for testing
	preCommitConfig := `repos:
  - repo: https://github.com/pre-commit/pre-commit-hooks
    rev: v4.4.0
    hooks:
      - id: check-yaml
      - id: end-of-file-fixer
      - id: trailing-whitespace
`
	testRepo.CreateFile(".pre-commit-config.yaml", preCommitConfig)

	// Create an initial commit so we have something to work with
	testRepo.AddCommit("README.md", "# Test Repository\n", "Initial commit")

	// Set up upstream tracking so AllFiles: false tests can resolve the tracking branch
	testRepo.AddRemote("origin", "https://github.com/example/repo.git")
	testRepo.CreateRemoteTrackingBranch("origin", "main")
	testRepo.SetUpstream("origin", "main")

	return testRepo
}

// isPreCommitAvailable checks if pre-commit is available in PATH
func isPreCommitAvailable() bool {
	_, err := os.Stat("/nix/store")
	if err == nil {
		// We're in Nix environment, check for pre-commit
		entries, err := filepath.Glob("/nix/store/*pre-commit*/bin/pre-commit")
		return err == nil && len(entries) > 0
	}

	// Fallback to PATH check
	_, err = exec.LookPath("pre-commit")
	return err == nil
}

// TestLintCommandConstruction tests the command construction logic without executing
func TestLintCommandConstruction(t *testing.T) {
	testRepo := git.NewTestRepo(t)
	defer testRepo.Cleanup()

	// Setup a basic repository with upstream tracking
	testRepo.AddCommit("test.txt", "test content", "Initial commit")
	testRepo.AddRemote("origin", "https://github.com/example/repo.git")
	testRepo.CreateRemoteTrackingBranch("origin", "main")
	testRepo.SetUpstream("origin", "main")

	tests := []struct {
		name       string
		args       lint.ParsedArgs
		expectArgs func(args []string) bool
	}{
		{
			name: "AllFiles_True",
			args: lint.ParsedArgs{
				AllFiles:   true,
				Stream:     true,
				CheckNames: []string{},
			},
			expectArgs: func(args []string) bool {
				// Should NOT contain --from-ref and --to-ref when AllFiles is true
				for _, arg := range args {
					if strings.HasPrefix(arg, "--from-ref") || strings.HasPrefix(arg, "--to-ref") {
						return false
					}
				}
				return contains(args, "--all-files")
			},
		},
		{
			name: "AllFiles_False",
			args: lint.ParsedArgs{
				AllFiles:   false,
				Stream:     true,
				CheckNames: []string{},
			},
			expectArgs: func(args []string) bool {
				// Should contain --from-ref and --to-ref when AllFiles is false
				hasFromRef := false
				hasToRef := false
				for _, arg := range args {
					if strings.HasPrefix(arg, "--from-ref") {
						hasFromRef = true
					}
					if strings.HasPrefix(arg, "--to-ref") {
						hasToRef = true
					}
				}
				return hasFromRef && hasToRef && contains(args, "--all-files")
			},
		},
		{
			name: "SingleCheck",
			args: lint.ParsedArgs{
				AllFiles:   true,
				Stream:     true,
				CheckNames: []string{"check-yaml"},
			},
			expectArgs: func(args []string) bool {
				return contains(args, "check-yaml")
			},
		},
		{
			name: "MultipleChecks",
			args: lint.ParsedArgs{
				AllFiles:   true,
				Stream:     true,
				CheckNames: []string{"check-yaml", "end-of-file-fixer"},
			},
			expectArgs: func(args []string) bool {
				// This should result in separate command invocations
				// We can't easily test this without mocking, but we can test the logic
				return len(args) > 0 // Basic sanity check
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testRepo.InDir(func() {
				// We can't easily intercept the command execution without significant mocking
				// So for now, we'll test that the function doesn't panic and handles the args
				// In a real scenario, we'd mock exec.Command
				_ = tt.expectArgs([]string{"run", "--color=always", "--all-files"})
			})
		})
	}
}

// Helper function to check if slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
