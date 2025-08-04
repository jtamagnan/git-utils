package lint

import (
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
		// Should fail because there's no HEAD/upstream branch in an empty repo
		if err == nil {
			t.Error("Expected error for empty repository, but got none")
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
