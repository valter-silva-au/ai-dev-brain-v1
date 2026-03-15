package internal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/valter-silva-au/ai-dev-brain/pkg/models"
)

func TestNewApp(t *testing.T) {
	// Create temporary workspace
	tmpDir := t.TempDir()

	// Test creating app
	app, err := NewApp(tmpDir)
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}

	// Verify app structure
	if app == nil {
		t.Fatal("NewApp() returned nil app")
	}

	// Verify base path
	if app.BasePath != tmpDir {
		t.Errorf("BasePath = %v, want %v", app.BasePath, tmpDir)
	}

	// Verify configuration subsystem
	if app.ConfigManager == nil {
		t.Error("ConfigManager is nil")
	}
	if app.MergedConfig == nil {
		t.Error("MergedConfig is nil")
	}

	// Verify storage subsystem
	if app.BacklogManager == nil {
		t.Error("BacklogManager is nil")
	}
	if app.ContextManager == nil {
		t.Error("ContextManager is nil")
	}
	if app.SessionStoreManager == nil {
		t.Error("SessionStoreManager is nil")
	}

	// Verify core services
	if app.TaskIDGenerator == nil {
		t.Error("TaskIDGenerator is nil")
	}
	if app.TemplateManager == nil {
		t.Error("TemplateManager is nil")
	}
	if app.TaskManager == nil {
		t.Error("TaskManager is nil")
	}

	// Verify integration subsystem
	if app.GitWorktreeManager == nil {
		t.Error("GitWorktreeManager is nil")
	}
	if app.TerminalStateWriter == nil {
		t.Error("TerminalStateWriter is nil")
	}

	// Verify observability subsystem
	if app.EventLog == nil {
		t.Error("EventLog is nil")
	}
	if app.MetricsCalculator == nil {
		t.Error("MetricsCalculator is nil")
	}
	if app.AlertEvaluator == nil {
		t.Error("AlertEvaluator is nil")
	}
}

func TestNewApp_EmptyBasePath(t *testing.T) {
	// Test with empty base path (should use ".")
	app, err := NewApp("")
	if err != nil {
		t.Fatalf("NewApp(\"\") error = %v", err)
	}

	if app.BasePath != "." {
		t.Errorf("BasePath = %v, want %v", app.BasePath, ".")
	}
}

func TestApp_Integration_ConfigurationLoading(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test .taskrc file
	taskrcContent := `task_id_prefix: "TEST"
base_branch: "develop"
reviewers:
  - "reviewer1"
  - "reviewer2"
`
	taskrcPath := filepath.Join(tmpDir, ".taskrc")
	if err := os.WriteFile(taskrcPath, []byte(taskrcContent), 0o644); err != nil {
		t.Fatalf("Failed to write .taskrc: %v", err)
	}

	// Initialize app
	app, err := NewApp(tmpDir)
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}

	// Verify configuration was loaded
	if app.MergedConfig == nil {
		t.Fatal("MergedConfig is nil")
	}

	// Check that repo config was merged
	if app.MergedConfig.Repo == nil {
		t.Fatal("MergedConfig.Repo is nil")
	}

	if app.MergedConfig.Repo.BaseBranch != "develop" {
		t.Errorf("MergedConfig.Repo.BaseBranch = %v, want %v", app.MergedConfig.Repo.BaseBranch, "develop")
	}

	if len(app.MergedConfig.Repo.Reviewers) != 2 {
		t.Errorf("MergedConfig.Repo.Reviewers length = %v, want %v", len(app.MergedConfig.Repo.Reviewers), 2)
	}
}

func TestApp_Integration_Adapters(t *testing.T) {
	tmpDir := t.TempDir()

	app, err := NewApp(tmpDir)
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}

	t.Run("BacklogStoreAdapter", func(t *testing.T) {
		// Test that we can add and retrieve tasks through the adapter
		task := models.NewTask("TEST-001", "Test Task", models.TaskTypeFeat)

		err := app.BacklogManager.AddTask(*task)
		if err != nil {
			t.Fatalf("BacklogManager.AddTask() error = %v", err)
		}

		retrieved, err := app.BacklogManager.GetTask("TEST-001")
		if err != nil {
			t.Fatalf("BacklogManager.GetTask() error = %v", err)
		}

		if retrieved.ID != "TEST-001" {
			t.Errorf("Retrieved task ID = %v, want %v", retrieved.ID, "TEST-001")
		}
	})

	t.Run("ContextStoreAdapter", func(t *testing.T) {
		// Create task directory first
		taskDir := filepath.Join(tmpDir, "tickets", "TEST-002")
		if err := os.MkdirAll(taskDir, 0o755); err != nil {
			t.Fatalf("Failed to create task directory: %v", err)
		}

		// Test writing and reading context
		content := "Test context content"
		err := app.ContextManager.WriteContext("TEST-002", content)
		if err != nil {
			t.Fatalf("ContextManager.WriteContext() error = %v", err)
		}

		retrieved, err := app.ContextManager.ReadContext("TEST-002")
		if err != nil {
			t.Fatalf("ContextManager.ReadContext() error = %v", err)
		}

		if retrieved != content {
			t.Errorf("Retrieved context = %v, want %v", retrieved, content)
		}
	})

	t.Run("EventLoggerAdapter", func(t *testing.T) {
		// Test that event logging works
		app.EventLog.Log("test.event", map[string]interface{}{
			"test_key": "test_value",
		})

		// Read events back
		events, err := app.EventLog.ReadAll()
		if err != nil {
			t.Fatalf("EventLog.ReadAll() error = %v", err)
		}

		// Should have at least one event
		if len(events) == 0 {
			t.Error("No events logged")
		}
	})
}

func TestApp_Cleanup(t *testing.T) {
	tmpDir := t.TempDir()

	app, err := NewApp(tmpDir)
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}

	// Test cleanup
	err = app.Cleanup()
	if err != nil {
		t.Errorf("Cleanup() error = %v", err)
	}
}

func TestAdapters_BacklogStore(t *testing.T) {
	tmpDir := t.TempDir()
	app, err := NewApp(tmpDir)
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}

	// Create adapter directly to test all methods
	adapter := &backlogStoreAdapter{manager: app.BacklogManager}

	task := models.NewTask("ADAPT-001", "Adapter Test", models.TaskTypeFeat)

	// Test AddTask
	if err := adapter.AddTask(*task); err != nil {
		t.Fatalf("AddTask() error = %v", err)
	}

	// Test GetTask
	got, err := adapter.GetTask("ADAPT-001")
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}
	if got.ID != "ADAPT-001" {
		t.Errorf("GetTask().ID = %v, want ADAPT-001", got.ID)
	}

	// Test UpdateTask
	task.Title = "Updated Title"
	if err := adapter.UpdateTask(*task); err != nil {
		t.Fatalf("UpdateTask() error = %v", err)
	}

	// Test Load
	backlog, err := adapter.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(backlog.Tasks) != 1 {
		t.Errorf("Load() tasks = %d, want 1", len(backlog.Tasks))
	}

	// Test Save
	if err := adapter.Save(backlog); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Test RemoveTask
	if err := adapter.RemoveTask("ADAPT-001"); err != nil {
		t.Fatalf("RemoveTask() error = %v", err)
	}
}

func TestAdapters_ContextStore(t *testing.T) {
	tmpDir := t.TempDir()
	app, err := NewApp(tmpDir)
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}

	adapter := &contextStoreAdapter{manager: app.ContextManager}

	// Create task directory
	taskDir := filepath.Join(tmpDir, "tickets", "CTX-001")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("Failed to create task dir: %v", err)
	}

	// Test WriteContext / ReadContext
	if err := adapter.WriteContext("CTX-001", "test context"); err != nil {
		t.Fatalf("WriteContext() error = %v", err)
	}
	ctx, err := adapter.ReadContext("CTX-001")
	if err != nil {
		t.Fatalf("ReadContext() error = %v", err)
	}
	if ctx != "test context" {
		t.Errorf("ReadContext() = %v, want 'test context'", ctx)
	}

	// Test AppendContext
	if err := adapter.AppendContext("CTX-001", "\nappended"); err != nil {
		t.Fatalf("AppendContext() error = %v", err)
	}

	// Test WriteNotes / ReadNotes
	if err := adapter.WriteNotes("CTX-001", "test notes"); err != nil {
		t.Fatalf("WriteNotes() error = %v", err)
	}
	notes, err := adapter.ReadNotes("CTX-001")
	if err != nil {
		t.Fatalf("ReadNotes() error = %v", err)
	}
	if notes != "test notes" {
		t.Errorf("ReadNotes() = %v, want 'test notes'", notes)
	}
}

func TestAdapters_WorktreeCreator(t *testing.T) {
	tmpDir := t.TempDir()
	app, err := NewApp(tmpDir)
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}

	adapter := &worktreeCreatorAdapter{manager: app.GitWorktreeManager, basePath: tmpDir}

	// Test with empty repoPath defaults to basePath
	// This will fail (no git repo at tmpDir) but exercises the code path
	err = adapter.CreateWorktree("TEST-001", "task/TEST-001", filepath.Join(tmpDir, "work", "TEST-001"), "")
	if err == nil {
		t.Error("CreateWorktree() with non-git basePath should fail")
	}

	// Test with explicit repoPath
	err = adapter.CreateWorktree("TEST-002", "task/TEST-002", filepath.Join(tmpDir, "work", "TEST-002"), "/nonexistent/repo")
	if err == nil {
		t.Error("CreateWorktree() with nonexistent repo should fail")
	}
}

func TestAdapters_WorktreeRemover(t *testing.T) {
	tmpDir := t.TempDir()
	app, err := NewApp(tmpDir)
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}

	adapter := &worktreeRemoverAdapter{manager: app.GitWorktreeManager}

	// Test with nonexistent path
	err = adapter.RemoveWorktree("/nonexistent/worktree")
	if err == nil {
		t.Error("RemoveWorktree() with nonexistent path should fail")
	}
}

func TestAdapters_EventLogger(t *testing.T) {
	tmpDir := t.TempDir()
	app, err := NewApp(tmpDir)
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}

	adapter := &eventLoggerAdapter{log: app.EventLog}

	// Test logging (should not panic)
	adapter.Log("test.event", map[string]interface{}{
		"key": "value",
	})

	// Verify event was written
	events, err := app.EventLog.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if len(events) == 0 {
		t.Error("No events after Log()")
	}
}

func TestAdapters_SessionCapturer(t *testing.T) {
	tmpDir := t.TempDir()
	app, err := NewApp(tmpDir)
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}

	adapter := &sessionCapturerAdapter{manager: app.SessionStoreManager}

	// Test CaptureSession (currently returns error in adapter)
	err = adapter.CaptureSession("TASK-001", "S-001", map[string]interface{}{})
	if err == nil {
		t.Error("CaptureSession() should return error (not fully implemented)")
	}
}

func TestAdapters_TerminalStateUpdater(t *testing.T) {
	tmpDir := t.TempDir()
	app, err := NewApp(tmpDir)
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}

	adapter := &terminalStateUpdaterAdapter{writer: app.TerminalStateWriter}

	// Test with status in state map
	err = adapter.WriteTerminalState(tmpDir, "TASK-001", map[string]interface{}{
		"status": "in_progress",
	})
	if err != nil {
		t.Fatalf("WriteTerminalState() error = %v", err)
	}

	// Test without status key (uses default)
	err = adapter.WriteTerminalState(tmpDir, "TASK-002", map[string]interface{}{
		"other_key": "value",
	})
	if err != nil {
		t.Fatalf("WriteTerminalState() with default status error = %v", err)
	}
}

func TestApp_GetSessionStore(t *testing.T) {
	tmpDir := t.TempDir()
	app, err := NewApp(tmpDir)
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}

	store := app.GetSessionStore()
	if store == nil {
		t.Error("GetSessionStore() returned nil")
	}
}

func TestApp_Integration_NoCircularImports(t *testing.T) {
	// This test verifies that the app can be constructed without circular import issues
	// If the test compiles and runs, it means there are no circular imports
	tmpDir := t.TempDir()

	app, err := NewApp(tmpDir)
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}

	if app == nil {
		t.Fatal("NewApp() returned nil")
	}

	// Verify all major components are initialized
	components := map[string]interface{}{
		"ConfigManager":      app.ConfigManager,
		"BacklogManager":     app.BacklogManager,
		"ContextManager":     app.ContextManager,
		"TaskIDGenerator":    app.TaskIDGenerator,
		"TemplateManager":    app.TemplateManager,
		"TaskManager":        app.TaskManager,
		"GitWorktreeManager": app.GitWorktreeManager,
		"EventLog":           app.EventLog,
	}

	for name, component := range components {
		if component == nil {
			t.Errorf("Component %s is nil", name)
		}
	}
}
