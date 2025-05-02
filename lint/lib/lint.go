package lint

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/jtamagnan/git-utils/git"
)

type ParsedArgs struct {
	AllFiles bool
	Stream bool
}

func Lint(args ParsedArgs) error {
	// TODO(jat): Allow the "from-ref" to be set to a specific commit or upstream branch

	// Get the upstream branch that we're tracking. TODO(jat): Consider using a merge-base
	repo, err := git.GetRepository()
	if err != nil { return err }
	branch, err := repo.Head()
	if err != nil { return err }
	upstreamBranch, err := branch.TrackingBranch()
	if err != nil { return err }

	writeTree, err := repo.WriteTree()

	var cliArgs []string
	cliArgs = append(cliArgs, "run")
	cliArgs = append(cliArgs, "--color=always")
	cliArgs = append(cliArgs, "--all-files")

	if !args.AllFiles {
		cliArgs = append(cliArgs, fmt.Sprintf("--from-ref=%s", upstreamBranch))
		cliArgs = append(cliArgs, fmt.Sprintf("--to-ref=%s", writeTree))
	}

	cmd := exec.Command(
		"pre-commit",
		cliArgs...,
	)

	if args.Stream {
		fmt.Printf("$ %s:", cmd.String())
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			return fmt.Errorf("Error running `%s`", cmd.String())
		}
	} else {
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("Error running pre-commit: `%s` \n%s", cmd.String(), out)
		}
	}

	return nil
}
