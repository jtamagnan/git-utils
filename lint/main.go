package main

import (
	lint "github.com/jtamagnan/git-utils/lint/lib"
	"os"

	"github.com/spf13/cobra"
)

func parseArgs(cmd *cobra.Command, args []string) (lint.ParsedArgs, error) {
	parsedArgs := lint.ParsedArgs{}
	parsedArgs.Stream = true

	allFiles, err := cmd.Flags().GetBool("all")
	if err != nil {
		return parsedArgs, err
	}
	parsedArgs.AllFiles = allFiles

	// Get check names from positional arguments if provided
	if len(args) > 0 {
		parsedArgs.CheckNames = args
	}

	return parsedArgs, nil
}

func runE(cmd *cobra.Command, args []string) error {
	parsedArgs, err := parseArgs(cmd, args)
	if err != nil {
		return err
	}

	err = lint.Lint(parsedArgs)
	if err != nil {
		return err
	}
	return nil
}

func generateCommand() *cobra.Command {
	var rootCmd = &cobra.Command{
		Use:          "git-lint [check-name...]",
		Short:        "Run pre-commit checks in this repository.",
		Long:         "Run pre-commit checks. Optionally specify one or more specific check names to run only those checks.",
		RunE:         runE,
		SilenceUsage: true,
	}

	rootCmd.Flags().BoolP("all", "a", false, "Run against all files")

	return rootCmd
}

func main() {
	rootCmd := generateCommand()
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
