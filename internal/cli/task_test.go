package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/valter-silva-au/ai-dev-brain/internal"
)

// TestTaskCommandsExist verifies that all task commands are registered
func TestTaskCommandsExist(t *testing.T) {
	rootCmd := NewRootCmd()

	expectedCommands := []string{
		"task",
	}

	for _, cmdName := range expectedCommands {
		found := false
		for _, cmd := range rootCmd.Commands() {
			if cmd.Name() == cmdName {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected command %q to be registered", cmdName)
		}
	}
}

// TestTaskSubcommandsExist verifies that all task subcommands are registered
func TestTaskSubcommandsExist(t *testing.T) {
	taskCmd := NewTaskCmd()

	expectedSubcommands := []string{
		"create",
		"resume",
		"archive",
		"unarchive",
		"cleanup",
		"status",
		"priority",
		"update",
	}

	for _, subcmdName := range expectedSubcommands {
		found := false
		for _, cmd := range taskCmd.Commands() {
			if cmd.Name() == subcmdName {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected subcommand %q to be registered under task", subcmdName)
		}
	}
}

// TestTaskCreateFlags verifies that task create has the expected flags
func TestTaskCreateFlags(t *testing.T) {
	cmd := newTaskCreateCmd()

	expectedFlags := []string{
		"type",
		"repo",
		"priority",
		"owner",
		"tags",
		"description",
		"acceptance",
	}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("Expected flag %q to be defined", flagName)
		}
	}
}

// TestTaskArchiveFlags verifies that task archive has the expected flags
func TestTaskArchiveFlags(t *testing.T) {
	cmd := newTaskArchiveCmd()

	expectedFlags := []string{
		"force",
		"keep-worktree",
	}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("Expected flag %q to be defined", flagName)
		}
	}
}

// TestTaskStatusFlags verifies that task status has the expected flags
func TestTaskStatusFlags(t *testing.T) {
	cmd := newTaskStatusCmd()

	expectedFlags := []string{
		"filter",
	}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("Expected flag %q to be defined", flagName)
		}
	}
}

// TestTaskPriorityFlags verifies that task priority has the expected flags
func TestTaskPriorityFlags(t *testing.T) {
	cmd := newTaskPriorityCmd()

	expectedFlags := []string{
		"priority",
	}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("Expected flag %q to be defined", flagName)
		}
	}
}

// TestTaskUpdateFlags verifies that task update has the expected flags
func TestTaskUpdateFlags(t *testing.T) {
	cmd := newTaskUpdateCmd()

	expectedFlags := []string{
		"status",
		"priority",
		"owner",
	}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("Expected flag %q to be defined", flagName)
		}
	}
}

// TestTaskCreateValidation tests validation of task create arguments
func TestTaskCreateValidation(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()

	// Initialize app
	app, err := internal.NewApp(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}
	defer app.Cleanup()

	// Set global App
	App = app

	tests := []struct {
		name        string
		args        []string
		taskType    string
		priority    string
		expectError bool
		skipExec    bool // Skip actual execution (for valid cases that require git)
	}{
		{
			name:        "invalid type",
			args:        []string{"test-branch"},
			taskType:    "invalid",
			priority:    "P2",
			expectError: true,
			skipExec:    false,
		},
		{
			name:        "invalid priority",
			args:        []string{"test-branch"},
			taskType:    "feat",
			priority:    "P9",
			expectError: true,
			skipExec:    false,
		},
		{
			name:        "missing branch arg",
			args:        []string{},
			taskType:    "feat",
			priority:    "P2",
			expectError: true,
			skipExec:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipExec {
				t.Skip("Skipping execution test (requires git)")
			}

			cmd := newTaskCreateCmd()
			cmd.SetArgs(tt.args)
			cmd.Flags().Set("type", tt.taskType)
			cmd.Flags().Set("priority", tt.priority)

			// Capture output
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)

			err := cmd.Execute()
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

// TestRootCommandVersion tests that version info is properly set
func TestRootCommandVersion(t *testing.T) {
	// Set test version info
	Version = "1.0.0"
	Commit = "abc123"
	Date = "2024-01-01"

	rootCmd := NewRootCmd()

	// Check version string contains expected info
	if rootCmd.Version == "" {
		t.Error("Expected version to be set")
	}

	// Version should contain all three pieces of info
	version := rootCmd.Version
	if version != "1.0.0 (commit: abc123, built: 2024-01-01)" {
		t.Errorf("Unexpected version format: %s", version)
	}
}

// TestResolveBasePath tests the base path resolution logic
func TestResolveBasePath(t *testing.T) {
	// Save original env
	originalAdbHome := os.Getenv("ADB_HOME")
	defer os.Setenv("ADB_HOME", originalAdbHome)

	t.Run("ADB_HOME set", func(t *testing.T) {
		tmpDir := t.TempDir()
		os.Setenv("ADB_HOME", tmpDir)

		// Create a simple main function test
		basePath, err := resolveBasePathHelper()
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if basePath != tmpDir {
			t.Errorf("Expected basePath %s, got %s", tmpDir, basePath)
		}
	})

	t.Run("ADB_HOME not set, .taskrc exists", func(t *testing.T) {
		os.Unsetenv("ADB_HOME")

		tmpDir := t.TempDir()
		configFile := filepath.Join(tmpDir, ".taskrc")
		os.WriteFile(configFile, []byte("test"), 0644)

		// Change to tmpDir
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		os.Chdir(tmpDir)

		basePath, err := resolveBasePathHelper()
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if basePath != tmpDir {
			t.Errorf("Expected basePath %s, got %s", tmpDir, basePath)
		}
	})

	t.Run("ADB_HOME not set, fallback to current dir", func(t *testing.T) {
		os.Unsetenv("ADB_HOME")

		tmpDir := t.TempDir()

		// Change to tmpDir
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		os.Chdir(tmpDir)

		basePath, err := resolveBasePathHelper()
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if basePath != tmpDir {
			t.Errorf("Expected basePath %s, got %s", tmpDir, basePath)
		}
	})
}

// resolveBasePathHelper is a copy of the resolveBasePath function from main.go for testing
func resolveBasePathHelper() (string, error) {
	// Check ADB_HOME environment variable
	if adbHome := os.Getenv("ADB_HOME"); adbHome != "" {
		absPath, err := filepath.Abs(adbHome)
		if err != nil {
			return "", err
		}
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			return "", err
		}
		return absPath, nil
	}

	// Walk up from current directory looking for .taskconfig
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	dir := currentDir
	for {
		// Check if .taskconfig exists in this directory
		configPath := filepath.Join(dir, ".taskconfig")
		if _, err := os.Stat(configPath); err == nil {
			return dir, nil
		}

		// Check if .taskrc exists (repo-level config)
		configPath = filepath.Join(dir, ".taskrc")
		if _, err := os.Stat(configPath); err == nil {
			return dir, nil
		}

		// Move up one directory
		parentDir := filepath.Dir(dir)
		if parentDir == dir {
			// Reached root, stop
			break
		}
		dir = parentDir
	}

	// Fallback to current directory
	return currentDir, nil
}
