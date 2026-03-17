package core

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/valter-silva-au/ai-dev-brain/pkg/models"
)

// Mock implementations

type MockBacklogStore struct {
	tasks   map[string]*models.Task
	loadErr error
	saveErr error
}

func NewMockBacklogStore() *MockBacklogStore {
	return &MockBacklogStore{
		tasks: make(map[string]*models.Task),
	}
}

func (m *MockBacklogStore) Load() (*models.Backlog, error) {
	if m.loadErr != nil {
		return nil, m.loadErr
	}
	backlog := models.NewBacklog()
	for _, task := range m.tasks {
		backlog.AddTask(*task)
	}
	return backlog, nil
}

func (m *MockBacklogStore) Save(backlog *models.Backlog) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.tasks = make(map[string]*models.Task)
	for i := range backlog.Tasks {
		taskCopy := backlog.Tasks[i]
		m.tasks[taskCopy.ID] = &taskCopy
	}
	return nil
}

func (m *MockBacklogStore) AddTask(task models.Task) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	if _, exists := m.tasks[task.ID]; exists {
		return fmt.Errorf("task with ID %s already exists", task.ID)
	}
	taskCopy := task
	m.tasks[task.ID] = &taskCopy
	return nil
}

func (m *MockBacklogStore) UpdateTask(task models.Task) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	if _, exists := m.tasks[task.ID]; !exists {
		return fmt.Errorf("task with ID %s not found", task.ID)
	}
	taskCopy := task
	m.tasks[task.ID] = &taskCopy
	return nil
}

func (m *MockBacklogStore) GetTask(id string) (*models.Task, error) {
	if m.loadErr != nil {
		return nil, m.loadErr
	}
	task, exists := m.tasks[id]
	if !exists {
		return nil, fmt.Errorf("task with ID %s not found", id)
	}
	taskCopy := *task
	return &taskCopy, nil
}

func (m *MockBacklogStore) RemoveTask(id string) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	if _, exists := m.tasks[id]; !exists {
		return fmt.Errorf("task with ID %s not found", id)
	}
	delete(m.tasks, id)
	return nil
}

type MockContextStore struct {
	contexts map[string]string
	notes    map[string]string
}

func NewMockContextStore() *MockContextStore {
	return &MockContextStore{
		contexts: make(map[string]string),
		notes:    make(map[string]string),
	}
}

func (m *MockContextStore) ReadContext(taskID string) (string, error) {
	return m.contexts[taskID], nil
}

func (m *MockContextStore) WriteContext(taskID string, content string) error {
	m.contexts[taskID] = content
	return nil
}

func (m *MockContextStore) AppendContext(taskID string, section string) error {
	m.contexts[taskID] += section
	return nil
}

func (m *MockContextStore) ReadNotes(taskID string) (string, error) {
	return m.notes[taskID], nil
}

func (m *MockContextStore) WriteNotes(taskID string, content string) error {
	m.notes[taskID] = content
	return nil
}

type MockWorktreeCreator struct {
	worktrees  map[string]string // taskID -> worktreePath
	createErr  error
	shouldFail bool
}

func NewMockWorktreeCreator() *MockWorktreeCreator {
	return &MockWorktreeCreator{
		worktrees: make(map[string]string),
	}
}

func (m *MockWorktreeCreator) CreateWorktree(taskID, branchName, worktreePath, repoPath string) error {
	if m.shouldFail || m.createErr != nil {
		if m.createErr != nil {
			return m.createErr
		}
		return fmt.Errorf("worktree creation failed")
	}
	m.worktrees[taskID] = worktreePath
	return nil
}

type MockWorktreeRemover struct {
	removed   []string
	removeErr error
}

func NewMockWorktreeRemover() *MockWorktreeRemover {
	return &MockWorktreeRemover{
		removed: []string{},
	}
}

func (m *MockWorktreeRemover) RemoveWorktree(worktreePath string) error {
	if m.removeErr != nil {
		return m.removeErr
	}
	m.removed = append(m.removed, worktreePath)
	return nil
}

type MockEventLogger struct {
	events []map[string]interface{}
}

func NewMockEventLogger() *MockEventLogger {
	return &MockEventLogger{
		events: []map[string]interface{}{},
	}
}

func (m *MockEventLogger) Log(eventType string, data map[string]interface{}) {
	event := make(map[string]interface{})
	event["type"] = eventType
	for k, v := range data {
		event[k] = v
	}
	m.events = append(m.events, event)
}

type MockSessionCapturer struct {
	sessions map[string]map[string]interface{}
}

func NewMockSessionCapturer() *MockSessionCapturer {
	return &MockSessionCapturer{
		sessions: make(map[string]map[string]interface{}),
	}
}

func (m *MockSessionCapturer) CaptureSession(taskID, sessionID string, data map[string]interface{}) error {
	key := fmt.Sprintf("%s:%s", taskID, sessionID)
	m.sessions[key] = data
	return nil
}

type MockTerminalStateUpdater struct {
	states map[string]map[string]interface{}
}

func NewMockTerminalStateUpdater() *MockTerminalStateUpdater {
	return &MockTerminalStateUpdater{
		states: make(map[string]map[string]interface{}),
	}
}

func (m *MockTerminalStateUpdater) WriteTerminalState(worktreePath string, taskID string, state map[string]interface{}) error {
	key := fmt.Sprintf("%s:%s", worktreePath, taskID)
	m.states[key] = state
	return nil
}

type MockTaskIDGenerator struct {
	counter int
	prefix  string
}

func NewMockTaskIDGenerator(prefix string) *MockTaskIDGenerator {
	return &MockTaskIDGenerator{
		counter: 0,
		prefix:  prefix,
	}
}

func (m *MockTaskIDGenerator) GenerateTaskID() (string, error) {
	m.counter++
	return fmt.Sprintf("%s-%05d", m.prefix, m.counter), nil
}

type MockTemplateManager struct{}

func NewMockTemplateManager() *MockTemplateManager {
	return &MockTemplateManager{}
}

func (m *MockTemplateManager) Render(templateType TemplateType, data interface{}) (string, error) {
	bytes, err := m.RenderBytes(templateType, data)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (m *MockTemplateManager) RenderBytes(templateType TemplateType, data interface{}) ([]byte, error) {
	// Simple mock implementation
	switch templateType {
	case TemplateTypeHandoff:
		return []byte("# Handoff Document\n\nTask archived."), nil
	case TemplateTypeStatus:
		return []byte("status: pending"), nil
	case TemplateTypeContext:
		return []byte("# Context\n\nTask context."), nil
	case TemplateTypeNotes:
		return []byte("# Notes\n\nTask notes."), nil
	case TemplateTypeDesign:
		return []byte("# Design\n\nTask design."), nil
	case TemplateTypeTaskContext:
		return []byte("# Task Context\n\nFor Claude."), nil
	default:
		return nil, fmt.Errorf("unknown template type: %s", templateType)
	}
}

// Helper function to create a test TaskManager
func createTestTaskManager(t *testing.T) (*TaskManager, *MockBacklogStore, *MockEventLogger, *MockWorktreeCreator, *MockWorktreeRemover, string) {
	t.Helper()

	// Create temp directories
	tempDir := t.TempDir()
	ticketsDir := filepath.Join(tempDir, "tickets")
	archivedDir := filepath.Join(tempDir, "_archived")
	worktreesDir := filepath.Join(tempDir, "worktrees")

	// Create mocks
	backlogStore := NewMockBacklogStore()
	contextStore := NewMockContextStore()
	worktreeCreator := NewMockWorktreeCreator()
	worktreeRemover := NewMockWorktreeRemover()
	eventLogger := NewMockEventLogger()
	sessionCapturer := NewMockSessionCapturer()
	terminalStateUpdater := NewMockTerminalStateUpdater()
	taskIDGenerator := NewMockTaskIDGenerator("TASK")
	templateManager := NewMockTemplateManager()

	tm := NewTaskManager(
		backlogStore,
		contextStore,
		worktreeCreator,
		worktreeRemover,
		eventLogger,
		sessionCapturer,
		terminalStateUpdater,
		taskIDGenerator,
		templateManager,
		ticketsDir,
		archivedDir,
		worktreesDir,
	)

	return tm, backlogStore, eventLogger, worktreeCreator, worktreeRemover, tempDir
}

func TestTaskManager_Create(t *testing.T) {
	tm, backlogStore, eventLogger, worktreeCreator, _, tempDir := createTestTaskManager(t)

	opts := CreateTaskOpts{
		Title:              "Test Task",
		Description:        "Test description",
		AcceptanceCriteria: []string{"AC1", "AC2"},
		TaskType:           models.TaskTypeFeat,
		Priority:           models.PriorityP1,
		Owner:              "test-owner",
		Tags:               []string{"tag1", "tag2"},
		Repo:               "test-repo",
	}

	task, err := tm.Create(opts)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify task was created
	if task.ID != "TASK-00001" {
		t.Errorf("Expected task ID TASK-00001, got %s", task.ID)
	}
	if task.Title != opts.Title {
		t.Errorf("Expected title %s, got %s", opts.Title, task.Title)
	}
	if task.Priority != opts.Priority {
		t.Errorf("Expected priority %s, got %s", opts.Priority, task.Priority)
	}
	if task.Owner != opts.Owner {
		t.Errorf("Expected owner %s, got %s", opts.Owner, task.Owner)
	}
	if task.Status != models.TaskStatusBacklog {
		t.Errorf("Expected status backlog, got %s", task.Status)
	}

	// Verify task in backlog
	storedTask, err := backlogStore.GetTask(task.ID)
	if err != nil {
		t.Fatalf("Failed to get task from backlog: %v", err)
	}
	if storedTask.ID != task.ID {
		t.Errorf("Task ID mismatch in backlog")
	}

	// Verify worktree was created
	if _, exists := worktreeCreator.worktrees[task.ID]; !exists {
		t.Errorf("Worktree was not created for task %s", task.ID)
	}

	// Verify event was logged
	if len(eventLogger.events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(eventLogger.events))
	} else {
		event := eventLogger.events[0]
		if event["type"] != "task.created" {
			t.Errorf("Expected event type task.created, got %s", event["type"])
		}
		if event["task_id"] != task.ID {
			t.Errorf("Expected task_id %s in event, got %s", task.ID, event["task_id"])
		}
	}

	// Verify directories were created
	taskDir := filepath.Join(tempDir, "tickets", task.ID)
	if _, err := os.Stat(taskDir); os.IsNotExist(err) {
		t.Errorf("Task directory was not created: %s", taskDir)
	}
}

func TestTaskManager_Resume(t *testing.T) {
	tm, backlogStore, eventLogger, _, _, _ := createTestTaskManager(t)

	// Create a task first
	opts := CreateTaskOpts{
		Title:    "Test Task",
		TaskType: models.TaskTypeFeat,
	}
	task, err := tm.Create(opts)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Clear events from creation
	eventLogger.events = []map[string]interface{}{}

	// Resume the task
	resumedTask, err := tm.Resume(task.ID)
	if err != nil {
		t.Fatalf("Resume failed: %v", err)
	}

	// Verify status changed to in_progress
	if resumedTask.Status != models.TaskStatusInProgress {
		t.Errorf("Expected status in_progress, got %s", resumedTask.Status)
	}

	// Verify backlog was updated
	storedTask, _ := backlogStore.GetTask(task.ID)
	if storedTask.Status != models.TaskStatusInProgress {
		t.Errorf("Backlog not updated with new status")
	}

	// Verify event was logged
	if len(eventLogger.events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(eventLogger.events))
	} else {
		event := eventLogger.events[0]
		if event["type"] != "task.status_changed" {
			t.Errorf("Expected event type task.status_changed, got %s", event["type"])
		}
	}
}

func TestTaskManager_Resume_ArchivedTask(t *testing.T) {
	tm, backlogStore, _, _, _, _ := createTestTaskManager(t)

	// Create a task and set it as archived
	task := models.NewTask("TASK-00001", "Test Task", models.TaskTypeFeat)
	task.Status = models.TaskStatusArchived
	_ = backlogStore.AddTask(*task)

	// Try to resume archived task
	_, err := tm.Resume(task.ID)
	if err == nil {
		t.Errorf("Expected error when resuming archived task, got nil")
	}
}

func TestTaskManager_Archive(t *testing.T) {
	tm, backlogStore, eventLogger, _, worktreeRemover, tempDir := createTestTaskManager(t)

	// Create a task with repo so worktree is created
	opts := CreateTaskOpts{
		Title:    "Test Task",
		TaskType: models.TaskTypeFeat,
		Repo:     "github.com/test/repo",
	}
	task, err := tm.Create(opts)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Clear events from creation
	eventLogger.events = []map[string]interface{}{}

	// Archive the task
	err = tm.Archive(task.ID)
	if err != nil {
		t.Fatalf("Archive failed: %v", err)
	}

	// Verify task status is archived
	storedTask, _ := backlogStore.GetTask(task.ID)
	if storedTask.Status != models.TaskStatusArchived {
		t.Errorf("Expected status archived, got %s", storedTask.Status)
	}

	// Verify task directory was moved to _archived
	archivedDir := filepath.Join(tempDir, "_archived", task.ID)
	if _, err := os.Stat(archivedDir); os.IsNotExist(err) {
		t.Errorf("Archived task directory not found: %s", archivedDir)
	}

	// Verify handoff.md was created
	handoffPath := filepath.Join(archivedDir, "handoff.md")
	if _, err := os.Stat(handoffPath); os.IsNotExist(err) {
		t.Errorf("handoff.md not found: %s", handoffPath)
	}

	// Verify worktree was removed
	if len(worktreeRemover.removed) != 1 {
		t.Errorf("Expected 1 worktree removal, got %d", len(worktreeRemover.removed))
	}

	// Verify event was logged
	if len(eventLogger.events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(eventLogger.events))
	} else {
		event := eventLogger.events[0]
		if event["type"] != "task.archived" {
			t.Errorf("Expected event type task.archived, got %s", event["type"])
		}
	}
}

func TestTaskManager_Unarchive(t *testing.T) {
	tm, backlogStore, eventLogger, _, _, tempDir := createTestTaskManager(t)

	// Create and archive a task
	opts := CreateTaskOpts{
		Title:    "Test Task",
		TaskType: models.TaskTypeFeat,
	}
	task, err := tm.Create(opts)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	err = tm.Archive(task.ID)
	if err != nil {
		t.Fatalf("Archive failed: %v", err)
	}

	// Clear events
	eventLogger.events = []map[string]interface{}{}

	// Unarchive the task
	err = tm.Unarchive(task.ID)
	if err != nil {
		t.Fatalf("Unarchive failed: %v", err)
	}

	// Verify task status is backlog
	storedTask, _ := backlogStore.GetTask(task.ID)
	if storedTask.Status != models.TaskStatusBacklog {
		t.Errorf("Expected status backlog, got %s", storedTask.Status)
	}

	// Verify task directory was moved back to tickets
	activeDir := filepath.Join(tempDir, "tickets", task.ID)
	if _, err := os.Stat(activeDir); os.IsNotExist(err) {
		t.Errorf("Active task directory not found: %s", activeDir)
	}

	// Verify event was logged
	if len(eventLogger.events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(eventLogger.events))
	} else {
		event := eventLogger.events[0]
		if event["type"] != "task.unarchived" {
			t.Errorf("Expected event type task.unarchived, got %s", event["type"])
		}
	}
}

func TestTaskManager_UpdateStatus(t *testing.T) {
	tm, backlogStore, eventLogger, _, _, _ := createTestTaskManager(t)

	// Create a task
	opts := CreateTaskOpts{
		Title:    "Test Task",
		TaskType: models.TaskTypeFeat,
	}
	task, err := tm.Create(opts)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Clear events
	eventLogger.events = []map[string]interface{}{}

	// Update status
	err = tm.UpdateStatus(task.ID, models.TaskStatusReview)
	if err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}

	// Verify status was updated
	storedTask, _ := backlogStore.GetTask(task.ID)
	if storedTask.Status != models.TaskStatusReview {
		t.Errorf("Expected status review, got %s", storedTask.Status)
	}

	// Verify event was logged
	if len(eventLogger.events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(eventLogger.events))
	} else {
		event := eventLogger.events[0]
		if event["type"] != "task.status_changed" {
			t.Errorf("Expected event type task.status_changed, got %s", event["type"])
		}
	}
}

func TestTaskManager_UpdatePriority(t *testing.T) {
	tm, backlogStore, eventLogger, _, _, _ := createTestTaskManager(t)

	// Create a task
	opts := CreateTaskOpts{
		Title:    "Test Task",
		TaskType: models.TaskTypeFeat,
		Priority: models.PriorityP2,
	}
	task, err := tm.Create(opts)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Clear events
	eventLogger.events = []map[string]interface{}{}

	// Update priority
	err = tm.UpdatePriority(task.ID, models.PriorityP0)
	if err != nil {
		t.Fatalf("UpdatePriority failed: %v", err)
	}

	// Verify priority was updated
	storedTask, _ := backlogStore.GetTask(task.ID)
	if storedTask.Priority != models.PriorityP0 {
		t.Errorf("Expected priority P0, got %s", storedTask.Priority)
	}

	// Verify event was logged
	if len(eventLogger.events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(eventLogger.events))
	} else {
		event := eventLogger.events[0]
		if event["type"] != "task.priority_changed" {
			t.Errorf("Expected event type task.priority_changed, got %s", event["type"])
		}
	}
}

func TestTaskManager_Cleanup(t *testing.T) {
	tm, backlogStore, eventLogger, _, worktreeRemover, _ := createTestTaskManager(t)

	// Create a task with repo so worktree is created
	opts := CreateTaskOpts{
		Title:    "Test Task",
		TaskType: models.TaskTypeFeat,
		Repo:     "github.com/test/repo",
	}
	task, err := tm.Create(opts)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Clear events
	eventLogger.events = []map[string]interface{}{}

	// Cleanup
	err = tm.Cleanup(task.ID)
	if err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	// Verify worktree was removed
	if len(worktreeRemover.removed) != 1 {
		t.Errorf("Expected 1 worktree removal, got %d", len(worktreeRemover.removed))
	}

	// Verify task worktree path was cleared
	storedTask, _ := backlogStore.GetTask(task.ID)
	if storedTask.WorktreePath != "" {
		t.Errorf("Expected empty worktree path, got %s", storedTask.WorktreePath)
	}

	// Verify event was logged
	if len(eventLogger.events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(eventLogger.events))
	} else {
		event := eventLogger.events[0]
		if event["type"] != "worktree.removed" {
			t.Errorf("Expected event type worktree.removed, got %s", event["type"])
		}
	}
}

func TestTaskManager_Delete(t *testing.T) {
	tm, backlogStore, eventLogger, _, worktreeRemover, tempDir := createTestTaskManager(t)

	// Create a task with repo so worktree is created
	opts := CreateTaskOpts{
		Title:    "Test Task",
		TaskType: models.TaskTypeFeat,
		Repo:     "github.com/test/repo",
	}
	task, err := tm.Create(opts)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Clear events
	eventLogger.events = []map[string]interface{}{}

	// Delete
	err = tm.Delete(task.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify worktree was removed
	if len(worktreeRemover.removed) != 1 {
		t.Errorf("Expected 1 worktree removal, got %d", len(worktreeRemover.removed))
	}

	// Verify task directory was removed
	taskDir := filepath.Join(tempDir, "tickets", task.ID)
	if _, err := os.Stat(taskDir); !os.IsNotExist(err) {
		t.Errorf("Task directory still exists: %s", taskDir)
	}

	// Verify task was removed from backlog
	_, err = backlogStore.GetTask(task.ID)
	if err == nil {
		t.Errorf("Task still exists in backlog")
	}

	// Verify event was logged
	if len(eventLogger.events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(eventLogger.events))
	} else {
		event := eventLogger.events[0]
		if event["type"] != "task.deleted" {
			t.Errorf("Expected event type task.deleted, got %s", event["type"])
		}
	}
}

func TestTaskManager_Create_WithoutRepo_NoWorktree(t *testing.T) {
	tm, backlogStore, _, worktreeCreator, _, tempDir := createTestTaskManager(t)

	// Create a task without specifying a repo
	opts := CreateTaskOpts{
		Title:    "Idea Task",
		TaskType: models.TaskTypeSpike,
	}

	task, err := tm.Create(opts)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify task was created in backlog
	storedTask, err := backlogStore.GetTask(task.ID)
	if err != nil {
		t.Fatalf("Failed to get task from backlog: %v", err)
	}
	if storedTask.ID != task.ID {
		t.Errorf("Task ID mismatch in backlog")
	}

	// Verify NO worktree was created
	if len(worktreeCreator.worktrees) != 0 {
		t.Errorf("Expected no worktrees, got %d", len(worktreeCreator.worktrees))
	}

	// Verify task has no worktree path or branch
	if task.WorktreePath != "" {
		t.Errorf("Expected empty worktree path, got %s", task.WorktreePath)
	}
	if task.Branch != "" {
		t.Errorf("Expected empty branch, got %s", task.Branch)
	}

	// Verify ticket directory was still created
	taskDir := filepath.Join(tempDir, "tickets", task.ID)
	if _, err := os.Stat(taskDir); os.IsNotExist(err) {
		t.Errorf("Task directory was not created: %s", taskDir)
	}
}

func TestTaskManager_Create_WithDefaults(t *testing.T) {
	tm, _, _, _, _, _ := createTestTaskManager(t)

	// Create with minimal options
	opts := CreateTaskOpts{
		Title: "Minimal Task",
	}

	task, err := tm.Create(opts)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify defaults were applied
	if task.Priority != models.PriorityP2 {
		t.Errorf("Expected default priority P2, got %s", task.Priority)
	}
	if task.Type != models.TaskTypeFeat {
		t.Errorf("Expected default type feat, got %s", task.Type)
	}
	if task.Status != models.TaskStatusBacklog {
		t.Errorf("Expected default status backlog, got %s", task.Status)
	}
}

func TestTaskManager_Create_RollbackOnWorktreeFailure(t *testing.T) {
	tm, backlogStore, _, worktreeCreator, _, tempDir := createTestTaskManager(t)

	// Set worktree creator to fail
	worktreeCreator.shouldFail = true

	opts := CreateTaskOpts{
		Title:    "Test Task",
		TaskType: models.TaskTypeFeat,
		Repo:     "github.com/test/repo",
	}

	_, err := tm.Create(opts)
	if err == nil {
		t.Fatalf("Expected error on worktree creation failure")
	}

	// Verify task was not added to backlog
	_, err = backlogStore.GetTask("TASK-00001")
	if err == nil {
		t.Errorf("Task should not exist in backlog after rollback")
	}

	// Verify task directory was cleaned up
	taskDir := filepath.Join(tempDir, "tickets", "TASK-00001")
	if _, err := os.Stat(taskDir); !os.IsNotExist(err) {
		t.Errorf("Task directory should have been cleaned up: %s", taskDir)
	}
}
