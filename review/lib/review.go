package review

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/google/go-github/v71/github"
	"github.com/jtamagnan/git-utils/editor"
	"github.com/jtamagnan/git-utils/git"
	lint "github.com/jtamagnan/git-utils/lint/lib"
	"github.com/jtamagnan/git-utils/review/lib/branch"
	"github.com/jtamagnan/git-utils/review/lib/commit"
	githubapi "github.com/jtamagnan/git-utils/review/lib/github"
	"github.com/jtamagnan/git-utils/review/lib/pr"
	"github.com/jtamagnan/git-utils/review/lib/template"
)

// ParsedArgs represents the parsed command line arguments
type ParsedArgs struct {
	NoVerify    bool
	OpenBrowser bool
	Draft       bool
}

// stripRemotePrefix removes the specific remote prefix from branch names (e.g., "origin/main" -> "main")
func stripRemotePrefix(branch, remote string) string {
	prefix := remote + "/"
	if strings.HasPrefix(branch, prefix) {
		return strings.TrimPrefix(branch, prefix)
	}
	// Return as-is if no remote prefix found
	return branch
}

// getPRDescription gets the initial PR description content from templates and opens editor
func getPRDescription() (string, error) {
	// Get the initial template content
	initialContent := template.FindPRTemplate()

	// Open editor with the template content for user to edit
	return editor.OpenEditor(initialContent)
}

// cleanupRemoteBranch deletes a remote branch if it was created for a new PR
func cleanupRemoteBranch(repo *git.Repository, upstream, remoteBranchName string) {
	fmt.Printf("Cleaning up remote branch: %s\n", remoteBranchName)

	// Delete the remote branch
	_, cleanupErr := repo.GitExec("push", upstream, "--delete", remoteBranchName)
	if cleanupErr != nil {
		fmt.Printf("Warning: Failed to delete remote branch %s: %v\n", remoteBranchName, cleanupErr)
	} else {
		fmt.Printf("Successfully deleted remote branch: %s\n", remoteBranchName)
	}
}

// Review performs the main review workflow
func Review(args ParsedArgs) error {
	// TODO(jat): Support adding labels

	//
	// Get current repository
	//
	repo, err := git.GetRepository()
	if err != nil {
		return err
	}

	//
	// Run pre-commit checks unless skipped
	//
	if args.NoVerify {
		fmt.Println("Skipping pre-commit checks")
	} else {
		err = lint.Lint(lint.ParsedArgs{})
		if err != nil {
			return err
		}
	}

	//
	// Get upstream remote and default branch
	//
	upstream, err := repo.Remote()
	if err != nil {
		return err
	}
	upstreamBranch, err := repo.GetDefaultBranch()
	if err != nil {
		return err
	}

	//
	// Get repository information from upstream URL
	//
	upstreamURL, err := repo.GetRemoteURL(upstream)
	if err != nil {
		return err
	}

	repoInfo, err := git.ParseRepositoryInfo(upstreamURL)
	if err != nil {
		return err
	}

	//
	// Determine the remote branch name to use and if the PR already
	// exists and is open
	//
	var remoteBranchName string
	var isNewPR bool
	existingPRNumber, err := pr.DetectExistingPR(repo, upstreamBranch)
	if err != nil {
		// No existing PR found, generate UUID branch name for new PR
		remoteBranchName, err = branch.GenerateUUIDBranchName()
		if err != nil {
			return err
		}
		isNewPR = true
		fmt.Printf("No existing PR found, will create new PR with branch: %s\n", remoteBranchName)
	} else {
		// Check if the existing PR is still open
		existingPR, err := githubapi.GetExistingPR(repoInfo.Owner, repoInfo.Name, existingPRNumber)
		if err != nil {
			return err
		}

		if existingPR.State != nil && *existingPR.State == "open" {
			// Existing open PR found, get the remote branch name from the PR
			remoteBranchName, err = githubapi.GetRemoteBranchFromPR(repoInfo.Owner, repoInfo.Name, existingPRNumber)
			if err != nil {
				return err
			}
			isNewPR = false
			fmt.Printf("Found existing open PR #%d, will update branch: %s\n", existingPRNumber, remoteBranchName)
		} else {
			// Existing PR is closed, create a new PR
			remoteBranchName, err = branch.GenerateUUIDBranchName()
			if err != nil {
				return err
			}
			isNewPR = true
			fmt.Printf("Found existing PR #%d but it's closed, will create new PR with branch: %s\n", existingPRNumber, remoteBranchName)
		}
	}

	//
	// Push changes to the determined remote branch
	//
	fmt.Printf("Pushing to %s %s\n", upstream, remoteBranchName)
	_, err = repo.GitExec("push", "--force", upstream, fmt.Sprintf("HEAD:%s", remoteBranchName))
	if err != nil {
		return err
	}

	// Set up cleanup for new PR branches in case of failure
	var prCreationSucceeded bool
	if isNewPR {
		defer func() {
			if !prCreationSucceeded {
				cleanupRemoteBranch(repo, upstream, remoteBranchName)
			}
		}()
	}

	//
	// Create the PR or get the existing one that we're working with.
	//
	var githubPR *github.PullRequest
	if isNewPR {
		//
		// Generate PR title from commit summaries
		//
		summaries := repo.RefSummaries(upstreamBranch)
		if len(summaries) == 0 {
			return fmt.Errorf("no commits found between HEAD and %s", upstreamBranch)
		}
		prTitle := summaries[0] // Use the oldest (first) commit summary

		//
		// Get the PR description
		//
		prDescription, err := getPRDescription()
		if err != nil {
			return err
		}

		//
		// Open the PR
		//
		baseBranch := stripRemotePrefix(upstreamBranch, upstream)
		githubPR, err = githubapi.CreatePR(repoInfo.Owner, repoInfo.Name, prTitle, remoteBranchName, baseBranch, prDescription, args.Draft)
		if err != nil {
			return err
		}

		//
		// Mark PR creation as successful to prevent branch deletion
		//
		prCreationSucceeded = true
		fmt.Printf("Created new PR #%d: %s\n", *githubPR.Number, *githubPR.HTMLURL)

		//
		// Update the oldest commit message with the PR URL
		//
		err = commit.UpdateOldestCommitWithPRURL(repo, upstreamBranch, *githubPR.HTMLURL)
		if err != nil {
			return err
		}

		//
		// Push again with the updated commit message
		//
		fmt.Printf("Pushing updated commits to %s %s\n", upstream, remoteBranchName)
		_, err = repo.GitExec("push", "--force", upstream, fmt.Sprintf("HEAD:%s", remoteBranchName))
		if err != nil {
			return err
		}
	} else {
		//
		// Get the PR
		//
		fmt.Printf("Found existing PR #%d\n", existingPRNumber)
		githubPR, err = githubapi.GetExistingPR(repoInfo.Owner, repoInfo.Name, existingPRNumber)
		if err != nil {
			return err
		}
	}

	//
	// Open browser to the PR if requested
	//
	if args.OpenBrowser {
		err = exec.Command("open", *githubPR.HTMLURL).Run()
		if err != nil {
			fmt.Printf("Failed to open browser: %v\n", err)
		}
	}

	fmt.Printf("PR URL: %s\n", *githubPR.HTMLURL)

	//
	// Clean exit to avoid any cleanup that might interfere with the PR
	//
	return nil
}
