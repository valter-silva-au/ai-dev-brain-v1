package models

import "time"

// TaskType represents the type of task
type TaskType string

const (
	TaskTypeFeat     TaskType = "feat"
	TaskTypeBug      TaskType = "bug"
	TaskTypeSpike    TaskType = "spike"
	TaskTypeRefactor TaskType = "refactor"
)

// TaskStatus represents the current status of a task
type TaskStatus string

const (
	TaskStatusBacklog    TaskStatus = "backlog"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusBlocked    TaskStatus = "blocked"
	TaskStatusReview     TaskStatus = "review"
	TaskStatusDone       TaskStatus = "done"
	TaskStatusArchived   TaskStatus = "archived"
)

// Priority represents the priority level of a task
type Priority string

const (
	PriorityP0 Priority = "P0"
	PriorityP1 Priority = "P1"
	PriorityP2 Priority = "P2"
	PriorityP3 Priority = "P3"
)

// Task represents a task in the system
type Task struct {
	ID           string            `yaml:"id"`
	Title        string            `yaml:"title"`
	Type         TaskType          `yaml:"type"`
	Source       string            `yaml:"source,omitempty"`
	Status       TaskStatus        `yaml:"status"`
	Priority     Priority          `yaml:"priority"`
	Owner        string            `yaml:"owner,omitempty"`
	Created      time.Time         `yaml:"created"`
	Updated      time.Time         `yaml:"updated"`
	Repo         string            `yaml:"repo,omitempty"`
	Branch       string            `yaml:"branch,omitempty"`
	WorktreePath string            `yaml:"worktree_path,omitempty"`
	TicketPath   string            `yaml:"ticket_path,omitempty"`
	Tags         []string          `yaml:"tags,omitempty"`
	BlockedBy    []string          `yaml:"blocked_by,omitempty"`
	Teams        []string          `yaml:"teams,omitempty"`
	TeamMetadata map[string]string `yaml:"team_metadata,omitempty"`
}

// NewTask creates a new task with default values
func NewTask(id, title string, taskType TaskType) *Task {
	now := time.Now().UTC()
	return &Task{
		ID:           id,
		Title:        title,
		Type:         taskType,
		Status:       TaskStatusBacklog,
		Priority:     PriorityP2,
		Created:      now,
		Updated:      now,
		Tags:         []string{},
		BlockedBy:    []string{},
		Teams:        []string{},
		TeamMetadata: make(map[string]string),
	}
}

// IsActive returns true if the task is in an active status
func (t *Task) IsActive() bool {
	return t.Status == TaskStatusInProgress || t.Status == TaskStatusReview || t.Status == TaskStatusBlocked
}

// IsBlocked returns true if the task is blocked
func (t *Task) IsBlocked() bool {
	return t.Status == TaskStatusBlocked || len(t.BlockedBy) > 0
}

// UpdateTimestamp updates the Updated timestamp to the current UTC time
func (t *Task) UpdateTimestamp() {
	t.Updated = time.Now().UTC()
}
