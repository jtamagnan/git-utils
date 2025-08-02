package github

import (
	"os"
	"testing"

	keychain "github.com/jtamagnan/git-utils/keychain/lib"
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
			_ = os.Setenv("GITHUB_TOKEN", originalToken)
		} else {
			_ = os.Unsetenv("GITHUB_TOKEN")
		}
	}()

	// Test with no token (will try keychain first, then env var)
	_ = os.Unsetenv("GITHUB_TOKEN")
	client, err := newAuthenticatedClient()
	if err == nil {
		t.Error("Expected error when no token is available, but got none")
	}
	if client != nil {
		t.Error("Expected nil client when no token is available, but got a client")
	}

	// Test with environment variable token
	_ = os.Setenv("GITHUB_TOKEN", "test-token")
	client, err = newAuthenticatedClient()
	if err != nil {
		t.Errorf("Expected no error when GITHUB_TOKEN is set, but got: %v", err)
	}
	if client == nil {
		t.Error("Expected client when GITHUB_TOKEN is set, but got nil")
	}
}

func TestGitHubTokenIntegration(t *testing.T) {
	// This test verifies that the github package correctly uses the keychain library
	// The actual functionality is tested in the keychain package

	// Save original GITHUB_TOKEN
	originalToken := os.Getenv("GITHUB_TOKEN")
	defer func() {
		if originalToken != "" {
			_ = os.Setenv("GITHUB_TOKEN", originalToken)
		} else {
			_ = os.Unsetenv("GITHUB_TOKEN")
		}
	}()

	// Test that keychain integration works
	_ = os.Setenv("GITHUB_TOKEN", "test-integration-token")
	token, err := keychain.GetGitHubToken()
	if err != nil {
		t.Errorf("Expected no error when env token is set, but got: %v", err)
	}
	if token != "test-integration-token" {
		t.Errorf("Expected token 'test-integration-token', but got: %s", token)
	}
}
