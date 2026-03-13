package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/valter-silva-au/ai-dev-brain/internal"
)

func TestSyncCommands(t *testing.T) {
	// Create temporary workspace
	tmpDir := t.TempDir()

	// Initialize app
	app, err := internal.NewApp(tmpDir)
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}
	defer app.Cleanup()

	// Set global app for CLI
	oldApp := App
	App = app
	defer func() { App = oldApp }()

	// Create initial backlog
	backlogPath := filepath.Join(tmpDir, "backlog.yaml")
	if err := os.WriteFile(backlogPath, []byte("tasks: []\n"), 0o644); err != nil {
		t.Fatalf("failed to create backlog: %v", err)
	}

	tests := []struct {
		name    string
		cmd     func() *cobra.Command
		args    []string
		wantErr bool
	}{
		{
			name:    "sync context",
			cmd:     newSyncContextCmd,
			args:    []string{},
			wantErr: false,
		},
		{
			name:    "sync repos",
			cmd:     newSyncReposCmd,
			args:    []string{},
			wantErr: false,
		},
		{
			name:    "sync claude-user",
			cmd:     newSyncClaudeUserCmd,
			args:    []string{},
			wantErr: false,
		},
		{
			name:    "sync claude-user dry-run",
			cmd:     newSyncClaudeUserCmd,
			args:    []string{"--dry-run"},
			wantErr: false,
		},
		{
			name:    "sync all",
			cmd:     newSyncAllCmd,
			args:    []string{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.cmd()
			cmd.SetArgs(tt.args)
			err := cmd.Execute()
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSyncTaskContext(t *testing.T) {
	// Create temporary workspace
	tmpDir := t.TempDir()

	// Initialize app
	app, err := internal.NewApp(tmpDir)
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}
	defer app.Cleanup()

	// Set global app for CLI
	oldApp := App
	App = app
	defer func() { App = oldApp }()

	// Create task directory
	taskID := "TASK-00001"
	taskDir := filepath.Join(tmpDir, "tickets", taskID)
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("failed to create task directory: %v", err)
	}

	cmd := newSyncTaskContextCmd()
	cmd.SetArgs([]string{taskID})
	if err := cmd.Execute(); err != nil {
		t.Errorf("Execute() error = %v", err)
	}

	// Verify context was created
	contextPath := filepath.Join(taskDir, "context.md")
	if _, err := os.Stat(contextPath); os.IsNotExist(err) {
		t.Errorf("context.md was not created")
	}
}
