package review

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/google/go-github/v71/github"
	"github.com/jtamagnan/git-utils/editor"
	"github.com/jtamagnan/git-utils/git"
	lint "github.com/jtamagnan/git-utils/lint/lib"
	"github.com/jtamagnan/git-utils/review/lib/branch"
	"github.com/jtamagnan/git-utils/review/lib/commit"
	githubapi "github.com/jtamagnan/git-utils/review/lib/github"
	"github.com/jtamagnan/git-utils/review/lib/parent"
	"github.com/jtamagnan/git-utils/review/lib/pr"
	"github.com/jtamagnan/git-utils/review/lib/template"
)

// StackParsedArgs represents the parsed command line arguments for the stack command
type StackParsedArgs struct {
	NoVerify    bool
	OpenBrowser bool
	Verbose     bool
	Parent      string
}

// stackGroup represents a group of commits that belong to one PR
type stackGroup struct {
	commits    []pr.StackCommitPR // commits in this group (oldest first)
	prNumber   int                // 0 if new PR needed
	prURL      string
	branchName string // remote branch for this PR
	baseBranch string // what this PR targets (GitHub base)
}

// Stack performs the stacked PR workflow
func Stack(args StackParsedArgs) error {
	repo, err := git.GetRepository()
	if err != nil {
		return err
	}

	// Run pre-commit checks
	if args.NoVerify {
		fmt.Println("Skipping pre-commit checks")
	} else {
		fmt.Println("Running pre-commit checks...")
		err = lint.Lint(lint.ParsedArgs{Stream: args.Verbose})
		if err != nil {
			return err
		}
	}

	// Get upstream remote
	upstream, err := repo.Remote()
	if err != nil {
		return fmt.Errorf("no upstream branch configured for current branch - run 'git branch --set-upstream-to=<remote>/<branch>' to set upstream")
	}

	upstreamURL, err := repo.GetRemoteURL(upstream)
	if err != nil {
		return err
	}

	repoInfo, err := git.ParseRepositoryInfo(upstreamURL)
	if err != nil {
		return err
	}

	// Resolve parent branch
	resolvedParent, err := parent.ResolveParent(repo, args.Parent, repoInfo.Owner, repoInfo.Name)
	if err != nil {
		return err
	}

	parentBranch := resolvedParent.GitRef
	fmt.Printf("Using parent branch: %s (GitHub base: %s)\n", parentBranch, resolvedParent.GitHubBase)

	// Get all commits with their PR info
	commits, err := pr.DetectAllPRs(repo, parentBranch)
	if err != nil {
		return err
	}

	fmt.Printf("Found %d commits in stack\n", len(commits))

	// Determine mode: any commit with a PR URL means update mode
	hasAnyPR := false
	for _, c := range commits {
		if c.PRNum > 0 {
			hasAnyPR = true
			break
		}
	}

	if hasAnyPR {
		// Update mode: group commits, absorbing orphans into their parent's PR
		groups := groupCommits(commits, resolvedParent.GitHubBase)
		return updateStack(repo, upstream, repoInfo, parentBranch, groups)
	}

	// Create mode: one group per commit
	var groups []stackGroup
	for _, c := range commits {
		groups = append(groups, stackGroup{
			commits: []pr.StackCommitPR{c},
		})
	}
	return createStack(repo, upstream, repoInfo, parentBranch, resolvedParent.GitHubBase, groups, args.OpenBrowser)
}

// groupCommits organizes commits into groups based on PR ownership.
// Each commit with a PR URL starts a new group. Commits with a bare
// "PR URL:" sentinel (WantsPR but no PRNum) also start a new group
// that will get a new PR. Commits without any PR marker join the
// previous group (or form a new group if they appear before any PR).
func groupCommits(commits []pr.StackCommitPR, defaultBase string) []stackGroup {
	var groups []stackGroup

	for _, c := range commits {
		if c.PRNum > 0 {
			// This commit owns a PR - start a new group
			groups = append(groups, stackGroup{
				commits:  []pr.StackCommitPR{c},
				prNumber: c.PRNum,
				prURL:    c.PRURL,
			})
		} else if c.WantsPR {
			// Bare "PR URL:" sentinel - start a new group that needs a new PR
			groups = append(groups, stackGroup{
				commits: []pr.StackCommitPR{c},
			})
		} else if len(groups) > 0 {
			// No PR marker - join the previous group
			groups[len(groups)-1].commits = append(groups[len(groups)-1].commits, c)
		} else {
			// Orphan commit before any PR - create new group
			groups = append(groups, stackGroup{
				commits: []pr.StackCommitPR{c},
			})
		}
	}

	return groups
}

// stackPRInfo holds the info needed to build the PR Stack section
type stackPRInfo struct {
	title    string
	prNumber int
}

// buildStackSection builds the "## PR Stack" markdown section.
// currentIndex is the 0-based index of the PR being described.
func buildStackSection(prs []stackPRInfo, currentIndex int) string {
	var b strings.Builder
	b.WriteString("## PR Stack\n")
	for i, p := range prs {
		if i == currentIndex {
			b.WriteString(fmt.Sprintf("%d. :star: `%s` (#%d)\n", i+1, p.title, p.prNumber))
		} else {
			b.WriteString(fmt.Sprintf("%d. `%s` (#%d)\n", i+1, p.title, p.prNumber))
		}
	}
	return b.String()
}

// stackSectionRegex matches an existing "## PR Stack" section (including trailing newlines)
var stackSectionRegex = regexp.MustCompile(`(?ms)^## PR Stack\n(?:.*\n)*?(?:\n|$)`)

// upsertStackSection replaces an existing PR Stack section in body, or appends one.
func upsertStackSection(body, section string) string {
	if stackSectionRegex.MatchString(body) {
		return strings.TrimSpace(stackSectionRegex.ReplaceAllString(body, section)) + "\n"
	}
	return strings.TrimSpace(body) + "\n\n" + section
}

// updateStackDescriptions updates all PRs in the stack with the PR Stack section
func updateStackDescriptions(owner, repo string, prs []stackPRInfo, prBodies map[int]string) {
	for i, p := range prs {
		section := buildStackSection(prs, i)
		body := upsertStackSection(prBodies[p.prNumber], section)
		err := githubapi.UpdatePRBody(owner, repo, p.prNumber, body)
		if err != nil {
			fmt.Printf("Warning: failed to update description for PR #%d: %v\n", p.prNumber, err)
		}
	}
}

// createStack creates a new PR for each commit (mode 1: no existing PRs)
func createStack(repo *git.Repository, upstream string, repoInfo *git.RepositoryInfo, parentBranch, defaultBase string, groups []stackGroup, openBrowser bool) error {
	var createdPRs []*github.PullRequest
	var prURLUpdates []commit.CommitPRURL
	previousBase := defaultBase

	initialContent := template.FindPRTemplate()

	for i, group := range groups {
		// Generate branch name
		branchName, err := branch.GenerateUUIDBranchName()
		if err != nil {
			return err
		}
		groups[i].branchName = branchName
		groups[i].baseBranch = previousBase

		// Push cumulative commits up to this group's last commit
		lastCommit := group.commits[len(group.commits)-1]
		fmt.Printf("\nPushing commits up to %s to branch %s\n", lastCommit.Hash[:8], branchName)
		_, err = repo.GitExec("push", "--force", upstream, fmt.Sprintf("%s:refs/heads/%s", lastCommit.Hash, branchName))
		if err != nil {
			return fmt.Errorf("error pushing to %s: %v", branchName, err)
		}

		// PR title from the first commit in the group
		prTitle := group.commits[0].Summary

		// Get PR description from editor
		fmt.Printf("\n--- PR #%d: %s ---\n", i+1, prTitle)
		prDescription, err := editor.OpenEditor(initialContent)
		if err != nil {
			return err
		}

		// Create PR
		githubPR, err := githubapi.CreatePR(repoInfo.Owner, repoInfo.Name, prTitle, branchName, previousBase, prDescription, false, nil, nil)
		if err != nil {
			return fmt.Errorf("error creating PR for group %d: %v", i+1, err)
		}

		createdPRs = append(createdPRs, githubPR)
		fmt.Printf("Created PR #%d: %s\n", *githubPR.Number, *githubPR.HTMLURL)

		// Record that the first commit in each group should get the PR URL
		prURLUpdates = append(prURLUpdates, commit.CommitPRURL{
			Hash:  group.commits[0].Hash,
			PRURL: *githubPR.HTMLURL,
		})

		// Next PR's base is this PR's branch
		previousBase = branchName
	}

	// Stamp all PR URLs in a single rebase pass
	fmt.Println("\nStamping PR URLs into commit messages...")
	err := commit.UpdateMultipleCommitsWithPRURLs(repo, parentBranch, prURLUpdates)
	if err != nil {
		return fmt.Errorf("error stamping PR URLs: %v", err)
	}

	// Re-push all branches with updated commit messages
	fmt.Println("Re-pushing branches with updated commit messages...")
	commits, err := pr.DetectAllPRs(repo, parentBranch)
	if err != nil {
		return fmt.Errorf("error re-reading commits after stamping: %v", err)
	}

	// Rebuild groups with updated hashes
	updatedGroups := groupCommits(commits, defaultBase)
	for i, group := range updatedGroups {
		if i >= len(groups) {
			break
		}
		lastCommit := group.commits[len(group.commits)-1]
		branchName := groups[i].branchName
		_, err = repo.GitExec("push", "--force", upstream, fmt.Sprintf("%s:refs/heads/%s", lastCommit.Hash, branchName))
		if err != nil {
			fmt.Printf("Warning: failed to re-push branch %s: %v\n", branchName, err)
		}
	}

	// Update all PR descriptions with the PR Stack section
	fmt.Println("Updating PR descriptions with stack info...")
	var stackInfos []stackPRInfo
	prBodies := make(map[int]string)
	for i, githubPR := range createdPRs {
		stackInfos = append(stackInfos, stackPRInfo{
			title:    groups[i].commits[0].Summary,
			prNumber: *githubPR.Number,
		})
		prBodies[*githubPR.Number] = githubPR.GetBody()
	}
	updateStackDescriptions(repoInfo.Owner, repoInfo.Name, stackInfos, prBodies)

	// Open browsers
	if openBrowser {
		for _, githubPR := range createdPRs {
			_ = exec.Command("open", *githubPR.HTMLURL).Run()
		}
	}

	// Print summary
	fmt.Println("\n--- Stack Summary ---")
	for i, githubPR := range createdPRs {
		fmt.Printf("  %d. PR #%d: %s\n", i+1, *githubPR.Number, *githubPR.HTMLURL)
	}

	return nil
}

// updateStack updates existing PRs and absorbs orphan commits (mode 2)
func updateStack(repo *git.Repository, upstream string, repoInfo *git.RepositoryInfo, parentBranch string, groups []stackGroup) error {
	var prURLUpdates []commit.CommitPRURL
	var allPRURLs []string

	initialContent := template.FindPRTemplate()

	// First pass: resolve branch names for existing PRs, create new PRs for orphan groups
	previousBase := ""
	for i, group := range groups {
		if i == 0 {
			// First group's base is the default branch
			defaultBranch, err := repo.GetDefaultBranch()
			if err != nil {
				return err
			}
			previousBase = stripRemotePrefix(defaultBranch, upstream)
		}

		groups[i].baseBranch = previousBase

		if group.prNumber > 0 {
			// Existing PR - get its branch name
			branchName, err := githubapi.GetRemoteBranchFromPR(repoInfo.Owner, repoInfo.Name, group.prNumber)
			if err != nil {
				return fmt.Errorf("error getting branch for PR #%d: %v", group.prNumber, err)
			}
			groups[i].branchName = branchName
			previousBase = branchName
		} else {
			// Orphan group - create a new PR
			branchName, err := branch.GenerateUUIDBranchName()
			if err != nil {
				return err
			}
			groups[i].branchName = branchName

			// Push commits up to this group
			lastCommit := group.commits[len(group.commits)-1]
			fmt.Printf("\nPushing new commits to branch %s\n", branchName)
			_, err = repo.GitExec("push", "--force", upstream, fmt.Sprintf("%s:refs/heads/%s", lastCommit.Hash, branchName))
			if err != nil {
				return fmt.Errorf("error pushing to %s: %v", branchName, err)
			}

			prTitle := group.commits[0].Summary

			fmt.Printf("\n--- New PR: %s ---\n", prTitle)
			prDescription, err := editor.OpenEditor(initialContent)
			if err != nil {
				return err
			}

			githubPR, err := githubapi.CreatePR(repoInfo.Owner, repoInfo.Name, prTitle, branchName, previousBase, prDescription, false, nil, nil)
			if err != nil {
				return fmt.Errorf("error creating PR: %v", err)
			}

			groups[i].prNumber = *githubPR.Number
			groups[i].prURL = *githubPR.HTMLURL
			fmt.Printf("Created PR #%d: %s\n", *githubPR.Number, *githubPR.HTMLURL)

			prURLUpdates = append(prURLUpdates, commit.CommitPRURL{
				Hash:  group.commits[0].Hash,
				PRURL: *githubPR.HTMLURL,
			})

			previousBase = branchName
		}
	}

	// Stamp any new PR URLs
	if len(prURLUpdates) > 0 {
		fmt.Println("\nStamping PR URLs into commit messages...")
		err := commit.UpdateMultipleCommitsWithPRURLs(repo, parentBranch, prURLUpdates)
		if err != nil {
			return fmt.Errorf("error stamping PR URLs: %v", err)
		}

		// Re-read commits after rebase changed hashes
		commits, err := pr.DetectAllPRs(repo, parentBranch)
		if err != nil {
			return fmt.Errorf("error re-reading commits: %v", err)
		}
		// Rebuild groups to get updated hashes
		defaultBranch, _ := repo.GetDefaultBranch()
		defaultBase := stripRemotePrefix(defaultBranch, upstream)
		updatedGroups := groupCommits(commits, defaultBase)

		// Preserve branch names from original groups
		for i := range updatedGroups {
			if i < len(groups) {
				updatedGroups[i].branchName = groups[i].branchName
				updatedGroups[i].baseBranch = groups[i].baseBranch
			}
		}
		groups = updatedGroups
	}

	// Second pass: push all branches, update PR bases, and collect PR info
	var stackInfos []stackPRInfo
	prBodies := make(map[int]string)

	for i, group := range groups {
		lastCommit := group.commits[len(group.commits)-1]
		fmt.Printf("Pushing to %s (PR #%d)\n", group.branchName, group.prNumber)
		_, err := repo.GitExec("push", "--force", upstream, fmt.Sprintf("%s:refs/heads/%s", lastCommit.Hash, group.branchName))
		if err != nil {
			return fmt.Errorf("error pushing to %s: %v", group.branchName, err)
		}

		// Update the PR base branch
		err = githubapi.UpdatePRBase(repoInfo.Owner, repoInfo.Name, group.prNumber, group.baseBranch)
		if err != nil {
			fmt.Printf("Warning: failed to update base for PR #%d: %v\n", group.prNumber, err)
		}

		// Collect PR info for stack description and summary
		stackInfos = append(stackInfos, stackPRInfo{
			title:    group.commits[0].Summary,
			prNumber: group.prNumber,
		})

		if group.prURL != "" {
			allPRURLs = append(allPRURLs, fmt.Sprintf("  %d. PR #%d: %s", i+1, group.prNumber, group.prURL))
			prBodies[group.prNumber] = "" // new PR, body already set during creation
		} else {
			githubPR, err := githubapi.GetExistingPR(repoInfo.Owner, repoInfo.Name, group.prNumber)
			if err == nil {
				allPRURLs = append(allPRURLs, fmt.Sprintf("  %d. PR #%d: %s", i+1, group.prNumber, *githubPR.HTMLURL))
				prBodies[group.prNumber] = githubPR.GetBody()
			}
		}
	}

	// Update all PR descriptions with the PR Stack section
	fmt.Println("Updating PR descriptions with stack info...")
	updateStackDescriptions(repoInfo.Owner, repoInfo.Name, stackInfos, prBodies)

	// Print summary
	fmt.Println("\n--- Stack Summary ---")
	for _, line := range allPRURLs {
		fmt.Println(line)
	}

	return nil
}
