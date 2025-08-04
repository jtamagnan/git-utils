package lint

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/jtamagnan/git-utils/git"
)

type ParsedArgs struct {
	AllFiles   bool
	Stream     bool
	CheckNames []string
}

func canLint() bool {
	repo, err := git.GetRepository()
	if err != nil {
		return false
	}

	workTree, err := repo.Worktree()
	if err != nil {
		return false
	}

	workDir := workTree.Filesystem.Root()
	configPath := filepath.Join(workDir, ".pre-commit-config.yaml")

	if _, err := os.Stat(configPath); err == nil {
		return true
	}

	configPathYml := filepath.Join(workDir, ".pre-commit-config.yml")
	if _, err := os.Stat(configPathYml); err == nil {
		return true
	}

	return false
}

func Lint(args ParsedArgs) error {
	if !canLint() {
		return nil
	}

	// TODO(jat): Allow the "from-ref" to be set to a specific commit or upstream branch

	// Get the upstream branch that we're tracking. TODO(jat): Consider using a merge-base
	repo, err := git.GetRepository()
	if err != nil {
		return err
	}
	branch, err := repo.Head()
	if err != nil {
		return err
	}
	upstreamBranch, err := branch.TrackingBranch()
	if err != nil {
		return err
	}

	writeTree, err := repo.WriteTree()
	if err != nil {
		return err
	}

	var baseArgs []string
	baseArgs = append(baseArgs, "run")
	baseArgs = append(baseArgs, "--color=always")
	baseArgs = append(baseArgs, "--all-files")

	if !args.AllFiles {
		baseArgs = append(baseArgs, fmt.Sprintf("--from-ref=%s", upstreamBranch))
		baseArgs = append(baseArgs, fmt.Sprintf("--to-ref=%s", writeTree))
	}

	// If no specific checks provided, run all checks
	if len(args.CheckNames) == 0 {
		return runPreCommit(baseArgs, args.Stream)
	}

	// Run each check separately
	for _, checkName := range args.CheckNames {
		cliArgs := make([]string, len(baseArgs))
		copy(cliArgs, baseArgs)
		cliArgs = append(cliArgs, checkName)

		err := runPreCommit(cliArgs, args.Stream)
		if err != nil {
			return err
		}
	}

	return nil
}

func runPreCommit(cliArgs []string, stream bool) error {
	cmd := exec.Command("pre-commit", cliArgs...)

	if stream {
		fmt.Printf("$ %s:\n", cmd.String())
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			return fmt.Errorf("error running `%s`", cmd.String())
		}
	} else {
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("error running pre-commit: `%s` \n%s", cmd.String(), out)
		}
	}

	return nil
}
