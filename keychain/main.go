package main

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strings"
	"syscall"

	keychain "github.com/jtamagnan/git-utils/keychain/lib"
	"golang.org/x/term"
)

func main() {
	fmt.Println("🔐 GitHub Token Keychain Setup for git-review")
	fmt.Println()

	// Check if we're on macOS
	if !isMacOS() {
		fmt.Println("❌ This tool is only supported on macOS")
		fmt.Println("Please use the GITHUB_TOKEN environment variable instead.")
		os.Exit(1)
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
			return
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
	tokenBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		fmt.Printf("\n❌ Error reading token: %v\n", err)
		os.Exit(1)
	}

	token := strings.TrimSpace(string(tokenBytes))
	fmt.Println() // New line after hidden input

	if token == "" {
		fmt.Println("❌ No token provided.")
		os.Exit(1)
	}

		// Validate token format (GitHub tokens start with ghp_, gho_, ghu_, ghs_, or ghr_)
	if !keychain.IsValidGitHubToken(token) {
		fmt.Println("⚠️  Warning: Token doesn't match expected GitHub format (should start with ghp_, gho_, etc.)")
		fmt.Print("Continue anyway? (y/N): ")

		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response != "y" && response != "yes" {
			fmt.Println("❌ Cancelled.")
			os.Exit(1)
		}
	}

	// Store in keychain
	if err := keychain.StoreTokenInKeychain(token); err != nil {
		fmt.Printf("❌ Failed to store token in keychain: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ GitHub token successfully stored in keychain!")
	fmt.Println()
	fmt.Println("🚀 You can now use git-review without setting GITHUB_TOKEN:")
	fmt.Println("   go run ./review")
	fmt.Println()
	fmt.Println("🔧 To update the token later, run this tool again:")
	fmt.Println("   go run ./keychain")
}

func isMacOS() bool {
	return runtime.GOOS == "darwin"
}
