package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestRepo represents a test git repository with helper methods
type TestRepo struct {
	Dir  string
	Repo *Repository
	t    *testing.T
}

// NewTestRepo creates a new temporary git repository for testing
func NewTestRepo(t *testing.T) *TestRepo {
	t.Helper()

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "git-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		if removeErr := os.RemoveAll(tempDir); removeErr != nil {
			t.Errorf("Failed to cleanup temp dir: %v", removeErr)
		}
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Set basic git config
	testRepo := &TestRepo{Dir: tempDir, t: t}
	testRepo.GitExec("config", "user.name", "Test User")
	testRepo.GitExec("config", "user.email", "test@example.com")
	testRepo.GitExec("config", "init.defaultBranch", "main")

	// Open the repository with our git package
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current dir: %v", err)
	}

	if err := os.Chdir(tempDir); err != nil {
		if removeErr := os.RemoveAll(tempDir); removeErr != nil {
			t.Errorf("Failed to cleanup temp dir: %v", removeErr)
		}
		t.Fatalf("Failed to change to temp dir: %v", err)
	}
	repo, err := GetRepository()
	if chdirErr := os.Chdir(oldDir); chdirErr != nil {
		t.Errorf("Failed to change back to original dir: %v", chdirErr)
	}

	if err != nil {
		if removeErr := os.RemoveAll(tempDir); removeErr != nil {
			t.Errorf("Failed to cleanup temp dir: %v", removeErr)
		}
		t.Fatalf("Failed to open test repo: %v", err)
	}

	testRepo.Repo = repo
	return testRepo
}

// Cleanup removes the test repository
func (tr *TestRepo) Cleanup() {
	tr.t.Helper()
	if err := os.RemoveAll(tr.Dir); err != nil {
		tr.t.Errorf("Failed to cleanup test repo: %v", err)
	}
}

// GitExec runs a git command in the test repository (public for external use)
func (tr *TestRepo) GitExec(args ...string) string {
	tr.t.Helper()
	out, err := tr.GitExecWithError(args...)
	if err != nil {
		tr.t.Fatalf("Git command failed: %v", err)
	}
	return out
}

// GitExecWithError runs a git command in the test repository and returns both output and error
func (tr *TestRepo) GitExecWithError(args ...string) (string, error) {
	tr.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = tr.Dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("git command failed: %s\nOutput: %s", cmd.String(), out)
	}
	return strings.TrimSpace(string(out)), nil
}

// CreateFile creates a file with given content
func (tr *TestRepo) CreateFile(filename, content string) {
	tr.t.Helper()
	filePath := filepath.Join(tr.Dir, filename)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		tr.t.Fatalf("Failed to create file %s: %v", filename, err)
	}
}

// AddCommit creates a file and commits it with the given message
func (tr *TestRepo) AddCommit(filename, content, message string) {
	tr.t.Helper()
	tr.CreateFile(filename, content)
	tr.GitExec("add", filename)
	tr.GitExec("commit", "-m", message)
}

// CreateBranch creates and switches to a new branch
func (tr *TestRepo) CreateBranch(branchName string) {
	tr.t.Helper()
	tr.GitExec("checkout", "-b", branchName)
}

// SwitchBranch switches to an existing branch
func (tr *TestRepo) SwitchBranch(branchName string) {
	tr.t.Helper()
	tr.GitExec("checkout", branchName)
}

// AddRemote adds a remote repository
func (tr *TestRepo) AddRemote(name, url string) {
	tr.t.Helper()
	tr.GitExec("remote", "add", name, url)
}

// SetUpstream sets the upstream branch for the current branch
func (tr *TestRepo) SetUpstream(remote, branch string) {
	tr.t.Helper()
	tr.GitExec("branch", "--set-upstream-to", fmt.Sprintf("%s/%s", remote, branch))
}

// CreateRemoteTrackingBranch creates a remote tracking branch
func (tr *TestRepo) CreateRemoteTrackingBranch(remote, branch string) {
	tr.t.Helper()
	// Create the remote ref
	tr.GitExec("update-ref", fmt.Sprintf("refs/remotes/%s/%s", remote, branch), "HEAD")
}

// GetCurrentBranch returns the current branch name
func (tr *TestRepo) GetCurrentBranch() string {
	tr.t.Helper()
	out := tr.GitExec("rev-parse", "--abbrev-ref", "HEAD")
	return out[:len(out)-1] // Remove trailing newline
}

// InDir executes a function in the context of the test repository directory
func (tr *TestRepo) InDir(fn func()) {
	tr.t.Helper()
	oldDir, err := os.Getwd()
	if err != nil {
		tr.t.Fatalf("Failed to get current dir: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldDir); err != nil {
			tr.t.Errorf("Failed to change back to original dir: %v", err)
		}
	}()

	if err := os.Chdir(tr.Dir); err != nil {
		tr.t.Fatalf("Failed to change to test dir: %v", err)
	}

	fn()
}

// RefreshRepo refreshes the git.Repository instance after git operations
func (tr *TestRepo) RefreshRepo() {
	tr.t.Helper()
	tr.InDir(func() {
		repo, err := GetRepository()
		if err != nil {
			tr.t.Fatalf("Failed to refresh repo: %v", err)
		}
		tr.Repo = repo
	})
}
