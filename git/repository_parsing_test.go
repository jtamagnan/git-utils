package git

import (
	"testing"
)

func TestParseRepositoryInfo(t *testing.T) {
	tests := []struct {
		name        string
		remoteURL   string
		expectOwner string
		expectRepo  string
		expectError bool
	}{
		{
			name:        "HTTPS URL with .git suffix",
			remoteURL:   "https://github.com/octocat/Hello-World.git",
			expectOwner: "octocat",
			expectRepo:  "Hello-World",
			expectError: false,
		},
		{
			name:        "HTTPS URL without .git suffix",
			remoteURL:   "https://github.com/microsoft/vscode",
			expectOwner: "microsoft",
			expectRepo:  "vscode",
			expectError: false,
		},
		{
			name:        "HTTPS URL with trailing slash",
			remoteURL:   "https://github.com/facebook/react/",
			expectOwner: "facebook",
			expectRepo:  "react",
			expectError: false,
		},
		{
			name:        "SSH URL with .git suffix",
			remoteURL:   "git@github.com:torvalds/linux.git",
			expectOwner: "torvalds",
			expectRepo:  "linux",
			expectError: false,
		},
		{
			name:        "SSH URL without .git suffix",
			remoteURL:   "git@github.com:golang/go",
			expectOwner: "golang",
			expectRepo:  "go",
			expectError: false,
		},
		{
			name:        "Repository with hyphens and underscores",
			remoteURL:   "https://github.com/user-name/repo_name-test.git",
			expectOwner: "user-name",
			expectRepo:  "repo_name-test",
			expectError: false,
		},
		{
			name:        "Repository with numbers",
			remoteURL:   "https://github.com/user123/repo456.git",
			expectOwner: "user123",
			expectRepo:  "repo456",
			expectError: false,
		},
		{
			name:        "Invalid URL - not GitHub",
			remoteURL:   "https://gitlab.com/user/repo.git",
			expectError: true,
		},
		{
			name:        "Invalid URL - malformed",
			remoteURL:   "not-a-url",
			expectError: true,
		},
		{
			name:        "Invalid URL - missing parts",
			remoteURL:   "https://github.com/user",
			expectError: true,
		},
		{
			name:        "Empty URL",
			remoteURL:   "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoInfo, err := ParseRepositoryInfo(tt.remoteURL)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for URL '%s', but got none", tt.remoteURL)
				}
				if repoInfo != nil {
					t.Errorf("Expected nil repoInfo on error, got %+v", repoInfo)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for URL '%s': %v", tt.remoteURL, err)
				return
			}

			if repoInfo == nil {
				t.Errorf("Expected repoInfo but got nil for URL '%s'", tt.remoteURL)
				return
			}

			if repoInfo.Owner != tt.expectOwner {
				t.Errorf("Expected owner '%s', got '%s' for URL '%s'", tt.expectOwner, repoInfo.Owner, tt.remoteURL)
			}

			if repoInfo.Name != tt.expectRepo {
				t.Errorf("Expected repo '%s', got '%s' for URL '%s'", tt.expectRepo, repoInfo.Name, tt.remoteURL)
			}
		})
	}
}

func TestGetRemoteURL(t *testing.T) {
	testRepo := NewTestRepo(t)
	defer testRepo.Cleanup()

	testRepo.InDir(func() {
		// Create initial commit
		testRepo.AddCommit("README.md", "# Test", "Initial commit")

		// Add remote
		testRemoteURL := "https://github.com/testuser/testrepo.git"
		testRepo.AddRemote("origin", testRemoteURL)

		testRepo.RefreshRepo()

		// Test GetRemoteURL
		remoteURL, err := testRepo.Repo.GetRemoteURL("origin")
		if err != nil {
			t.Fatalf("GetRemoteURL failed: %v", err)
		}

		// Git might convert HTTPS to SSH format, so we just verify we get a valid URL
		if remoteURL == "" {
			t.Errorf("Expected non-empty remote URL, got empty string")
		}

		// Test parsing the URL (should work with both HTTPS and SSH formats)
		repoInfo, err := ParseRepositoryInfo(remoteURL)
		if err != nil {
			t.Fatalf("ParseRepositoryInfo failed for URL '%s': %v", remoteURL, err)
		}

		if repoInfo.Owner != "testuser" {
			t.Errorf("Expected owner 'testuser', got '%s'", repoInfo.Owner)
		}

		if repoInfo.Name != "testrepo" {
			t.Errorf("Expected repo 'testrepo', got '%s'", repoInfo.Name)
		}
	})
}
