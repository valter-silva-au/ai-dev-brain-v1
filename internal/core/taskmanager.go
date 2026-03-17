package core

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/valter-silva-au/ai-dev-brain/pkg/models"
)

// BacklogStore defines the interface for managing task backlogs
type BacklogStore interface {
	Load() (*models.Backlog, error)
	Save(backlog *models.Backlog) error
	AddTask(task models.Task) error
	UpdateTask(task models.Task) error
	GetTask(id string) (*models.Task, error)
	RemoveTask(id string) error
}

// ContextStore defines the interface for managing task-specific context
type ContextStore interface {
	ReadContext(taskID string) (string, error)
	WriteContext(taskID string, content string) error
	AppendContext(taskID string, section string) error
	ReadNotes(taskID string) (string, error)
	WriteNotes(taskID string, content string) error
}

// WorktreeCreator defines the interface for creating git worktrees
type WorktreeCreator interface {
	CreateWorktree(taskID, branchName, worktreePath, repoPath string) error
}

// WorktreeRemover defines the interface for removing git worktrees
type WorktreeRemover interface {
	RemoveWorktree(worktreePath string) error
}

// EventLogger defines the interface for logging events
type EventLogger interface {
	Log(eventType string, data map[string]interface{})
}

// SessionCapturer defines the interface for capturing session data
type SessionCapturer interface {
	CaptureSession(taskID, sessionID string, data map[string]interface{}) error
}

// TerminalStateUpdater defines the interface for updating terminal state
type TerminalStateUpdater interface {
	WriteTerminalState(worktreePath string, taskID string, state map[string]interface{}) error
}

// CreateTaskOpts contains options for creating a new task
type CreateTaskOpts struct {
	Title              string
	Description        string
	AcceptanceCriteria []string
	TaskType           models.TaskType
	Priority           models.Priority
	Owner              string
	Tags               []string
	Prefix             string
	Repo               string
}

// TaskManager orchestrates the task lifecycle
type TaskManager struct {
	backlogStore         BacklogStore
	contextStore         ContextStore
	worktreeCreator      WorktreeCreator
	worktreeRemover      WorktreeRemover
	eventLogger          EventLogger
	sessionCapturer      SessionCapturer
	terminalStateUpdater TerminalStateUpdater
	taskIDGenerator      TaskIDGenerator
	templateManager      TemplateManager
	ticketsDir           string
	archivedDir          string
	worktreesDir         string
}

// NewTaskManager creates a new task manager
func NewTaskManager(
	backlogStore BacklogStore,
	contextStore ContextStore,
	worktreeCreator WorktreeCreator,
	worktreeRemover WorktreeRemover,
	eventLogger EventLogger,
	sessionCapturer SessionCapturer,
	terminalStateUpdater TerminalStateUpdater,
	taskIDGenerator TaskIDGenerator,
	templateManager TemplateManager,
	ticketsDir string,
	archivedDir string,
	worktreesDir string,
) *TaskManager {
	return &TaskManager{
		backlogStore:         backlogStore,
		contextStore:         contextStore,
		worktreeCreator:      worktreeCreator,
		worktreeRemover:      worktreeRemover,
		eventLogger:          eventLogger,
		sessionCapturer:      sessionCapturer,
		terminalStateUpdater: terminalStateUpdater,
		taskIDGenerator:      taskIDGenerator,
		templateManager:      templateManager,
		ticketsDir:           ticketsDir,
		archivedDir:          archivedDir,
		worktreesDir:         worktreesDir,
	}
}

// Create creates a new task with full lifecycle initialization
func (tm *TaskManager) Create(opts CreateTaskOpts) (*models.Task, error) {
	// Set defaults
	if opts.Prefix == "" {
		opts.Prefix = "TASK"
	}
	if opts.Priority == "" {
		opts.Priority = models.PriorityP2
	}
	if opts.TaskType == "" {
		opts.TaskType = models.TaskTypeFeat
	}

	// Generate task ID
	taskID, err := tm.taskIDGenerator.GenerateTaskID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate task ID: %w", err)
	}

	// Create task model
	task := models.NewTask(taskID, opts.Title, opts.TaskType)
	task.Priority = opts.Priority
	task.Owner = opts.Owner
	task.Tags = opts.Tags
	task.Repo = opts.Repo
	task.Status = models.TaskStatusBacklog

	// Add to backlog
	if err := tm.backlogStore.AddTask(*task); err != nil {
		return nil, fmt.Errorf("failed to add task to backlog: %w", err)
	}

	// Bootstrap directories and files (without task-context.md — worktree doesn't exist yet)
	bootstrapConfig := BootstrapConfig{
		TaskID:             taskID,
		Title:              opts.Title,
		Description:        opts.Description,
		AcceptanceCriteria: opts.AcceptanceCriteria,
		Status:             string(task.Status),
		TicketsDir:         tm.ticketsDir,
		WorktreeDir:        "", // Empty — task-context.md generated after worktree creation
	}

	result, err := BootstrapSystem(bootstrapConfig, tm.templateManager)
	if err != nil {
		// Rollback: remove from backlog
		_ = tm.backlogStore.RemoveTask(taskID)
		return nil, fmt.Errorf("failed to bootstrap task: %w", err)
	}

	// Update task with paths
	task.TicketPath = result.TaskDir

	// Create worktree only when a repository is specified
	if opts.Repo != "" && tm.worktreeCreator != nil {
		branchName := fmt.Sprintf("task/%s", taskID)
		worktreePath := filepath.Join(tm.worktreesDir, taskID)
		if err := tm.worktreeCreator.CreateWorktree(taskID, branchName, worktreePath, opts.Repo); err != nil {
			// Rollback: remove task dir and backlog entry
			_ = os.RemoveAll(result.TaskDir)
			_ = tm.backlogStore.RemoveTask(taskID)
			return nil, fmt.Errorf("failed to create worktree: %w", err)
		}
		task.WorktreePath = worktreePath
		task.Branch = branchName

		// Generate task-context.md inside the worktree now that it exists
		if err := generateTaskContext(worktreePath, tm.templateManager, bootstrapConfig); err != nil {
			// Non-fatal: log but continue — worktree is usable without task-context
			fmt.Fprintf(os.Stderr, "Warning: failed to generate task-context.md in worktree: %v\n", err)
		}
	}

	// Update backlog with worktree info
	if err := tm.backlogStore.UpdateTask(*task); err != nil {
		// Rollback: remove worktree and task dir
		if tm.worktreeRemover != nil && task.WorktreePath != "" {
			_ = tm.worktreeRemover.RemoveWorktree(task.WorktreePath)
		}
		_ = os.RemoveAll(result.TaskDir)
		_ = tm.backlogStore.RemoveTask(taskID)
		return nil, fmt.Errorf("failed to update task in backlog: %w", err)
	}

	// Write terminal state (only if worktree was created)
	if tm.terminalStateUpdater != nil && task.WorktreePath != "" {
		terminalState := map[string]interface{}{
			"task_id":    taskID,
			"status":     task.Status,
			"created_at": task.Created,
		}
		if err := tm.terminalStateUpdater.WriteTerminalState(task.WorktreePath, taskID, terminalState); err != nil {
			// Non-fatal: log but continue
			fmt.Fprintf(os.Stderr, "Warning: failed to write terminal state: %v\n", err)
		}
	}

	// Log event
	if tm.eventLogger != nil {
		tm.eventLogger.Log("task.created", map[string]interface{}{
			"task_id":  taskID,
			"title":    opts.Title,
			"priority": opts.Priority,
			"owner":    opts.Owner,
		})
	}

	return task, nil
}

// Resume loads a task and promotes it from backlog to in_progress
func (tm *TaskManager) Resume(taskID string) (*models.Task, error) {
	// Load task from backlog
	task, err := tm.backlogStore.GetTask(taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to load task: %w", err)
	}

	// Check not archived
	if task.Status == models.TaskStatusArchived {
		return nil, fmt.Errorf("cannot resume archived task %s", taskID)
	}

	// Promote to in_progress if currently in backlog
	if task.Status == models.TaskStatusBacklog {
		task.Status = models.TaskStatusInProgress
		task.UpdateTimestamp()
		if err := tm.backlogStore.UpdateTask(*task); err != nil {
			return nil, fmt.Errorf("failed to update task status: %w", err)
		}

		// Log event
		if tm.eventLogger != nil {
			tm.eventLogger.Log("task.status_changed", map[string]interface{}{
				"task_id":    taskID,
				"old_status": models.TaskStatusBacklog,
				"new_status": models.TaskStatusInProgress,
			})
		}
	}

	return task, nil
}

// Archive generates handoff.md, moves ticket to _archived/, removes worktree, and updates status
func (tm *TaskManager) Archive(taskID string) error {
	// Load task
	task, err := tm.backlogStore.GetTask(taskID)
	if err != nil {
		return fmt.Errorf("failed to load task: %w", err)
	}

	// Generate handoff.md from template
	handoffData := map[string]interface{}{
		"TaskID":     taskID,
		"Title":      task.Title,
		"Status":     task.Status,
		"Priority":   task.Priority,
		"Owner":      task.Owner,
		"Created":    task.Created.Format(time.RFC3339),
		"Updated":    task.Updated.Format(time.RFC3339),
		"ArchivedAt": time.Now().UTC().Format(time.RFC3339),
	}

	handoffContent, err := tm.templateManager.RenderBytes(TemplateTypeHandoff, handoffData)
	if err != nil {
		return fmt.Errorf("failed to render handoff template: %w", err)
	}

	// Write handoff.md to task directory
	taskDir := filepath.Join(tm.ticketsDir, taskID)
	handoffPath := filepath.Join(taskDir, "handoff.md")
	if err := os.WriteFile(handoffPath, handoffContent, 0o644); err != nil {
		return fmt.Errorf("failed to write handoff.md: %w", err)
	}

	// Move ticket to _archived/
	archivedTaskDir := filepath.Join(tm.archivedDir, taskID)
	if err := os.MkdirAll(tm.archivedDir, 0o755); err != nil {
		return fmt.Errorf("failed to create archived directory: %w", err)
	}

	if err := os.Rename(taskDir, archivedTaskDir); err != nil {
		return fmt.Errorf("failed to move task to archived: %w", err)
	}

	// Remove worktree
	if tm.worktreeRemover != nil && task.WorktreePath != "" {
		if err := tm.worktreeRemover.RemoveWorktree(task.WorktreePath); err != nil {
			// Non-fatal: log but continue
			fmt.Fprintf(os.Stderr, "Warning: failed to remove worktree: %v\n", err)
		}
	}

	// Update task status to archived
	task.Status = models.TaskStatusArchived
	task.TicketPath = archivedTaskDir
	task.WorktreePath = ""
	task.UpdateTimestamp()
	if err := tm.backlogStore.UpdateTask(*task); err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	// Log event
	if tm.eventLogger != nil {
		tm.eventLogger.Log("task.archived", map[string]interface{}{
			"task_id":      taskID,
			"archived_at":  time.Now().UTC(),
			"archived_dir": archivedTaskDir,
		})
	}

	return nil
}

// Unarchive moves a task back from _archived/ to active tickets
func (tm *TaskManager) Unarchive(taskID string) error {
	// Load task
	task, err := tm.backlogStore.GetTask(taskID)
	if err != nil {
		return fmt.Errorf("failed to load task: %w", err)
	}

	// Check if archived
	if task.Status != models.TaskStatusArchived {
		return fmt.Errorf("task %s is not archived", taskID)
	}

	// Move ticket back from _archived/
	archivedTaskDir := filepath.Join(tm.archivedDir, taskID)
	activeTaskDir := filepath.Join(tm.ticketsDir, taskID)

	if err := os.Rename(archivedTaskDir, activeTaskDir); err != nil {
		return fmt.Errorf("failed to move task from archived: %w", err)
	}

	// Update task status to backlog
	task.Status = models.TaskStatusBacklog
	task.TicketPath = activeTaskDir
	task.UpdateTimestamp()
	if err := tm.backlogStore.UpdateTask(*task); err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	// Log event
	if tm.eventLogger != nil {
		tm.eventLogger.Log("task.unarchived", map[string]interface{}{
			"task_id":       taskID,
			"unarchived_at": time.Now().UTC(),
		})
	}

	return nil
}

// UpdateStatus updates the status of a task
func (tm *TaskManager) UpdateStatus(taskID string, newStatus models.TaskStatus) error {
	// Load task
	task, err := tm.backlogStore.GetTask(taskID)
	if err != nil {
		return fmt.Errorf("failed to load task: %w", err)
	}

	oldStatus := task.Status
	task.Status = newStatus
	task.UpdateTimestamp()

	if err := tm.backlogStore.UpdateTask(*task); err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	// Log event
	if tm.eventLogger != nil {
		tm.eventLogger.Log("task.status_changed", map[string]interface{}{
			"task_id":    taskID,
			"old_status": oldStatus,
			"new_status": newStatus,
		})
	}

	return nil
}

// UpdatePriority updates the priority of a task
func (tm *TaskManager) UpdatePriority(taskID string, newPriority models.Priority) error {
	// Load task
	task, err := tm.backlogStore.GetTask(taskID)
	if err != nil {
		return fmt.Errorf("failed to load task: %w", err)
	}

	oldPriority := task.Priority
	task.Priority = newPriority
	task.UpdateTimestamp()

	if err := tm.backlogStore.UpdateTask(*task); err != nil {
		return fmt.Errorf("failed to update task priority: %w", err)
	}

	// Log event
	if tm.eventLogger != nil {
		tm.eventLogger.Log("task.priority_changed", map[string]interface{}{
			"task_id":      taskID,
			"old_priority": oldPriority,
			"new_priority": newPriority,
		})
	}

	return nil
}

// Cleanup removes only the worktree for a task
func (tm *TaskManager) Cleanup(taskID string) error {
	// Load task
	task, err := tm.backlogStore.GetTask(taskID)
	if err != nil {
		return fmt.Errorf("failed to load task: %w", err)
	}

	// Remove worktree if it exists
	if tm.worktreeRemover != nil && task.WorktreePath != "" {
		if err := tm.worktreeRemover.RemoveWorktree(task.WorktreePath); err != nil {
			return fmt.Errorf("failed to remove worktree: %w", err)
		}

		// Update task to clear worktree path
		task.WorktreePath = ""
		task.UpdateTimestamp()
		if err := tm.backlogStore.UpdateTask(*task); err != nil {
			return fmt.Errorf("failed to update task: %w", err)
		}

		// Log event
		if tm.eventLogger != nil {
			tm.eventLogger.Log("worktree.removed", map[string]interface{}{
				"task_id": taskID,
			})
		}
	}

	return nil
}

// Delete performs full removal of a task (worktree, ticket directory, and backlog entry)
func (tm *TaskManager) Delete(taskID string) error {
	// Load task
	task, err := tm.backlogStore.GetTask(taskID)
	if err != nil {
		return fmt.Errorf("failed to load task: %w", err)
	}

	// Remove worktree
	if tm.worktreeRemover != nil && task.WorktreePath != "" {
		if err := tm.worktreeRemover.RemoveWorktree(task.WorktreePath); err != nil {
			// Non-fatal: log but continue
			fmt.Fprintf(os.Stderr, "Warning: failed to remove worktree: %v\n", err)
		}
	}

	// Remove ticket directory (check both active and archived)
	taskDir := filepath.Join(tm.ticketsDir, taskID)
	archivedTaskDir := filepath.Join(tm.archivedDir, taskID)

	if _, err := os.Stat(taskDir); err == nil {
		if err := os.RemoveAll(taskDir); err != nil {
			return fmt.Errorf("failed to remove task directory: %w", err)
		}
	}

	if _, err := os.Stat(archivedTaskDir); err == nil {
		if err := os.RemoveAll(archivedTaskDir); err != nil {
			return fmt.Errorf("failed to remove archived task directory: %w", err)
		}
	}

	// Remove from backlog
	if err := tm.backlogStore.RemoveTask(taskID); err != nil {
		return fmt.Errorf("failed to remove task from backlog: %w", err)
	}

	// Log event
	if tm.eventLogger != nil {
		tm.eventLogger.Log("task.deleted", map[string]interface{}{
			"task_id":    taskID,
			"deleted_at": time.Now().UTC(),
		})
	}

	return nil
}
