package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/valter-silva-au/ai-dev-brain/pkg/models"
	"gopkg.in/yaml.v3"
)

// BacklogManager defines the interface for managing task backlogs
type BacklogManager interface {
	Load() (*models.Backlog, error)
	Save(backlog *models.Backlog) error
	AddTask(task models.Task) error
	UpdateTask(task models.Task) error
	GetTask(id string) (*models.Task, error)
	RemoveTask(id string) error
}

// FileBacklogManager implements BacklogManager with file-based storage
type FileBacklogManager struct {
	filePath string
	mu       sync.RWMutex // file-level locking for concurrent safety
}

// NewFileBacklogManager creates a new file-based backlog manager
func NewFileBacklogManager(filePath string) *FileBacklogManager {
	return &FileBacklogManager{
		filePath: filePath,
	}
}

// loadUnsafe reads the backlog from the YAML file without acquiring locks
func (fbm *FileBacklogManager) loadUnsafe() (*models.Backlog, error) {
	// Check if file exists
	if _, err := os.Stat(fbm.filePath); os.IsNotExist(err) {
		// Return empty backlog if file doesn't exist
		return models.NewBacklog(), nil
	}

	// Read the file
	data, err := os.ReadFile(fbm.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read backlog file: %w", err)
	}

	// Handle empty file
	if len(data) == 0 {
		return models.NewBacklog(), nil
	}

	// Parse YAML
	var backlog models.Backlog
	if err := yaml.Unmarshal(data, &backlog); err != nil {
		return nil, fmt.Errorf("failed to parse backlog YAML: %w", err)
	}

	// Ensure Tasks slice is not nil
	if backlog.Tasks == nil {
		backlog.Tasks = []models.Task{}
	}

	return &backlog, nil
}

// Load reads the backlog from the YAML file
func (fbm *FileBacklogManager) Load() (*models.Backlog, error) {
	fbm.mu.RLock()
	defer fbm.mu.RUnlock()
	return fbm.loadUnsafe()
}

// saveUnsafe writes the backlog to the YAML file without acquiring locks
func (fbm *FileBacklogManager) saveUnsafe(backlog *models.Backlog) error {
	// Ensure directory exists
	dir := filepath.Dir(fbm.filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(backlog)
	if err != nil {
		return fmt.Errorf("failed to marshal backlog to YAML: %w", err)
	}

	// Write to file with proper permissions
	if err := os.WriteFile(fbm.filePath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write backlog file: %w", err)
	}

	return nil
}

// Save writes the backlog to the YAML file
func (fbm *FileBacklogManager) Save(backlog *models.Backlog) error {
	fbm.mu.Lock()
	defer fbm.mu.Unlock()
	return fbm.saveUnsafe(backlog)
}

// AddTask adds a new task to the backlog
func (fbm *FileBacklogManager) AddTask(task models.Task) error {
	fbm.mu.Lock()
	defer fbm.mu.Unlock()

	backlog, err := fbm.loadUnsafe()
	if err != nil {
		return fmt.Errorf("failed to load backlog: %w", err)
	}

	// Check if task already exists
	if backlog.FindTaskByID(task.ID) != nil {
		return fmt.Errorf("task with ID %s already exists", task.ID)
	}

	backlog.AddTask(task)

	if err := fbm.saveUnsafe(backlog); err != nil {
		return fmt.Errorf("failed to save backlog: %w", err)
	}

	return nil
}

// UpdateTask updates an existing task in the backlog
func (fbm *FileBacklogManager) UpdateTask(task models.Task) error {
	fbm.mu.Lock()
	defer fbm.mu.Unlock()

	backlog, err := fbm.loadUnsafe()
	if err != nil {
		return fmt.Errorf("failed to load backlog: %w", err)
	}

	if !backlog.UpdateTask(task) {
		return fmt.Errorf("task with ID %s not found", task.ID)
	}

	if err := fbm.saveUnsafe(backlog); err != nil {
		return fmt.Errorf("failed to save backlog: %w", err)
	}

	return nil
}

// GetTask retrieves a task by ID from the backlog
func (fbm *FileBacklogManager) GetTask(id string) (*models.Task, error) {
	fbm.mu.RLock()
	defer fbm.mu.RUnlock()

	backlog, err := fbm.loadUnsafe()
	if err != nil {
		return nil, fmt.Errorf("failed to load backlog: %w", err)
	}

	task := backlog.FindTaskByID(id)
	if task == nil {
		return nil, fmt.Errorf("task with ID %s not found", id)
	}

	// Return a copy to avoid external modifications
	taskCopy := *task
	return &taskCopy, nil
}

// RemoveTask removes a task from the backlog by ID
func (fbm *FileBacklogManager) RemoveTask(id string) error {
	fbm.mu.Lock()
	defer fbm.mu.Unlock()

	backlog, err := fbm.loadUnsafe()
	if err != nil {
		return fmt.Errorf("failed to load backlog: %w", err)
	}

	if !backlog.RemoveTask(id) {
		return fmt.Errorf("task with ID %s not found", id)
	}

	if err := fbm.saveUnsafe(backlog); err != nil {
		return fmt.Errorf("failed to save backlog: %w", err)
	}

	return nil
}
