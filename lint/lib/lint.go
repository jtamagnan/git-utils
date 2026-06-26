package lint

import (
	"errors"
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

// lintCommand returns "prek" if it is installed, otherwise "pre-commit".
func lintCommand() string {
	if _, err := exec.LookPath("prek"); err == nil {
		return "prek"
	}
	return "pre-commit"
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
	if err != nil || upstreamBranch == "" {
		return fmt.Errorf("no upstream branch configured for current branch - run 'git branch --set-upstream-to=<remote>/<branch>' to set upstream")
	}

	writeTree, err := repo.WriteTree()
	if err != nil {
		return err
	}

	var baseArgs []string
	baseArgs = append(baseArgs, "run")
	baseArgs = append(baseArgs, "--color=always")

	if args.AllFiles {
		baseArgs = append(baseArgs, "--all-files")
	} else {
		baseArgs = append(baseArgs, fmt.Sprintf("--from-ref=%s", upstreamBranch))
		baseArgs = append(baseArgs, fmt.Sprintf("--to-ref=%s", writeTree))
	}

	lintCmd := lintCommand()

	// If no specific checks provided, run all checks
	if len(args.CheckNames) == 0 {
		return runLintCommand(lintCmd, baseArgs, args.Stream)
	}

	// Run each check separately and collect all errors
	var errs []error
	for _, checkName := range args.CheckNames {
		cliArgs := make([]string, len(baseArgs))
		copy(cliArgs, baseArgs)
		cliArgs = append(cliArgs, checkName)

		err := runLintCommand(lintCmd, cliArgs, args.Stream)
		if err != nil {
			errs = append(errs, fmt.Errorf("check %q failed: %w", checkName, err))
		}
	}

	// Return all errors joined together, or nil if no errors
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func runLintCommand(lintCmd string, cliArgs []string, stream bool) error {
	cmd := exec.Command(lintCmd, cliArgs...)

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
			return fmt.Errorf("error running `%s` \n%s", cmd.String(), out)
		}
	}

	return nil
}
