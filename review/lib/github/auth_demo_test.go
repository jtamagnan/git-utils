package github

import (
	keychain "github.com/jtamagnan/git-utils/keychain/lib"
	"os"
	"strings"
	"testing"
)

// TestAuthenticationFailureDemo demonstrates the authentication behavior
func TestAuthenticationFailureDemo(t *testing.T) {
	// This test demonstrates what happens when no GitHub token is available
	// Note: If you have tokens in your keychain, this test may not behave as expected

	// Unset environment variable to force using keychain (if available)
	originalToken := os.Getenv("GITHUB_TOKEN")
	defer func() {
		if originalToken != "" {
			_ = os.Setenv("GITHUB_TOKEN", originalToken)
		} else {
			_ = os.Unsetenv("GITHUB_TOKEN")
		}
	}()

	_ = os.Unsetenv("GITHUB_TOKEN")

	// Check if we have a keychain token
	if keychain.HasExistingToken() {
		t.Log("Keychain token exists - this test will demonstrate API calls with authentication")
		t.Log("The calls will fail with 404 (not found) rather than authentication errors")

		// With authentication, we'll get 404 errors for non-existent repos
		_, err := CreatePR("testowner", "testrepo", "Test PR", "feature", "main", "Test description", false, []string{}, []string{})
		if err == nil {
			t.Fatal("Expected error for non-existent repository, but got success")
		}

		if strings.Contains(err.Error(), "404 Not Found") {
			t.Logf("Expected 404 error for non-existent repo (authentication worked): %v", err)
		} else {
			t.Errorf("Expected 404 error, but got: %v", err)
		}
		return
	}

	// No keychain token - test actual authentication failure
	t.Log("No keychain token found - testing authentication failure scenarios")

	_, err := CreatePR("testowner", "testrepo", "Test PR", "feature", "main", "Test description", false, []string{}, []string{})

	if err == nil {
		t.Fatal("Expected authentication error when GITHUB_TOKEN is not set, but got success")
	}

	expectedMessage := "GitHub token not found"
	if !strings.Contains(err.Error(), expectedMessage) {
		t.Errorf("Expected error message to contain '%s', but got: %v", expectedMessage, err)
	}

	t.Logf("Correct authentication failure: %v", err)

	// Try to get existing PR - should also fail
	_, err = GetExistingPR("testowner", "testrepo", 123)

	if err == nil {
		t.Fatal("Expected authentication error when GITHUB_TOKEN is not set, but got success")
	}

	if !strings.Contains(err.Error(), expectedMessage) {
		t.Errorf("Expected error message to contain '%s', but got: %v", expectedMessage, err)
	}

	t.Logf("Correct authentication failure for GetExistingPR: %v", err)

	// Try to get remote branch - should also fail
	_, err = GetRemoteBranchFromPR("testowner", "testrepo", 123)

	if err == nil {
		t.Fatal("Expected authentication error when GITHUB_TOKEN is not set, but got success")
	}

	if !strings.Contains(err.Error(), expectedMessage) {
		t.Errorf("Expected error message to contain '%s', but got: %v", expectedMessage, err)
	}

	t.Logf("Correct authentication failure for GetRemoteBranchFromPR: %v", err)
}
