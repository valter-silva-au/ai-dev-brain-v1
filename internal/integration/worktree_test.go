package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeRepoPath(t *testing.T) {
	manager := NewGitWorktreeManager("")

	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "HTTPS URL",
			input:    "https://github.com/org/repo.git",
			expected: "github.com/org/repo",
			wantErr:  false,
		},
		{
			name:     "HTTPS URL without .git",
			input:    "https://github.com/org/repo",
			expected: "github.com/org/repo",
			wantErr:  false,
		},
		{
			name:     "HTTP URL",
			input:    "http://github.com/org/repo.git",
			expected: "github.com/org/repo",
			wantErr:  false,
		},
		{
			name:     "SSH URL",
			input:    "git@github.com:org/repo.git",
			expected: "github.com/org/repo",
			wantErr:  false,
		},
		{
			name:     "SSH URL without .git",
			input:    "git@github.com:org/repo",
			expected: "github.com/org/repo",
			wantErr:  false,
		},
		{
			name:     "Absolute local path",
			input:    "/home/user/repos/myrepo",
			expected: "/home/user/repos/myrepo",
			wantErr:  false,
		},
		{
			name:     "Relative local path",
			input:    "./local/repo",
			expected: "./local/repo",
			wantErr:  false,
		},
		{
			name:     "Relative parent path",
			input:    "../parent/repo",
			expected: "../parent/repo",
			wantErr:  false,
		},
		{
			name:     "Platform/org/repo format",
			input:    "github.com/org/repo",
			expected: "github.com/org/repo",
			wantErr:  false,
		},
		{
			name:     "Empty path",
			input:    "",
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := manager.NormalizeRepoPath(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NormalizeRepoPath() expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("NormalizeRepoPath() unexpected error: %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("NormalizeRepoPath() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCreateWorktreeLocalRepo(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Initialize a git repo
	repoDir := filepath.Join(tempDir, "test-repo")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatalf("Failed to create repo directory: %v", err)
	}

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = repoDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to init git repo: %v: %s", err, string(output))
	}

	// Configure git user
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = repoDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to config git user email: %v: %s", err, string(output))
	}

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = repoDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to config git user name: %v: %s", err, string(output))
	}

	// Create initial commit
	testFile := filepath.Join(repoDir, "README.md")
	if err := os.WriteFile(testFile, []byte("# Test Repo\n"), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd = exec.Command("git", "add", "README.md")
	cmd.Dir = repoDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to add file: %v: %s", err, string(output))
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = repoDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to commit: %v: %s", err, string(output))
	}

	// Create main branch (git init creates 'master' by default in some versions)
	cmd = exec.Command("git", "branch", "-M", "main")
	cmd.Dir = repoDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to create main branch: %v: %s", err, string(output))
	}

	// Create worktree manager
	workBase := filepath.Join(tempDir, "workspace")
	manager := NewGitWorktreeManager(workBase)

	// Create worktree for a task
	taskID := "TASK-001"
	worktreePath, err := manager.CreateWorktree(taskID, repoDir, "main")
	if err != nil {
		t.Fatalf("CreateWorktree() failed: %v", err)
	}

	// Verify worktree was created
	expectedPath := filepath.Join(workBase, "work", taskID)
	if worktreePath != expectedPath {
		t.Errorf("Expected worktree at %s, got %s", expectedPath, worktreePath)
	}

	// Verify worktree directory exists
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		t.Errorf("Worktree directory does not exist: %s", worktreePath)
	}

	// Verify .git file exists
	gitFile := filepath.Join(worktreePath, ".git")
	if _, err := os.Stat(gitFile); os.IsNotExist(err) {
		t.Errorf(".git file does not exist: %s", gitFile)
	}

	// Verify README.md exists in worktree
	readmeInWorktree := filepath.Join(worktreePath, "README.md")
	if _, err := os.Stat(readmeInWorktree); os.IsNotExist(err) {
		t.Errorf("README.md does not exist in worktree: %s", readmeInWorktree)
	}
}

func TestCreateWorktreeEmptyParams(t *testing.T) {
	manager := NewGitWorktreeManager("")

	tests := []struct {
		name       string
		taskID     string
		repoPath   string
		baseBranch string
		wantErr    bool
		errContains string
	}{
		{
			name:        "Empty taskID",
			taskID:      "",
			repoPath:    "/some/repo",
			baseBranch:  "main",
			wantErr:     true,
			errContains: "taskID cannot be empty",
		},
		{
			name:        "Empty repoPath",
			taskID:      "TASK-001",
			repoPath:    "",
			baseBranch:  "main",
			wantErr:     true,
			errContains: "repoPath cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := manager.CreateWorktree(tt.taskID, tt.repoPath, tt.baseBranch)
			if tt.wantErr {
				if err == nil {
					t.Errorf("CreateWorktree() expected error but got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("CreateWorktree() error = %v, want error containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("CreateWorktree() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestRemoveWorktree(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Initialize a git repo
	repoDir := filepath.Join(tempDir, "test-repo")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatalf("Failed to create repo directory: %v", err)
	}

	// Initialize git repo with initial commit
	setupGitRepo(t, repoDir)

	// Create worktree manager
	workBase := filepath.Join(tempDir, "workspace")
	manager := NewGitWorktreeManager(workBase)

	// Create worktree
	taskID := "TASK-002"
	worktreePath, err := manager.CreateWorktree(taskID, repoDir, "main")
	if err != nil {
		t.Fatalf("CreateWorktree() failed: %v", err)
	}

	// Verify worktree exists
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		t.Fatalf("Worktree was not created: %s", worktreePath)
	}

	// Remove worktree
	if err := manager.RemoveWorktree(worktreePath); err != nil {
		t.Fatalf("RemoveWorktree() failed: %v", err)
	}

	// Verify worktree was removed
	if _, err := os.Stat(worktreePath); !os.IsNotExist(err) {
		t.Errorf("Worktree still exists after removal: %s", worktreePath)
	}
}

func TestRemoveWorktreeErrors(t *testing.T) {
	manager := NewGitWorktreeManager("")

	tests := []struct {
		name        string
		path        string
		wantErr     bool
		errContains string
	}{
		{
			name:        "Empty path",
			path:        "",
			wantErr:     true,
			errContains: "worktreePath cannot be empty",
		},
		{
			name:        "Non-existent path",
			path:        "/nonexistent/path",
			wantErr:     true,
			errContains: "worktree does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.RemoveWorktree(tt.path)
			if tt.wantErr {
				if err == nil {
					t.Errorf("RemoveWorktree() expected error but got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("RemoveWorktree() error = %v, want error containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("RemoveWorktree() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestListWorktrees(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Initialize a git repo
	repoDir := filepath.Join(tempDir, "test-repo")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatalf("Failed to create repo directory: %v", err)
	}

	// Initialize git repo with initial commit
	setupGitRepo(t, repoDir)

	// Create worktree manager
	workBase := filepath.Join(tempDir, "workspace")
	manager := NewGitWorktreeManager(workBase)

	// Create multiple worktrees
	taskIDs := []string{"TASK-003", "TASK-004"}
	for _, taskID := range taskIDs {
		if _, err := manager.CreateWorktree(taskID, repoDir, "main"); err != nil {
			t.Fatalf("CreateWorktree(%s) failed: %v", taskID, err)
		}
	}

	// List worktrees
	worktrees, err := manager.ListWorktrees(repoDir)
	if err != nil {
		t.Fatalf("ListWorktrees() failed: %v", err)
	}

	// Verify we have at least 3 worktrees (main repo + 2 task worktrees)
	if len(worktrees) < 3 {
		t.Errorf("Expected at least 3 worktrees, got %d", len(worktrees))
	}

	// Verify main repo is listed
	foundMain := false
	for _, wt := range worktrees {
		if wt.Path == repoDir {
			foundMain = true
			break
		}
	}
	if !foundMain {
		t.Errorf("Main repository not found in worktree list")
	}

	// Verify task worktrees are listed
	for _, taskID := range taskIDs {
		expectedPath := filepath.Join(workBase, "work", taskID)
		found := false
		for _, wt := range worktrees {
			if wt.Path == expectedPath {
				found = true
				// Verify branch name
				expectedBranch := "refs/heads/task/" + taskID
				if wt.Branch != expectedBranch {
					t.Errorf("Expected branch %s for worktree %s, got %s", expectedBranch, taskID, wt.Branch)
				}
				// Verify commit is set
				if wt.Commit == "" {
					t.Errorf("Commit hash not set for worktree %s", taskID)
				}
				break
			}
		}
		if !found {
			t.Errorf("Worktree for %s not found in list", taskID)
		}
	}
}

func TestListWorktreesErrors(t *testing.T) {
	manager := NewGitWorktreeManager("")

	tests := []struct {
		name        string
		repoPath    string
		wantErr     bool
		errContains string
	}{
		{
			name:        "Empty repo path",
			repoPath:    "",
			wantErr:     true,
			errContains: "repoPath cannot be empty",
		},
		{
			name:        "Non-git directory",
			repoPath:    os.TempDir(),
			wantErr:     true,
			errContains: "failed to list worktrees",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := manager.ListWorktrees(tt.repoPath)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ListWorktrees() expected error but got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ListWorktrees() error = %v, want error containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("ListWorktrees() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestGetWorktreeForTask(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Initialize a git repo
	repoDir := filepath.Join(tempDir, "test-repo")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatalf("Failed to create repo directory: %v", err)
	}

	// Initialize git repo with initial commit
	setupGitRepo(t, repoDir)

	// Create worktree manager
	workBase := filepath.Join(tempDir, "workspace")
	manager := NewGitWorktreeManager(workBase)

	// Create worktree for a task
	taskID := "TASK-005"
	expectedPath, err := manager.CreateWorktree(taskID, repoDir, "main")
	if err != nil {
		t.Fatalf("CreateWorktree() failed: %v", err)
	}

	// Test GetWorktreeForTask with existing task
	path, exists, err := manager.GetWorktreeForTask(taskID)
	if err != nil {
		t.Errorf("GetWorktreeForTask() unexpected error: %v", err)
	}
	if !exists {
		t.Errorf("GetWorktreeForTask() exists = false, want true")
	}
	if path != expectedPath {
		t.Errorf("GetWorktreeForTask() path = %s, want %s", path, expectedPath)
	}

	// Test GetWorktreeForTask with non-existent task
	nonExistentTaskID := "TASK-999"
	path, exists, err = manager.GetWorktreeForTask(nonExistentTaskID)
	if err != nil {
		t.Errorf("GetWorktreeForTask() unexpected error: %v", err)
	}
	if exists {
		t.Errorf("GetWorktreeForTask() exists = true, want false")
	}
	if path != "" {
		t.Errorf("GetWorktreeForTask() path = %s, want empty string", path)
	}
}

func TestGetWorktreeForTaskErrors(t *testing.T) {
	manager := NewGitWorktreeManager("")

	// Test with empty taskID
	_, _, err := manager.GetWorktreeForTask("")
	if err == nil {
		t.Errorf("GetWorktreeForTask() expected error with empty taskID, got nil")
	}
	if !strings.Contains(err.Error(), "taskID cannot be empty") {
		t.Errorf("GetWorktreeForTask() error = %v, want error containing 'taskID cannot be empty'", err)
	}
}

func TestCreateWorktreeDefaultBaseBranch(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Initialize a git repo
	repoDir := filepath.Join(tempDir, "test-repo")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatalf("Failed to create repo directory: %v", err)
	}

	// Initialize git repo with initial commit
	setupGitRepo(t, repoDir)

	// Create worktree manager
	workBase := filepath.Join(tempDir, "workspace")
	manager := NewGitWorktreeManager(workBase)

	// Create worktree with empty baseBranch (should default to "main")
	taskID := "TASK-006"
	_, err := manager.CreateWorktree(taskID, repoDir, "")
	if err != nil {
		t.Fatalf("CreateWorktree() with empty baseBranch failed: %v", err)
	}

	// Verify worktree was created
	expectedPath := filepath.Join(workBase, "work", taskID)
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Worktree was not created with default baseBranch: %s", expectedPath)
	}
}

func TestCreateWorktreeAlreadyExists(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Initialize a git repo
	repoDir := filepath.Join(tempDir, "test-repo")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatalf("Failed to create repo directory: %v", err)
	}

	// Initialize git repo with initial commit
	setupGitRepo(t, repoDir)

	// Create worktree manager
	workBase := filepath.Join(tempDir, "workspace")
	manager := NewGitWorktreeManager(workBase)

	// Create worktree
	taskID := "TASK-007"
	_, err := manager.CreateWorktree(taskID, repoDir, "main")
	if err != nil {
		t.Fatalf("CreateWorktree() failed: %v", err)
	}

	// Try to create worktree again with same taskID
	_, err = manager.CreateWorktree(taskID, repoDir, "main")
	if err == nil {
		t.Errorf("CreateWorktree() expected error when worktree already exists, got nil")
	}
	if !strings.Contains(err.Error(), "worktree already exists") {
		t.Errorf("CreateWorktree() error = %v, want error containing 'worktree already exists'", err)
	}
}

// setupGitRepo is a helper function to initialize a git repo with an initial commit
func setupGitRepo(t *testing.T, repoDir string) {
	t.Helper()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = repoDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to init git repo: %v: %s", err, string(output))
	}

	// Configure git user
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = repoDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to config git user email: %v: %s", err, string(output))
	}

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = repoDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to config git user name: %v: %s", err, string(output))
	}

	// Create initial commit
	testFile := filepath.Join(repoDir, "README.md")
	if err := os.WriteFile(testFile, []byte("# Test Repo\n"), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd = exec.Command("git", "add", "README.md")
	cmd.Dir = repoDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to add file: %v: %s", err, string(output))
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = repoDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to commit: %v: %s", err, string(output))
	}

	// Create main branch
	cmd = exec.Command("git", "branch", "-M", "main")
	cmd.Dir = repoDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to create main branch: %v: %s", err, string(output))
	}
}
