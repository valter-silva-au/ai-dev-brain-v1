package core

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// BootstrapConfig contains configuration for bootstrapping a task
type BootstrapConfig struct {
	// TaskID is the unique task identifier (e.g., "TASK-00001")
	TaskID string
	// Title is the task title
	Title string
	// Description is the task description
	Description string
	// AcceptanceCriteria is a list of acceptance criteria
	AcceptanceCriteria []string
	// Dependencies is a list of task dependencies
	Dependencies []string
	// RelatedTasks is information about related tasks
	RelatedTasks string
	// Status is the initial task status
	Status string
	// TicketsDir is the base directory for tickets (e.g., "tickets")
	TicketsDir string
	// WorktreeDir is the worktree directory (e.g., ".") for creating .claude/rules/
	WorktreeDir string
}

// BootstrapResult contains the paths created during bootstrapping
type BootstrapResult struct {
	// TaskDir is the path to the task directory
	TaskDir string
	// StatusFile is the path to status.yaml
	StatusFile string
	// ContextFile is the path to context.md
	ContextFile string
	// NotesFile is the path to notes.md
	NotesFile string
	// DesignFile is the path to design.md
	DesignFile string
	// SessionsDir is the path to sessions directory
	SessionsDir string
	// KnowledgeDir is the path to knowledge directory
	KnowledgeDir string
	// DecisionsFile is the path to knowledge/decisions.yaml
	DecisionsFile string
	// TaskContextFile is the path to .claude/rules/task-context.md
	TaskContextFile string
}

// BootstrapSystem scaffolds a new task's directory structure
// It creates:
// - tickets/TASK-XXXXX/ directory
// - status.yaml, context.md, notes.md, design.md files
// - sessions/ and knowledge/ subdirectories
// - knowledge/decisions.yaml (initially empty)
// - .claude/rules/task-context.md in the worktree
func BootstrapSystem(config BootstrapConfig, tm TemplateManager) (*BootstrapResult, error) {
	if config.TaskID == "" {
		return nil, fmt.Errorf("TaskID is required")
	}
	if config.Title == "" {
		return nil, fmt.Errorf("Title is required")
	}
	if config.TicketsDir == "" {
		config.TicketsDir = "tickets"
	}
	if config.WorktreeDir == "" {
		config.WorktreeDir = "."
	}
	if config.Status == "" {
		config.Status = "pending"
	}

	// Create result structure
	result := &BootstrapResult{}

	// Create task directory
	taskDir := filepath.Join(config.TicketsDir, config.TaskID)
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create task directory: %w", err)
	}
	result.TaskDir = taskDir

	// Create sessions subdirectory
	sessionsDir := filepath.Join(taskDir, "sessions")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create sessions directory: %w", err)
	}
	result.SessionsDir = sessionsDir

	// Create knowledge subdirectory
	knowledgeDir := filepath.Join(taskDir, "knowledge")
	if err := os.MkdirAll(knowledgeDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create knowledge directory: %w", err)
	}
	result.KnowledgeDir = knowledgeDir

	// Get current timestamp
	now := time.Now().Format(time.RFC3339)

	// Prepare template data
	templateData := map[string]interface{}{
		"TaskID":             config.TaskID,
		"Title":              config.Title,
		"Description":        config.Description,
		"AcceptanceCriteria": config.AcceptanceCriteria,
		"Dependencies":       config.Dependencies,
		"RelatedTasks":       config.RelatedTasks,
		"Status":             config.Status,
		"CreatedAt":          now,
		"UpdatedAt":          now,
		"Context":            config.Description,
		"Notes":              "",
		"References":         "",
		"Overview":           "",
		"Components":         "",
		"DataFlow":           "",
		"ImplementationPlan": "",
		"TechnicalDecisions": "",
		"OpenQuestions":      "",
	}

	// Create status.yaml
	statusFile := filepath.Join(taskDir, "status.yaml")
	if err := renderTemplateToFile(tm, TemplateTypeStatus, templateData, statusFile); err != nil {
		return nil, fmt.Errorf("failed to create status.yaml: %w", err)
	}
	result.StatusFile = statusFile

	// Create context.md
	contextFile := filepath.Join(taskDir, "context.md")
	if err := renderTemplateToFile(tm, TemplateTypeContext, templateData, contextFile); err != nil {
		return nil, fmt.Errorf("failed to create context.md: %w", err)
	}
	result.ContextFile = contextFile

	// Create notes.md
	notesFile := filepath.Join(taskDir, "notes.md")
	if err := renderTemplateToFile(tm, TemplateTypeNotes, templateData, notesFile); err != nil {
		return nil, fmt.Errorf("failed to create notes.md: %w", err)
	}
	result.NotesFile = notesFile

	// Create design.md
	designFile := filepath.Join(taskDir, "design.md")
	if err := renderTemplateToFile(tm, TemplateTypeDesign, templateData, designFile); err != nil {
		return nil, fmt.Errorf("failed to create design.md: %w", err)
	}
	result.DesignFile = designFile

	// Create knowledge/decisions.yaml (initially empty)
	decisionsFile := filepath.Join(knowledgeDir, "decisions.yaml")
	if err := os.WriteFile(decisionsFile, []byte("# Task Decisions\n# This file tracks key decisions made during task development\n\ndecisions: []\n"), 0o644); err != nil {
		return nil, fmt.Errorf("failed to create decisions.yaml: %w", err)
	}
	result.DecisionsFile = decisionsFile

	// Create .claude/rules/task-context.md in worktree
	claudeRulesDir := filepath.Join(config.WorktreeDir, ".claude", "rules")
	if err := os.MkdirAll(claudeRulesDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create .claude/rules directory: %w", err)
	}

	taskContextFile := filepath.Join(claudeRulesDir, "task-context.md")
	if err := renderTemplateToFile(tm, TemplateTypeTaskContext, templateData, taskContextFile); err != nil {
		return nil, fmt.Errorf("failed to create task-context.md: %w", err)
	}
	result.TaskContextFile = taskContextFile

	return result, nil
}

// renderTemplateToFile renders a template and writes it to a file
func renderTemplateToFile(tm TemplateManager, templateType TemplateType, data interface{}, filePath string) error {
	content, err := tm.RenderBytes(templateType, data)
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	if err := os.WriteFile(filePath, content, 0o644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
