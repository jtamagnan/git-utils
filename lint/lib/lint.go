package lint

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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

	repo, err := git.GetRepository()
	if err != nil {
		return err
	}

	if args.AllFiles {
		return runChecks(repo, args, nil)
	}

	branch, err := repo.Head()
	if err != nil {
		return err
	}
	upstreamBranch, err := branch.TrackingBranch()
	if err != nil || upstreamBranch == "" {
		return fmt.Errorf("no upstream branch configured for current branch - run 'git branch --set-upstream-to=<remote>/<branch>' to set upstream")
	}

	// Commit staged files, then stage+commit tracked modifications.
	// This prevents prek from stashing tracked changes out from under us.
	stagedCommit, err := repo.GitExec("commit", "--allow-empty", "-m", "git-lint: staged")
	if err != nil {
		return fmt.Errorf("failed to create temporary staged commit: %w", err)
	}
	_ = stagedCommit

	// Stage all tracked modifications and commit them
	repo.GitExec("add", "-u")
	trackedCommit, err := repo.GitExec("commit", "--allow-empty", "-m", "git-lint: tracked")
	if err != nil {
		// Reset the first commit before returning
		repo.GitExec("reset", "--soft", "HEAD~1")
		return fmt.Errorf("failed to create temporary tracked commit: %w", err)
	}
	_ = trackedCommit

	toRef := "HEAD"
	fromRef := upstreamBranch

	lintErr := runChecks(repo, args, &refRange{from: fromRef, to: toRef})

	// Capture any linter fixups before resetting
	stashed := false
	if hasDirtyWorktree(repo) {
		if _, err := repo.GitExec("stash", "push", "-m", "git-lint: fixups"); err == nil {
			stashed = true
		}
	}

	// Reset: undo tracked commit (unstaged), then undo staged commit (keep staged)
	repo.GitExec("reset", "HEAD~1")
	repo.GitExec("reset", "--soft", "HEAD~1")

	// Re-apply linter fixups
	if stashed {
		repo.GitExec("stash", "pop")
	}

	return lintErr
}

type refRange struct {
	from string
	to   string
}

func hasDirtyWorktree(repo *git.Repository) bool {
	status, err := repo.GitExec("status", "--porcelain")
	if err != nil {
		return false
	}
	return strings.TrimSpace(status) != ""
}

func runChecks(repo *git.Repository, args ParsedArgs, refs *refRange) error {
	var baseArgs []string
	baseArgs = append(baseArgs, "run")
	baseArgs = append(baseArgs, "--color=always")

	if args.AllFiles {
		baseArgs = append(baseArgs, "--all-files")
	} else if refs != nil {
		baseArgs = append(baseArgs, fmt.Sprintf("--from-ref=%s", refs.from))
		baseArgs = append(baseArgs, fmt.Sprintf("--to-ref=%s", refs.to))
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
