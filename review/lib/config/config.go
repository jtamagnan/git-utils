package config

import (
	"fmt"
	"strings"

	"github.com/jtamagnan/git-utils/git"
	review "github.com/jtamagnan/git-utils/review/lib"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// CommaString represents a comma-separated list of strings
type CommaString []string

// String returns the comma-separated string representation
func (cs CommaString) String() string {
	return strings.Join(cs, ",")
}

// Set parses a comma-separated string and updates the CommaString
func (cs *CommaString) Set(value string) error {
	if value == "" {
		*cs = CommaString{}
		return nil
	}

	parts := strings.Split(value, ",")
	var result []string
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		// Filter out empty strings
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	*cs = CommaString(result)
	return nil
}

// Type returns the type name for cobra flag usage
func (cs *CommaString) Type() string {
	return "commastring"
}

// ToStringSlice converts CommaString to []string
func (cs CommaString) ToStringSlice() []string {
	return []string(cs)
}

// FromStringSlice creates a CommaString from []string
func FromStringSlice(slice []string) CommaString {
	return CommaString(slice)
}

// ParseCommaString creates a CommaString from a comma-separated string
func ParseCommaString(value string) CommaString {
	var cs CommaString
	_ = cs.Set(value) // Set never returns an error, but satisfy the linter
	return cs
}

// Append adds new items to the CommaString, avoiding duplicates
func (cs *CommaString) Append(items ...string) {
	existing := make(map[string]bool)
	for _, item := range *cs {
		existing[item] = true
	}

	for _, item := range items {
		if item != "" && !existing[item] {
			*cs = append(*cs, item)
			existing[item] = true
		}
	}
}

// Contains checks if the CommaString contains a specific item
func (cs CommaString) Contains(item string) bool {
	for _, existing := range cs {
		if existing == item {
			return true
		}
	}
	return false
}

// FlagConfig represents a complete flag configuration
type FlagConfig struct {
	Name        string
	Shorthand   string
	Type        string // "bool", "string", or "commastring"
	Default     interface{}
	Description string
}

// Define all flags in one place - single source of truth
var flagConfigs = []FlagConfig{
	{
		Name:        "no-verify",
		Shorthand:   "v",
		Type:        "bool",
		Default:     false,
		Description: "Skip the pre-push checks",
	},
	{
		Name:        "open-browser",
		Shorthand:   "b",
		Type:        "bool",
		Default:     true,
		Description: "Open the pull request in the browser",
	},
	{
		Name:        "draft",
		Shorthand:   "d",
		Type:        "bool",
		Default:     false,
		Description: "Create the pull request as a draft",
	},
	{
		Name:        "labels",
		Shorthand:   "l",
		Type:        "commastring",
		Default:     CommaString{},
		Description: "Comma-separated list of labels to add to the PR (e.g., 'bug,enhancement')",
	},
}

// splitAndTrim splits a comma-separated string and trims whitespace from each element
func splitAndTrim(value string) []string {
	return ParseCommaString(value).ToStringSlice()
}

// getGitConfigList reads a git config value, splits it by commas, and calls the handler if non-empty
func getGitConfigList(key string, handler func([]string)) {
	if value, err := git.GetConfig(key); err == nil && value != "" {
		handler(splitAndTrim(value))
	}
}

// getGitConfigString reads a git config value and calls the handler if non-empty
func getGitConfigString(key string, handler func(string)) {
	if value, err := git.GetConfig(key); err == nil && value != "" {
		handler(value)
	}
}

// InitConfig sets up Viper configuration with defaults, environment variables, and config files
func InitConfig() {
	// Set defaults from flag configurations
	for _, flag := range flagConfigs {
		if flag.Type == "commastring" {
			// For CommaString types, set as empty slice in viper
			viper.SetDefault(flag.Name, []string{})
		} else {
			viper.SetDefault(flag.Name, flag.Default)
		}
	}

	// Support environment variables with REVIEW_ prefix
	viper.SetEnvPrefix("REVIEW")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	// Set up global config file locations (user-level only)
	viper.SetConfigName("git-review")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("$HOME/.config") // ~/.config/git-review.yaml
	viper.AddConfigPath("$HOME")         // ~/.git-review.yaml

	// Try to read user-level config file
	if err := viper.ReadInConfig(); err == nil {
		fmt.Printf("Using config file: %s\n", viper.ConfigFileUsed())
	}

	// Note: We intentionally do NOT read project-level config files for
	// behavioral settings like open-browser, draft, labels, etc.
	// These should remain user preferences, not project settings.
}

// LoadProjectConfig reads project-specific settings that ARE allowed per repository
func LoadProjectConfig() {
	// Default reviewers for this project
	getGitConfigList("review.default-reviewers", func(reviewers []string) {
		viper.Set("project.default-reviewers", reviewers)
	})

	// Additional project-specific labels (these are ADDED to user labels, not replaced)
	getGitConfigList("review.labels", func(projectLabels []string) {
		existingLabels := viper.GetStringSlice("labels")

		// Use CommaString for clean merging without duplicates
		labels := FromStringSlice(existingLabels)
		labels.Append(projectLabels...)

		viper.Set("labels", labels.ToStringSlice())
	})

	// Custom branch prefix for this project
	getGitConfigString("review.branch-prefix", func(branchPrefix string) {
		viper.Set("project.branch-prefix", branchPrefix)
	})

	// Note: Behavioral settings like open-browser, draft, no-verify are intentionally
	// NOT configurable via git config to maintain user control and prevent
	// projects from overriding user workflow preferences.
}

// ParseArgs converts Viper configuration and command-line flags into ParsedArgs
func ParseArgs(cmd *cobra.Command, _ []string) (review.ParsedArgs, error) {
	// Viper automatically handles the precedence:
	// 1. Command-line flags (highest)
	// 2. Environment variables
	// 3. User config files
	// 4. Git config (project-specific settings)
	// 5. Defaults (lowest)

	// Handle CommaString parsing for labels
	// If labels was provided as a comma-separated string via command line, parse it
	if cmd.Flags().Changed("labels") {
		labelStr := viper.GetString("labels")
		if labelStr != "" {
			labels := ParseCommaString(labelStr).ToStringSlice()
			viper.Set("labels", labels)
		}
	}

	// Use flag names from configuration to get values
	parsedArgs := review.ParsedArgs{
		NoVerify:    viper.GetBool("no-verify"),
		OpenBrowser: viper.GetBool("open-browser"),
		Draft:       viper.GetBool("draft"),
		Labels:      viper.GetStringSlice("labels"),
	}

	return parsedArgs, nil
}

// SetupFlags defines and binds command-line flags to Viper using the flag configurations
func SetupFlags(cmd *cobra.Command) {
	for _, flag := range flagConfigs {
		switch flag.Type {
		case "bool":
			cmd.Flags().BoolP(flag.Name, flag.Shorthand, flag.Default.(bool), flag.Description)
		case "string":
			cmd.Flags().StringP(flag.Name, flag.Shorthand, flag.Default.(string), flag.Description)
		case "commastring":
			// For CommaString, we register it as a string flag but with custom parsing
			defaultVal := flag.Default.(CommaString).String()
			cmd.Flags().StringP(flag.Name, flag.Shorthand, defaultVal, flag.Description)
		}

		// Bind to viper for automatic precedence handling
		_ = viper.BindPFlag(flag.Name, cmd.Flags().Lookup(flag.Name))
	}
}
