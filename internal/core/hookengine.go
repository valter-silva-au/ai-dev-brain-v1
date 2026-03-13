package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/valter-silva-au/ai-dev-brain/internal/hooks"
)

// HookEngine processes Claude Code hook events with hybrid shell/Go architecture
type HookEngine struct {
	basePath string
	tracker  *hooks.ChangeTracker
}

// NewHookEngine creates a new hook engine
func NewHookEngine(basePath string) *HookEngine {
	return &HookEngine{
		basePath: basePath,
		tracker:  hooks.NewChangeTracker(basePath),
	}
}

// PreventRecursion checks if we're in a recursive hook invocation
func (he *HookEngine) PreventRecursion() bool {
	return os.Getenv("ADB_HOOK_ACTIVE") == "1"
}

// ProcessPreToolUse handles PreToolUse hooks - blocking validation
func (he *HookEngine) ProcessPreToolUse(event *hooks.PreToolUseEvent) error {
	if he.PreventRecursion() {
		return nil
	}

	// Block edits to vendor/ and go.sum
	if event.ToolName == "Edit" || event.ToolName == "Write" {
		if filePath, ok := event.Parameters["file_path"].(string); ok {
			if strings.Contains(filePath, "/vendor/") || strings.HasSuffix(filePath, "go.sum") {
				return fmt.Errorf("blocked: modifications to vendor/ and go.sum are not allowed")
			}
		}
	}

	return nil
}

// ProcessPostToolUse handles PostToolUse hooks - non-blocking actions
func (he *HookEngine) ProcessPostToolUse(event *hooks.PostToolUseEvent) error {
	if he.PreventRecursion() {
		return nil
	}

	// Auto-format Go files after Edit/Write
	if event.ToolName == "Edit" || event.ToolName == "Write" {
		if filePath, ok := event.Parameters["file_path"].(string); ok {
			if strings.HasSuffix(filePath, ".go") {
				if err := he.formatGoFile(filePath); err != nil {
					// Non-blocking: log warning but don't fail
					fmt.Fprintf(os.Stderr, "Warning: failed to format %s: %v\n", filePath, err)
				}
			}

			// Track the change
			operation := "modified"
			if event.ToolName == "Write" {
				operation = "created"
			}
			if err := he.tracker.TrackChange(filePath, operation); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to track change: %v\n", err)
			}
		}
	}

	return nil
}

// ProcessStop handles Stop hooks - advisory checks
func (he *HookEngine) ProcessStop() error {
	if he.PreventRecursion() {
		return nil
	}

	warnings := []string{}

	// Check for uncommitted changes
	if hasUncommitted, err := he.hasUncommittedChanges(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to check git status: %v\n", err)
	} else if hasUncommitted {
		warnings = append(warnings, "Uncommitted changes detected")
	}

	// Check build status
	if err := he.checkBuild(); err != nil {
		warnings = append(warnings, fmt.Sprintf("Build check failed: %v", err))
	}

	// Check go vet
	if err := he.checkVet(); err != nil {
		warnings = append(warnings, fmt.Sprintf("go vet found issues: %v", err))
	}

	// Update context.md with session summary
	if err := he.updateContextOnStop(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to update context: %v\n", err)
	}

	// Print warnings (advisory only)
	if len(warnings) > 0 {
		fmt.Fprintf(os.Stderr, "\n⚠️  Advisory warnings:\n")
		for _, w := range warnings {
			fmt.Fprintf(os.Stderr, "  - %s\n", w)
		}
		fmt.Fprintf(os.Stderr, "\n")
	}

	return nil
}

// ProcessTaskCompleted handles TaskCompleted hooks - two-phase approach
func (he *HookEngine) ProcessTaskCompleted(event *hooks.TaskCompletedEvent) error {
	if he.PreventRecursion() {
		return nil
	}

	// Phase A: Blocking quality gates
	if err := he.phaseAQualityGates(event); err != nil {
		return fmt.Errorf("quality gate failed: %w", err)
	}

	// Phase B: Non-blocking knowledge extraction
	if err := he.phaseBKnowledgeExtraction(event); err != nil {
		// Non-blocking: log but continue
		fmt.Fprintf(os.Stderr, "Warning: knowledge extraction failed: %v\n", err)
	}

	return nil
}

// ProcessSessionEnd handles SessionEnd hooks
func (he *HookEngine) ProcessSessionEnd(event *hooks.SessionEndEvent) error {
	if he.PreventRecursion() {
		return nil
	}

	// Capture transcript if available
	if transcript, ok := event.Metadata["transcript"].(string); ok {
		taskID := he.getCurrentTaskID()
		if taskID != "" {
			taskDir := filepath.Join(he.basePath, "tickets", taskID)
			if err := hooks.CaptureTranscript(taskDir, event.SessionID, transcript); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to capture transcript: %v\n", err)
			}
		}
	}

	// Update context.md with session summary
	if err := he.updateContextOnSessionEnd(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to update context: %v\n", err)
	}

	return nil
}

// formatGoFile runs gofmt on a Go file
func (he *HookEngine) formatGoFile(filePath string) error {
	cmd := exec.Command("gofmt", "-w", filePath)
	cmd.Env = append(os.Environ(), "ADB_HOOK_ACTIVE=1")
	return cmd.Run()
}

// hasUncommittedChanges checks for uncommitted git changes
func (he *HookEngine) hasUncommittedChanges() (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Env = append(os.Environ(), "ADB_HOOK_ACTIVE=1")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return len(output) > 0, nil
}

// checkBuild runs go build to verify the code compiles
func (he *HookEngine) checkBuild() error {
	cmd := exec.Command("go", "build", "./...")
	cmd.Env = append(os.Environ(), "ADB_HOOK_ACTIVE=1")
	cmd.Dir = he.basePath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", output)
	}
	return nil
}

// checkVet runs go vet
func (he *HookEngine) checkVet() error {
	cmd := exec.Command("go", "vet", "./...")
	cmd.Env = append(os.Environ(), "ADB_HOOK_ACTIVE=1")
	cmd.Dir = he.basePath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", output)
	}
	return nil
}

// phaseAQualityGates performs blocking quality checks
func (he *HookEngine) phaseAQualityGates(event *hooks.TaskCompletedEvent) error {
	// Check tests pass
	cmd := exec.Command("go", "test", "./...", "-count=1")
	cmd.Env = append(os.Environ(), "ADB_HOOK_ACTIVE=1")
	cmd.Dir = he.basePath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("tests failed: %s", output)
	}

	// Check build
	if err := he.checkBuild(); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	// Check vet
	if err := he.checkVet(); err != nil {
		return fmt.Errorf("go vet failed: %w", err)
	}

	return nil
}

// phaseBKnowledgeExtraction performs non-blocking knowledge work
func (he *HookEngine) phaseBKnowledgeExtraction(event *hooks.TaskCompletedEvent) error {
	taskDir := filepath.Join(he.basePath, "tickets", event.TaskID)

	// Get tracked changes
	changes, err := he.tracker.GetChanges()
	if err != nil {
		return err
	}

	// Build summary
	summary := fmt.Sprintf("Task %s completed with %d file changes", event.TaskID, len(changes))
	if len(changes) > 0 {
		summary += "\n\nModified files:\n"
		for _, change := range changes {
			summary += fmt.Sprintf("- %s (%s)\n", change.FilePath, change.Operation)
		}
	}

	// Update context.md
	if err := hooks.UpdateContextFile(taskDir, summary); err != nil {
		return err
	}

	// Clear tracked changes
	if err := he.tracker.Clear(); err != nil {
		return err
	}

	return nil
}

// updateContextOnStop updates context.md when stopping
func (he *HookEngine) updateContextOnStop() error {
	taskID := he.getCurrentTaskID()
	if taskID == "" {
		return nil
	}

	taskDir := filepath.Join(he.basePath, "tickets", taskID)
	changes, err := he.tracker.GetChanges()
	if err != nil {
		return err
	}

	if len(changes) == 0 {
		return nil
	}

	summary := fmt.Sprintf("Session paused with %d file changes", len(changes))
	return hooks.UpdateContextFile(taskDir, summary)
}

// updateContextOnSessionEnd updates context.md when session ends
func (he *HookEngine) updateContextOnSessionEnd(event *hooks.SessionEndEvent) error {
	taskID := he.getCurrentTaskID()
	if taskID == "" {
		return nil
	}

	taskDir := filepath.Join(he.basePath, "tickets", taskID)
	summary := fmt.Sprintf("Session %s ended (duration: %.2fs)", event.SessionID, event.Duration)
	return hooks.UpdateContextFile(taskDir, summary)
}

// getCurrentTaskID gets the current task ID from environment or git branch
func (he *HookEngine) getCurrentTaskID() string {
	// Try environment variable first
	if taskID := os.Getenv("ADB_TASK_ID"); taskID != "" {
		return taskID
	}

	// Try to extract from git branch (e.g., task/TASK-001)
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Env = append(os.Environ(), "ADB_HOOK_ACTIVE=1")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	branch := strings.TrimSpace(string(output))
	if strings.HasPrefix(branch, "task/") {
		return strings.TrimPrefix(branch, "task/")
	}

	return ""
}
