package observability

import (
	"path/filepath"
	"testing"
	"time"
)

func TestMetricsCalculator_ComputeMetrics(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, ".adb_events.jsonl")

	el := NewEventLog(logPath)
	mc := NewMetricsCalculator(el)

	// Log various events
	el.Log(EventTaskCreated, map[string]interface{}{
		"task_id": "TASK-001",
		"type":    "feat",
		"status":  "backlog",
	})

	el.Log(EventTaskCreated, map[string]interface{}{
		"task_id": "TASK-002",
		"type":    "bug",
		"status":  "backlog",
	})

	el.Log(EventTaskStatusChanged, map[string]interface{}{
		"task_id":    "TASK-001",
		"old_status": "backlog",
		"new_status": "in_progress",
	})

	el.Log(EventTaskCompleted, map[string]interface{}{
		"task_id": "TASK-001",
	})

	el.Log(EventAgentSessionStarted, map[string]interface{}{
		"session_id": "session-1",
	})

	el.Log(EventWorktreeCreated, map[string]interface{}{
		"path": "/tmp/worktree1",
	})

	// Compute metrics
	metrics, err := mc.ComputeMetrics()
	if err != nil {
		t.Fatalf("Failed to compute metrics: %v", err)
	}

	// Verify metrics
	if metrics.TasksCreated != 2 {
		t.Errorf("Expected 2 tasks created, got %d", metrics.TasksCreated)
	}

	if metrics.TasksCompleted != 1 {
		t.Errorf("Expected 1 task completed, got %d", metrics.TasksCompleted)
	}

	if metrics.TasksByType["feat"] != 1 {
		t.Errorf("Expected 1 feat task, got %d", metrics.TasksByType["feat"])
	}

	if metrics.TasksByType["bug"] != 1 {
		t.Errorf("Expected 1 bug task, got %d", metrics.TasksByType["bug"])
	}

	if metrics.AgentSessions != 1 {
		t.Errorf("Expected 1 agent session, got %d", metrics.AgentSessions)
	}

	if metrics.WorktreesCreated != 1 {
		t.Errorf("Expected 1 worktree created, got %d", metrics.WorktreesCreated)
	}

	// Verify status counts (backlog should have 1, in_progress should be 0 after completion)
	if metrics.TasksByStatus["backlog"] != 1 {
		t.Errorf("Expected 1 task in backlog status, got %d", metrics.TasksByStatus["backlog"])
	}

	if metrics.TasksByStatus["in_progress"] != 1 {
		t.Errorf("Expected 1 task in in_progress status, got %d", metrics.TasksByStatus["in_progress"])
	}
}

func TestMetricsCalculator_StatusHistory(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, ".adb_events.jsonl")

	el := NewEventLog(logPath)
	mc := NewMetricsCalculator(el)

	// Create task and change status multiple times
	el.Log(EventTaskCreated, map[string]interface{}{
		"task_id": "TASK-001",
		"status":  "backlog",
	})

	el.Log(EventTaskStatusChanged, map[string]interface{}{
		"task_id":    "TASK-001",
		"old_status": "backlog",
		"new_status": "in_progress",
	})

	el.Log(EventTaskStatusChanged, map[string]interface{}{
		"task_id":    "TASK-001",
		"old_status": "in_progress",
		"new_status": "blocked",
	})

	el.Log(EventTaskStatusChanged, map[string]interface{}{
		"task_id":    "TASK-001",
		"old_status": "blocked",
		"new_status": "in_progress",
	})

	// Compute metrics
	metrics, err := mc.ComputeMetrics()
	if err != nil {
		t.Fatalf("Failed to compute metrics: %v", err)
	}

	// Verify status history
	history := metrics.TaskStatusHistory["TASK-001"]
	if len(history) != 3 {
		t.Fatalf("Expected 3 status changes, got %d", len(history))
	}

	expectedChanges := []struct {
		old string
		new string
	}{
		{"backlog", "in_progress"},
		{"in_progress", "blocked"},
		{"blocked", "in_progress"},
	}

	for i, expected := range expectedChanges {
		if history[i].OldStatus != expected.old {
			t.Errorf("Change %d: expected old status %s, got %s", i, expected.old, history[i].OldStatus)
		}
		if history[i].NewStatus != expected.new {
			t.Errorf("Change %d: expected new status %s, got %s", i, expected.new, history[i].NewStatus)
		}
	}
}

func TestMetricsCalculator_GetTaskDuration(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, ".adb_events.jsonl")

	el := NewEventLog(logPath)
	mc := NewMetricsCalculator(el)

	// Create task with initial status
	el.Log(EventTaskCreated, map[string]interface{}{
		"task_id": "TASK-001",
		"status":  "backlog",
	})

	// Wait a bit
	time.Sleep(10 * time.Millisecond)

	// Change status
	el.Log(EventTaskStatusChanged, map[string]interface{}{
		"task_id":    "TASK-001",
		"old_status": "backlog",
		"new_status": "in_progress",
	})

	// Wait a bit more
	time.Sleep(10 * time.Millisecond)

	// Get duration in current status
	duration, err := mc.GetTaskDuration("TASK-001", "in_progress")
	if err != nil {
		t.Fatalf("Failed to get task duration: %v", err)
	}

	if duration < 10*time.Millisecond {
		t.Errorf("Expected duration >= 10ms, got %v", duration)
	}

	// Duration for old status should be 0
	duration, err = mc.GetTaskDuration("TASK-001", "backlog")
	if err != nil {
		t.Fatalf("Failed to get task duration: %v", err)
	}

	if duration != 0 {
		t.Errorf("Expected 0 duration for old status, got %v", duration)
	}
}

func TestMetricsCalculator_GetTasksInStatus(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, ".adb_events.jsonl")

	el := NewEventLog(logPath)
	mc := NewMetricsCalculator(el)

	// Create multiple tasks with different statuses
	el.Log(EventTaskCreated, map[string]interface{}{
		"task_id": "TASK-001",
		"status":  "backlog",
	})

	el.Log(EventTaskCreated, map[string]interface{}{
		"task_id": "TASK-002",
		"status":  "backlog",
	})

	el.Log(EventTaskCreated, map[string]interface{}{
		"task_id": "TASK-003",
		"status":  "backlog",
	})

	// Move one task to in_progress
	el.Log(EventTaskStatusChanged, map[string]interface{}{
		"task_id":    "TASK-001",
		"old_status": "backlog",
		"new_status": "in_progress",
	})

	// Move another to blocked
	el.Log(EventTaskStatusChanged, map[string]interface{}{
		"task_id":    "TASK-002",
		"old_status": "backlog",
		"new_status": "blocked",
	})

	// Get tasks in backlog
	backlogTasks, err := mc.GetTasksInStatus("backlog")
	if err != nil {
		t.Fatalf("Failed to get tasks in status: %v", err)
	}

	if len(backlogTasks) != 1 {
		t.Errorf("Expected 1 task in backlog, got %d", len(backlogTasks))
	}

	// Get tasks in progress
	inProgressTasks, err := mc.GetTasksInStatus("in_progress")
	if err != nil {
		t.Fatalf("Failed to get tasks in status: %v", err)
	}

	if len(inProgressTasks) != 1 {
		t.Errorf("Expected 1 task in progress, got %d", len(inProgressTasks))
	}

	// Get blocked tasks
	blockedTasks, err := mc.GetTasksInStatus("blocked")
	if err != nil {
		t.Fatalf("Failed to get tasks in status: %v", err)
	}

	if len(blockedTasks) != 1 {
		t.Errorf("Expected 1 blocked task, got %d", len(blockedTasks))
	}
}

func TestMetricsCalculator_EmptyLog(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, ".adb_events.jsonl")

	el := NewEventLog(logPath)
	mc := NewMetricsCalculator(el)

	// Compute metrics from empty log
	metrics, err := mc.ComputeMetrics()
	if err != nil {
		t.Fatalf("Failed to compute metrics: %v", err)
	}

	if metrics.TasksCreated != 0 {
		t.Errorf("Expected 0 tasks created, got %d", metrics.TasksCreated)
	}

	if metrics.TasksCompleted != 0 {
		t.Errorf("Expected 0 tasks completed, got %d", metrics.TasksCompleted)
	}

	if metrics.AgentSessions != 0 {
		t.Errorf("Expected 0 agent sessions, got %d", metrics.AgentSessions)
	}
}

func TestMetricsCalculator_AllEventTypes(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, ".adb_events.jsonl")

	el := NewEventLog(logPath)
	mc := NewMetricsCalculator(el)

	// Log all event types
	el.Log(EventTaskCreated, map[string]interface{}{"task_id": "TASK-001"})
	el.Log(EventTaskCompleted, map[string]interface{}{"task_id": "TASK-001"})
	el.Log(EventTaskStatusChanged, map[string]interface{}{
		"task_id":    "TASK-001",
		"old_status": "backlog",
		"new_status": "done",
	})
	el.Log(EventAgentSessionStarted, map[string]interface{}{"session": "1"})
	el.Log(EventKnowledgeExtracted, map[string]interface{}{"item": "1"})
	el.Log(EventWorktreeCreated, map[string]interface{}{"path": "/tmp/wt1"})
	el.Log(EventWorktreeRemoved, map[string]interface{}{"path": "/tmp/wt1"})

	// Compute metrics
	metrics, err := mc.ComputeMetrics()
	if err != nil {
		t.Fatalf("Failed to compute metrics: %v", err)
	}

	// Verify all counters
	if metrics.TasksCreated != 1 {
		t.Errorf("Expected 1 task created, got %d", metrics.TasksCreated)
	}

	if metrics.TasksCompleted != 1 {
		t.Errorf("Expected 1 task completed, got %d", metrics.TasksCompleted)
	}

	if metrics.AgentSessions != 1 {
		t.Errorf("Expected 1 agent session, got %d", metrics.AgentSessions)
	}

	if metrics.KnowledgeExtracts != 1 {
		t.Errorf("Expected 1 knowledge extract, got %d", metrics.KnowledgeExtracts)
	}

	if metrics.WorktreesCreated != 1 {
		t.Errorf("Expected 1 worktree created, got %d", metrics.WorktreesCreated)
	}

	if metrics.WorktreesRemoved != 1 {
		t.Errorf("Expected 1 worktree removed, got %d", metrics.WorktreesRemoved)
	}
}
