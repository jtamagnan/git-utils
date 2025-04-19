package lint

import (
	"fmt"
	"os/exec"

	"github.com/jtamagnan/git-utils/git"
	"github.com/jtamagnan/git-utils/lint"
)

type ParsedArgs struct {
}

func Lint(args ParsedArgs) error {
	// Get the branch
	repo, err := git.GetRepository()
	if err != nil { return err }

	branch, err := repo.Head()
	if err != nil { return err }

	trackingBranch, err := branch.TrackingBranch()
	if err != nil { return err }

	writeTree, err := repo.WriteTree()

	cmd := exec.Command(
		"pre-commit",
		"run",
		"--color=always",
		fmt.Sprintf("--from-ref=%s", trackingBranch),
		fmt.Sprintf("--to-ref=%s", writeTree),
		"--all-files",
	)
	fmt.Println(cmd.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Error running pre-commit: `%s` \n %s", cmd.String(), out)
	}
	fmt.Println(string(out))
	return nil
}
