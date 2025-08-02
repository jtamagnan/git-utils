package main

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strings"
	"syscall"

	keychain "github.com/jtamagnan/git-utils/keychain/lib"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func runE(cmd *cobra.Command, args []string) error {
	fmt.Println("🔐 GitHub Token Keychain Setup for git-review")
	fmt.Println()

	// Check if we're on macOS
	if !isMacOS() {
		fmt.Println("❌ This tool is only supported on macOS")
		fmt.Println("Please use the GITHUB_TOKEN environment variable instead.")
		return fmt.Errorf("macOS required")
	}

	// Check if token already exists
	if keychain.HasExistingToken() {
		fmt.Println("🔍 Found existing GitHub token in keychain.")
		fmt.Print("Do you want to update it? (y/N): ")

		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response != "y" && response != "yes" {
			fmt.Println("✅ Keeping existing token.")
			return nil
		}
	}

	// Prompt for token
	fmt.Println()
	fmt.Println("📝 Please create a GitHub Personal Access Token:")
	fmt.Println("   1. Go to: https://github.com/settings/tokens")
	fmt.Println("   2. Click 'Generate new token (classic)'")
	fmt.Println("   3. Select scope: 'repo' (Full control of private repositories)")
	fmt.Println("   4. Copy the generated token")
	fmt.Println()

	fmt.Print("🔑 Enter your GitHub token (input will be hidden): ")

	// Read token securely (hidden input)
	tokenBytes, err := term.ReadPassword(syscall.Stdin)
	if err != nil {
		return fmt.Errorf("error reading token: %v", err)
	}

	token := strings.TrimSpace(string(tokenBytes))
	fmt.Println() // New line after hidden input

	if token == "" {
		return fmt.Errorf("no token provided")
	}

	// Validate token format (GitHub tokens start with ghp_, gho_, ghu_, ghs_, or ghr_)
	if !keychain.IsValidGitHubToken(token) {
		fmt.Println("⚠️  Warning: Token doesn't match expected GitHub format (should start with ghp_, gho_, etc.)")
		fmt.Print("Continue anyway? (y/N): ")

		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response != "y" && response != "yes" {
			return fmt.Errorf("cancelled by user")
		}
	}

	// Store in keychain
	if err := keychain.StoreTokenInKeychain(token); err != nil {
		return fmt.Errorf("failed to store token in keychain: %v", err)
	}

	fmt.Println("✅ GitHub token successfully stored in keychain!")
	fmt.Println()
	fmt.Println("🚀 You can now use git-review without setting GITHUB_TOKEN:")
	fmt.Println("   git review")
	fmt.Println()
	fmt.Println("🔧 To update the token later, run this tool again:")
	fmt.Println("   git keychain")

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

func isMacOS() bool {
	return runtime.GOOS == "darwin"
}
