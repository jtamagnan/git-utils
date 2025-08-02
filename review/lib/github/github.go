package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v71/github"
	keychain "github.com/jtamagnan/git-utils/keychain/lib"
)

// newAuthenticatedClient creates a GitHub client with token authentication
func newAuthenticatedClient() (*github.Client, error) {
	token, err := keychain.GetGitHubToken()
	if err != nil {
		return nil, err
	}

	return github.NewClient(nil).WithAuthToken(token), nil
}

// GetRemoteBranchFromPR gets the remote branch name from an existing PR
func GetRemoteBranchFromPR(owner, repo string, prNumber int) (string, error) {
	client, err := newAuthenticatedClient()
	if err != nil {
		return "", err
	}

	// Get the PR details
	pr, _, err := client.PullRequests.Get(context.Background(), owner, repo, prNumber)
	if err != nil {
		return "", fmt.Errorf("failed to get PR #%d: %v", prNumber, err)
	}

	// Return the head branch name (the branch the PR is coming from)
	if pr.Head != nil && pr.Head.Ref != nil {
		return *pr.Head.Ref, nil
	}

	return "", fmt.Errorf("PR #%d has no head branch information", prNumber)
}

// GetExistingPR fetches an existing pull request by number
func GetExistingPR(owner, repo string, prNumber int) (*github.PullRequest, error) {
	client, err := newAuthenticatedClient()
	if err != nil {
		return nil, err
	}

	// Get the PR
	pr, _, err := client.PullRequests.Get(context.Background(), owner, repo, prNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get PR #%d: %v", prNumber, err)
	}

	return pr, nil
}

// AddLabelsToIssue adds labels to an issue or pull request
func AddLabelsToIssue(owner, repo string, issueNumber int, labels []string) error {
	if len(labels) == 0 {
		return nil // Nothing to do
	}

	client, err := newAuthenticatedClient()
	if err != nil {
		return err
	}

	// Add labels to the issue/PR
	_, _, err = client.Issues.AddLabelsToIssue(context.Background(), owner, repo, issueNumber, labels)
	if err != nil {
		return fmt.Errorf("failed to add labels to issue #%d: %v", issueNumber, err)
	}

	return nil
}

// CreatePR creates a new pull request and optionally adds labels
func CreatePR(owner, repo, title, head, base, body string, draft bool, labels []string) (*github.PullRequest, error) {
	client, err := newAuthenticatedClient()
	if err != nil {
		return nil, err
	}

	prRequest := &github.NewPullRequest{
		Title: github.Ptr(title),
		Head:  github.Ptr(head),
		Base:  github.Ptr(base),
		Body:  github.Ptr(body),
		Draft: github.Ptr(draft),
	}

	pr, _, err := client.PullRequests.Create(context.Background(), owner, repo, prRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to create PR: %v", err)
	}

	// Add labels if provided (PRs are treated as issues for labeling)
	if len(labels) > 0 {
		_ = AddLabelsToIssue(owner, repo, *pr.Number, labels)
	}

	return pr, nil
}
