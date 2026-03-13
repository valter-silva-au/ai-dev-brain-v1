package core

import (
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// ProjectInitializer handles full workspace scaffolding for new projects
type ProjectInitializer interface {
	// InitializeProject creates a complete workspace with all scaffolding
	InitializeProject(path string, options InitOptions) error
}

// InitOptions configures project initialization
type InitOptions struct {
	Name         string // Project name
	AIProvider   string // AI provider (claude, gpt)
	TaskIDPrefix string // Task ID prefix (TASK, FEAT, etc)
	GitInit      bool   // Initialize git repository
	WithBMAD     bool   // Include BMAD artifacts templates
}

// FileProjectInitializer implements ProjectInitializer using embedded templates
type FileProjectInitializer struct {
	templatesFS embed.FS // Embedded filesystem for templates
}

// NewFileProjectInitializer creates a new project initializer
func NewFileProjectInitializer(templatesFS embed.FS) *FileProjectInitializer {
	return &FileProjectInitializer{
		templatesFS: templatesFS,
	}
}

// InitializeProject creates a complete workspace from templates
func (pi *FileProjectInitializer) InitializeProject(path string, options InitOptions) error {
	// Resolve absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(absPath, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Initialize git if requested
	if options.GitInit {
		if err := pi.initGit(absPath); err != nil {
			return fmt.Errorf("failed to initialize git: %w", err)
		}
	}

	// Create directory structure
	if err := pi.createDirectories(absPath); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	// Create configuration files
	if err := pi.createConfig(absPath, options); err != nil {
		return fmt.Errorf("failed to create config: %w", err)
	}

	// Create .claude/ setup
	if err := pi.createClaudeSetup(absPath); err != nil {
		return fmt.Errorf("failed to create Claude setup: %w", err)
	}

	// Create BMAD artifacts if requested
	if options.WithBMAD {
		if err := pi.createBMADArtifacts(absPath, options.Name); err != nil {
			return fmt.Errorf("failed to create BMAD artifacts: %w", err)
		}
	}

	return nil
}

// initGit initializes a git repository
func (pi *FileProjectInitializer) initGit(path string) error {
	cmd := exec.Command("git", "init")
	cmd.Dir = path
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git init failed: %w", err)
	}

	// Create .gitignore
	gitignorePath := filepath.Join(path, ".gitignore")
	gitignoreContent := `# AI Dev Brain
.adb_terminal_state.json
.task_counter
.events.jsonl
work/
sessions/

# IDE
.vscode/
.idea/

# OS
.DS_Store
Thumbs.db

# Build
*.exe
*.dll
*.so
*.dylib
*.test
*.out
`
	if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0o644); err != nil {
		return fmt.Errorf("failed to create .gitignore: %w", err)
	}

	return nil
}

// createDirectories creates the standard directory structure
func (pi *FileProjectInitializer) createDirectories(path string) error {
	dirs := []string{
		"tickets",
		"tickets/_archived",
		"work",
		"sessions",
		".adb",
		".claude",
		".claude/rules",
		"docs",
		"docs/bmad",
	}

	for _, dir := range dirs {
		dirPath := filepath.Join(path, dir)
		if err := os.MkdirAll(dirPath, 0o755); err != nil {
			return fmt.Errorf("failed to create %s: %w", dir, err)
		}
	}

	return nil
}

// createConfig creates .taskconfig and .taskrc files
func (pi *FileProjectInitializer) createConfig(path string, options InitOptions) error {
	// Create backlog.yaml
	backlogPath := filepath.Join(path, "backlog.yaml")
	backlogContent := "tasks: []\n"
	if err := os.WriteFile(backlogPath, []byte(backlogContent), 0o644); err != nil {
		return fmt.Errorf("failed to create backlog.yaml: %w", err)
	}

	// Set defaults
	projectName := options.Name
	if projectName == "" {
		projectName = filepath.Base(path)
	}

	aiProvider := options.AIProvider
	if aiProvider == "" {
		aiProvider = "claude"
	}

	taskIDPrefix := options.TaskIDPrefix
	if taskIDPrefix == "" {
		taskIDPrefix = "TASK"
	}

	// Create .taskrc
	taskrcPath := filepath.Join(path, ".taskrc")
	taskrcContent := fmt.Sprintf(`# AI Dev Brain Repository Configuration
# This file configures the workspace for AI-assisted development

name: "%s"
ai_provider: "%s"
task_id_prefix: "%s"

build:
  command: "go build ./..."
  test_command: "go test ./... -count=1"

git:
  worktree_dir: "work"
  default_branch: "main"

hooks:
  enabled: true
  on_task_create: true
  on_status_change: true
`, projectName, aiProvider, taskIDPrefix)

	if err := os.WriteFile(taskrcPath, []byte(taskrcContent), 0o644); err != nil {
		return fmt.Errorf("failed to create .taskrc: %w", err)
	}

	return nil
}

// createClaudeSetup creates .claude/ directory and files
func (pi *FileProjectInitializer) createClaudeSetup(path string) error {
	// Create project_context.md
	projectContextPath := filepath.Join(path, ".claude", "project_context.md")
	projectContextContent := `# Project Context

## Overview
This workspace uses AI Dev Brain for task management and AI-assisted development.

## Structure
- ` + "`tickets/`" + ` - Task-specific context and notes
- ` + "`work/`" + ` - Git worktrees for task isolation
- ` + "`sessions/`" + ` - Captured session data
- ` + "`backlog.yaml`" + ` - Task backlog
- ` + "`.taskrc`" + ` - Workspace configuration

## Commands
- ` + "`adb task create`" + ` - Create new task with worktree
- ` + "`adb task resume <task-id>`" + ` - Resume task
- ` + "`adb task status`" + ` - View all tasks
- ` + "`adb team <name> <prompt>`" + ` - Launch multi-agent orchestration
- ` + "`adb agents`" + ` - List available agents
- ` + "`adb mcp check`" + ` - Validate MCP server health
`

	if err := os.WriteFile(projectContextPath, []byte(projectContextContent), 0o644); err != nil {
		return fmt.Errorf("failed to create project_context.md: %w", err)
	}

	// Create rules/task-isolation.md
	taskIsolationPath := filepath.Join(path, ".claude", "rules", "task-isolation.md")
	taskIsolationContent := `# Task Isolation Rules

## Git Worktrees
Each task gets an isolated git worktree in ` + "`work/TASK-XXXXX/`" + `.

## Context Management
- Read task context from ` + "`tickets/TASK-XXXXX/context.md`" + `
- Update notes in ` + "`tickets/TASK-XXXXX/notes.md`" + `
- Follow handoff instructions in ` + "`tickets/TASK-XXXXX/handoff.md`" + `

## Testing
Always run tests before committing: ` + "`go test ./... -count=1`" + `
`

	if err := os.WriteFile(taskIsolationPath, []byte(taskIsolationContent), 0o644); err != nil {
		return fmt.Errorf("failed to create task-isolation.md: %w", err)
	}

	return nil
}

// createBMADArtifacts creates BMAD template files
func (pi *FileProjectInitializer) createBMADArtifacts(path string, projectName string) error {
	if projectName == "" {
		projectName = filepath.Base(path)
	}

	// Create PRD template
	prdPath := filepath.Join(path, "docs", "bmad", "PRD.md")
	prdContent := fmt.Sprintf(`# Product Requirements Document: %s

## 1. Executive Summary
[Brief overview of the product and its goals]

## 2. Problem Statement
[What problem are we solving?]

## 3. Goals and Objectives
- Goal 1
- Goal 2
- Goal 3

## 4. User Stories
### As a [user type]
I want [goal]
So that [benefit]

## 5. Functional Requirements
- FR1: [Requirement]
- FR2: [Requirement]

## 6. Non-Functional Requirements
- Performance: [Criteria]
- Security: [Criteria]
- Scalability: [Criteria]

## 7. Acceptance Criteria
- [ ] Criterion 1
- [ ] Criterion 2

## 8. Success Metrics
- Metric 1: [Target]
- Metric 2: [Target]
`, projectName)

	if err := os.WriteFile(prdPath, []byte(prdContent), 0o644); err != nil {
		return fmt.Errorf("failed to create PRD.md: %w", err)
	}

	// Create Product Brief template
	briefPath := filepath.Join(path, "docs", "bmad", "product-brief.md")
	briefContent := fmt.Sprintf(`# Product Brief: %s

## Vision
[One-sentence vision statement]

## Target Users
[Who is this for?]

## Key Features
1. Feature 1
2. Feature 2
3. Feature 3

## Competitive Advantage
[What makes this unique?]

## Timeline
- Phase 1: [Date]
- Phase 2: [Date]
- Launch: [Date]
`, projectName)

	if err := os.WriteFile(briefPath, []byte(briefContent), 0o644); err != nil {
		return fmt.Errorf("failed to create product-brief.md: %w", err)
	}

	// Create Technical Specification template
	techSpecPath := filepath.Join(path, "docs", "bmad", "tech-spec.md")
	techSpecContent := fmt.Sprintf(`# Technical Specification: %s

## 1. System Architecture
[High-level architecture diagram/description]

## 2. Technology Stack
- Backend: [Technologies]
- Frontend: [Technologies]
- Database: [Technologies]
- Infrastructure: [Technologies]

## 3. API Design
### Endpoints
- `+"`GET /api/resource`"+` - [Description]
- `+"`POST /api/resource`"+` - [Description]

## 4. Data Models
### Model 1
`+"```"+`
{
  "field1": "type",
  "field2": "type"
}
`+"```"+`

## 5. Security Considerations
- Authentication: [Method]
- Authorization: [Method]
- Data Encryption: [Method]

## 6. Performance Requirements
- Response Time: [Target]
- Throughput: [Target]
- Concurrent Users: [Target]

## 7. Testing Strategy
- Unit Tests: [Coverage target]
- Integration Tests: [Approach]
- E2E Tests: [Approach]
`, projectName)

	if err := os.WriteFile(techSpecPath, []byte(techSpecContent), 0o644); err != nil {
		return fmt.Errorf("failed to create tech-spec.md: %w", err)
	}

	// Create Architecture Document template
	archPath := filepath.Join(path, "docs", "bmad", "architecture-doc.md")
	archContent := fmt.Sprintf(`# Architecture Document: %s

## 1. System Overview
[High-level description of the system]

## 2. Architecture Principles
- Principle 1: [Description]
- Principle 2: [Description]

## 3. Component Diagram
[Diagram showing system components and their relationships]

## 4. Components
### Component 1
- **Purpose**: [Description]
- **Responsibilities**: [List]
- **Dependencies**: [List]

## 5. Data Flow
[Description of how data flows through the system]

## 6. Integration Points
- External Service 1: [Description]
- External Service 2: [Description]

## 7. Scalability Strategy
[How the system scales]

## 8. Deployment Architecture
[Infrastructure and deployment setup]

## 9. Monitoring and Observability
- Metrics: [What we measure]
- Logging: [What we log]
- Tracing: [Distributed tracing approach]
`, projectName)

	if err := os.WriteFile(archPath, []byte(archContent), 0o644); err != nil {
		return fmt.Errorf("failed to create architecture-doc.md: %w", err)
	}

	// Create Quality Gate Checklist
	qgPath := filepath.Join(path, "docs", "bmad", "quality-gates.md")
	qgContent := `# Quality Gate Checklist

## Pre-Development
- [ ] PRD reviewed and approved
- [ ] Technical specification reviewed
- [ ] Architecture design reviewed
- [ ] Security review completed
- [ ] Dependencies identified

## Development
- [ ] Code follows style guidelines
- [ ] Unit tests written (>80% coverage)
- [ ] Integration tests written
- [ ] Code reviewed by peers
- [ ] Documentation updated

## Pre-Production
- [ ] All tests passing
- [ ] Performance benchmarks met
- [ ] Security scan passed
- [ ] Load testing completed
- [ ] Monitoring/alerting configured

## Production Readiness
- [ ] Deployment plan documented
- [ ] Rollback plan documented
- [ ] Runbook created
- [ ] On-call rotation scheduled
- [ ] Stakeholders notified

## Post-Deployment
- [ ] Monitoring dashboard active
- [ ] Error rates acceptable
- [ ] Performance metrics acceptable
- [ ] User feedback collected
- [ ] Post-mortem scheduled (if needed)
`

	if err := os.WriteFile(qgPath, []byte(qgContent), 0o644); err != nil {
		return fmt.Errorf("failed to create quality-gates.md: %w", err)
	}

	return nil
}
