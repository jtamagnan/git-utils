package lint

import (
	"context"
	"fmt"

	"github.com/jtamagnan/git-utils/editor"
	"github.com/jtamagnan/git-utils/git"
	lint "github.com/jtamagnan/git-utils/lint/lib"

	"github.com/google/go-github/v71/github"
)

type ParsedArgs struct {
	NoVerify bool
	OpenBrowser bool
	Draft bool
}


func getPRDescription() (string, error) {
	// Look for GitHub PR template in common locations
	templateContent := findPRTemplate()

	// Open editor with template content as initial text
	return editor.OpenEditor(templateContent)
}



func Review(args ParsedArgs) error {
	// TODO(jat): Support adding labels

	// Get the current repository
	repo, err := git.GetRepository()
	if err != nil { return err }

	// Run pre-commit
	if args.NoVerify {
		fmt.Println("Skipping pre-commit checks")
	} else {
		err = lint.Lint(lint.ParsedArgs{})
		if err != nil { return err }
	}

	// Decide the upstream to open a pull request against
	upstream, err := repo.Remote()
	if err != nil { return err }

	// Get the default branch
	upstreamBranch, err := repo.GetDefaultBranch()
	if err != nil { return err }

	// Check if there's already a PR associated with this branch
	existingPRNumber, err := detectExistingPR(repo, upstreamBranch)

	// Determine the remote branch name to use
	var remoteBranchName string
	var isNewPR bool

	if err != nil {
		// No existing PR found, generate UUID branch name for new PR
		remoteBranchName = generateUUIDBranchName()
		isNewPR = true
		fmt.Printf("No existing PR found, will create new PR with branch: %s\n", remoteBranchName)
	} else {
		// Existing PR found, get the remote branch name from the PR
		remoteBranchName, err = getRemoteBranchFromPR(existingPRNumber)
		if err != nil { return err }
		isNewPR = false
		fmt.Printf("Found existing PR #%d, will update branch: %s\n", existingPRNumber, remoteBranchName)
	}

	// Push changes to the determined remote branch
	fmt.Printf("Pushing to %s %s\n", upstream, remoteBranchName)
	_, err = repo.GitExec(
		"push",
		"--force",
		upstream,
		fmt.Sprintf("HEAD:%s", remoteBranchName),
	)
	if err != nil { return err }

	// Get commit summaries to use for PR title
	summaries := repo.RefSummaries(upstreamBranch)
	if len(summaries) == 0 {
		return fmt.Errorf("no commits found between %s and HEAD - nothing to create a pull request for", upstreamBranch)
	}

	// Use the first element which is the oldest/first commit summary (RefSummaries returns oldest to newest)
	prTitle := summaries[0]

	var pr *github.PullRequest
	if isNewPR {
		// Create new PR
		prDescription, err := getPRDescription()
		if err != nil { return err }

		client := github.NewClient(nil)
		prRequest := &github.NewPullRequest{
			Title: 	github.Ptr(prTitle),
			Head: 	github.Ptr(remoteBranchName),
			Base: 	github.Ptr(upstreamBranch),
			Body: 	github.Ptr(prDescription),
			Draft: 	github.Ptr(args.Draft),
		}
		pr, _, err = client.PullRequests.Create(context.Background(), "owner", "repo", prRequest)
		if err != nil { return err }
		fmt.Printf("Created new PR #%d: %s\n", *pr.Number, *pr.HTMLURL)
	} else {
		// Get existing PR (pushing will automatically update it)
		pr, err = getExistingPR(existingPRNumber)
		if err != nil { return err }
		fmt.Printf("Will update existing PR #%d: %s\n", existingPRNumber, *pr.HTMLURL)
	}

	// Update the oldest commit message with the PR URL (for new PRs or if URL is missing)
	err = updateOldestCommitWithPRURL(repo, upstreamBranch, *pr.HTMLURL)
	if err != nil { return err }

	// Push again with the updated commit message
	fmt.Printf("Pushing updated commits to %s %s\n", upstream, remoteBranchName)
	_, err = repo.GitExec(
		"push",
		"--force",
		upstream,
		fmt.Sprintf("HEAD:%s", remoteBranchName),
	)
	if err != nil { return err }

	if !args.OpenBrowser {
		url := pr.HTMLURL
		fmt.Println("Opening PR in browser:", url)
		err = editor.OpenBrowser(*url)
		if err != nil { return err }
	}

	// Exit cleanly
	return nil
}
