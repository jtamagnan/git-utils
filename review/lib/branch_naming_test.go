package lint

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestGetUserIdentifier(t *testing.T) {
	// Save original USER env var
	originalUser := os.Getenv("USER")
	defer os.Setenv("USER", originalUser)

	// Test fallback to USER environment variable
	os.Setenv("USER", "testuser")
	userID, err := getUserIdentifier()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if userID != "testuser" {
		t.Errorf("Expected 'testuser', got '%s'", userID)
	}

	// Test with different user name
	os.Setenv("USER", "alice")
	userID, err = getUserIdentifier()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if userID != "alice" {
		t.Errorf("Expected 'alice', got '%s'", userID)
	}

	// Test with empty USER (should return error)
	os.Setenv("USER", "")
	userID, err = getUserIdentifier()
	if err == nil {
		t.Errorf("Expected error for empty USER, but got userID: '%s'", userID)
	}
	if userID != "" {
		t.Errorf("Expected empty userID on error, got '%s'", userID)
	}
}

func TestGenerateUUIDBranchName(t *testing.T) {
	// Save original USER env var
	originalUser := os.Getenv("USER")
	defer os.Setenv("USER", originalUser)

	// Set a known user for consistent testing
	os.Setenv("USER", "testuser")

	branch1, err := generateUUIDBranchName()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	// Verify it starts with the expected prefix
	if !strings.HasPrefix(branch1, "testuser/pr/") {
		t.Errorf("Expected branch name to start with 'testuser/pr/', got: %s", branch1)
	}

	// Verify it matches the expected UUID pattern
	expectedPattern := `^testuser/pr/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`
	matched, _ := regexp.MatchString(expectedPattern, branch1)
	if !matched {
		t.Errorf("Branch name doesn't match UUID pattern. Expected format: testuser/pr/XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX, got: %s", branch1)
	}
}

func TestGenerateUUIDBranchNameWithDifferentUsers(t *testing.T) {
	// Save original USER env var
	originalUser := os.Getenv("USER")
	defer os.Setenv("USER", originalUser)

	// Test with different users
	users := []string{"alice", "bob", "charlie"}

	for _, user := range users {
		os.Setenv("USER", user)
		branch, err := generateUUIDBranchName()
		if err != nil {
			t.Fatalf("Expected no error for user %s, got: %v", user, err)
		}

		expectedPrefix := user + "/pr/"
		if !strings.HasPrefix(branch, expectedPrefix) {
			t.Errorf("Expected branch to start with '%s', got: %s", expectedPrefix, branch)
		}

		// Verify the UUID part after the user prefix
		uuidPart := strings.TrimPrefix(branch, expectedPrefix)
		expectedPattern := `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`
		matched, err := regexp.MatchString(expectedPattern, uuidPart)
		if err != nil {
			t.Fatalf("Failed to compile regex: %v", err)
		}
		if !matched {
			t.Errorf("UUID part doesn't match expected pattern for user %s. Got: %s", user, uuidPart)
		}
	}
}

func TestGenerateUUIDBranchNameUniqueness(t *testing.T) {
	// Save original USER env var
	originalUser := os.Getenv("USER")
	defer os.Setenv("USER", originalUser)

	// Set a known user for testing
	os.Setenv("USER", "testuser")

	// Generate multiple UUIDs and verify they're unique
	generatedUUIDs := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		uuid, err := generateUUIDBranchName()
		if err != nil {
			t.Fatalf("Expected no error on iteration %d, got: %v", i, err)
		}
		if generatedUUIDs[uuid] {
			t.Fatalf("Generated duplicate UUID: %s", uuid)
		}
		generatedUUIDs[uuid] = true
	}
}

func TestGenerateUUIDBranchNameError(t *testing.T) {
	// Save original USER env var
	originalUser := os.Getenv("USER")
	defer os.Setenv("USER", originalUser)

	// Clear USER environment variable
	os.Setenv("USER", "")

	// Should return error when no user identifier is available
	branch, err := generateUUIDBranchName()
	if err == nil {
		t.Errorf("Expected error when no user identifier available, but got branch: %s", branch)
	}
	if branch != "" {
		t.Errorf("Expected empty branch name on error, got: %s", branch)
	}

	// Error message should be helpful
	if !strings.Contains(err.Error(), "no user identifier found") {
		t.Errorf("Expected helpful error message, got: %v", err)
	}
}