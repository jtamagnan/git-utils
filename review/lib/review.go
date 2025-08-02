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
	// Get the PR description
	return editor.OpenEditor("Testing")
}

func Review(args ParsedArgs) error {
	// TODO(jat): Support adding labels
	// TODO(jat): Support getting the description from the commit message

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
	//
	// TODO(jat): Better upstream branch name:
	//  - Use a "review-at" commit message
	//  - Use the current branch name
	//  - Pick a name using a UUID
	upstream, err := repo.Remote()
	if err != nil { return err }

	developerBranch, err := repo.Head()
	if err != nil { return err }
	developerBranchName := developerBranch.Name()

	// Push my changes to origin
	fmt.Println("Pushing to", upstream, developerBranchName.String())
	repo.GitExec(
		"push",
		"--force",
		upstream,
		fmt.Sprintf("HEAD:%s", developerBranchName.String()),
	)

	// pr description
	prDescription, err := getPRDescription()
	if err != nil { return err }

	// Get the default branch
	upstreamBranch, err := repo.GetDefaultBranch()
	if err != nil { return err }

	// Get commit summaries to use for PR title
	summaries := repo.RefSummaries(upstreamBranch)
	if len(summaries) == 0 {
		return fmt.Errorf("no commits found between %s and HEAD - nothing to create a pull request for", upstreamBranch)
	}
	
	// Use the first element which is the oldest/first commit summary (RefSummaries returns oldest to newest)
	prTitle := summaries[0]

	// TODO(jat): Don't open a PR if there is already an associated PR
	// Create the PR
	client := github.NewClient(nil)
	prRequest := &github.NewPullRequest{
		Title: 	github.Ptr(prTitle),
		Head: 	github.Ptr(developerBranchName.Short()),
		Base: 	github.Ptr(upstreamBranch),
		Body: 	github.Ptr(prDescription),
		Draft: 	github.Ptr(args.Draft),
	}
	pr, _, err := client.PullRequests.Create(context.Background(), "owner", "repo", prRequest)
	if err != nil { return err }

	// TODO(jat): Update commit message with the PR URL
	// TODO(jat): Push again

	if !args.OpenBrowser {
		url := pr.HTMLURL
		fmt.Println("Opening PR in browser:", url)
		err = editor.OpenBrowser(*url)
		if err != nil { return err }
	}

	// Exit cleanly
	return nil
}
