package integration

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// validTaskID matches safe task IDs: alphanumeric with dashes and underscores
var validTaskID = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

// GitWorktreeManager manages git worktrees for multi-repo task isolation
type GitWorktreeManager interface {
	// CreateWorktree creates a new git worktree for a task
	// repoPath can be a local path, HTTPS URL, SSH URL, or platform/org/repo identifier
	// Creates worktree at basePath/work/{taskID} with new branch from baseBranch
	CreateWorktree(taskID, repoPath, baseBranch string) (string, error)

	// RemoveWorktree force-removes a worktree, resolving parent repo from .git file
	RemoveWorktree(worktreePath string) error

	// ListWorktrees returns all worktrees by parsing 'git worktree list --porcelain'
	ListWorktrees(repoPath string) ([]WorktreeInfo, error)

	// GetWorktreeForTask checks if a worktree exists for a task at basePath/work/{taskID}
	GetWorktreeForTask(taskID string) (string, bool, error)

	// NormalizeRepoPath converts various URL formats to canonical platform/org/repo
	NormalizeRepoPath(repoPath string) (string, error)
}

// WorktreeInfo represents information about a git worktree
type WorktreeInfo struct {
	Path   string
	Branch string
	Commit string
	Bare   bool
}

// DefaultGitWorktreeManager implements GitWorktreeManager
type DefaultGitWorktreeManager struct {
	basePath string // Base path for repos and worktrees
}

// NewGitWorktreeManager creates a new GitWorktreeManager
// basePath is the base directory where repos/ and work/ subdirectories will be created
func NewGitWorktreeManager(basePath string) GitWorktreeManager {
	if basePath == "" {
		basePath = "."
	}
	return &DefaultGitWorktreeManager{
		basePath: basePath,
	}
}

// NormalizeRepoPath converts various URL formats to canonical platform/org/repo
// Handles:
// - HTTPS URLs: https://github.com/org/repo.git -> github.com/org/repo
// - SSH URLs: git@github.com:org/repo.git -> github.com/org/repo
// - Relative paths: ./local/repo -> ./local/repo (unchanged)
// - Absolute paths: /path/to/repo -> /path/to/repo (unchanged)
func (m *DefaultGitWorktreeManager) NormalizeRepoPath(repoPath string) (string, error) {
	if repoPath == "" {
		return "", fmt.Errorf("repoPath cannot be empty")
	}

	// Handle local paths (relative or absolute)
	if strings.HasPrefix(repoPath, "/") || strings.HasPrefix(repoPath, "./") || strings.HasPrefix(repoPath, "../") {
		return repoPath, nil
	}

	// Handle HTTPS URLs: https://github.com/org/repo.git
	if strings.HasPrefix(repoPath, "https://") || strings.HasPrefix(repoPath, "http://") {
		// Remove protocol
		path := strings.TrimPrefix(repoPath, "https://")
		path = strings.TrimPrefix(path, "http://")
		// Remove .git suffix
		path = strings.TrimSuffix(path, ".git")
		return path, nil
	}

	// Handle SSH URLs: git@github.com:org/repo.git
	if strings.HasPrefix(repoPath, "git@") {
		// Format: git@platform:org/repo.git
		parts := strings.SplitN(repoPath, "@", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid SSH URL format: %s", repoPath)
		}
		// Split by colon
		hostRepo := strings.SplitN(parts[1], ":", 2)
		if len(hostRepo) != 2 {
			return "", fmt.Errorf("invalid SSH URL format: %s", repoPath)
		}
		platform := hostRepo[0]
		repo := strings.TrimSuffix(hostRepo[1], ".git")
		return fmt.Sprintf("%s/%s", platform, repo), nil
	}

	// Assume it's already in platform/org/repo format
	return repoPath, nil
}

// CreateWorktree creates a new git worktree for a task
func (m *DefaultGitWorktreeManager) CreateWorktree(taskID, repoPath, baseBranch string) (string, error) {
	if taskID == "" {
		return "", fmt.Errorf("taskID cannot be empty")
	}
	if repoPath == "" {
		return "", fmt.Errorf("repoPath cannot be empty")
	}
	// Validate taskID to prevent path traversal
	if !validTaskID.MatchString(taskID) {
		return "", fmt.Errorf("taskID contains invalid characters (must be alphanumeric with dashes/underscores): %s", taskID)
	}
	if strings.Contains(taskID, "..") {
		return "", fmt.Errorf("taskID contains path traversal sequence: %s", taskID)
	}
	if baseBranch == "" {
		baseBranch = "main"
	}

	// Normalize the repo path
	normalizedPath, err := m.NormalizeRepoPath(repoPath)
	if err != nil {
		return "", fmt.Errorf("failed to normalize repo path: %w", err)
	}

	// Determine if this is a local path or remote repo
	var repoDir string
	if strings.HasPrefix(normalizedPath, "/") || strings.HasPrefix(normalizedPath, "./") || strings.HasPrefix(normalizedPath, "../") {
		// Local path - use as-is
		repoDir = normalizedPath
	} else {
		// Remote repo - clone to repos/{platform}/{org}/{repo}
		repoDir = filepath.Join(m.basePath, "repos", normalizedPath)

		// Check if repo already exists
		if _, err := os.Stat(filepath.Join(repoDir, ".git")); os.IsNotExist(err) {
			// Repo doesn't exist, clone it
			if err := m.cloneRepo(repoPath, repoDir); err != nil {
				return "", fmt.Errorf("failed to clone repo: %w", err)
			}
		}
	}

	// Verify repo directory exists and is a git repo
	if _, err := os.Stat(filepath.Join(repoDir, ".git")); os.IsNotExist(err) {
		return "", fmt.Errorf("not a git repository: %s", repoDir)
	}

	// Create work directory if it doesn't exist
	workDir := filepath.Join(m.basePath, "work")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create work directory: %w", err)
	}

	// Create worktree at work/{taskID}
	worktreePath := filepath.Join(workDir, taskID)

	// Check if worktree already exists
	if _, err := os.Stat(worktreePath); err == nil {
		return "", fmt.Errorf("worktree already exists at: %s", worktreePath)
	}

	// Create new branch name for the task
	branchName := fmt.Sprintf("task/%s", taskID)

	// Create worktree with new branch
	cmd := exec.Command("git", "worktree", "add", "-b", branchName, worktreePath, baseBranch)
	cmd.Dir = repoDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to create worktree: %w: %s", err, string(output))
	}

	return worktreePath, nil
}

// cloneRepo clones a repository with HTTPS first, SSH fallback
func (m *DefaultGitWorktreeManager) cloneRepo(repoPath, targetDir string) error {
	// Create parent directory
	parentDir := filepath.Dir(targetDir)
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// Try HTTPS first
	var httpsURL string
	if strings.HasPrefix(repoPath, "https://") || strings.HasPrefix(repoPath, "http://") {
		httpsURL = repoPath
	} else if strings.HasPrefix(repoPath, "git@") {
		// Convert SSH to HTTPS
		// git@github.com:org/repo.git -> https://github.com/org/repo.git
		parts := strings.SplitN(repoPath, "@", 2)
		if len(parts) == 2 {
			hostRepo := strings.Replace(parts[1], ":", "/", 1)
			httpsURL = fmt.Sprintf("https://%s", hostRepo)
		}
	} else {
		// Assume platform/org/repo format, default to GitHub HTTPS
		httpsURL = fmt.Sprintf("https://%s.git", repoPath)
	}

	// Try cloning with HTTPS
	cmd := exec.Command("git", "clone", httpsURL, targetDir)
	output, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}

	// HTTPS failed, try SSH
	var sshURL string
	if strings.HasPrefix(repoPath, "git@") {
		sshURL = repoPath
	} else if strings.HasPrefix(repoPath, "https://") || strings.HasPrefix(repoPath, "http://") {
		// Convert HTTPS to SSH
		// https://github.com/org/repo.git -> git@github.com:org/repo.git
		url := strings.TrimPrefix(repoPath, "https://")
		url = strings.TrimPrefix(url, "http://")
		parts := strings.SplitN(url, "/", 2)
		if len(parts) == 2 {
			sshURL = fmt.Sprintf("git@%s:%s", parts[0], parts[1])
		}
	} else {
		// Assume platform/org/repo format, default to GitHub SSH
		parts := strings.SplitN(repoPath, "/", 2)
		if len(parts) == 2 {
			sshURL = fmt.Sprintf("git@%s:%s.git", parts[0], parts[1])
		}
	}

	if sshURL == "" {
		return fmt.Errorf("failed to clone with HTTPS: %w: %s", err, string(output))
	}

	// Try cloning with SSH
	cmd = exec.Command("git", "clone", sshURL, targetDir)
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to clone with HTTPS and SSH: %s", string(output))
	}

	return nil
}

// RemoveWorktree force-removes a worktree, resolving parent repo from .git file
func (m *DefaultGitWorktreeManager) RemoveWorktree(worktreePath string) error {
	if worktreePath == "" {
		return fmt.Errorf("worktreePath cannot be empty")
	}

	// Check if worktree exists
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		return fmt.Errorf("worktree does not exist: %s", worktreePath)
	}

	// Read .git file to find parent repo
	gitFile := filepath.Join(worktreePath, ".git")
	content, err := os.ReadFile(gitFile)
	if err != nil {
		return fmt.Errorf("failed to read .git file: %w", err)
	}

	// Parse .git file: "gitdir: /path/to/parent/repo/.git/worktrees/name"
	gitdirLine := strings.TrimSpace(string(content))
	if !strings.HasPrefix(gitdirLine, "gitdir: ") {
		return fmt.Errorf("invalid .git file format: %s", gitdirLine)
	}

	gitdir := strings.TrimPrefix(gitdirLine, "gitdir: ")
	// Navigate up from .git/worktrees/name to parent repo
	// gitdir is something like: /path/to/parent/.git/worktrees/taskname
	parentGitDir := filepath.Dir(filepath.Dir(gitdir))
	parentRepo := filepath.Dir(parentGitDir)

	// Remove worktree using git
	cmd := exec.Command("git", "worktree", "remove", "--force", worktreePath)
	cmd.Dir = parentRepo
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to remove worktree: %w: %s", err, string(output))
	}

	return nil
}

// ListWorktrees returns all worktrees by parsing 'git worktree list --porcelain'
func (m *DefaultGitWorktreeManager) ListWorktrees(repoPath string) ([]WorktreeInfo, error) {
	if repoPath == "" {
		return nil, fmt.Errorf("repoPath cannot be empty")
	}

	// Run git worktree list --porcelain
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w: %s", err, string(output))
	}

	// Parse porcelain output
	worktrees := []WorktreeInfo{}
	lines := strings.Split(string(output), "\n")

	var current *WorktreeInfo
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			// Empty line marks end of a worktree entry
			if current != nil {
				worktrees = append(worktrees, *current)
				current = nil
			}
			continue
		}

		if strings.HasPrefix(line, "worktree ") {
			// Start of a new worktree entry
			current = &WorktreeInfo{
				Path: strings.TrimPrefix(line, "worktree "),
			}
		} else if strings.HasPrefix(line, "HEAD ") && current != nil {
			current.Commit = strings.TrimPrefix(line, "HEAD ")
		} else if strings.HasPrefix(line, "branch ") && current != nil {
			current.Branch = strings.TrimPrefix(line, "branch ")
		} else if line == "bare" && current != nil {
			current.Bare = true
		}
	}

	// Add last entry if exists
	if current != nil {
		worktrees = append(worktrees, *current)
	}

	return worktrees, nil
}

// GetWorktreeForTask checks if a worktree exists for a task at basePath/work/{taskID}
func (m *DefaultGitWorktreeManager) GetWorktreeForTask(taskID string) (string, bool, error) {
	if taskID == "" {
		return "", false, fmt.Errorf("taskID cannot be empty")
	}

	worktreePath := filepath.Join(m.basePath, "work", taskID)

	// Check if path exists
	info, err := os.Stat(worktreePath)
	if os.IsNotExist(err) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("failed to check worktree path: %w", err)
	}

	// Verify it's a directory
	if !info.IsDir() {
		return "", false, fmt.Errorf("worktree path exists but is not a directory: %s", worktreePath)
	}

	// Verify it has a .git file (characteristic of a worktree)
	gitFile := filepath.Join(worktreePath, ".git")
	if _, err := os.Stat(gitFile); os.IsNotExist(err) {
		return "", false, fmt.Errorf("worktree path exists but is not a git worktree: %s", worktreePath)
	}

	return worktreePath, true, nil
}
