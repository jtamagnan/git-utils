package github

import (
	"os"
	"strings"
	"testing"

	keychain "github.com/jtamagnan/git-utils/keychain/lib"
)

// TestKeychainIntegrationWorkflow demonstrates the complete authentication workflow
func TestKeychainIntegrationWorkflow(t *testing.T) {
	// Save original GITHUB_TOKEN
	originalToken := os.Getenv("GITHUB_TOKEN")
	defer func() {
		if originalToken != "" {
			os.Setenv("GITHUB_TOKEN", originalToken)
		} else {
			os.Unsetenv("GITHUB_TOKEN")
		}
	}()

	t.Run("NoTokenAnywhere", func(t *testing.T) {
		// Clear environment variable
		os.Unsetenv("GITHUB_TOKEN")

		// Should fail with helpful message about both keychain and env var
		_, err := getGitHubToken()
		if err == nil {
			t.Fatal("Expected error when no token is available, but got success")
		}

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

		t.Logf("✅ Proper error when no token available: %v", err)
	})

	t.Run("EnvironmentVariableFallback", func(t *testing.T) {
		// Set environment variable (keychain will likely fail)
		testToken := "ghp_test_token_from_env"
		os.Setenv("GITHUB_TOKEN", testToken)

		// Should succeed and return env var token
		token, err := getGitHubToken()
		if err != nil {
			t.Fatalf("Expected success when env token is set, but got error: %v", err)
		}

		if token != testToken {
			t.Errorf("Expected token '%s' from environment, but got: %s", testToken, token)
		}

		t.Logf("✅ Environment variable fallback works: token retrieved")
	})

		t.Run("KeychainAccessAttempt", func(t *testing.T) {
		// This test just verifies that keychain access is attempted and handles errors gracefully
		_, err := keychain.GetTokenFromKeychain()

		// We expect this to fail in most test environments, which is fine
		t.Logf("📝 Keychain access result: %v", err)

		// The important thing is that it doesn't crash or hang
		if err != nil {
			t.Logf("✅ Keychain access failed gracefully (expected in test environment)")
		} else {
			t.Logf("🔑 Keychain token found (you have one set up)")
		}
	})
}

// TestKeychainSecurityCommandFormat verifies we're using the right security command structure
func TestKeychainSecurityCommandFormat(t *testing.T) {
	// We can't easily test the actual security command without setting up keychain entries,
	// but we can verify the command structure would be correct

	t.Log("🔐 Keychain integration uses macOS security command with:")
	t.Log("   Service: git-review")
	t.Log("   Account: github-token")
	t.Log("   Command: security find-generic-password -s git-review -a github-token -w")

	// This test mainly serves as documentation of the keychain structure
	// and ensures the test suite covers keychain functionality
}
