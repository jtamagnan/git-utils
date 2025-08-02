package github

import (
	"os"
	"strings"
	"testing"
)

// TestAuthenticationFailureDemo demonstrates the authentication behavior
func TestAuthenticationFailureDemo(t *testing.T) {
	// Save original GITHUB_TOKEN
	originalToken := os.Getenv("GITHUB_TOKEN")
	defer func() {
		if originalToken != "" {
			_ = os.Setenv("GITHUB_TOKEN", originalToken)
		} else {
			_ = os.Unsetenv("GITHUB_TOKEN")
		}
	}()

	// Clear the token to simulate a user without authentication
	_ = os.Unsetenv("GITHUB_TOKEN")

	// Try to create a PR - should fail with clear error message
	_, err := CreatePR("testowner", "testrepo", "Test PR", "feature", "main", "Test description", false, []string{})

	if err == nil {
		t.Fatal("Expected authentication error when GITHUB_TOKEN is not set, but got success")
	}

	expectedMessage := "GitHub token not found"
	if !strings.Contains(err.Error(), expectedMessage) {
		t.Errorf("Expected error message to contain '%s', but got: %v", expectedMessage, err)
	}

	t.Logf("✅ Correct authentication failure: %v", err)

	// Try to get existing PR - should also fail
	_, err = GetExistingPR("testowner", "testrepo", 123)

	if err == nil {
		t.Fatal("Expected authentication error when GITHUB_TOKEN is not set, but got success")
	}

	if !strings.Contains(err.Error(), expectedMessage) {
		t.Errorf("Expected error message to contain '%s', but got: %v", expectedMessage, err)
	}

	t.Logf("✅ Correct authentication failure for GetExistingPR: %v", err)

	// Try to get remote branch - should also fail
	_, err = GetRemoteBranchFromPR("testowner", "testrepo", 123)

	if err == nil {
		t.Fatal("Expected authentication error when GITHUB_TOKEN is not set, but got success")
	}

	if !strings.Contains(err.Error(), expectedMessage) {
		t.Errorf("Expected error message to contain '%s', but got: %v", expectedMessage, err)
	}

	t.Logf("✅ Correct authentication failure for GetRemoteBranchFromPR: %v", err)
}
