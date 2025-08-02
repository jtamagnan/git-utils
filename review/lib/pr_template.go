package lint

import (
	"embed"
	"os"
)

//go:embed default_pull_request_template.md
var defaultTemplate embed.FS

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

	// No template found, return default content from embedded file
	content, err := defaultTemplate.ReadFile("default_pull_request_template.md")
	if err != nil {
		// This should never happen since the file is embedded at compile time
		panic("Failed to read embedded default template: " + err.Error())
	}

	return string(content)
}