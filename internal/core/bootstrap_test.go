package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/valter-silva-au/ai-dev-brain/templates/claude"
)

func TestBootstrapSystem(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create template manager
	tm, err := NewEmbedTemplateManager(claude.FS)
	if err != nil {
		t.Fatalf("Failed to create template manager: %v", err)
	}

	// Create bootstrap config
	config := BootstrapConfig{
		TaskID:      "TASK-00001",
		Title:       "Test Task",
		Description: "This is a test task for bootstrapping",
		AcceptanceCriteria: []string{
			"Criterion 1",
			"Criterion 2",
			"Criterion 3",
		},
		Dependencies: []string{
			"TASK-00000",
		},
		RelatedTasks: "Related to TASK-00002",
		Status:       "pending",
		TicketsDir:   filepath.Join(tempDir, "tickets"),
		WorktreeDir:  tempDir,
	}

	// Run bootstrap
	result, err := BootstrapSystem(config, tm)
	if err != nil {
		t.Fatalf("BootstrapSystem() failed: %v", err)
	}

	// Verify task directory was created
	if result.TaskDir == "" {
		t.Error("TaskDir should not be empty")
	}
	if _, err := os.Stat(result.TaskDir); os.IsNotExist(err) {
		t.Errorf("Task directory was not created: %s", result.TaskDir)
	}

	// Verify sessions directory was created
	if result.SessionsDir == "" {
		t.Error("SessionsDir should not be empty")
	}
	if _, err := os.Stat(result.SessionsDir); os.IsNotExist(err) {
		t.Errorf("Sessions directory was not created: %s", result.SessionsDir)
	}

	// Verify knowledge directory was created
	if result.KnowledgeDir == "" {
		t.Error("KnowledgeDir should not be empty")
	}
	if _, err := os.Stat(result.KnowledgeDir); os.IsNotExist(err) {
		t.Errorf("Knowledge directory was not created: %s", result.KnowledgeDir)
	}

	// Verify status.yaml was created
	if result.StatusFile == "" {
		t.Error("StatusFile should not be empty")
	}
	if _, err := os.Stat(result.StatusFile); os.IsNotExist(err) {
		t.Errorf("Status file was not created: %s", result.StatusFile)
	}

	// Verify context.md was created
	if result.ContextFile == "" {
		t.Error("ContextFile should not be empty")
	}
	if _, err := os.Stat(result.ContextFile); os.IsNotExist(err) {
		t.Errorf("Context file was not created: %s", result.ContextFile)
	}

	// Verify notes.md was created
	if result.NotesFile == "" {
		t.Error("NotesFile should not be empty")
	}
	if _, err := os.Stat(result.NotesFile); os.IsNotExist(err) {
		t.Errorf("Notes file was not created: %s", result.NotesFile)
	}

	// Verify design.md was created
	if result.DesignFile == "" {
		t.Error("DesignFile should not be empty")
	}
	if _, err := os.Stat(result.DesignFile); os.IsNotExist(err) {
		t.Errorf("Design file was not created: %s", result.DesignFile)
	}

	// Verify decisions.yaml was created
	if result.DecisionsFile == "" {
		t.Error("DecisionsFile should not be empty")
	}
	if _, err := os.Stat(result.DecisionsFile); os.IsNotExist(err) {
		t.Errorf("Decisions file was not created: %s", result.DecisionsFile)
	}

	// Verify task-context.md was created in .claude/rules/
	if result.TaskContextFile == "" {
		t.Error("TaskContextFile should not be empty")
	}
	if _, err := os.Stat(result.TaskContextFile); os.IsNotExist(err) {
		t.Errorf("Task context file was not created: %s", result.TaskContextFile)
	}

	// Verify expected path structure
	expectedTaskDir := filepath.Join(tempDir, "tickets", "TASK-00001")
	if result.TaskDir != expectedTaskDir {
		t.Errorf("Expected TaskDir to be %s, got %s", expectedTaskDir, result.TaskDir)
	}

	expectedTaskContextFile := filepath.Join(tempDir, ".claude", "rules", "task-context.md")
	if result.TaskContextFile != expectedTaskContextFile {
		t.Errorf("Expected TaskContextFile to be %s, got %s", expectedTaskContextFile, result.TaskContextFile)
	}
}

func TestBootstrapSystemFileContents(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create template manager
	tm, err := NewEmbedTemplateManager(claude.FS)
	if err != nil {
		t.Fatalf("Failed to create template manager: %v", err)
	}

	// Create bootstrap config
	config := BootstrapConfig{
		TaskID:      "TASK-99999",
		Title:       "Content Test Task",
		Description: "Testing file contents",
		AcceptanceCriteria: []string{
			"Must have content A",
			"Must have content B",
		},
		Dependencies: []string{
			"TASK-99998",
		},
		RelatedTasks: "Related to TASK-99997",
		Status:       "in_progress",
		TicketsDir:   filepath.Join(tempDir, "tickets"),
		WorktreeDir:  tempDir,
	}

	// Run bootstrap
	result, err := BootstrapSystem(config, tm)
	if err != nil {
		t.Fatalf("BootstrapSystem() failed: %v", err)
	}

	// Test status.yaml content
	statusContent, err := os.ReadFile(result.StatusFile)
	if err != nil {
		t.Fatalf("Failed to read status file: %v", err)
	}
	statusStr := string(statusContent)
	expectedInStatus := []string{
		"task_id: TASK-99999",
		"title: Content Test Task",
		"status: in_progress",
	}
	for _, expected := range expectedInStatus {
		if !strings.Contains(statusStr, expected) {
			t.Errorf("Expected status.yaml to contain %q, but it didn't.\nContent:\n%s", expected, statusStr)
		}
	}

	// Test context.md content
	contextContent, err := os.ReadFile(result.ContextFile)
	if err != nil {
		t.Fatalf("Failed to read context file: %v", err)
	}
	contextStr := string(contextContent)
	expectedInContext := []string{
		"# Context: Content Test Task",
		"**Task ID:** TASK-99999",
		"Testing file contents",
		"- [ ] Must have content A",
		"- [ ] Must have content B",
		"- TASK-99998",
		"Related to TASK-99997",
	}
	for _, expected := range expectedInContext {
		if !strings.Contains(contextStr, expected) {
			t.Errorf("Expected context.md to contain %q, but it didn't.\nContent:\n%s", expected, contextStr)
		}
	}

	// Test notes.md content
	notesContent, err := os.ReadFile(result.NotesFile)
	if err != nil {
		t.Fatalf("Failed to read notes file: %v", err)
	}
	notesStr := string(notesContent)
	expectedInNotes := []string{
		"# Notes: Content Test Task",
		"**Task ID:** TASK-99999",
		"Testing file contents",
		"- [ ] Must have content A",
		"- [ ] Must have content B",
	}
	for _, expected := range expectedInNotes {
		if !strings.Contains(notesStr, expected) {
			t.Errorf("Expected notes.md to contain %q, but it didn't.\nContent:\n%s", expected, notesStr)
		}
	}

	// Test design.md content
	designContent, err := os.ReadFile(result.DesignFile)
	if err != nil {
		t.Fatalf("Failed to read design file: %v", err)
	}
	designStr := string(designContent)
	expectedInDesign := []string{
		"# Design Document: Content Test Task",
		"**Task ID:** TASK-99999",
	}
	for _, expected := range expectedInDesign {
		if !strings.Contains(designStr, expected) {
			t.Errorf("Expected design.md to contain %q, but it didn't.\nContent:\n%s", expected, designStr)
		}
	}

	// Test decisions.yaml content
	decisionsContent, err := os.ReadFile(result.DecisionsFile)
	if err != nil {
		t.Fatalf("Failed to read decisions file: %v", err)
	}
	decisionsStr := string(decisionsContent)
	expectedInDecisions := []string{
		"# Task Decisions",
		"decisions: []",
	}
	for _, expected := range expectedInDecisions {
		if !strings.Contains(decisionsStr, expected) {
			t.Errorf("Expected decisions.yaml to contain %q, but it didn't.\nContent:\n%s", expected, decisionsStr)
		}
	}

	// Test task-context.md content
	taskContextContent, err := os.ReadFile(result.TaskContextFile)
	if err != nil {
		t.Fatalf("Failed to read task-context file: %v", err)
	}
	taskContextStr := string(taskContextContent)
	expectedInTaskContext := []string{
		"# Task Context",
		"You are currently working on **TASK-99999: Content Test Task**",
		"Testing file contents",
		"- [ ] Must have content A",
		"- [ ] Must have content B",
		"**Status:** in_progress",
		"tickets/TASK-99999/",
	}
	for _, expected := range expectedInTaskContext {
		if !strings.Contains(taskContextStr, expected) {
			t.Errorf("Expected task-context.md to contain %q, but it didn't.\nContent:\n%s", expected, taskContextStr)
		}
	}
}

func TestBootstrapSystemMissingTaskID(t *testing.T) {
	tempDir := t.TempDir()

	tm, err := NewEmbedTemplateManager(claude.FS)
	if err != nil {
		t.Fatalf("Failed to create template manager: %v", err)
	}

	config := BootstrapConfig{
		TaskID:      "", // Missing TaskID
		Title:       "Test Task",
		Description: "Test",
		TicketsDir:  filepath.Join(tempDir, "tickets"),
		WorktreeDir: tempDir,
	}

	_, err = BootstrapSystem(config, tm)
	if err == nil {
		t.Error("Expected error when TaskID is missing, got nil")
	}
	if !strings.Contains(err.Error(), "TaskID is required") {
		t.Errorf("Expected error to mention TaskID, got: %v", err)
	}
}

func TestBootstrapSystemMissingTitle(t *testing.T) {
	tempDir := t.TempDir()

	tm, err := NewEmbedTemplateManager(claude.FS)
	if err != nil {
		t.Fatalf("Failed to create template manager: %v", err)
	}

	config := BootstrapConfig{
		TaskID:      "TASK-00001",
		Title:       "", // Missing Title
		Description: "Test",
		TicketsDir:  filepath.Join(tempDir, "tickets"),
		WorktreeDir: tempDir,
	}

	_, err = BootstrapSystem(config, tm)
	if err == nil {
		t.Error("Expected error when Title is missing, got nil")
	}
	if !strings.Contains(err.Error(), "Title is required") {
		t.Errorf("Expected error to mention Title, got: %v", err)
	}
}

func TestBootstrapSystemDefaultValues(t *testing.T) {
	tempDir := t.TempDir()

	tm, err := NewEmbedTemplateManager(claude.FS)
	if err != nil {
		t.Fatalf("Failed to create template manager: %v", err)
	}

	// Config with minimal values (should use defaults)
	config := BootstrapConfig{
		TaskID:      "TASK-00002",
		Title:       "Minimal Config Task",
		Description: "Test defaults",
		// TicketsDir and WorktreeDir not set - should default
		// Status not set - should default to "pending"
	}

	// Change to temp directory for this test
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWd)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	result, err := BootstrapSystem(config, tm)
	if err != nil {
		t.Fatalf("BootstrapSystem() with defaults failed: %v", err)
	}

	// Verify status defaults to "pending"
	statusContent, err := os.ReadFile(result.StatusFile)
	if err != nil {
		t.Fatalf("Failed to read status file: %v", err)
	}
	statusStr := string(statusContent)
	if !strings.Contains(statusStr, "status: pending") {
		t.Errorf("Expected default status to be 'pending', content:\n%s", statusStr)
	}

	// Verify directories were created relative to current directory
	if _, err := os.Stat(result.TaskDir); os.IsNotExist(err) {
		t.Errorf("Task directory was not created with default TicketsDir")
	}
}

func TestBootstrapSystemEmptyAcceptanceCriteria(t *testing.T) {
	tempDir := t.TempDir()

	tm, err := NewEmbedTemplateManager(claude.FS)
	if err != nil {
		t.Fatalf("Failed to create template manager: %v", err)
	}

	config := BootstrapConfig{
		TaskID:             "TASK-00003",
		Title:              "Empty Criteria Task",
		Description:        "Test empty criteria",
		AcceptanceCriteria: []string{}, // Empty
		TicketsDir:         filepath.Join(tempDir, "tickets"),
		WorktreeDir:        tempDir,
	}

	result, err := BootstrapSystem(config, tm)
	if err != nil {
		t.Fatalf("BootstrapSystem() failed: %v", err)
	}

	// Verify notes.md shows default criteria
	notesContent, err := os.ReadFile(result.NotesFile)
	if err != nil {
		t.Fatalf("Failed to read notes file: %v", err)
	}
	notesStr := string(notesContent)
	if !strings.Contains(notesStr, "- [ ] Define acceptance criteria") {
		t.Errorf("Expected default acceptance criteria when list is empty.\nContent:\n%s", notesStr)
	}

	// Verify context.md shows default criteria
	contextContent, err := os.ReadFile(result.ContextFile)
	if err != nil {
		t.Fatalf("Failed to read context file: %v", err)
	}
	contextStr := string(contextContent)
	if !strings.Contains(contextStr, "- [ ] Define acceptance criteria") {
		t.Errorf("Expected default acceptance criteria in context.md when list is empty.\nContent:\n%s", contextStr)
	}
}

func TestBootstrapSystemEmptyDependencies(t *testing.T) {
	tempDir := t.TempDir()

	tm, err := NewEmbedTemplateManager(claude.FS)
	if err != nil {
		t.Fatalf("Failed to create template manager: %v", err)
	}

	config := BootstrapConfig{
		TaskID:       "TASK-00004",
		Title:        "Empty Dependencies Task",
		Description:  "Test empty dependencies",
		Dependencies: []string{}, // Empty
		TicketsDir:   filepath.Join(tempDir, "tickets"),
		WorktreeDir:  tempDir,
	}

	result, err := BootstrapSystem(config, tm)
	if err != nil {
		t.Fatalf("BootstrapSystem() failed: %v", err)
	}

	// Verify context.md shows default dependencies
	contextContent, err := os.ReadFile(result.ContextFile)
	if err != nil {
		t.Fatalf("Failed to read context file: %v", err)
	}
	contextStr := string(contextContent)
	if !strings.Contains(contextStr, "- No dependencies") {
		t.Errorf("Expected default dependencies message when list is empty.\nContent:\n%s", contextStr)
	}
}

func TestBootstrapSystemDirectoryStructure(t *testing.T) {
	tempDir := t.TempDir()

	tm, err := NewEmbedTemplateManager(claude.FS)
	if err != nil {
		t.Fatalf("Failed to create template manager: %v", err)
	}

	config := BootstrapConfig{
		TaskID:      "TASK-12345",
		Title:       "Directory Structure Test",
		Description: "Testing directory structure",
		TicketsDir:  filepath.Join(tempDir, "tickets"),
		WorktreeDir: tempDir,
	}

	result, err := BootstrapSystem(config, tm)
	if err != nil {
		t.Fatalf("BootstrapSystem() failed: %v", err)
	}

	// Verify exact directory structure
	expectedDirs := []string{
		filepath.Join(tempDir, "tickets", "TASK-12345"),
		filepath.Join(tempDir, "tickets", "TASK-12345", "sessions"),
		filepath.Join(tempDir, "tickets", "TASK-12345", "knowledge"),
		filepath.Join(tempDir, ".claude"),
		filepath.Join(tempDir, ".claude", "rules"),
	}

	for _, dir := range expectedDirs {
		info, err := os.Stat(dir)
		if os.IsNotExist(err) {
			t.Errorf("Expected directory does not exist: %s", dir)
			continue
		}
		if err != nil {
			t.Errorf("Error checking directory %s: %v", dir, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("Expected %s to be a directory", dir)
		}
	}

	// Verify exact file structure
	expectedFiles := []string{
		filepath.Join(tempDir, "tickets", "TASK-12345", "status.yaml"),
		filepath.Join(tempDir, "tickets", "TASK-12345", "context.md"),
		filepath.Join(tempDir, "tickets", "TASK-12345", "notes.md"),
		filepath.Join(tempDir, "tickets", "TASK-12345", "design.md"),
		filepath.Join(tempDir, "tickets", "TASK-12345", "knowledge", "decisions.yaml"),
		filepath.Join(tempDir, ".claude", "rules", "task-context.md"),
	}

	for _, file := range expectedFiles {
		info, err := os.Stat(file)
		if os.IsNotExist(err) {
			t.Errorf("Expected file does not exist: %s", file)
			continue
		}
		if err != nil {
			t.Errorf("Error checking file %s: %v", file, err)
			continue
		}
		if info.IsDir() {
			t.Errorf("Expected %s to be a file, not a directory", file)
		}
	}

	// Verify result paths match expected
	if result.TaskDir != filepath.Join(tempDir, "tickets", "TASK-12345") {
		t.Errorf("Unexpected TaskDir: %s", result.TaskDir)
	}
	if result.SessionsDir != filepath.Join(tempDir, "tickets", "TASK-12345", "sessions") {
		t.Errorf("Unexpected SessionsDir: %s", result.SessionsDir)
	}
	if result.KnowledgeDir != filepath.Join(tempDir, "tickets", "TASK-12345", "knowledge") {
		t.Errorf("Unexpected KnowledgeDir: %s", result.KnowledgeDir)
	}
}
