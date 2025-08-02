package main

import (
	"os"
	"strconv"
	"strings"

	"github.com/jtamagnan/git-utils/git"
	review "github.com/jtamagnan/git-utils/review/lib"
	"github.com/spf13/cobra"
)

// getOpenBrowserDefault gets the default value for --open-browser from git config
func getOpenBrowserDefault() bool {
	// Try to get the setting from git config
	if value, err := git.GetConfig("review.openBrowser"); err == nil && value != "" {
		// Parse boolean value (git config uses "true"/"false" strings)
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}

	// Default to true if not configured or invalid
	return true
}

func parseArgs(cmd *cobra.Command, _ []string) (review.ParsedArgs, error) {
	parsedArgs := review.ParsedArgs{}

	noVerify, err := cmd.Flags().GetBool("no-verify")
	if err != nil {
		return parsedArgs, err
	}
	parsedArgs.NoVerify = noVerify

	openBrowser, err := cmd.Flags().GetBool("open-browser")
	if err != nil {
		return parsedArgs, err
	}
	parsedArgs.OpenBrowser = openBrowser

	draft, err := cmd.Flags().GetBool("draft")
	if err != nil {
		return parsedArgs, err
	}
	parsedArgs.Draft = draft

	labels, err := cmd.Flags().GetString("labels")
	if err != nil {
		return parsedArgs, err
	}
	// Parse comma-separated labels
	if labels != "" {
		parsedArgs.Labels = strings.Split(labels, ",")
		// Trim whitespace from each label
		for i, label := range parsedArgs.Labels {
			parsedArgs.Labels[i] = strings.TrimSpace(label)
		}
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
		RunE:  runE,
	}

	// Get the default value for open-browser from git config
	openBrowserDefault := getOpenBrowserDefault()

	rootCmd.Flags().BoolP("no-verify", "v", false, "Skip the pre-push checks")
	rootCmd.Flags().BoolP("open-browser", "b", openBrowserDefault, "Open the pull request in the browser (default from git config review.openBrowser)")
	rootCmd.Flags().BoolP("draft", "d", false, "Create the pull request as a draft")
	rootCmd.Flags().StringP("labels", "l", "", "Comma-separated list of labels to add to the PR (e.g., 'bug,enhancement')")
	// TODO(jat): Learn to use viper
	return rootCmd
}

func main() {
	rootCmd := generateCommand()
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
