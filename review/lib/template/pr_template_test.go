package template

import (
	"os"
	"strings"
	"testing"

	"github.com/jtamagnan/git-utils/git"
)

func TestFindPRTemplate(t *testing.T) {
	testRepo := git.NewTestRepo(t)
	defer testRepo.Cleanup()

	testRepo.InDir(func() {
		// Test 1: No template files exist - should return default from embedded file
		defaultContent := FindPRTemplate()
		if !isDefaultTemplate(defaultContent) {
			t.Errorf("Expected default template from embedded file, got: %q", defaultContent)
		}

		// Test 2: Template in .github/ directory
		err := os.MkdirAll(".github", 0755)
		if err != nil {
			t.Fatalf("Failed to create .github directory: %v", err)
		}

		githubTemplate := "## GitHub Template\n\nThis is a custom template from .github/"
		err = os.WriteFile(".github/pull_request_template.md", []byte(githubTemplate), 0644)
		if err != nil {
			t.Fatalf("Failed to write .github template: %v", err)
		}

		content := FindPRTemplate()
		if content != githubTemplate {
			t.Errorf("Expected GitHub template, got: %q", content)
		}

		// Test 3: Template in root directory (should be lower priority)
		rootTemplate := "## Root Template\n\nThis is from the root directory"
		err = os.WriteFile("pull_request_template.md", []byte(rootTemplate), 0644)
		if err != nil {
			t.Fatalf("Failed to write root template: %v", err)
		}

		// Should still prefer .github/ version
		content = FindPRTemplate()
		if content != githubTemplate {
			t.Errorf("Expected GitHub template to have priority, got: %q", content)
		}

		// Test 4: Remove .github template, should fall back to root
		err = os.Remove(".github/pull_request_template.md")
		if err != nil {
			t.Fatalf("Failed to remove .github template: %v", err)
		}

		content = FindPRTemplate()
		if content != rootTemplate {
			t.Errorf("Expected root template after removing .github version, got: %q", content)
		}

		// Test 5: Remove all templates, should return default again
		err = os.Remove("pull_request_template.md")
		if err != nil {
			t.Fatalf("Failed to remove root template: %v", err)
		}

		content = FindPRTemplate()
		if !isDefaultTemplate(content) {
			t.Errorf("Expected default template after removing all custom templates, got: %q", content)
		}
	})
}

// isDefaultTemplate checks if the content matches our default template structure
func isDefaultTemplate(content string) bool {
	expectedSections := []string{
		"## Summary",
		"## Motivation",
		"## Risks",
		"## Test plan",
		"## Rollout plan",
		"## Is this safe to roll back?",
	}

	for _, section := range expectedSections {
		if !strings.Contains(content, section) {
			return false
		}
	}

	return true
}