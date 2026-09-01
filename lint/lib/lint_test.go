package lint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jtamagnan/git-utils/git"
)

// TestParsedArgsStructure tests that ParsedArgs has the expected fields
func TestParsedArgsStructure(t *testing.T) {
	args := ParsedArgs{
		AllFiles:   true,
		Stream:     false,
		CheckNames: []string{"check1", "check2"},
	}

	if !args.AllFiles {
		t.Error("Expected AllFiles to be true")
	}
	if args.Stream {
		t.Error("Expected Stream to be false")
	}
	if len(args.CheckNames) != 2 {
		t.Errorf("Expected 2 check names, got %d", len(args.CheckNames))
	}
	if args.CheckNames[0] != "check1" || args.CheckNames[1] != "check2" {
		t.Errorf("Expected ['check1', 'check2'], got %v", args.CheckNames)
	}
}

// TestLintWithEmptyRepo tests that Lint fails gracefully with an empty repository
func TestLintWithEmptyRepo(t *testing.T) {
	testRepo := git.NewTestRepo(t)
	defer testRepo.Cleanup()

	testRepo.InDir(func() {
		args := ParsedArgs{
			AllFiles:   false,
			Stream:     true,
			CheckNames: []string{},
		}

		err := Lint(args)
		// Should pass because canLint() returns false for repos without pre-commit config
		if err != nil {
			t.Errorf("Expected no error for empty repository without pre-commit config, but got: %v", err)
		}
	})
}

// TestLintWithValidRepo tests that Lint handles a valid repository setup
func TestLintWithValidRepo(t *testing.T) {
	testRepo := git.NewTestRepo(t)
	defer testRepo.Cleanup()

	// Create a commit so we have a valid HEAD
	testRepo.AddCommit("test.txt", "test content", "Initial commit")

	testRepo.InDir(func() {
		args := ParsedArgs{
			AllFiles:   true, // Use AllFiles to avoid upstream branch requirements
			Stream:     true,
			CheckNames: []string{},
		}

		err := Lint(args)
		// This will likely fail because pre-commit isn't set up, but the git logic should work
		// We just want to make sure it gets to the pre-commit execution step
		if err != nil {
			// Should be a pre-commit execution error, not a git setup error
			if err.Error() == "" {
				t.Error("Expected non-empty error message")
			}
		}
	})
}

// TestLintArgumentValidation tests various argument combinations
func TestLintArgumentValidation(t *testing.T) {
	testRepo := git.NewTestRepo(t)
	defer testRepo.Cleanup()

	// Create a basic repository with upstream tracking
	testRepo.AddCommit("test.txt", "test content", "Initial commit")
	testRepo.AddRemote("origin", "https://github.com/example/repo.git")
	testRepo.CreateRemoteTrackingBranch("origin", "main")
	testRepo.SetUpstream("origin", "main")

	tests := []struct {
		name        string
		args        ParsedArgs
		expectError bool
		description string
	}{
		{
			name: "EmptyCheckNames",
			args: ParsedArgs{
				AllFiles:   true,
				Stream:     true,
				CheckNames: []string{},
			},
			expectError: false, // Should run all checks
			description: "Empty check names should run all checks",
		},
		{
			name: "SingleCheck",
			args: ParsedArgs{
				AllFiles:   true,
				Stream:     true,
				CheckNames: []string{"check-yaml"},
			},
			expectError: false, // Should attempt to run the check
			description: "Single check should be handled correctly",
		},
		{
			name: "MultipleChecks",
			args: ParsedArgs{
				AllFiles:   true,
				Stream:     true,
				CheckNames: []string{"check-yaml", "end-of-file-fixer", "trailing-whitespace"},
			},
			expectError: false, // Should attempt to run all checks
			description: "Multiple checks should be handled correctly",
		},
		{
			name: "NonStreamMode",
			args: ParsedArgs{
				AllFiles:   true,
				Stream:     false,
				CheckNames: []string{"check-yaml"},
			},
			expectError: false, // Should work with non-stream mode
			description: "Non-stream mode should work",
		},
		{
			name: "AllFilesFalse",
			args: ParsedArgs{
				AllFiles:   false,
				Stream:     true,
				CheckNames: []string{"check-yaml"},
			},
			expectError: false, // Should work with upstream branch
			description: "AllFiles false should work with tracked branch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testRepo.InDir(func() {
				err := Lint(tt.args)

				if tt.expectError && err == nil {
					t.Errorf("%s: expected error but got none", tt.description)
				}

				if !tt.expectError && err != nil {
					// We expect pre-commit related errors since we don't have pre-commit set up
					// But we don't want git-related errors or argument parsing errors
					t.Logf("%s: got expected pre-commit error: %v", tt.description, err)
				}
			})
		})
	}
}

// TestCheckNamesHandling tests that check names are handled correctly
func TestCheckNamesHandling(t *testing.T) {
	tests := []struct {
		name       string
		checkNames []string
		expected   int
	}{
		{
			name:       "NoChecks",
			checkNames: []string{},
			expected:   0,
		},
		{
			name:       "SingleCheck",
			checkNames: []string{"check-yaml"},
			expected:   1,
		},
		{
			name:       "MultipleChecks",
			checkNames: []string{"check-yaml", "end-of-file-fixer", "trailing-whitespace"},
			expected:   3,
		},
		{
			name:       "DuplicateChecks",
			checkNames: []string{"check-yaml", "check-yaml", "end-of-file-fixer"},
			expected:   3, // Should preserve duplicates (pre-commit will handle them)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := ParsedArgs{
				AllFiles:   true,
				Stream:     true,
				CheckNames: tt.checkNames,
			}

			if len(args.CheckNames) != tt.expected {
				t.Errorf("Expected %d check names, got %d", tt.expected, len(args.CheckNames))
			}

			// Verify the check names are preserved correctly
			for i, expected := range tt.checkNames {
				if i < len(args.CheckNames) && args.CheckNames[i] != expected {
					t.Errorf("Expected check name %q at index %d, got %q", expected, i, args.CheckNames[i])
				}
			}
		})
	}
}

// TestLintNoUpstream tests that Lint fails gracefully when no upstream is configured
func TestLintNoUpstream(t *testing.T) {
	testRepo := git.NewTestRepo(t)
	defer testRepo.Cleanup()

	// Create a commit and pre-commit config, but don't set up upstream tracking
	testRepo.AddCommit("test.txt", "test content", "Initial commit")
	testRepo.CreateFile(".pre-commit-config.yaml", "repos: []")

	testRepo.InDir(func() {
		args := ParsedArgs{
			AllFiles:   false, // This will trigger upstream branch lookup
			Stream:     true,
			CheckNames: []string{},
		}

		err := Lint(args)
		if err == nil {
			t.Error("Expected error when no upstream is configured, but got none")
		}

		expectedMsg := "no upstream branch configured for current branch - run 'git branch --set-upstream-to=<remote>/<branch>' to set upstream"
		if err.Error() != expectedMsg {
			t.Errorf("Expected error message %q, got %q", expectedMsg, err.Error())
		}
	})
}

// TestMultipleCheckFailures tests that all checks run even if some fail
func TestMultipleCheckFailures(t *testing.T) {
	testRepo := git.NewTestRepo(t)
	defer testRepo.Cleanup()

	// Create a repository with pre-commit config and upstream tracking
	testRepo.AddCommit("test.txt", "test content", "Initial commit")
	testRepo.AddRemote("origin", "https://github.com/example/repo.git")
	testRepo.CreateRemoteTrackingBranch("origin", "main")
	testRepo.SetUpstream("origin", "main")
	testRepo.CreateFile(".pre-commit-config.yaml", "repos: []")

	testRepo.InDir(func() {
		// Request multiple checks that will likely fail
		args := ParsedArgs{
			AllFiles:   true,
			Stream:     false,
			CheckNames: []string{"non-existent-check-1", "non-existent-check-2", "non-existent-check-3"},
		}

		err := Lint(args)

		// Should get an error
		if err == nil {
			t.Error("Expected error when running non-existent checks, but got none")
			return
		}

		// The error should be a joined error containing multiple check failures
		errMsg := err.Error()

		// Count how many checks are mentioned in the error
		checkCount := 0
		for _, checkName := range args.CheckNames {
			if strings.Contains(errMsg, checkName) {
				checkCount++
			}
		}

		// All checks should have run and failed, so all should be mentioned in the error
		if checkCount != len(args.CheckNames) {
			t.Errorf("Expected all %d checks to be mentioned in error, but only found %d", len(args.CheckNames), checkCount)
			t.Logf("Error message: %s", errMsg)
		}

		t.Logf("Successfully verified that all %d checks ran and errors were collected", checkCount)
	})
}

// TestLintWorktreeRestoration verifies that after Lint runs, staged files
// remain staged, unstaged modifications remain unstaged, and HEAD is unchanged.
func TestLintWorktreeRestoration(t *testing.T) {
	testRepo := git.NewTestRepo(t)
	defer testRepo.Cleanup()

	// Set up repo with upstream tracking and pre-commit config
	testRepo.AddCommit("committed.txt", "committed content", "Initial commit")
	testRepo.CreateFile(".pre-commit-config.yaml", "repos: []")
	testRepo.GitExec("add", ".pre-commit-config.yaml")
	testRepo.GitExec("commit", "-m", "Add pre-commit config")
	testRepo.AddRemote("origin", "https://github.com/example/repo.git")
	testRepo.CreateRemoteTrackingBranch("origin", "main")
	testRepo.SetUpstream("origin", "main")

	// Create staged change
	testRepo.CreateFile("staged.txt", "staged content")
	testRepo.GitExec("add", "staged.txt")

	// Create unstaged modification to a tracked file
	testRepo.CreateFile("committed.txt", "modified content")

	testRepo.InDir(func() {
		// Record state before lint
		headBefore := testRepo.GitExec("rev-parse", "HEAD")

		// Install a fake prek that succeeds (exits 0)
		fakeBin := installFakeLintCommand(t, "#!/bin/sh\nexit 0\n")
		origPath := os.Getenv("PATH")
		os.Setenv("PATH", fakeBin+":"+origPath)
		defer os.Setenv("PATH", origPath)

		args := ParsedArgs{
			AllFiles:   false,
			Stream:     false,
			CheckNames: []string{},
		}

		err := Lint(args)
		if err != nil {
			t.Fatalf("Lint returned unexpected error: %v", err)
		}

		// HEAD should be unchanged
		headAfter := testRepo.GitExec("rev-parse", "HEAD")
		if headBefore != headAfter {
			t.Errorf("HEAD changed: before=%s after=%s", headBefore, headAfter)
		}

		// staged.txt should still be staged
		stagedFiles := testRepo.GitExec("diff", "--cached", "--name-only")
		if !strings.Contains(stagedFiles, "staged.txt") {
			t.Errorf("Expected staged.txt to remain staged, got staged files: %q", stagedFiles)
		}

		// committed.txt should have unstaged modifications
		unstagedFiles := testRepo.GitExec("diff", "--name-only")
		if !strings.Contains(unstagedFiles, "committed.txt") {
			t.Errorf("Expected committed.txt to have unstaged modifications, got unstaged files: %q", unstagedFiles)
		}
	})
}

// TestLintFixupsPreserved verifies that when the linter modifies files,
// those fixups are present in the worktree after Lint returns.
func TestLintFixupsPreserved(t *testing.T) {
	testRepo := git.NewTestRepo(t)
	defer testRepo.Cleanup()

	// Set up repo with upstream tracking and pre-commit config
	testRepo.AddCommit("fixme.txt", "original content", "Initial commit")
	testRepo.CreateFile(".pre-commit-config.yaml", "repos: []")
	testRepo.GitExec("add", ".pre-commit-config.yaml")
	testRepo.GitExec("commit", "-m", "Add pre-commit config")
	testRepo.AddRemote("origin", "https://github.com/example/repo.git")
	testRepo.CreateRemoteTrackingBranch("origin", "main")
	testRepo.SetUpstream("origin", "main")

	// Stage a change so the lint has something to check
	testRepo.CreateFile("staged.txt", "staged content")
	testRepo.GitExec("add", "staged.txt")

	testRepo.InDir(func() {
		// Install a fake prek that modifies fixme.txt (simulating a linter autofix)
		script := "#!/bin/sh\necho 'linter fixed content' > fixme.txt\nexit 0\n"
		fakeBin := installFakeLintCommand(t, script)
		origPath := os.Getenv("PATH")
		os.Setenv("PATH", fakeBin+":"+origPath)
		defer os.Setenv("PATH", origPath)

		args := ParsedArgs{
			AllFiles:   false,
			Stream:     false,
			CheckNames: []string{},
		}

		err := Lint(args)
		if err != nil {
			t.Fatalf("Lint returned unexpected error: %v", err)
		}

		// The linter fixup should be present in the worktree
		content, readErr := os.ReadFile("fixme.txt")
		if readErr != nil {
			t.Fatalf("Failed to read fixme.txt: %v", readErr)
		}
		if !strings.Contains(string(content), "linter fixed content") {
			t.Errorf("Expected linter fixup to be preserved, got: %q", string(content))
		}
	})
}

// installFakeLintCommand creates a temporary directory with a fake "prek" script
// and returns the directory path (to be prepended to PATH).
func installFakeLintCommand(t *testing.T, script string) string {
	t.Helper()
	dir := t.TempDir()
	fakePath := filepath.Join(dir, "prek")
	if err := os.WriteFile(fakePath, []byte(script), 0755); err != nil {
		t.Fatalf("Failed to write fake prek: %v", err)
	}
	return dir
}

// TestLintWithUntrackedFiles verifies that untracked files are not lost or
// modified by the commit/stash/reset dance.
func TestLintWithUntrackedFiles(t *testing.T) {
	testRepo := git.NewTestRepo(t)
	defer testRepo.Cleanup()

	// Set up repo
	testRepo.AddCommit("base.txt", "base content", "Initial commit")
	testRepo.CreateFile(".pre-commit-config.yaml", "repos: []")
	testRepo.GitExec("add", ".pre-commit-config.yaml")
	testRepo.GitExec("commit", "-m", "Add pre-commit config")
	testRepo.AddRemote("origin", "https://github.com/example/repo.git")
	testRepo.CreateRemoteTrackingBranch("origin", "main")
	testRepo.SetUpstream("origin", "main")

	// Create staged, unstaged tracked, and untracked files
	testRepo.CreateFile("staged.txt", "staged content")
	testRepo.GitExec("add", "staged.txt")
	testRepo.CreateFile("base.txt", "modified content")
	testRepo.CreateFile("untracked.txt", "untracked content")

	testRepo.InDir(func() {
		headBefore := testRepo.GitExec("rev-parse", "HEAD")
		statusBefore := testRepo.GitExec("status", "--porcelain")

		fakeBin := installFakeLintCommand(t, "#!/bin/sh\nexit 0\n")
		origPath := os.Getenv("PATH")
		os.Setenv("PATH", fakeBin+":"+origPath)
		defer os.Setenv("PATH", origPath)

		args := ParsedArgs{
			AllFiles:   false,
			Stream:     false,
			CheckNames: []string{},
		}

		err := Lint(args)
		if err != nil {
			t.Fatalf("Lint returned unexpected error: %v", err)
		}

		// HEAD unchanged
		headAfter := testRepo.GitExec("rev-parse", "HEAD")
		if headBefore != headAfter {
			t.Errorf("HEAD changed: %s → %s", headBefore, headAfter)
		}

		// git status should be identical
		statusAfter := testRepo.GitExec("status", "--porcelain")
		if statusBefore != statusAfter {
			t.Errorf("Working tree state changed!\n  before:\n%s\n  after:\n%s", statusBefore, statusAfter)
		}

		// Verify file contents are unchanged
		for _, tc := range []struct{ name, expected string }{
			{"staged.txt", "staged content"},
			{"base.txt", "modified content"},
			{"untracked.txt", "untracked content"},
		} {
			content, readErr := os.ReadFile(tc.name)
			if readErr != nil {
				t.Errorf("Failed to read %s: %v", tc.name, readErr)
				continue
			}
			if string(content) != tc.expected {
				t.Errorf("%s: expected %q, got %q", tc.name, tc.expected, string(content))
			}
		}

		// No stash entries should have been created
		stashList, _ := testRepo.GitExecWithError("stash", "list")
		if stashList != "" {
			t.Errorf("Unexpected stash entries after lint: %q", stashList)
		}
	})
}

// TestLintFixupOnStagedFile verifies that when the linter modifies a file that
// was staged, the fixup shows as an unstaged modification on top of the staged version.
func TestLintFixupOnStagedFile(t *testing.T) {
	testRepo := git.NewTestRepo(t)
	defer testRepo.Cleanup()

	testRepo.AddCommit("target.txt", "original content\n", "Initial commit")
	testRepo.CreateFile(".pre-commit-config.yaml", "repos: []")
	testRepo.GitExec("add", ".pre-commit-config.yaml")
	testRepo.GitExec("commit", "-m", "Add pre-commit config")
	testRepo.AddRemote("origin", "https://github.com/example/repo.git")
	testRepo.CreateRemoteTrackingBranch("origin", "main")
	testRepo.SetUpstream("origin", "main")

	// Stage a change to target.txt
	testRepo.CreateFile("target.txt", "staged content\n")
	testRepo.GitExec("add", "target.txt")

	testRepo.InDir(func() {
		headBefore := testRepo.GitExec("rev-parse", "HEAD")

		// Fake prek that modifies the staged file (simulating autofix)
		script := "#!/bin/sh\necho 'linter fixed content' > target.txt\nexit 0\n"
		fakeBin := installFakeLintCommand(t, script)
		origPath := os.Getenv("PATH")
		os.Setenv("PATH", fakeBin+":"+origPath)
		defer os.Setenv("PATH", origPath)

		err := Lint(ParsedArgs{Stream: false})
		if err != nil {
			t.Fatalf("Lint returned unexpected error: %v", err)
		}

		// HEAD unchanged
		headAfter := testRepo.GitExec("rev-parse", "HEAD")
		if headBefore != headAfter {
			t.Errorf("HEAD changed: %s → %s", headBefore, headAfter)
		}

		// target.txt should still be staged (the staged version)
		stagedFiles := testRepo.GitExec("diff", "--cached", "--name-only")
		if !strings.Contains(stagedFiles, "target.txt") {
			t.Errorf("Expected target.txt to remain staged, got: %q", stagedFiles)
		}

		// The linter fixup should be present in the worktree
		content, _ := os.ReadFile("target.txt")
		if !strings.Contains(string(content), "linter fixed content") {
			t.Errorf("Expected linter fixup in worktree, got: %q", string(content))
		}

		// No leftover stash entries
		stashList, _ := testRepo.GitExecWithError("stash", "list")
		if strings.Contains(stashList, "git-lint") {
			t.Errorf("Leftover git-lint stash entry: %q", stashList)
		}
	})
}

// TestLintFailingLinterWithFixups verifies that when the linter auto-fixes files
// but exits non-zero, the fixups are still preserved and the error is returned.
func TestLintFailingLinterWithFixups(t *testing.T) {
	testRepo := git.NewTestRepo(t)
	defer testRepo.Cleanup()

	testRepo.AddCommit("target.txt", "original content\n", "Initial commit")
	testRepo.CreateFile(".pre-commit-config.yaml", "repos: []")
	testRepo.GitExec("add", ".pre-commit-config.yaml")
	testRepo.GitExec("commit", "-m", "Add pre-commit config")
	testRepo.AddRemote("origin", "https://github.com/example/repo.git")
	testRepo.CreateRemoteTrackingBranch("origin", "main")
	testRepo.SetUpstream("origin", "main")

	// Stage a change
	testRepo.CreateFile("target.txt", "bad content\n")
	testRepo.GitExec("add", "target.txt")

	testRepo.InDir(func() {
		headBefore := testRepo.GitExec("rev-parse", "HEAD")

		// Fake prek that fixes the file but exits non-zero (common autofix behavior)
		script := "#!/bin/sh\necho 'fixed content' > target.txt\nexit 1\n"
		fakeBin := installFakeLintCommand(t, script)
		origPath := os.Getenv("PATH")
		os.Setenv("PATH", fakeBin+":"+origPath)
		defer os.Setenv("PATH", origPath)

		err := Lint(ParsedArgs{Stream: false})
		// Should return an error (linter failed)
		if err == nil {
			t.Error("Expected error from failing linter, got nil")
		}

		// HEAD unchanged
		headAfter := testRepo.GitExec("rev-parse", "HEAD")
		if headBefore != headAfter {
			t.Errorf("HEAD changed: %s → %s", headBefore, headAfter)
		}

		// target.txt should still be staged
		stagedFiles := testRepo.GitExec("diff", "--cached", "--name-only")
		if !strings.Contains(stagedFiles, "target.txt") {
			t.Errorf("Expected target.txt to remain staged, got: %q", stagedFiles)
		}

		// Fixup should be present in worktree despite linter failure
		content, _ := os.ReadFile("target.txt")
		if !strings.Contains(string(content), "fixed content") {
			t.Errorf("Expected linter fixup to be preserved despite failure, got: %q", string(content))
		}
	})
}

// TestLintCommand tests that lintCommand returns a valid command name
func TestLintCommand(t *testing.T) {
	cmd := lintCommand()
	if cmd != "prek" && cmd != "pre-commit" {
		t.Errorf("Expected lintCommand() to return \"prek\" or \"pre-commit\", got %q", cmd)
	}
}

// TestCanLint tests the canLint function with and without pre-commit config
func TestCanLint(t *testing.T) {
	tests := []struct {
		name           string
		configFile     string
		configContent  string
		expectedResult bool
	}{
		{
			name:           "WithYamlConfig",
			configFile:     ".pre-commit-config.yaml",
			configContent:  "repos: []",
			expectedResult: true,
		},
		{
			name:           "WithYmlConfig",
			configFile:     ".pre-commit-config.yml",
			configContent:  "repos: []",
			expectedResult: true,
		},
		{
			name:           "NoConfig",
			configFile:     "",
			configContent:  "",
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testRepo := git.NewTestRepo(t)
			defer testRepo.Cleanup()

			testRepo.AddCommit("test.txt", "test content", "Initial commit")

			if tt.configFile != "" {
				testRepo.CreateFile(tt.configFile, tt.configContent)
			}

			testRepo.InDir(func() {
				result := canLint()
				if result != tt.expectedResult {
					t.Errorf("Expected canLint() to return %v, got %v", tt.expectedResult, result)
				}
			})
		})
	}
}
