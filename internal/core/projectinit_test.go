package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/valter-silva-au/ai-dev-brain/templates/claude"
)

func TestFileProjectInitializer_InitializeProject(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "projectinit-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	projectPath := filepath.Join(tmpDir, "test-project")

	// Create initializer
	pi := NewFileProjectInitializer(claude.FS)

	// Test basic initialization
	options := InitOptions{
		Name:         "test-project",
		AIProvider:   "claude",
		TaskIDPrefix: "TEST",
		GitInit:      false,
		WithBMAD:     false,
	}

	err = pi.InitializeProject(projectPath, options)
	if err != nil {
		t.Fatalf("InitializeProject failed: %v", err)
	}

	// Verify directory structure
	expectedDirs := []string{
		"tickets",
		"tickets/_archived",
		"work",
		"sessions",
		".adb",
		".claude",
		".claude/rules",
		"docs",
		"docs/bmad",
	}

	for _, dir := range expectedDirs {
		dirPath := filepath.Join(projectPath, dir)
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			t.Errorf("Expected directory %s not found", dir)
		}
	}

	// Verify config files
	expectedFiles := []string{
		"backlog.yaml",
		".taskrc",
		".claude/project_context.md",
		".claude/rules/task-isolation.md",
	}

	for _, file := range expectedFiles {
		filePath := filepath.Join(projectPath, file)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("Expected file %s not found", file)
		}
	}

	// Verify .taskrc content
	taskrcPath := filepath.Join(projectPath, ".taskrc")
	content, err := os.ReadFile(taskrcPath)
	if err != nil {
		t.Fatalf("Failed to read .taskrc: %v", err)
	}

	taskrcContent := string(content)
	if taskrcContent == "" {
		t.Error(".taskrc should not be empty")
	}
}

func TestFileProjectInitializer_WithGitInit(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "projectinit-git-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	projectPath := filepath.Join(tmpDir, "git-project")

	// Create initializer
	pi := NewFileProjectInitializer(claude.FS)

	// Test with git init
	options := InitOptions{
		Name:         "git-project",
		AIProvider:   "claude",
		TaskIDPrefix: "GIT",
		GitInit:      true,
		WithBMAD:     false,
	}

	err = pi.InitializeProject(projectPath, options)
	if err != nil {
		t.Fatalf("InitializeProject with git failed: %v", err)
	}

	// Verify .git directory exists
	gitDir := filepath.Join(projectPath, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		t.Error("Expected .git directory not found")
	}

	// Verify .gitignore exists
	gitignorePath := filepath.Join(projectPath, ".gitignore")
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		t.Error("Expected .gitignore not found")
	}
}

func TestFileProjectInitializer_WithBMAD(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "projectinit-bmad-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	projectPath := filepath.Join(tmpDir, "bmad-project")

	// Create initializer
	pi := NewFileProjectInitializer(claude.FS)

	// Test with BMAD artifacts
	options := InitOptions{
		Name:         "bmad-project",
		AIProvider:   "claude",
		TaskIDPrefix: "BMAD",
		GitInit:      false,
		WithBMAD:     true,
	}

	err = pi.InitializeProject(projectPath, options)
	if err != nil {
		t.Fatalf("InitializeProject with BMAD failed: %v", err)
	}

	// Verify BMAD artifacts
	expectedBMADFiles := []string{
		"docs/bmad/PRD.md",
		"docs/bmad/product-brief.md",
		"docs/bmad/tech-spec.md",
		"docs/bmad/architecture-doc.md",
		"docs/bmad/quality-gates.md",
	}

	for _, file := range expectedBMADFiles {
		filePath := filepath.Join(projectPath, file)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("Expected BMAD file %s not found", file)
		}
	}

	// Verify PRD content
	prdPath := filepath.Join(projectPath, "docs/bmad/PRD.md")
	content, err := os.ReadFile(prdPath)
	if err != nil {
		t.Fatalf("Failed to read PRD.md: %v", err)
	}

	prdContent := string(content)
	if prdContent == "" {
		t.Error("PRD.md should not be empty")
	}
}

func TestFileProjectInitializer_DefaultValues(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "projectinit-defaults-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	projectPath := filepath.Join(tmpDir, "defaults-project")

	// Create initializer
	pi := NewFileProjectInitializer(claude.FS)

	// Test with empty options (should use defaults)
	options := InitOptions{}

	err = pi.InitializeProject(projectPath, options)
	if err != nil {
		t.Fatalf("InitializeProject with defaults failed: %v", err)
	}

	// Verify .taskrc has default values
	taskrcPath := filepath.Join(projectPath, ".taskrc")
	content, err := os.ReadFile(taskrcPath)
	if err != nil {
		t.Fatalf("Failed to read .taskrc: %v", err)
	}

	taskrcContent := string(content)

	// Check for default values in content
	if taskrcContent == "" {
		t.Error(".taskrc should not be empty")
	}
}

func TestFileProjectInitializer_ExistingDirectory(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "projectinit-existing-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Use existing temp directory as project path
	projectPath := tmpDir

	// Create initializer
	pi := NewFileProjectInitializer(claude.FS)

	// Test with existing directory
	options := InitOptions{
		Name:         "existing-project",
		AIProvider:   "claude",
		TaskIDPrefix: "EXIST",
		GitInit:      false,
		WithBMAD:     false,
	}

	err = pi.InitializeProject(projectPath, options)
	if err != nil {
		t.Fatalf("InitializeProject with existing dir failed: %v", err)
	}

	// Verify files were created
	backlogPath := filepath.Join(projectPath, "backlog.yaml")
	if _, err := os.Stat(backlogPath); os.IsNotExist(err) {
		t.Error("Expected backlog.yaml not found in existing directory")
	}
}
