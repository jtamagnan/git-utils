package review

import (
	"testing"

	"github.com/jtamagnan/git-utils/review/lib/pr"
)

func TestGroupCommits_AllOrphans(t *testing.T) {
	// groupCommits is only used in update mode. When all commits are orphans,
	// they all merge into one group (first orphan creates a group, rest join it).
	commits := []pr.StackCommitPR{
		{Hash: "aaa", Summary: "First", PRURL: "", PRNum: 0},
		{Hash: "bbb", Summary: "Second", PRURL: "", PRNum: 0},
		{Hash: "ccc", Summary: "Third", PRURL: "", PRNum: 0},
	}

	groups := groupCommits(commits, "main")

	if len(groups) != 1 {
		t.Errorf("Expected 1 group (all orphans merge), got %d", len(groups))
	}
	if len(groups[0].commits) != 3 {
		t.Errorf("Expected 3 commits in group, got %d", len(groups[0].commits))
	}
}

func TestGroupCommits_AllExisting(t *testing.T) {
	commits := []pr.StackCommitPR{
		{Hash: "aaa", Summary: "First", PRURL: "https://github.com/o/r/pull/1", PRNum: 1},
		{Hash: "bbb", Summary: "Second", PRURL: "https://github.com/o/r/pull/2", PRNum: 2},
		{Hash: "ccc", Summary: "Third", PRURL: "https://github.com/o/r/pull/3", PRNum: 3},
	}

	groups := groupCommits(commits, "main")

	if len(groups) != 3 {
		t.Fatalf("Expected 3 groups, got %d", len(groups))
	}

	for i, g := range groups {
		if g.prNumber != i+1 {
			t.Errorf("Group %d: expected prNumber %d, got %d", i, i+1, g.prNumber)
		}
		if len(g.commits) != 1 {
			t.Errorf("Group %d: expected 1 commit, got %d", i, len(g.commits))
		}
	}
}

func TestGroupCommits_OrphanAbsorbed(t *testing.T) {
	// B is new (no PR URL), inserted after A which has PR#1
	commits := []pr.StackCommitPR{
		{Hash: "aaa", Summary: "First", PRURL: "https://github.com/o/r/pull/1", PRNum: 1},
		{Hash: "bbb", Summary: "New commit", PRURL: "", PRNum: 0},
		{Hash: "ccc", Summary: "Third", PRURL: "https://github.com/o/r/pull/2", PRNum: 2},
	}

	groups := groupCommits(commits, "main")

	if len(groups) != 2 {
		t.Fatalf("Expected 2 groups, got %d", len(groups))
	}

	// First group: A + B (orphan B absorbed into A's PR)
	if groups[0].prNumber != 1 {
		t.Errorf("Group 0: expected prNumber 1, got %d", groups[0].prNumber)
	}
	if len(groups[0].commits) != 2 {
		t.Errorf("Group 0: expected 2 commits, got %d", len(groups[0].commits))
	}

	// Second group: C
	if groups[1].prNumber != 2 {
		t.Errorf("Group 1: expected prNumber 2, got %d", groups[1].prNumber)
	}
	if len(groups[1].commits) != 1 {
		t.Errorf("Group 1: expected 1 commit, got %d", len(groups[1].commits))
	}
}

func TestGroupCommits_OrphanBeforeFirst(t *testing.T) {
	// A is new (no PR URL), B has PR#1
	commits := []pr.StackCommitPR{
		{Hash: "aaa", Summary: "New first commit", PRURL: "", PRNum: 0},
		{Hash: "bbb", Summary: "Existing", PRURL: "https://github.com/o/r/pull/1", PRNum: 1},
	}

	groups := groupCommits(commits, "main")

	if len(groups) != 2 {
		t.Fatalf("Expected 2 groups, got %d", len(groups))
	}

	// First group: orphan A (needs new PR)
	if groups[0].prNumber != 0 {
		t.Errorf("Group 0: expected prNumber 0 (new PR), got %d", groups[0].prNumber)
	}
	if len(groups[0].commits) != 1 {
		t.Errorf("Group 0: expected 1 commit, got %d", len(groups[0].commits))
	}

	// Second group: B with existing PR
	if groups[1].prNumber != 1 {
		t.Errorf("Group 1: expected prNumber 1, got %d", groups[1].prNumber)
	}
}
