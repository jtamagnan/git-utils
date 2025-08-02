package lint

import (
	"os"
)

// findPRTemplate looks for GitHub PR templates in standard locations
func findPRTemplate() string {
	// Common locations for GitHub PR templates (in order of preference)
	templatePaths := []string{
		".github/pull_request_template.md",
		".github/PULL_REQUEST_TEMPLATE.md",
		"pull_request_template.md",
		"PULL_REQUEST_TEMPLATE.md",
	}

	for _, path := range templatePaths {
		if content, err := os.ReadFile(path); err == nil {
			// Found a template, return its content
			return string(content)
		}
	}

	// No template found, return default content
	return "## Description\n\nBrief description of changes.\n\n## Changes\n\n- \n\n## Testing\n\n- \n"
}