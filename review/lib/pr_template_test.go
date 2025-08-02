package lint

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
		defaultContent := findPRTemplate()
		// The default should now come from default_pull_request_template.md
		if !isDefaultTemplate(defaultContent) {
			t.Errorf("Expected default template from embedded file, got: %q", defaultContent)
		}

		// Test 2: Create .github directory and add template
		err := os.MkdirAll(".github", 0755)
		if err != nil {
			t.Fatalf("Failed to create .github directory: %v", err)
		}

		templateContent := "## What does this PR do?\n\nDescribe your changes here.\n\n## Checklist\n\n- [ ] Tests added\n- [ ] Documentation updated\n"
		err = os.WriteFile(".github/pull_request_template.md", []byte(templateContent), 0644)
		if err != nil {
			t.Fatalf("Failed to write template file: %v", err)
		}

		// Should now return the template content
		foundContent := findPRTemplate()
		if foundContent != templateContent {
			t.Errorf("Expected template content %q, got %q", templateContent, foundContent)
		}

		// Test 3: Test root directory template (simpler test)
		rootContent := "# ROOT TEMPLATE\n\nTemplate in root directory.\n"
		err = os.WriteFile("pull_request_template.md", []byte(rootContent), 0644)
		if err != nil {
			t.Fatalf("Failed to write root template: %v", err)
		}

		// Should still prefer .github/ template over root
		foundContent = findPRTemplate()
		if foundContent != templateContent {
			t.Errorf("Expected .github template to have priority over root, got: %q", foundContent)
		}

		// Test 4: Remove .github template, should fall back to root
		err = os.Remove(".github/pull_request_template.md")
		if err != nil {
			t.Fatalf("Failed to remove .github template: %v", err)
		}

		foundContent = findPRTemplate()
		if foundContent != rootContent {
			t.Errorf("Expected root template after removing .github template, got: %q", foundContent)
		}

		// Test 5: Remove root template, should fall back to default
		err = os.Remove("pull_request_template.md")
		if err != nil {
			t.Fatalf("Failed to remove root template: %v", err)
		}

		foundContent = findPRTemplate()
		if !isDefaultTemplate(foundContent) {
			t.Errorf("Expected default template after removing all templates, got: %q", foundContent)
		}
	})
}

// isDefaultTemplate checks if the content matches our default template structure
func isDefaultTemplate(content string) bool {
	// Check for key sections that should be in our default template
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