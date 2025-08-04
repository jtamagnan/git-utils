package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	keychain "github.com/jtamagnan/git-utils/keychain/lib"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func runE(cmd *cobra.Command, args []string) error {
	fmt.Println("GitHub Token Keychain Setup for git-review")
	fmt.Println()

	if runtime.GOOS != "darwin" {
		fmt.Println("This tool is only supported on macOS")
		return fmt.Errorf("macOS required")
	}

	// Check if token already exists
	if keychain.HasExistingToken() {
		fmt.Println("GitHub token already found in keychain.")

		// Prompt user if they want to keep it or replace it
		fmt.Print("Do you want to keep the existing token? (y/n): ")
		var response string
		_, _ = fmt.Scanln(&response)

		if strings.ToLower(response) == "y" || strings.ToLower(response) == "yes" {
			fmt.Println("Keeping existing token.")
			return nil
		}
	}

	// Prompt user to create token
	fmt.Println("Please create a GitHub Personal Access Token:")
	fmt.Println("1. Go to: https://github.com/settings/tokens")
	fmt.Println("2. Click 'Generate new token' -> 'Generate new token (classic)'")
	fmt.Println("3. Give it a descriptive name (e.g., 'git-review CLI')")
	fmt.Println("4. Select scopes: 'repo' (for private repos) or 'public_repo' (for public only)")
	fmt.Println("5. Click 'Generate token' and copy the token")
	fmt.Println()
	fmt.Print("Enter your GitHub token (input will be hidden): ")

	// Read token from stdin (hidden input)
	tokenBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println() // New line after hidden input
	if err != nil {
		return fmt.Errorf("failed to read token: %v", err)
	}

	token := strings.TrimSpace(string(tokenBytes))
	if token == "" {
		return fmt.Errorf("no token provided")
	}

	// Validate token format using the keychain library function
	if !keychain.IsValidGitHubToken(token) {
		fmt.Println("Warning: Token doesn't match expected GitHub format (should start with ghp_, gho_, etc.)")
		fmt.Print("Continue anyway? (y/n): ")
		var response string
		_, _ = fmt.Scanln(&response)
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			return fmt.Errorf("token storage cancelled")
		}
	}

	// Store in keychain
	err = keychain.StoreTokenInKeychain(token)
	if err != nil {
		return fmt.Errorf("failed to store token in keychain: %v", err)
	}

	fmt.Println("GitHub token successfully stored in keychain!")
	fmt.Println()
	fmt.Println("You can now use git-review without setting GITHUB_TOKEN:")
	fmt.Println("  git review")
	fmt.Println()
	fmt.Println("To update the token later, run this tool again:")
	fmt.Println("  git keychain")

	return nil
}

func generateCommand() *cobra.Command {
	var rootCmd = &cobra.Command{
		Use:   "git-keychain",
		Short: "Manage GitHub tokens securely in macOS keychain",
		Long: `Store and manage GitHub Personal Access Tokens securely in the macOS keychain.

This tool allows you to securely store your GitHub token in the macOS keychain
instead of using environment variables. The stored token will be automatically
used by git-review and other git-utils tools.`,
		RunE:         runE,
		SilenceUsage: true,
	}

	return rootCmd
}

func main() {
	rootCmd := generateCommand()
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
