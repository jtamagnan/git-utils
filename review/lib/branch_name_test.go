package review

import (
	"testing"
)

func TestStripRemotePrefix(t *testing.T) {
	tests := []struct {
		branch   string
		remote   string
		expected string
	}{
		// Simple branch names (no prefix)
		{"main", "origin", "main"},
		{"develop", "origin", "develop"},
		{"feature-branch", "upstream", "feature-branch"},

		// Branch names with correct remote prefix
		{"origin/main", "origin", "main"},
		{"origin/develop", "origin", "develop"},
		{"upstream/main", "upstream", "main"},
		{"fork/feature-branch", "fork", "feature-branch"},

		// Branch names with different remote (no change)
		{"origin/main", "upstream", "origin/main"},
		{"upstream/develop", "origin", "upstream/develop"},

		// Branch names with multiple slashes (feature branches)
		{"origin/feature/my-feature", "origin", "feature/my-feature"},
		{"upstream/bugfix/issue-123", "upstream", "bugfix/issue-123"},

		// Edge cases
		{"", "origin", ""},
		{"single", "origin", "single"},
		{"origin/a/b/c/d", "origin", "a/b/c/d"},

		// No remote prefix
		{"main", "", "main"},
		{"develop", "", "develop"},
	}

	for _, test := range tests {
		result := stripRemotePrefix(test.branch, test.remote)
		if result != test.expected {
			t.Errorf("stripRemotePrefix(%q, %q) = %q, expected %q",
				test.branch, test.remote, result, test.expected)
		}
	}
}
