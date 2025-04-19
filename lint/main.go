package main

import (
	"os"
	lint "github.com/jtamagnan/git-utils/lint/lib"

	"github.com/spf13/cobra"
)

func parseArgs(cmd *cobra.Command, _ []string) (lint.ParsedArgs, error) {
	parsedArgs := lint.ParsedArgs{}

	// toggle, err := cmd.Flags().GetBool("toggle")
	// if err != nil { return parsedArgs, err }
	// parsedArgs.toggle = toggle

	return parsedArgs, nil
}

func runE(cmd *cobra.Command, args []string) error {
	parsedArgs, err := parseArgs(cmd, args)
	if err != nil { return err }

	err = lint.Lint(parsedArgs)
	if err != nil { return err }
	return nil
}

func generateCommand() (*cobra.Command) {
	var rootCmd = &cobra.Command{
		Use:   "git-lint",
		Short: "Run pre-commit checks in this repositoyr.",
		RunE: runE,
	}

	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
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
