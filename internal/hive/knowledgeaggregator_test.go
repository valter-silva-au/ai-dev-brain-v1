package hive

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/valter-silva-au/ai-dev-brain/pkg/models"
)

// createKnowledgeFile creates a test knowledge index YAML file in the specified directory.
func createKnowledgeFile(t *testing.T, projectPath string, entries []knowledgeEntry) {
	t.Helper()

	knowledgeDir := filepath.Join(projectPath, "docs", "knowledge")
	if err := os.MkdirAll(knowledgeDir, 0o755); err != nil {
		t.Fatalf("failed to create knowledge directory: %v", err)
	}

	// Build YAML content
	yamlContent := "version: \"1.0\"\nentries:\n"
	for _, entry := range entries {
		yamlContent += "  - id: " + entry.ID + "\n"
		yamlContent += "    type: " + entry.Type + "\n"
		yamlContent += "    topic: " + entry.Topic + "\n"
		yamlContent += "    summary: " + entry.Summary + "\n"
		yamlContent += "    detail: " + entry.Detail + "\n"
		yamlContent += "    source_task: " + entry.SourceTask + "\n"
		yamlContent += "    source_type: " + entry.SourceType + "\n"
		yamlContent += "    date: \"" + entry.Date + "\"\n"
	}

	indexPath := filepath.Join(knowledgeDir, "index.yaml")
	if err := os.WriteFile(indexPath, []byte(yamlContent), 0o644); err != nil {
		t.Fatalf("failed to write knowledge index: %v", err)
	}
}

// knowledgeEntry represents a test knowledge entry.
type knowledgeEntry struct {
	ID         string
	Type       string
	Topic      string
	Summary    string
	Detail     string
	SourceTask string
	SourceType string
	Date       string
}

func TestKnowledgeAggregator_IndexEmptyRegistry(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	registry := NewProjectRegistry(basePath)

	// Create aggregator with empty registry
	aggregator := NewKnowledgeAggregator(basePath, registry)

	// Index with no projects
	err := aggregator.Index()
	if err != nil {
		t.Fatalf("Index() error = %v", err)
	}

	// Search should return no entries
	results, err := aggregator.SearchAcrossProjects("", SearchOptions{})
	if err != nil {
		t.Fatalf("SearchAcrossProjects() error = %v", err)
	}

	if len(results) != 0 {
		t.Errorf("SearchAcrossProjects() returned %d entries, want 0", len(results))
	}
}

func TestKnowledgeAggregator_IndexSingleProject(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	registry := NewProjectRegistry(basePath)

	// Create a test project directory
	projectPath := filepath.Join(basePath, "test-project")
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}

	// Create knowledge entries
	entries := []knowledgeEntry{
		{
			ID:         "K-00001",
			Type:       "decision",
			Topic:      "architecture",
			Summary:    "Use local interfaces to avoid import cycles",
			Detail:     "detailed explanation 1",
			SourceTask: "TASK-00001",
			SourceType: "task_archive",
			Date:       "2026-02-01",
		},
		{
			ID:         "K-00002",
			Type:       "pattern",
			Topic:      "testing",
			Summary:    "Use t.TempDir() for test isolation",
			Detail:     "detailed explanation 2",
			SourceTask: "TASK-00002",
			SourceType: "task_archive",
			Date:       "2026-02-02",
		},
		{
			ID:         "K-00003",
			Type:       "gotcha",
			Topic:      "golang",
			Summary:    "Defer in loops can cause memory leaks",
			Detail:     "detailed explanation 3",
			SourceTask: "TASK-00003",
			SourceType: "task_archive",
			Date:       "2026-02-03",
		},
	}
	createKnowledgeFile(t, projectPath, entries)

	// Register the project
	project := models.Project{
		Name:     "test-project",
		RepoPath: projectPath,
		Status:   models.ProjectActive,
	}
	if err := registry.Register(project); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Create aggregator and index
	aggregator := NewKnowledgeAggregator(basePath, registry)
	err := aggregator.Index()
	if err != nil {
		t.Fatalf("Index() error = %v", err)
	}

	// Search for all entries
	results, err := aggregator.SearchAcrossProjects("", SearchOptions{})
	if err != nil {
		t.Fatalf("SearchAcrossProjects() error = %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("SearchAcrossProjects() returned %d entries, want 3", len(results))
	}

	// Verify namespaced IDs
	for i, result := range results {
		expectedID := "test-project::K-" + strings.TrimPrefix(entries[i].ID, "K-")
		if result.ID != expectedID {
			t.Errorf("Entry %d: ID = %v, want %v", i, result.ID, expectedID)
		}
		if result.Project != "test-project" {
			t.Errorf("Entry %d: Project = %v, want test-project", i, result.Project)
		}
		if result.ProjectPath != projectPath {
			t.Errorf("Entry %d: ProjectPath = %v, want %v", i, result.ProjectPath, projectPath)
		}
		if result.LocalID != entries[i].ID {
			t.Errorf("Entry %d: LocalID = %v, want %v", i, result.LocalID, entries[i].ID)
		}
	}
}

func TestKnowledgeAggregator_IndexMultipleProjects(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	registry := NewProjectRegistry(basePath)

	// Create first project
	project1Path := filepath.Join(basePath, "project-1")
	if err := os.MkdirAll(project1Path, 0o755); err != nil {
		t.Fatalf("failed to create project1 directory: %v", err)
	}
	entries1 := []knowledgeEntry{
		{
			ID:         "K-00001",
			Type:       "decision",
			Topic:      "architecture",
			Summary:    "Project 1 decision",
			Detail:     "detail 1",
			SourceTask: "TASK-00001",
			SourceType: "task_archive",
			Date:       "2026-02-01",
		},
		{
			ID:         "K-00002",
			Type:       "pattern",
			Topic:      "testing",
			Summary:    "Project 1 pattern",
			Detail:     "detail 2",
			SourceTask: "TASK-00002",
			SourceType: "task_archive",
			Date:       "2026-02-02",
		},
	}
	createKnowledgeFile(t, project1Path, entries1)

	// Create second project
	project2Path := filepath.Join(basePath, "project-2")
	if err := os.MkdirAll(project2Path, 0o755); err != nil {
		t.Fatalf("failed to create project2 directory: %v", err)
	}
	entries2 := []knowledgeEntry{
		{
			ID:         "K-00101",
			Type:       "decision",
			Topic:      "database",
			Summary:    "Project 2 decision",
			Detail:     "detail 101",
			SourceTask: "TASK-00101",
			SourceType: "task_archive",
			Date:       "2026-02-10",
		},
	}
	createKnowledgeFile(t, project2Path, entries2)

	// Register both projects
	if err := registry.Register(models.Project{Name: "project-1", RepoPath: project1Path, Status: models.ProjectActive}); err != nil {
		t.Fatalf("Register project-1 error = %v", err)
	}
	if err := registry.Register(models.Project{Name: "project-2", RepoPath: project2Path, Status: models.ProjectActive}); err != nil {
		t.Fatalf("Register project-2 error = %v", err)
	}

	// Create aggregator and index
	aggregator := NewKnowledgeAggregator(basePath, registry)
	err := aggregator.Index()
	if err != nil {
		t.Fatalf("Index() error = %v", err)
	}

	// Search for all entries
	results, err := aggregator.SearchAcrossProjects("", SearchOptions{})
	if err != nil {
		t.Fatalf("SearchAcrossProjects() error = %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("SearchAcrossProjects() returned %d entries, want 3", len(results))
	}

	// Verify entries from both projects
	project1Count := 0
	project2Count := 0
	for _, result := range results {
		if result.Project == "project-1" {
			project1Count++
		} else if result.Project == "project-2" {
			project2Count++
		}
	}

	if project1Count != 2 {
		t.Errorf("Found %d entries from project-1, want 2", project1Count)
	}
	if project2Count != 1 {
		t.Errorf("Found %d entries from project-2, want 1", project2Count)
	}
}

func TestKnowledgeAggregator_SearchByQuery(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	registry := NewProjectRegistry(basePath)

	// Create a project with knowledge
	projectPath := filepath.Join(basePath, "test-project")
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}

	entries := []knowledgeEntry{
		{
			ID:         "K-00001",
			Type:       "decision",
			Topic:      "architecture",
			Summary:    "Use microservices architecture",
			Detail:     "detailed explanation",
			SourceTask: "TASK-00001",
			SourceType: "task_archive",
			Date:       "2026-02-01",
		},
		{
			ID:         "K-00002",
			Type:       "pattern",
			Topic:      "testing",
			Summary:    "Use unit tests for all functions",
			Detail:     "detailed explanation",
			SourceTask: "TASK-00002",
			SourceType: "task_archive",
			Date:       "2026-02-02",
		},
		{
			ID:         "K-00003",
			Type:       "gotcha",
			Topic:      "golang",
			Summary:    "Avoid goroutine leaks",
			Detail:     "detailed explanation",
			SourceTask: "TASK-00003",
			SourceType: "task_archive",
			Date:       "2026-02-03",
		},
	}
	createKnowledgeFile(t, projectPath, entries)

	if err := registry.Register(models.Project{Name: "test-project", RepoPath: projectPath, Status: models.ProjectActive}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	aggregator := NewKnowledgeAggregator(basePath, registry)
	if err := aggregator.Index(); err != nil {
		t.Fatalf("Index() error = %v", err)
	}

	// Search for "microservices"
	results, err := aggregator.SearchAcrossProjects("microservices", SearchOptions{})
	if err != nil {
		t.Fatalf("SearchAcrossProjects() error = %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("SearchAcrossProjects('microservices') returned %d entries, want 1", len(results))
	}

	if !strings.Contains(results[0].Summary, "microservices") {
		t.Errorf("Result summary = %v, want to contain 'microservices'", results[0].Summary)
	}
}

func TestKnowledgeAggregator_SearchCaseInsensitive(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	registry := NewProjectRegistry(basePath)

	// Create a project with knowledge
	projectPath := filepath.Join(basePath, "test-project")
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}

	entries := []knowledgeEntry{
		{
			ID:         "K-00001",
			Type:       "decision",
			Topic:      "Architecture",
			Summary:    "Use MICROSERVICES for scalability",
			Detail:     "detailed explanation",
			SourceTask: "TASK-00001",
			SourceType: "task_archive",
			Date:       "2026-02-01",
		},
	}
	createKnowledgeFile(t, projectPath, entries)

	if err := registry.Register(models.Project{Name: "test-project", RepoPath: projectPath, Status: models.ProjectActive}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	aggregator := NewKnowledgeAggregator(basePath, registry)
	if err := aggregator.Index(); err != nil {
		t.Fatalf("Index() error = %v", err)
	}

	// Test different case combinations
	testCases := []string{
		"microservices",
		"MICROSERVICES",
		"MicroServices",
		"architecture",
		"ARCHITECTURE",
	}

	for _, query := range testCases {
		results, err := aggregator.SearchAcrossProjects(query, SearchOptions{})
		if err != nil {
			t.Fatalf("SearchAcrossProjects(%q) error = %v", query, err)
		}

		if len(results) != 1 {
			t.Errorf("SearchAcrossProjects(%q) returned %d entries, want 1", query, len(results))
		}
	}
}

func TestKnowledgeAggregator_SearchFilterByProject(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	registry := NewProjectRegistry(basePath)

	// Create two projects
	project1Path := filepath.Join(basePath, "project-1")
	if err := os.MkdirAll(project1Path, 0o755); err != nil {
		t.Fatalf("failed to create project1 directory: %v", err)
	}
	createKnowledgeFile(t, project1Path, []knowledgeEntry{
		{
			ID:         "K-00001",
			Type:       "decision",
			Topic:      "architecture",
			Summary:    "Project 1 decision",
			Detail:     "detail 1",
			SourceTask: "TASK-00001",
			SourceType: "task_archive",
			Date:       "2026-02-01",
		},
	})

	project2Path := filepath.Join(basePath, "project-2")
	if err := os.MkdirAll(project2Path, 0o755); err != nil {
		t.Fatalf("failed to create project2 directory: %v", err)
	}
	createKnowledgeFile(t, project2Path, []knowledgeEntry{
		{
			ID:         "K-00002",
			Type:       "decision",
			Topic:      "database",
			Summary:    "Project 2 decision",
			Detail:     "detail 2",
			SourceTask: "TASK-00002",
			SourceType: "task_archive",
			Date:       "2026-02-02",
		},
	})

	if err := registry.Register(models.Project{Name: "project-1", RepoPath: project1Path, Status: models.ProjectActive}); err != nil {
		t.Fatalf("Register project-1 error = %v", err)
	}
	if err := registry.Register(models.Project{Name: "project-2", RepoPath: project2Path, Status: models.ProjectActive}); err != nil {
		t.Fatalf("Register project-2 error = %v", err)
	}

	aggregator := NewKnowledgeAggregator(basePath, registry)
	if err := aggregator.Index(); err != nil {
		t.Fatalf("Index() error = %v", err)
	}

	// Search with project filter
	results, err := aggregator.SearchAcrossProjects("", SearchOptions{
		ProjectFilter: []string{"project-1"},
	})
	if err != nil {
		t.Fatalf("SearchAcrossProjects() error = %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("SearchAcrossProjects() with project filter returned %d entries, want 1", len(results))
	}

	if results[0].Project != "project-1" {
		t.Errorf("Result project = %v, want project-1", results[0].Project)
	}
}

func TestKnowledgeAggregator_SearchFilterByType(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	registry := NewProjectRegistry(basePath)

	// Create a project with mixed types
	projectPath := filepath.Join(basePath, "test-project")
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}

	entries := []knowledgeEntry{
		{
			ID:         "K-00001",
			Type:       "decision",
			Topic:      "architecture",
			Summary:    "Decision 1",
			Detail:     "detail 1",
			SourceTask: "TASK-00001",
			SourceType: "task_archive",
			Date:       "2026-02-01",
		},
		{
			ID:         "K-00002",
			Type:       "decision",
			Topic:      "database",
			Summary:    "Decision 2",
			Detail:     "detail 2",
			SourceTask: "TASK-00002",
			SourceType: "task_archive",
			Date:       "2026-02-02",
		},
		{
			ID:         "K-00003",
			Type:       "pattern",
			Topic:      "testing",
			Summary:    "Pattern 1",
			Detail:     "detail 3",
			SourceTask: "TASK-00003",
			SourceType: "task_archive",
			Date:       "2026-02-03",
		},
	}
	createKnowledgeFile(t, projectPath, entries)

	if err := registry.Register(models.Project{Name: "test-project", RepoPath: projectPath, Status: models.ProjectActive}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	aggregator := NewKnowledgeAggregator(basePath, registry)
	if err := aggregator.Index(); err != nil {
		t.Fatalf("Index() error = %v", err)
	}

	// Search with type filter for "decision"
	results, err := aggregator.SearchAcrossProjects("", SearchOptions{
		TypeFilter: []string{"decision"},
	})
	if err != nil {
		t.Fatalf("SearchAcrossProjects() error = %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("SearchAcrossProjects() with type filter returned %d entries, want 2", len(results))
	}

	// Verify all results are decisions
	for _, result := range results {
		if result.Type != "decision" {
			t.Errorf("Result type = %v, want decision", result.Type)
		}
	}
}

func TestKnowledgeAggregator_SearchWithLimit(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	registry := NewProjectRegistry(basePath)

	// Create a project with multiple entries
	projectPath := filepath.Join(basePath, "test-project")
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}

	entries := []knowledgeEntry{
		{
			ID:         "K-00001",
			Type:       "decision",
			Topic:      "architecture",
			Summary:    "Decision 1",
			Detail:     "detail 1",
			SourceTask: "TASK-00001",
			SourceType: "task_archive",
			Date:       "2026-02-01",
		},
		{
			ID:         "K-00002",
			Type:       "decision",
			Topic:      "database",
			Summary:    "Decision 2",
			Detail:     "detail 2",
			SourceTask: "TASK-00002",
			SourceType: "task_archive",
			Date:       "2026-02-02",
		},
		{
			ID:         "K-00003",
			Type:       "pattern",
			Topic:      "testing",
			Summary:    "Pattern 1",
			Detail:     "detail 3",
			SourceTask: "TASK-00003",
			SourceType: "task_archive",
			Date:       "2026-02-03",
		},
		{
			ID:         "K-00004",
			Type:       "gotcha",
			Topic:      "golang",
			Summary:    "Gotcha 1",
			Detail:     "detail 4",
			SourceTask: "TASK-00004",
			SourceType: "task_archive",
			Date:       "2026-02-04",
		},
	}
	createKnowledgeFile(t, projectPath, entries)

	if err := registry.Register(models.Project{Name: "test-project", RepoPath: projectPath, Status: models.ProjectActive}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	aggregator := NewKnowledgeAggregator(basePath, registry)
	if err := aggregator.Index(); err != nil {
		t.Fatalf("Index() error = %v", err)
	}

	// Search with limit=2
	results, err := aggregator.SearchAcrossProjects("", SearchOptions{
		Limit: 2,
	})
	if err != nil {
		t.Fatalf("SearchAcrossProjects() error = %v", err)
	}

	if len(results) != 2 {
		t.Errorf("SearchAcrossProjects() with limit=2 returned %d entries, want 2", len(results))
	}
}

func TestKnowledgeAggregator_GetDecisionsForTopic(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	registry := NewProjectRegistry(basePath)

	// Create a project with decisions
	projectPath := filepath.Join(basePath, "test-project")
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}

	entries := []knowledgeEntry{
		{
			ID:         "K-00001",
			Type:       "decision",
			Topic:      "architecture-microservices",
			Summary:    "Use microservices",
			Detail:     "detail 1",
			SourceTask: "TASK-00001",
			SourceType: "task_archive",
			Date:       "2026-02-01",
		},
		{
			ID:         "K-00002",
			Type:       "decision",
			Topic:      "architecture-monolith",
			Summary:    "Avoid monoliths",
			Detail:     "detail 2",
			SourceTask: "TASK-00002",
			SourceType: "task_archive",
			Date:       "2026-02-02",
		},
		{
			ID:         "K-00003",
			Type:       "pattern",
			Topic:      "architecture",
			Summary:    "Pattern 1",
			Detail:     "detail 3",
			SourceTask: "TASK-00003",
			SourceType: "task_archive",
			Date:       "2026-02-03",
		},
		{
			ID:         "K-00004",
			Type:       "decision",
			Topic:      "database",
			Summary:    "Use PostgreSQL",
			Detail:     "detail 4",
			SourceTask: "TASK-00004",
			SourceType: "task_archive",
			Date:       "2026-02-04",
		},
	}
	createKnowledgeFile(t, projectPath, entries)

	if err := registry.Register(models.Project{Name: "test-project", RepoPath: projectPath, Status: models.ProjectActive}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	aggregator := NewKnowledgeAggregator(basePath, registry)
	if err := aggregator.Index(); err != nil {
		t.Fatalf("Index() error = %v", err)
	}

	// Get decisions for "architecture" topic
	results, err := aggregator.GetDecisionsForTopic("architecture")
	if err != nil {
		t.Fatalf("GetDecisionsForTopic() error = %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("GetDecisionsForTopic('architecture') returned %d entries, want 2", len(results))
	}

	// Verify all results are decisions with architecture in topic
	for _, result := range results {
		if result.Type != "decision" {
			t.Errorf("Result type = %v, want decision", result.Type)
		}
		if !strings.Contains(strings.ToLower(result.Topic), "architecture") {
			t.Errorf("Result topic = %v, want to contain 'architecture'", result.Topic)
		}
	}
}

func TestKnowledgeAggregator_SaveAndLoad(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	registry := NewProjectRegistry(basePath)

	// Create a project with knowledge
	projectPath := filepath.Join(basePath, "test-project")
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}

	entries := []knowledgeEntry{
		{
			ID:         "K-00001",
			Type:       "decision",
			Topic:      "architecture",
			Summary:    "Decision 1",
			Detail:     "detail 1",
			SourceTask: "TASK-00001",
			SourceType: "task_archive",
			Date:       "2026-02-01",
		},
		{
			ID:         "K-00002",
			Type:       "pattern",
			Topic:      "testing",
			Summary:    "Pattern 1",
			Detail:     "detail 2",
			SourceTask: "TASK-00002",
			SourceType: "task_archive",
			Date:       "2026-02-02",
		},
	}
	createKnowledgeFile(t, projectPath, entries)

	if err := registry.Register(models.Project{Name: "test-project", RepoPath: projectPath, Status: models.ProjectActive}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Create first aggregator, index and save
	aggregator1 := NewKnowledgeAggregator(basePath, registry)
	if err := aggregator1.Index(); err != nil {
		t.Fatalf("Index() error = %v", err)
	}

	if err := aggregator1.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Create second aggregator and load
	aggregator2 := NewKnowledgeAggregator(basePath, registry)
	if err := aggregator2.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Search to verify loaded entries
	results, err := aggregator2.SearchAcrossProjects("", SearchOptions{})
	if err != nil {
		t.Fatalf("SearchAcrossProjects() error = %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("SearchAcrossProjects() after Load() returned %d entries, want 2", len(results))
	}

	// Verify entry contents
	for i, result := range results {
		if result.Project != "test-project" {
			t.Errorf("Entry %d: Project = %v, want test-project", i, result.Project)
		}
		if result.Type != entries[i].Type {
			t.Errorf("Entry %d: Type = %v, want %v", i, result.Type, entries[i].Type)
		}
		if result.Summary != entries[i].Summary {
			t.Errorf("Entry %d: Summary = %v, want %v", i, result.Summary, entries[i].Summary)
		}
	}
}

func TestKnowledgeAggregator_ProjectWithNoKnowledge(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	registry := NewProjectRegistry(basePath)

	// Create a project directory without knowledge
	projectPath := filepath.Join(basePath, "empty-project")
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}

	// Register the project without creating knowledge files
	if err := registry.Register(models.Project{Name: "empty-project", RepoPath: projectPath, Status: models.ProjectActive}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Create aggregator and index (should skip gracefully)
	aggregator := NewKnowledgeAggregator(basePath, registry)
	err := aggregator.Index()
	if err != nil {
		t.Fatalf("Index() error = %v, want nil (should skip project without knowledge)", err)
	}

	// Search should return no entries
	results, err := aggregator.SearchAcrossProjects("", SearchOptions{})
	if err != nil {
		t.Fatalf("SearchAcrossProjects() error = %v", err)
	}

	if len(results) != 0 {
		t.Errorf("SearchAcrossProjects() returned %d entries, want 0", len(results))
	}
}

func TestKnowledgeAggregator_SearchWithSinceFilter(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	registry := NewProjectRegistry(basePath)

	// Create a project with entries at different dates
	projectPath := filepath.Join(basePath, "test-project")
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}

	entries := []knowledgeEntry{
		{
			ID:         "K-00001",
			Type:       "decision",
			Topic:      "architecture",
			Summary:    "Old decision",
			Detail:     "detail 1",
			SourceTask: "TASK-00001",
			SourceType: "task_archive",
			Date:       "2026-01-01",
		},
		{
			ID:         "K-00002",
			Type:       "decision",
			Topic:      "database",
			Summary:    "Recent decision",
			Detail:     "detail 2",
			SourceTask: "TASK-00002",
			SourceType: "task_archive",
			Date:       "2026-03-01",
		},
	}
	createKnowledgeFile(t, projectPath, entries)

	if err := registry.Register(models.Project{Name: "test-project", RepoPath: projectPath, Status: models.ProjectActive}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	aggregator := NewKnowledgeAggregator(basePath, registry)
	if err := aggregator.Index(); err != nil {
		t.Fatalf("Index() error = %v", err)
	}

	// Search with Since filter (after 2026-02-01)
	since, _ := time.Parse("2006-01-02", "2026-02-01")
	results, err := aggregator.SearchAcrossProjects("", SearchOptions{
		Since: since,
	})
	if err != nil {
		t.Fatalf("SearchAcrossProjects() error = %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("SearchAcrossProjects() with Since filter returned %d entries, want 1", len(results))
	}

	if results[0].Summary != "Recent decision" {
		t.Errorf("Result summary = %v, want 'Recent decision'", results[0].Summary)
	}
}
