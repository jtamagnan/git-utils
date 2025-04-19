package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

type ParsedArgs struct {
	toggle bool
}

func parseArgs(cmd *cobra.Command, _ []string) (ParsedArgs, error) {
	parsedArgs := ParsedArgs{}

	toggle, err := cmd.Flags().GetBool("toggle")
	if err != nil { return parsedArgs, err }
	parsedArgs.toggle = toggle

	return parsedArgs, nil
}

func run(args ParsedArgs) error {
	fmt.Printf("%#v", args)
	return nil
}

func runE(cmd *cobra.Command, args []string) error {
	parsedArgs, err := parseArgs(cmd, args)
	if err != nil { return err }

	err = run(parsedArgs)
	if err != nil { return err }
	return nil
}

func generateCommand() (*cobra.Command) {
	var rootCmd = &cobra.Command{
		Use:   "directorylist",
		Short: "List the size of a local directory.",
		Long:  `This command will display the size of a directory with several different options.`,
		RunE: runE,
	}

	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	return rootCmd
}

func Execute() {
	rootCmd := generateCommand()
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
