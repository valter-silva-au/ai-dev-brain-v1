package internal

import (
	"fmt"
	"path/filepath"

	"github.com/valter-silva-au/ai-dev-brain/internal/core"
	"github.com/valter-silva-au/ai-dev-brain/internal/integration"
	"github.com/valter-silva-au/ai-dev-brain/internal/observability"
	"github.com/valter-silva-au/ai-dev-brain/internal/storage"
	"github.com/valter-silva-au/ai-dev-brain/pkg/models"
	"github.com/valter-silva-au/ai-dev-brain/templates/claude"
)

// App is the application's dependency injection container
// It wires all subsystems together using the adapter pattern to prevent circular imports
type App struct {
	// ===== Configuration =====
	BasePath           string
	ConfigManager      core.ConfigurationManager
	MergedConfig       *models.MergedConfig

	// ===== Storage =====
	BacklogManager     storage.BacklogManager
	ContextManager     storage.ContextManager
	SessionStoreManager storage.SessionStoreManager

	// ===== Core Services =====
	TaskIDGenerator    core.TaskIDGenerator
	TemplateManager    core.TemplateManager
	TaskManager        *core.TaskManager

	// ===== Integration =====
	GitWorktreeManager integration.GitWorktreeManager
	TerminalStateWriter integration.TerminalStateWriter

	// ===== Observability =====
	EventLog           *observability.EventLog
	MetricsCalculator  *observability.MetricsCalculator
	AlertEvaluator     *observability.AlertEvaluator
}

// Adapters bridge core interfaces to real implementations
// This prevents circular imports: core defines interfaces, implementations live elsewhere

// backlogStoreAdapter adapts storage.BacklogManager to core.BacklogStore
type backlogStoreAdapter struct {
	manager storage.BacklogManager
}

func (a *backlogStoreAdapter) Load() (*models.Backlog, error) {
	return a.manager.Load()
}

func (a *backlogStoreAdapter) Save(backlog *models.Backlog) error {
	return a.manager.Save(backlog)
}

func (a *backlogStoreAdapter) AddTask(task models.Task) error {
	return a.manager.AddTask(task)
}

func (a *backlogStoreAdapter) UpdateTask(task models.Task) error {
	return a.manager.UpdateTask(task)
}

func (a *backlogStoreAdapter) GetTask(id string) (*models.Task, error) {
	return a.manager.GetTask(id)
}

func (a *backlogStoreAdapter) RemoveTask(id string) error {
	return a.manager.RemoveTask(id)
}

// contextStoreAdapter adapts storage.ContextManager to core.ContextStore
type contextStoreAdapter struct {
	manager storage.ContextManager
}

func (a *contextStoreAdapter) ReadContext(taskID string) (string, error) {
	return a.manager.ReadContext(taskID)
}

func (a *contextStoreAdapter) WriteContext(taskID string, content string) error {
	return a.manager.WriteContext(taskID, content)
}

func (a *contextStoreAdapter) AppendContext(taskID string, section string) error {
	return a.manager.AppendContext(taskID, section)
}

func (a *contextStoreAdapter) ReadNotes(taskID string) (string, error) {
	return a.manager.ReadNotes(taskID)
}

func (a *contextStoreAdapter) WriteNotes(taskID string, content string) error {
	return a.manager.WriteNotes(taskID, content)
}

// worktreeCreatorAdapter adapts integration.GitWorktreeManager to core.WorktreeCreator
type worktreeCreatorAdapter struct {
	manager  integration.GitWorktreeManager
	basePath string
}

func (a *worktreeCreatorAdapter) CreateWorktree(taskID, branchName, worktreePath string) error {
	// For local repo, use current directory
	// This assumes we're working with the current git repository
	repoPath := "."
	baseBranch := "main"

	// The GitWorktreeManager already handles creating the worktree at the right path
	// We just need to call it with the taskID
	_, err := a.manager.CreateWorktree(taskID, repoPath, baseBranch)
	return err
}

// worktreeRemoverAdapter adapts integration.GitWorktreeManager to core.WorktreeRemover
type worktreeRemoverAdapter struct {
	manager integration.GitWorktreeManager
}

func (a *worktreeRemoverAdapter) RemoveWorktree(worktreePath string) error {
	return a.manager.RemoveWorktree(worktreePath)
}

// eventLoggerAdapter adapts observability.EventLog to core.EventLogger
type eventLoggerAdapter struct {
	log *observability.EventLog
}

func (a *eventLoggerAdapter) Log(eventType string, data map[string]interface{}) {
	a.log.Log(observability.EventType(eventType), data)
}

// sessionCapturerAdapter adapts storage.SessionStoreManager to core.SessionCapturer
type sessionCapturerAdapter struct {
	manager storage.SessionStoreManager
}

func (a *sessionCapturerAdapter) CaptureSession(taskID, sessionID string, data map[string]interface{}) error {
	// This is a simplified implementation - in a real system, we'd convert the data map
	// to a proper CapturedSession struct. For now, we'll return an error if not implemented.
	return fmt.Errorf("session capture not fully implemented in adapter")
}

// terminalStateUpdaterAdapter adapts integration.TerminalStateWriter to core.TerminalStateUpdater
type terminalStateUpdaterAdapter struct {
	writer integration.TerminalStateWriter
}

func (a *terminalStateUpdaterAdapter) WriteTerminalState(worktreePath string, taskID string, state map[string]interface{}) error {
	// Convert map to TerminalState struct
	status := "active"
	if s, ok := state["status"].(string); ok {
		status = s
	}

	ts := integration.TerminalState{
		WorktreePath: worktreePath,
		TaskID:       taskID,
		Status:       status,
		LastUpdated:  "", // Will be set by the writer
	}

	return a.writer.WriteState(ts)
}

// NewApp creates and wires all application subsystems in dependency order
// basePath is the root directory for the workspace (e.g., "." or "/path/to/workspace")
func NewApp(basePath string) (*App, error) {
	if basePath == "" {
		basePath = "."
	}

	app := &App{
		BasePath: basePath,
	}

	// ===== Configuration =====
	// Load configuration from .taskconfig and .taskrc
	globalConfigPath := "" // Uses default ~/.taskconfig
	repoConfigPath := filepath.Join(basePath, ".taskrc")
	app.ConfigManager = core.NewViperConfigManager(globalConfigPath, repoConfigPath)

	config, err := app.ConfigManager.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}
	app.MergedConfig = config

	// ===== Storage =====
	// Backlog manager - stores tasks in backlog.yaml
	backlogPath := filepath.Join(basePath, "backlog.yaml")
	app.BacklogManager = storage.NewFileBacklogManager(backlogPath)

	// Context manager - manages task-specific context and notes
	ticketsDir := filepath.Join(basePath, "tickets")
	app.ContextManager = storage.NewFileContextManager(ticketsDir)

	// Session store manager - manages captured sessions
	sessionsDir := filepath.Join(basePath, "sessions")
	app.SessionStoreManager = storage.NewFileSessionStoreManager(sessionsDir)

	// ===== Core Services =====
	// Task ID generator - generates sequential task IDs
	counterFile := filepath.Join(basePath, ".task_counter")
	prefix := "TASK"
	if app.MergedConfig != nil && app.MergedConfig.Global != nil && app.MergedConfig.Global.TaskIDPrefix != "" {
		prefix = app.MergedConfig.Global.TaskIDPrefix
	}
	app.TaskIDGenerator = core.NewFileTaskIDGenerator(counterFile, prefix)

	// Template manager - renders templates from embedded filesystem
	templateManager, err := core.NewEmbedTemplateManager(claude.FS)
	if err != nil {
		return nil, fmt.Errorf("failed to create template manager: %w", err)
	}
	app.TemplateManager = templateManager

	// ===== Integration =====
	// Git worktree manager - manages git worktrees for task isolation
	app.GitWorktreeManager = integration.NewGitWorktreeManager(basePath)

	// Terminal state writer - manages terminal state for VS Code integration
	terminalStateFile := "" // Uses default ~/.adb_terminal_state.json
	app.TerminalStateWriter = integration.NewTerminalStateWriter(terminalStateFile)

	// ===== Observability =====
	// Event log - append-only JSONL event logging
	eventLogPath := filepath.Join(basePath, ".events.jsonl")
	app.EventLog = observability.NewEventLog(eventLogPath)

	// Metrics calculator - computes metrics on-demand from event log
	app.MetricsCalculator = observability.NewMetricsCalculator(app.EventLog)

	// Alert evaluator - evaluates alert conditions against thresholds
	app.AlertEvaluator = observability.NewAlertEvaluator(nil, app.MetricsCalculator)

	// ===== Task Manager (wires everything together) =====
	// Create adapters
	backlogStoreAdpt := &backlogStoreAdapter{manager: app.BacklogManager}
	contextStoreAdpt := &contextStoreAdapter{manager: app.ContextManager}
	worktreeCreatorAdpt := &worktreeCreatorAdapter{manager: app.GitWorktreeManager, basePath: basePath}
	worktreeRemoverAdpt := &worktreeRemoverAdapter{manager: app.GitWorktreeManager}
	eventLoggerAdpt := &eventLoggerAdapter{log: app.EventLog}
	sessionCapturerAdpt := &sessionCapturerAdapter{manager: app.SessionStoreManager}
	terminalStateUpdaterAdpt := &terminalStateUpdaterAdapter{writer: app.TerminalStateWriter}

	// Create task manager with all dependencies
	archivedDir := filepath.Join(basePath, "tickets", "_archived")
	worktreesDir := filepath.Join(basePath, "work")

	app.TaskManager = core.NewTaskManager(
		backlogStoreAdpt,
		contextStoreAdpt,
		worktreeCreatorAdpt,
		worktreeRemoverAdpt,
		eventLoggerAdpt,
		sessionCapturerAdpt,
		terminalStateUpdaterAdpt,
		app.TaskIDGenerator,
		app.TemplateManager,
		ticketsDir,
		archivedDir,
		worktreesDir,
	)

	return app, nil
}

// GetSessionStore returns the session store manager
func (app *App) GetSessionStore() storage.SessionStoreManager {
	return app.SessionStoreManager
}

// Cleanup performs cleanup operations (optional, for graceful shutdown)
func (app *App) Cleanup() error {
	// Future: close any open resources, flush buffers, etc.
	return nil
}
