package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/valter-silva-au/ai-dev-brain/internal"
)

func TestInitWorkspace(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	workspaceDir := filepath.Join(tmpDir, "test-workspace")

	// Initialize app (needed for CLI)
	app, err := internal.NewApp(tmpDir)
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}
	defer app.Cleanup()

	oldApp := App
	App = app
	defer func() { App = oldApp }()

	// Create workspace init command
	cmd := newInitWorkspaceCmd()
	cmd.SetArgs([]string{workspaceDir, "--name", "test-workspace"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Verify directories were created
	dirs := []string{"tickets", "work", "sessions", ".adb"}
	for _, dir := range dirs {
		dirPath := filepath.Join(workspaceDir, dir)
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			t.Errorf("directory %s was not created", dir)
		}
	}

	// Verify files were created
	files := []string{"backlog.yaml", ".taskrc", "README.md"}
	for _, file := range files {
		filePath := filepath.Join(workspaceDir, file)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("file %s was not created", file)
		}
	}
}

func TestInitClaude(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	// Initialize app
	app, err := internal.NewApp(tmpDir)
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}
	defer app.Cleanup()

	oldApp := App
	App = app
	defer func() { App = oldApp }()

	// Create claude init command
	cmd := newInitClaudeCmd()
	cmd.SetArgs([]string{tmpDir})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Verify files were created
	claudePath := filepath.Join(tmpDir, "CLAUDE.md")
	if _, err := os.Stat(claudePath); os.IsNotExist(err) {
		t.Errorf("CLAUDE.md was not created")
	}

	userContextPath := filepath.Join(tmpDir, ".adb", "claude-user.md")
	if _, err := os.Stat(userContextPath); os.IsNotExist(err) {
		t.Errorf("claude-user.md was not created")
	}
}
