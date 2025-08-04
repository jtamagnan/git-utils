package main

import (
	"os"

	review "github.com/jtamagnan/git-utils/review/lib"
	"github.com/jtamagnan/git-utils/review/lib/config"
	"github.com/spf13/cobra"
)

func runE(cmd *cobra.Command, args []string) error {
	parsedArgs, err := config.ParseArgs(cmd, args)
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
			config.InitConfig()
		},
		RunE: runE,
	}

	// Set up flags using the config library
	config.SetupFlags(rootCmd)

	return rootCmd
}

func main() {
	rootCmd := generateCommand()
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
