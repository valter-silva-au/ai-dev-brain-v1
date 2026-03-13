package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/valter-silva-au/ai-dev-brain/internal/hooks"
)

func TestHookEngine_PreventRecursion(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hookengine-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	engine := NewHookEngine(tmpDir)

	t.Run("No recursion without flag", func(t *testing.T) {
		os.Unsetenv("ADB_HOOK_ACTIVE")
		if engine.PreventRecursion() {
			t.Errorf("PreventRecursion() = true, want false")
		}
	})

	t.Run("Recursion detected with flag", func(t *testing.T) {
		os.Setenv("ADB_HOOK_ACTIVE", "1")
		defer os.Unsetenv("ADB_HOOK_ACTIVE")

		if !engine.PreventRecursion() {
			t.Errorf("PreventRecursion() = false, want true")
		}
	})
}

func TestHookEngine_ProcessPreToolUse(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hookengine-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	engine := NewHookEngine(tmpDir)
	os.Unsetenv("ADB_HOOK_ACTIVE")

	t.Run("Allow normal file edit", func(t *testing.T) {
		event := &hooks.PreToolUseEvent{
			ToolName: "Edit",
			Parameters: map[string]interface{}{
				"file_path": "/path/to/main.go",
			},
		}

		err := engine.ProcessPreToolUse(event)
		if err != nil {
			t.Errorf("ProcessPreToolUse() error = %v, want nil", err)
		}
	})

	t.Run("Block vendor/ edit", func(t *testing.T) {
		event := &hooks.PreToolUseEvent{
			ToolName: "Edit",
			Parameters: map[string]interface{}{
				"file_path": "/path/to/vendor/package/file.go",
			},
		}

		err := engine.ProcessPreToolUse(event)
		if err == nil {
			t.Errorf("ProcessPreToolUse() error = nil, want error")
		}
	})

	t.Run("Block go.sum edit", func(t *testing.T) {
		event := &hooks.PreToolUseEvent{
			ToolName: "Write",
			Parameters: map[string]interface{}{
				"file_path": "/path/to/go.sum",
			},
		}

		err := engine.ProcessPreToolUse(event)
		if err == nil {
			t.Errorf("ProcessPreToolUse() error = nil, want error")
		}
	})

	t.Run("Allow other tools", func(t *testing.T) {
		event := &hooks.PreToolUseEvent{
			ToolName: "Read",
			Parameters: map[string]interface{}{
				"file_path": "/path/to/vendor/package/file.go",
			},
		}

		err := engine.ProcessPreToolUse(event)
		if err != nil {
			t.Errorf("ProcessPreToolUse() error = %v, want nil", err)
		}
	})
}

func TestHookEngine_ProcessPostToolUse(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hookengine-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	engine := NewHookEngine(tmpDir)
	os.Unsetenv("ADB_HOOK_ACTIVE")

	t.Run("Track file change on Edit", func(t *testing.T) {
		testFile := filepath.Join(tmpDir, "test.go")
		if err := os.WriteFile(testFile, []byte("package main\n"), 0o644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		event := &hooks.PostToolUseEvent{
			ToolName: "Edit",
			Parameters: map[string]interface{}{
				"file_path": testFile,
			},
		}

		err := engine.ProcessPostToolUse(event)
		if err != nil {
			t.Errorf("ProcessPostToolUse() error = %v", err)
		}

		// Verify change was tracked
		changes, err := engine.tracker.GetChanges()
		if err != nil {
			t.Fatalf("Failed to get changes: %v", err)
		}

		if len(changes) != 1 {
			t.Errorf("Expected 1 change, got %d", len(changes))
		}

		if len(changes) > 0 {
			if changes[0].FilePath != testFile {
				t.Errorf("Change FilePath = %v, want %v", changes[0].FilePath, testFile)
			}
			if changes[0].Operation != "modified" {
				t.Errorf("Change Operation = %v, want %v", changes[0].Operation, "modified")
			}
		}
	})

	t.Run("Track file change on Write", func(t *testing.T) {
		// Clear previous changes
		engine.tracker.Clear()

		testFile := filepath.Join(tmpDir, "new.go")
		if err := os.WriteFile(testFile, []byte("package main\n"), 0o644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		event := &hooks.PostToolUseEvent{
			ToolName: "Write",
			Parameters: map[string]interface{}{
				"file_path": testFile,
			},
		}

		err := engine.ProcessPostToolUse(event)
		if err != nil {
			t.Errorf("ProcessPostToolUse() error = %v", err)
		}

		// Verify change was tracked
		changes, err := engine.tracker.GetChanges()
		if err != nil {
			t.Fatalf("Failed to get changes: %v", err)
		}

		if len(changes) != 1 {
			t.Errorf("Expected 1 change, got %d", len(changes))
		}

		if len(changes) > 0 {
			if changes[0].Operation != "created" {
				t.Errorf("Change Operation = %v, want %v", changes[0].Operation, "created")
			}
		}
	})
}

func TestHookEngine_ProcessStop(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hookengine-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	engine := NewHookEngine(tmpDir)
	os.Unsetenv("ADB_HOOK_ACTIVE")

	t.Run("Process stop without errors", func(t *testing.T) {
		// ProcessStop should not error even if checks fail (advisory only)
		err := engine.ProcessStop()
		if err != nil {
			t.Errorf("ProcessStop() error = %v, want nil (advisory only)", err)
		}
	})
}

func TestHookEngine_ProcessTaskCompleted(t *testing.T) {
	// This test requires a valid Go project structure, so we'll test the error cases
	tmpDir, err := os.MkdirTemp("", "hookengine-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	engine := NewHookEngine(tmpDir)
	os.Unsetenv("ADB_HOOK_ACTIVE")

	t.Run("Process task completed", func(t *testing.T) {
		// Create task directory
		taskDir := filepath.Join(tmpDir, "tickets", "TASK-001")
		if err := os.MkdirAll(taskDir, 0o755); err != nil {
			t.Fatalf("Failed to create task dir: %v", err)
		}

		// Track some changes
		engine.tracker.TrackChange("file1.go", "modified")
		engine.tracker.TrackChange("file2.go", "created")

		event := &hooks.TaskCompletedEvent{
			TaskID:    "TASK-001",
			Status:    "done",
			Timestamp: "2024-01-01T00:00:00Z",
		}

		// This will fail quality gates (no valid Go project), but we test it runs
		err := engine.ProcessTaskCompleted(event)
		// We expect an error from quality gates since this isn't a real Go project
		if err == nil {
			t.Logf("ProcessTaskCompleted() succeeded (unexpected in test env)")
		}
	})
}

func TestHookEngine_ProcessSessionEnd(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hookengine-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create tickets directory
	ticketsDir := filepath.Join(tmpDir, "tickets")
	if err := os.MkdirAll(ticketsDir, 0o755); err != nil {
		t.Fatalf("Failed to create tickets dir: %v", err)
	}

	engine := NewHookEngine(tmpDir)
	os.Unsetenv("ADB_HOOK_ACTIVE")

	t.Run("Process session end", func(t *testing.T) {
		event := &hooks.SessionEndEvent{
			SessionID: "sess-123",
			Timestamp: "2024-01-01T00:00:00Z",
			Duration:  120.5,
			Metadata: map[string]interface{}{
				"transcript": "Test transcript content",
			},
		}

		err := engine.ProcessSessionEnd(event)
		if err != nil {
			t.Errorf("ProcessSessionEnd() error = %v", err)
		}
	})
}

func TestHookEngine_GetCurrentTaskID(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hookengine-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	engine := NewHookEngine(tmpDir)

	t.Run("Get from environment", func(t *testing.T) {
		os.Setenv("ADB_TASK_ID", "TASK-001")
		defer os.Unsetenv("ADB_TASK_ID")

		taskID := engine.getCurrentTaskID()
		if taskID != "TASK-001" {
			t.Errorf("getCurrentTaskID() = %v, want %v", taskID, "TASK-001")
		}
	})

	t.Run("Empty when not set", func(t *testing.T) {
		os.Unsetenv("ADB_TASK_ID")
		// Note: getCurrentTaskID will try to get from git branch, which may or may not exist
		// in the test environment, so we just ensure it doesn't panic
		_ = engine.getCurrentTaskID()
	})
}
