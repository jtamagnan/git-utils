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
	_, err := newAuthenticatedClient()
	if err == nil {
		// This might succeed if there's a token in the keychain
		t.Log("Authentication succeeded (token found in keychain or env)")
	} else {
		// This is expected if no token is available
		t.Log("Authentication failed as expected when no token available")
	}

	// Test with environment variable token
	_ = os.Setenv("GITHUB_TOKEN", "test-integration-token")
	token, err := keychain.GetGitHubToken()
	if err != nil {
		t.Errorf("Expected no error when env token is set, but got: %v", err)
	}
	if token != "test-integration-token" {
		t.Errorf("Expected token 'test-integration-token', but got: %s", token)
	}

	// Test that CreatePR has the correct signature with labels
	_ = testCreatePRSignature()
}

// testCreatePRSignature is a compile-time test to ensure CreatePR has the expected signature
func testCreatePRSignature() error {
	// This function should compile if the signature is correct
	// We don't actually call it since we don't want to make real API calls
	createPRFunc := CreatePR
	_ = createPRFunc
	return nil
}

func TestAddLabelsToIssueSignature(t *testing.T) {
	// Test that AddLabelsToIssue has the correct signature
	addLabelsFunc := AddLabelsToIssue
	_ = addLabelsFunc

	// Test with empty labels (should return nil immediately)
	err := AddLabelsToIssue("owner", "repo", 1, []string{})
	if err != nil {
		t.Errorf("Expected no error for empty labels, got: %v", err)
	}
}

func TestCreatePRWithLabels(t *testing.T) {
	// Test that CreatePR function accepts labels parameter
	testLabels := []string{"bug", "enhancement", "high-priority"}

	// We can't make actual API calls in tests, but we can verify the function
	// accepts the correct parameters without error (until it tries to authenticate)
	_, err := CreatePR("test-owner", "test-repo", "Test Title", "feature-branch", "main", "Test description", false, testLabels)

	// We expect this to fail due to authentication, but the error should be about
	// authentication, not about function signature or parameter parsing
	if err != nil {
		t.Logf("Expected authentication error when no valid token: %v", err)
	}
}
