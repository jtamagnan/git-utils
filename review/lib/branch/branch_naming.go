package branch

import (
	"crypto/rand"
	"fmt"
	"os"

	"github.com/jtamagnan/git-utils/git"
)

// getUserIdentifier gets the user identifier from git config or environment variable
func getUserIdentifier() (string, error) {
	// First try git config
	if identifier, err := git.GetConfig("review.user-identifier"); err == nil && identifier != "" {
		return identifier, nil
	}

	// Fall back to USER environment variable
	if user := os.Getenv("USER"); user != "" {
		return user, nil
	}

	// Return error if no identifier found
	return "", fmt.Errorf("no user identifier found: set 'git config review.user-identifier <name>' or ensure USER environment variable is set")
}

// GenerateUUIDBranchName creates a user-prefixed UUID-based branch name for new PRs
func GenerateUUIDBranchName() (string, error) {
	// Get user identifier
	userID, err := getUserIdentifier()
	if err != nil {
		return "", err
	}

	// Generate a simple UUID-like string for branch naming
	bytes := make([]byte, 16)
	rand.Read(bytes)

	// Format as: user/pr/UUID (8-4-4-4-12 characters)
	return fmt.Sprintf("%s/pr/%x-%x-%x-%x-%x",
		userID, bytes[0:4], bytes[4:6], bytes[6:8], bytes[8:10], bytes[10:16]), nil
}
