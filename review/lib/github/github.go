package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

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

// RequestReviewers requests reviewers for a pull request
func RequestReviewers(owner, repo string, prNumber int, reviewers []string) error {
	if len(reviewers) == 0 {
		return nil // Nothing to do
	}

	client, err := newAuthenticatedClient()
	if err != nil {
		return err
	}

	// Request reviewers for the PR
	reviewersRequest := github.ReviewersRequest{
		Reviewers: reviewers,
	}

	_, _, err = client.PullRequests.RequestReviewers(context.Background(), owner, repo, prNumber, reviewersRequest)
	if err != nil {
		return fmt.Errorf("failed to request reviewers for PR #%d: %v", prNumber, err)
	}

	return nil
}

// EnableAutoMerge enables automerge for a pull request using GitHub's GraphQL API
func EnableAutoMerge(owner, repo string, prNumber int) error {
	client, err := newAuthenticatedClient()
	if err != nil {
		return err
	}

	// Step 1: Get the pull request node ID (required for GraphQL)
	pr, _, err := client.PullRequests.Get(context.Background(), owner, repo, prNumber)
	if err != nil {
		return fmt.Errorf("failed to get PR #%d: %v", prNumber, err)
	}

	if pr.NodeID == nil {
		return fmt.Errorf("PR #%d has no node ID", prNumber)
	}

	// Step 2: Enable auto-merge using GraphQL mutation
	token, err := keychain.GetGitHubToken()
	if err != nil {
		return err
	}

	mutation := `
		mutation($pullRequestId: ID!, $mergeMethod: PullRequestMergeMethod!) {
			enablePullRequestAutoMerge(input: {
				pullRequestId: $pullRequestId,
				mergeMethod: $mergeMethod
			}) {
				pullRequest {
					id
					autoMergeRequest {
						mergeMethod
						enabledAt
					}
				}
			}
		}
	`

	requestBody := map[string]interface{}{
		"query": mutation,
		"variables": map[string]interface{}{
			"pullRequestId": *pr.NodeID,
			"mergeMethod":   "MERGE",
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal GraphQL request: %v", err)
	}

	req, err := http.NewRequestWithContext(context.Background(), "POST", "https://api.github.com/graphql", bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute GraphQL request: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GraphQL request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response to check for GraphQL errors
	var graphQLResponse struct {
		Data struct {
			EnablePullRequestAutoMerge struct {
				PullRequest struct {
					ID               string
					AutoMergeRequest struct {
						MergeMethod string
						EnabledAt   string
					}
				}
			}
		}
		Errors []struct {
			Message string
		}
	}

	err = json.Unmarshal(body, &graphQLResponse)
	if err != nil {
		return fmt.Errorf("failed to parse GraphQL response: %v", err)
	}

	if len(graphQLResponse.Errors) > 0 {
		return fmt.Errorf("GraphQL errors: %s", graphQLResponse.Errors[0].Message)
	}

	return nil
}

// UpdatePRBase updates the base branch of an existing pull request
func UpdatePRBase(owner, repo string, prNumber int, newBase string) error {
	client, err := newAuthenticatedClient()
	if err != nil {
		return err
	}

	update := &github.PullRequest{
		Base: &github.PullRequestBranch{
			Ref: github.Ptr(newBase),
		},
	}

	_, _, err = client.PullRequests.Edit(context.Background(), owner, repo, prNumber, update)
	if err != nil {
		return fmt.Errorf("failed to update base branch for PR #%d: %v", prNumber, err)
	}

	return nil
}

// UpdatePRBody updates the body/description of an existing pull request
func UpdatePRBody(owner, repo string, prNumber int, body string) error {
	client, err := newAuthenticatedClient()
	if err != nil {
		return err
	}

	update := &github.PullRequest{
		Body: github.Ptr(body),
	}

	_, _, err = client.PullRequests.Edit(context.Background(), owner, repo, prNumber, update)
	if err != nil {
		return fmt.Errorf("failed to update body for PR #%d: %v", prNumber, err)
	}

	return nil
}

// CreatePR creates a new pull request and optionally adds labels and reviewers
func CreatePR(owner, repo, title, head, base, body string, draft bool, labels []string, reviewers []string) (*github.PullRequest, error) {
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

	// Request reviewers if provided
	if len(reviewers) > 0 {
		err = RequestReviewers(owner, repo, *pr.Number, reviewers)
		if err != nil {
			return nil, err
		}
	}

	return pr, nil
}
