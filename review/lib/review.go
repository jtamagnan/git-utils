package review

import (
	"fmt"

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

// getPRDescription gets the initial PR description content from templates and opens editor
func getPRDescription() (string, error) {
	// Get the initial template content
	initialContent := template.FindPRTemplate()

	// Open editor with the template content for user to edit
	return editor.OpenEditor(initialContent)
}

// Review performs the main review workflow
func Review(args ParsedArgs) error {
	// TODO(jat): Support adding labels

	//
	// Get current repository
	//
	repo, err := git.GetRepository()
	if err != nil { return err }

	//
	// Run pre-commit checks unless skipped
	//
	if args.NoVerify {
		fmt.Println("Skipping pre-commit checks")
	} else {
		err = lint.Lint(lint.ParsedArgs{})
		if err != nil { return err }
	}

	//
	// Get upstream remote and default branch
	//
	upstream, err := repo.Remote()
	if err != nil { return err }
	upstreamBranch, err := repo.GetDefaultBranch()
	if err != nil { return err }

	//
	// Determine the remote branch name to use and if the PR already
	// exists
	//
	var remoteBranchName string
	var isNewPR bool
	existingPRNumber, err := pr.DetectExistingPR(repo, upstreamBranch)
	if err != nil {
		// No existing PR found, generate UUID branch name for new PR
		remoteBranchName, err = branch.GenerateUUIDBranchName()
		if err != nil { return err }
		isNewPR = true
		fmt.Printf("No existing PR found, will create new PR with branch: %s\n", remoteBranchName)
	} else {
		// Existing PR found, get the remote branch name from the PR
		remoteBranchName, err = githubapi.GetRemoteBranchFromPR(existingPRNumber)
		if err != nil { return err }
		isNewPR = false
		fmt.Printf("Found existing PR #%d, will update branch: %s\n", existingPRNumber, remoteBranchName)
	}

	//
	// Push changes to the determined remote branch
	//
	fmt.Printf("Pushing to %s %s\n", upstream, remoteBranchName)
	_, err = repo.GitExec("push", "--force", upstream, fmt.Sprintf("HEAD:%s", remoteBranchName))
	if err != nil { return err }

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
		// Get the PR descirption
		//
		prDescription, err := getPRDescription()
		if err != nil { return err }

		//
		// Open the PR
		//
		githubPR, err = githubapi.CreatePR(prTitle, remoteBranchName, upstreamBranch, prDescription, args.Draft)
		if err != nil { return err }
		fmt.Printf("Created new PR #%d: %s\n", *githubPR.Number, *githubPR.HTMLURL)

		//
		// Update the oldest commit message with the PR URL
		//
		err = commit.UpdateOldestCommitWithPRURL(repo, upstreamBranch, *githubPR.HTMLURL)
		if err != nil { return err }

		//
		// Push again with the updated commit message
		//
		fmt.Printf("Pushing updated commits to %s %s\n", upstream, remoteBranchName)
		_, err = repo.GitExec("push", "--force", upstream, fmt.Sprintf("HEAD:%s", remoteBranchName))
		if err != nil { return err }
	} else {
		//
		// Get the PR
		//
		fmt.Printf("Found existing PR #%d\n", existingPRNumber)
		githubPR, err = githubapi.GetExistingPR(existingPRNumber)
		if err != nil { return err }
	}

	//
	// Open browser to the PR if requested
	//
	if args.OpenBrowser {
		_, err = repo.GitExec("open", *githubPR.HTMLURL)
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
