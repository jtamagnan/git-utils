package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/jtamagnan/git-utils/git"
	review "github.com/jtamagnan/git-utils/review/lib"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// initConfig sets up Viper configuration
func initConfig() {
	// Set hardcoded defaults for user-level settings
	viper.SetDefault("open-browser", true)
	viper.SetDefault("draft", false)
	viper.SetDefault("no-verify", false)
	viper.SetDefault("labels", []string{})

	// Support environment variables with REVIEW_ prefix
	viper.SetEnvPrefix("REVIEW")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	// Set up global config file locations (user-level only)
	// These settings should NOT be configurable per repository
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

// getProjectConfig reads project-specific settings that ARE allowed per repository
func getProjectConfig() {
	// Read git config values for project-specific settings

	// Default reviewers for this project
	if reviewers, err := git.GetConfig("review.defaultReviewers"); err == nil && reviewers != "" {
		reviewerList := strings.Split(reviewers, ",")
		// Trim whitespace from each reviewer
		for i, reviewer := range reviewerList {
			reviewerList[i] = strings.TrimSpace(reviewer)
		}
		viper.Set("project.default-reviewers", reviewerList)
	}

	// Additional project-specific labels (these are ADDED to user labels, not replaced)
	if projectLabels, err := git.GetConfig("review.projectLabels"); err == nil && projectLabels != "" {
		labelList := strings.Split(projectLabels, ",")
		// Trim whitespace from each label
		for i, label := range labelList {
			labelList[i] = strings.TrimSpace(label)
		}

		// Merge with existing labels (user-defined labels take precedence)
		existingLabels := viper.GetStringSlice("labels")
		allLabels := append(existingLabels, labelList...)
		viper.Set("labels", allLabels)
	}

	// Custom branch prefix for this project (defaults to user identifier)
	if branchPrefix, err := git.GetConfig("review.branchPrefix"); err == nil && branchPrefix != "" {
		viper.Set("project.branch-prefix", branchPrefix)
	}

	// Note: Behavioral settings like open-browser, draft, no-verify are intentionally
	// NOT configurable via git config to maintain user control and prevent
	// projects from overriding user workflow preferences.
}

func parseArgs(cmd *cobra.Command, _ []string) (review.ParsedArgs, error) {
	// Viper automatically handles the precedence:
	// 1. Command-line flags (highest)
	// 2. Environment variables
	// 3. User config files
	// 4. Defaults (lowest)

	// Handle labels parsing from comma-separated string
	var labels []string
	if labelsStr := viper.GetString("labels"); labelsStr != "" {
		labels = strings.Split(labelsStr, ",")
		// Trim whitespace from each label
		for i, label := range labels {
			labels[i] = strings.TrimSpace(label)
		}
	}

	parsedArgs := review.ParsedArgs{
		NoVerify:    viper.GetBool("no-verify"),
		OpenBrowser: viper.GetBool("open-browser"),
		Draft:       viper.GetBool("draft"),
		Labels:      labels,
	}

	return parsedArgs, nil
}

func runE(cmd *cobra.Command, args []string) error {
	parsedArgs, err := parseArgs(cmd, args)
	if err != nil {
		return err
	}

	err = review.Review(parsedArgs)
	if err != nil {
		return err
	}
	return nil
}

func generateCommand() *cobra.Command {
	var rootCmd = &cobra.Command{
		Use:   "git-review",
		Short: "Open a pull request for this repository.",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			initConfig()
			getProjectConfig()
		},
		RunE: runE,
	}

	// Define flags with hardcoded defaults (Viper will override these after config is loaded)
	rootCmd.Flags().BoolP("no-verify", "v", false, "Skip the pre-push checks")
	rootCmd.Flags().BoolP("open-browser", "b", true, "Open the pull request in the browser")
	rootCmd.Flags().BoolP("draft", "d", false, "Create the pull request as a draft")
	rootCmd.Flags().StringP("labels", "l", "", "Comma-separated list of labels to add to the PR (e.g., 'bug,enhancement')")

	// Bind flags to viper for automatic precedence handling
	_ = viper.BindPFlag("no-verify", rootCmd.Flags().Lookup("no-verify"))
	_ = viper.BindPFlag("open-browser", rootCmd.Flags().Lookup("open-browser"))
	_ = viper.BindPFlag("draft", rootCmd.Flags().Lookup("draft"))
	_ = viper.BindPFlag("labels", rootCmd.Flags().Lookup("labels"))

	return rootCmd
}

func main() {
	rootCmd := generateCommand()
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
