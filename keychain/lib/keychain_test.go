package keychain

import (
	"os"
	"strings"
	"testing"
)

func TestGetGitHubTokenSources(t *testing.T) {
	// Save original GITHUB_TOKEN
	originalToken := os.Getenv("GITHUB_TOKEN")
	defer func() {
		if originalToken != "" {
			_ = os.Setenv("GITHUB_TOKEN", originalToken)
		} else {
			_ = os.Unsetenv("GITHUB_TOKEN")
		}
	}()

	// Test fallback to environment variable when keychain fails
	_ = os.Unsetenv("GITHUB_TOKEN")
	_, err := GetGitHubToken()
	if err == nil {
		t.Log("Token found (this is OK if you have one in keychain)")
	} else {
		if !strings.Contains(err.Error(), "GitHub token not found") {
			t.Errorf("Expected helpful error message, got: %v", err)
		}
		t.Logf("Expected error when no token sources available: %v", err)
	}

	// Test environment variable works
	_ = os.Setenv("GITHUB_TOKEN", "env-token")
	token, err := GetGitHubToken()
	if err != nil {
		t.Errorf("Expected no error when env token is set, but got: %v", err)
	}
	if token != "env-token" {
		t.Errorf("Expected token 'env-token', but got: %s", token)
	}
}

func TestGetTokenFromKeychain(t *testing.T) {
	// This test just verifies the function exists and handles errors gracefully
	// We can't easily test the actual keychain without setting up test credentials
	_, err := GetTokenFromKeychain()
	// Should return an error since we don't have a test token in keychain
	if err == nil {
		t.Log("Keychain token found (this is fine if you have one set up)")
	} else {
		t.Logf("Keychain token not found (expected): %v", err)
	}
}

func TestIsValidGitHubToken(t *testing.T) {
	tests := []struct {
		token string
		valid bool
	}{
		{"ghp_abc123def456", true},
		{"gho_xyz789", true},
		{"ghu_test", true},
		{"ghs_secret", true},
		{"ghr_refresh", true},
		{"invalid_token", false},
		{"", false},
		{"random_string", false},
		{"github_token_123", false},
	}

	for _, test := range tests {
		result := IsValidGitHubToken(test.token)
		if result != test.valid {
			t.Errorf("IsValidGitHubToken(%q) = %v, expected %v", test.token, result, test.valid)
		}
	}
}

func TestHasExistingTokenHandlesErrors(t *testing.T) {
	// This test verifies that HasExistingToken doesn't crash and returns a boolean
	result := HasExistingToken()
	t.Logf("HasExistingToken() returned: %v", result)

	// The result can be true or false depending on whether user has a token,
	// but it should not panic or return an error
}

func TestErrorMessageFormat(t *testing.T) {
	// Save original GITHUB_TOKEN
	originalToken := os.Getenv("GITHUB_TOKEN")
	defer func() {
		if originalToken != "" {
			_ = os.Setenv("GITHUB_TOKEN", originalToken)
		} else {
			_ = os.Unsetenv("GITHUB_TOKEN")
		}
	}()

	// Clear environment variable to trigger error
	_ = os.Unsetenv("GITHUB_TOKEN")

	// This will only fail if keychain also fails (which is expected in test environment)
	_, err := GetGitHubToken()
	if err != nil {
		errorMsg := err.Error()

		if !strings.Contains(errorMsg, "GitHub token not found") {
			t.Errorf("Expected error to mention 'GitHub token not found', got: %v", err)
		}

		if !strings.Contains(errorMsg, "keychain") {
			t.Errorf("Expected error to mention keychain option, got: %v", err)
		}

		if !strings.Contains(errorMsg, "GITHUB_TOKEN") {
			t.Errorf("Expected error to mention GITHUB_TOKEN option, got: %v", err)
		}

		if !strings.Contains(errorMsg, "go run ./keychain") {
			t.Errorf("Expected error to mention correct keychain command, got: %v", err)
		}

		t.Logf("Proper error message format: %v", err)
	} else {
		t.Log("Token found in keychain - error message test skipped")
	}
}
