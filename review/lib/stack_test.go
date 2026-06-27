package review

import (
	"strings"
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

func TestGroupCommits_SentinelStartsNewGroup(t *testing.T) {
	// B has a bare "PR URL:" sentinel — it should start its own group
	commits := []pr.StackCommitPR{
		{Hash: "aaa", Summary: "First", PRURL: "https://github.com/o/r/pull/1", PRNum: 1, WantsPR: true},
		{Hash: "bbb", Summary: "New PR here", PRURL: "", PRNum: 0, WantsPR: true},
		{Hash: "ccc", Summary: "Third", PRURL: "https://github.com/o/r/pull/2", PRNum: 2, WantsPR: true},
	}

	groups := groupCommits(commits, "main")

	if len(groups) != 3 {
		t.Fatalf("Expected 3 groups (sentinel should not be absorbed), got %d", len(groups))
	}

	// First group: A with existing PR
	if groups[0].prNumber != 1 {
		t.Errorf("Group 0: expected prNumber 1, got %d", groups[0].prNumber)
	}

	// Second group: B (sentinel, needs new PR)
	if groups[1].prNumber != 0 {
		t.Errorf("Group 1: expected prNumber 0 (new PR), got %d", groups[1].prNumber)
	}
	if len(groups[1].commits) != 1 {
		t.Errorf("Group 1: expected 1 commit, got %d", len(groups[1].commits))
	}

	// Third group: C with existing PR
	if groups[2].prNumber != 2 {
		t.Errorf("Group 2: expected prNumber 2, got %d", groups[2].prNumber)
	}
}

func TestGroupCommits_SentinelWithFollowingOrphan(t *testing.T) {
	// B has sentinel, C is a plain orphan — C should be absorbed into B's group
	commits := []pr.StackCommitPR{
		{Hash: "aaa", Summary: "First", PRURL: "https://github.com/o/r/pull/1", PRNum: 1, WantsPR: true},
		{Hash: "bbb", Summary: "New PR here", PRURL: "", PRNum: 0, WantsPR: true},
		{Hash: "ccc", Summary: "Follow-up", PRURL: "", PRNum: 0, WantsPR: false},
	}

	groups := groupCommits(commits, "main")

	if len(groups) != 2 {
		t.Fatalf("Expected 2 groups, got %d", len(groups))
	}

	// Second group: B + C (orphan C absorbed into sentinel B's group)
	if groups[1].prNumber != 0 {
		t.Errorf("Group 1: expected prNumber 0, got %d", groups[1].prNumber)
	}
	if len(groups[1].commits) != 2 {
		t.Errorf("Group 1: expected 2 commits, got %d", len(groups[1].commits))
	}
}

func TestBuildStackSection(t *testing.T) {
	prs := []stackPRInfo{
		{title: "Add auth module", prNumber: 10},
		{title: "Add user profile", prNumber: 11},
		{title: "Add settings page", prNumber: 12},
	}

	// Test highlighting the second PR
	section := buildStackSection(prs, 1)

	expected := "---\n## PR Stack\n" +
		"1. #10\n" +
		"2. :star: #11\n" +
		"3. #12\n"

	if section != expected {
		t.Errorf("buildStackSection mismatch.\nExpected:\n%s\nGot:\n%s", expected, section)
	}

	// Test highlighting the first PR
	section = buildStackSection(prs, 0)
	if !strings.HasPrefix(section, "---\n## PR Stack\n1. :star:") {
		t.Errorf("Expected first entry to be starred, got:\n%s", section)
	}
}

func TestBuildStackSection_SinglePR(t *testing.T) {
	prs := []stackPRInfo{
		{title: "Solo PR", prNumber: 42},
	}

	section := buildStackSection(prs, 0)
	expected := "---\n## PR Stack\n1. :star: #42\n"

	if section != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, section)
	}
}

func TestUpsertStackSection_Append(t *testing.T) {
	body := "Some PR description.\n\n## Test plan\n- [ ] Test it"
	section := "---\n## PR Stack\n1. :star: #1\n"

	result := upsertStackSection(body, section)

	if !strings.Contains(result, "## Test plan") {
		t.Error("Original body content was lost")
	}
	if !strings.Contains(result, "## PR Stack") {
		t.Error("Stack section was not appended")
	}
	if !strings.Contains(result, ":star: #1") {
		t.Error("Stack entry not found")
	}
}

func TestUpsertStackSection_Replace(t *testing.T) {
	body := "Some description.\n\n## PR Stack\n1. :star: #1\n2. #2\n\n## Test plan\n- [ ] done"
	section := "---\n## PR Stack\n1. #1\n2. :star: #2\n3. #3\n"

	result := upsertStackSection(body, section)

	if !strings.Contains(result, ":star: #2") {
		t.Errorf("New stack entry not found in:\n%s", result)
	}
	if !strings.Contains(result, "## Test plan") {
		t.Errorf("Test plan section was lost in:\n%s", result)
	}
	if !strings.Contains(result, "Some description.") {
		t.Errorf("Original description was lost in:\n%s", result)
	}
}

func TestUpsertStackSection_ReplaceWithSeparator(t *testing.T) {
	// Existing body already has --- before ## PR Stack
	body := "Some description.\n\n---\n## PR Stack\n1. :star: #1\n2. #2\n\n## Test plan\n- [ ] done"
	section := "---\n## PR Stack\n1. #1\n2. :star: #2\n3. #3\n"

	result := upsertStackSection(body, section)

	// Should not have duplicate ---
	if strings.Contains(result, "---\n---") {
		t.Errorf("Duplicate --- separators in:\n%s", result)
	}
	if !strings.Contains(result, ":star: #2") {
		t.Errorf("New stack entry not found in:\n%s", result)
	}
	if !strings.Contains(result, "## Test plan") {
		t.Errorf("Test plan section was lost in:\n%s", result)
	}
	if !strings.Contains(result, "Some description.") {
		t.Errorf("Original description was lost in:\n%s", result)
	}
}
