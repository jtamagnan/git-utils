package keychain

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// GetGitHubToken retrieves the GitHub token from keychain or environment
func GetGitHubToken() (string, error) {
	// First try macOS keychain
	if token, err := GetTokenFromKeychain(); err == nil && token != "" {
		return token, nil
	}

	// Fall back to environment variable
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		return token, nil
	}

	return "", fmt.Errorf("GitHub token not found. Please either:\n" +
		"  1. Add token to keychain: go run ./keychain\n" +
		"  2. Set environment variable: export GITHUB_TOKEN=your_token")
}

// GetTokenFromKeychain retrieves the GitHub token from macOS keychain
func GetTokenFromKeychain() (string, error) {
	cmd := exec.Command("security", "find-generic-password",
		"-s", "git-review",           // service name
		"-a", "github-token",         // account name
		"-w")                         // return password only

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	token := strings.TrimSpace(string(output))
	if token == "" {
		return "", fmt.Errorf("empty token in keychain")
	}

	return token, nil
}

// StoreTokenInKeychain stores a GitHub token in the macOS keychain
func StoreTokenInKeychain(token string) error {
	// Delete existing entry if it exists
	deleteCmd := exec.Command("security", "delete-generic-password",
		"-s", "git-review",
		"-a", "github-token")
	deleteCmd.Run() // Ignore errors - entry might not exist

	// Add new entry
	cmd := exec.Command("security", "add-generic-password",
		"-s", "git-review",                    // service name
		"-a", "github-token",                  // account name
		"-l", "GitHub Token for git-review",   // label (shown in Keychain Access)
		"-D", "application password",          // kind
		"-w", token)                           // password (the token)

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("security command failed: %v\nOutput: %s", err, output)
	}

	return nil
}

// HasExistingToken checks if a GitHub token already exists in the keychain
func HasExistingToken() bool {
	cmd := exec.Command("security", "find-generic-password",
		"-s", "git-review",
		"-a", "github-token")

	return cmd.Run() == nil
}

// IsValidGitHubToken checks if a token matches the expected GitHub format
func IsValidGitHubToken(token string) bool {
	validPrefixes := []string{"ghp_", "gho_", "ghu_", "ghs_", "ghr_"}
	for _, prefix := range validPrefixes {
		if strings.HasPrefix(token, prefix) {
			return true
		}
	}
	return false
}
