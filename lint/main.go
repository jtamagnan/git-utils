package main

import (
	"os"
	lint "github.com/jtamagnan/git-utils/lint/lib"

	"github.com/spf13/cobra"
)

func parseArgs(cmd *cobra.Command, _ []string) (lint.ParsedArgs, error) {
	parsedArgs := lint.ParsedArgs{}
	parsedArgs.Stream = true

	allFiles, err := cmd.Flags().GetBool("all")
	if err != nil { return parsedArgs, err }
	parsedArgs.AllFiles = allFiles

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
