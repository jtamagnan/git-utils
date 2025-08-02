package github

import (
	"os"
	"testing"
)

func TestGitHubPackageExists(t *testing.T) {
	// This is a simple test to ensure the package compiles
	// Real tests would require GitHub API mocking which is beyond scope
	t.Log("GitHub API package is properly structured")
}

func TestAuthenticatedClientRequiresToken(t *testing.T) {
	// Save original GITHUB_TOKEN
	originalToken := os.Getenv("GITHUB_TOKEN")
	defer func() {
		if originalToken != "" {
			os.Setenv("GITHUB_TOKEN", originalToken)
		} else {
			os.Unsetenv("GITHUB_TOKEN")
		}
	}()

	// Test with no token
	os.Unsetenv("GITHUB_TOKEN")
	client, err := newAuthenticatedClient()
	if err == nil {
		t.Error("Expected error when GITHUB_TOKEN is not set, but got none")
	}
	if client != nil {
		t.Error("Expected nil client when GITHUB_TOKEN is not set, but got a client")
	}

	// Test with token
	os.Setenv("GITHUB_TOKEN", "test-token")
	client, err = newAuthenticatedClient()
	if err != nil {
		t.Errorf("Expected no error when GITHUB_TOKEN is set, but got: %v", err)
	}
	if client == nil {
		t.Error("Expected client when GITHUB_TOKEN is set, but got nil")
	}
}
