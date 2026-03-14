package hive

import (
	"testing"
	"time"

	"github.com/valter-silva-au/ai-dev-brain/pkg/models"
)

func TestProjectRegistry_RegisterAndGet(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	registry := NewProjectRegistry(basePath)

	// Create a test project
	project := models.Project{
		Name:                "test-project",
		RepoPath:            "/path/to/repo",
		Purpose:             "Testing the registry",
		TechStack:           []string{"go", "yaml"},
		Status:              models.ProjectActive,
		Tags:                []string{"test", "unit"},
		DefaultAI:           "claude",
		RelatedProjects:     []string{"related-project"},
		KnowledgeEntryCount: 5,
		DecisionCount:       3,
		ActiveTaskCount:     2,
	}

	// Register the project
	err := registry.Register(project)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Get by name
	retrieved, err := registry.Get("test-project")
	if err != nil {
		t.Fatalf("Get() by name error = %v", err)
	}
	if retrieved == nil {
		t.Fatal("Get() by name returned nil")
	}

	// Verify all fields
	if retrieved.Name != project.Name {
		t.Errorf("Name = %v, want %v", retrieved.Name, project.Name)
	}
	if retrieved.RepoPath != project.RepoPath {
		t.Errorf("RepoPath = %v, want %v", retrieved.RepoPath, project.RepoPath)
	}
	if retrieved.Purpose != project.Purpose {
		t.Errorf("Purpose = %v, want %v", retrieved.Purpose, project.Purpose)
	}
	if retrieved.Status != project.Status {
		t.Errorf("Status = %v, want %v", retrieved.Status, project.Status)
	}
	if len(retrieved.TechStack) != len(project.TechStack) {
		t.Errorf("TechStack length = %v, want %v", len(retrieved.TechStack), len(project.TechStack))
	}
	if len(retrieved.Tags) != len(project.Tags) {
		t.Errorf("Tags length = %v, want %v", len(retrieved.Tags), len(project.Tags))
	}

	// Get by repo path
	retrievedByPath, err := registry.Get("/path/to/repo")
	if err != nil {
		t.Fatalf("Get() by repo path error = %v", err)
	}
	if retrievedByPath == nil {
		t.Fatal("Get() by repo path returned nil")
	}
	if retrievedByPath.Name != project.Name {
		t.Errorf("Get by repo path: Name = %v, want %v", retrievedByPath.Name, project.Name)
	}
}

func TestProjectRegistry_RegisterUpdate(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	registry := NewProjectRegistry(basePath)

	// Register initial project
	project1 := models.Project{
		Name:     "test-project",
		RepoPath: "/path/to/repo",
		Purpose:  "Initial purpose",
		Status:   models.ProjectActive,
		Tags:     []string{"tag1"},
	}

	err := registry.Register(project1)
	if err != nil {
		t.Fatalf("Register() initial error = %v", err)
	}

	// Register same project with updated fields
	project2 := models.Project{
		Name:     "test-project-updated",
		RepoPath: "/path/to/repo", // Same repo path
		Purpose:  "Updated purpose",
		Status:   models.ProjectPaused,
		Tags:     []string{"tag2"},
	}

	err = registry.Register(project2)
	if err != nil {
		t.Fatalf("Register() update error = %v", err)
	}

	// List all projects to verify no duplicates
	allProjects, err := registry.List(models.ProjectFilter{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(allProjects) != 1 {
		t.Errorf("List() returned %d projects, want 1 (should update, not duplicate)", len(allProjects))
	}

	// Verify updated values
	retrieved, err := registry.Get("/path/to/repo")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if retrieved.Name != "test-project-updated" {
		t.Errorf("Name = %v, want test-project-updated", retrieved.Name)
	}
	if retrieved.Purpose != "Updated purpose" {
		t.Errorf("Purpose = %v, want 'Updated purpose'", retrieved.Purpose)
	}
	if retrieved.Status != models.ProjectPaused {
		t.Errorf("Status = %v, want %v", retrieved.Status, models.ProjectPaused)
	}
}

func TestProjectRegistry_ListAll(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	registry := NewProjectRegistry(basePath)

	// Register 3 projects
	projects := []models.Project{
		{
			Name:     "project-1",
			RepoPath: "/path/to/repo1",
			Purpose:  "Purpose 1",
			Status:   models.ProjectActive,
		},
		{
			Name:     "project-2",
			RepoPath: "/path/to/repo2",
			Purpose:  "Purpose 2",
			Status:   models.ProjectActive,
		},
		{
			Name:     "project-3",
			RepoPath: "/path/to/repo3",
			Purpose:  "Purpose 3",
			Status:   models.ProjectArchived,
		},
	}

	for _, p := range projects {
		if err := registry.Register(p); err != nil {
			t.Fatalf("Register() error = %v", err)
		}
	}

	// List with empty filter
	allProjects, err := registry.List(models.ProjectFilter{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(allProjects) != 3 {
		t.Errorf("List() returned %d projects, want 3", len(allProjects))
	}

	// Verify all project names are present
	names := make(map[string]bool)
	for _, p := range allProjects {
		names[p.Name] = true
	}

	for _, expected := range []string{"project-1", "project-2", "project-3"} {
		if !names[expected] {
			t.Errorf("List() missing project %s", expected)
		}
	}
}

func TestProjectRegistry_ListFilterByStatus(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	registry := NewProjectRegistry(basePath)

	// Register 3 projects with different statuses
	projects := []models.Project{
		{
			Name:     "active-project-1",
			RepoPath: "/path/to/active1",
			Status:   models.ProjectActive,
		},
		{
			Name:     "active-project-2",
			RepoPath: "/path/to/active2",
			Status:   models.ProjectActive,
		},
		{
			Name:     "archived-project",
			RepoPath: "/path/to/archived",
			Status:   models.ProjectArchived,
		},
	}

	for _, p := range projects {
		if err := registry.Register(p); err != nil {
			t.Fatalf("Register() error = %v", err)
		}
	}

	// Filter by active status
	filter := models.ProjectFilter{
		Status: models.ProjectActive,
	}

	activeProjects, err := registry.List(filter)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(activeProjects) != 2 {
		t.Errorf("List() returned %d active projects, want 2", len(activeProjects))
	}

	// Verify all returned projects are active
	for _, p := range activeProjects {
		if p.Status != models.ProjectActive {
			t.Errorf("List() returned project with status %v, want %v", p.Status, models.ProjectActive)
		}
	}
}

func TestProjectRegistry_ListFilterByTags(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	registry := NewProjectRegistry(basePath)

	// Register projects with different tags
	projects := []models.Project{
		{
			Name:     "project-1",
			RepoPath: "/path/to/repo1",
			Tags:     []string{"backend", "api"},
			Status:   models.ProjectActive,
		},
		{
			Name:     "project-2",
			RepoPath: "/path/to/repo2",
			Tags:     []string{"frontend", "ui"},
			Status:   models.ProjectActive,
		},
		{
			Name:     "project-3",
			RepoPath: "/path/to/repo3",
			Tags:     []string{"backend", "database"},
			Status:   models.ProjectActive,
		},
	}

	for _, p := range projects {
		if err := registry.Register(p); err != nil {
			t.Fatalf("Register() error = %v", err)
		}
	}

	// Filter by "backend" tag
	filter := models.ProjectFilter{
		Tags: []string{"backend"},
	}

	backendProjects, err := registry.List(filter)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(backendProjects) != 2 {
		t.Errorf("List() returned %d projects with 'backend' tag, want 2", len(backendProjects))
	}

	// Verify returned projects have the backend tag
	for _, p := range backendProjects {
		hasBackendTag := false
		for _, tag := range p.Tags {
			if tag == "backend" {
				hasBackendTag = true
				break
			}
		}
		if !hasBackendTag {
			t.Errorf("List() returned project %s without 'backend' tag", p.Name)
		}
	}
}

func TestProjectRegistry_ListFilterByTechStack(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	registry := NewProjectRegistry(basePath)

	// Register projects with different tech stacks
	projects := []models.Project{
		{
			Name:      "go-project-1",
			RepoPath:  "/path/to/go1",
			TechStack: []string{"go", "postgresql"},
			Status:    models.ProjectActive,
		},
		{
			Name:      "python-project",
			RepoPath:  "/path/to/python",
			TechStack: []string{"python", "django"},
			Status:    models.ProjectActive,
		},
		{
			Name:      "go-project-2",
			RepoPath:  "/path/to/go2",
			TechStack: []string{"go", "redis"},
			Status:    models.ProjectActive,
		},
	}

	for _, p := range projects {
		if err := registry.Register(p); err != nil {
			t.Fatalf("Register() error = %v", err)
		}
	}

	// Filter by "go" tech stack
	filter := models.ProjectFilter{
		TechStack: []string{"go"},
	}

	goProjects, err := registry.List(filter)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(goProjects) != 2 {
		t.Errorf("List() returned %d projects with 'go' tech stack, want 2", len(goProjects))
	}

	// Verify returned projects have go in their tech stack
	for _, p := range goProjects {
		hasGo := false
		for _, tech := range p.TechStack {
			if tech == "go" {
				hasGo = true
				break
			}
		}
		if !hasGo {
			t.Errorf("List() returned project %s without 'go' in tech stack", p.Name)
		}
	}
}

func TestProjectRegistry_SaveAndLoad(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()

	// Create first registry and register projects
	registry1 := NewProjectRegistry(basePath)

	projects := []models.Project{
		{
			Name:      "project-1",
			RepoPath:  "/path/to/repo1",
			Purpose:   "Purpose 1",
			TechStack: []string{"go"},
			Status:    models.ProjectActive,
			Tags:      []string{"tag1"},
		},
		{
			Name:      "project-2",
			RepoPath:  "/path/to/repo2",
			Purpose:   "Purpose 2",
			TechStack: []string{"python"},
			Status:    models.ProjectArchived,
			Tags:      []string{"tag2"},
		},
	}

	for _, p := range projects {
		if err := registry1.Register(p); err != nil {
			t.Fatalf("Register() error = %v", err)
		}
	}

	// Save to disk
	if err := registry1.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Create new registry at same path and load
	registry2 := NewProjectRegistry(basePath)
	if err := registry2.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// List all projects from the loaded registry
	loadedProjects, err := registry2.List(models.ProjectFilter{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(loadedProjects) != len(projects) {
		t.Errorf("Load() returned %d projects, want %d", len(loadedProjects), len(projects))
	}

	// Verify projects match (compare by name and key fields)
	projectMap := make(map[string]models.Project)
	for _, p := range loadedProjects {
		projectMap[p.Name] = p
	}

	for _, original := range projects {
		loaded, exists := projectMap[original.Name]
		if !exists {
			t.Errorf("Load() missing project %s", original.Name)
			continue
		}

		if loaded.RepoPath != original.RepoPath {
			t.Errorf("Project %s: RepoPath = %v, want %v", original.Name, loaded.RepoPath, original.RepoPath)
		}
		if loaded.Purpose != original.Purpose {
			t.Errorf("Project %s: Purpose = %v, want %v", original.Name, loaded.Purpose, original.Purpose)
		}
		if loaded.Status != original.Status {
			t.Errorf("Project %s: Status = %v, want %v", original.Name, loaded.Status, original.Status)
		}
	}
}

func TestProjectRegistry_GetNotFound(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	registry := NewProjectRegistry(basePath)

	// Try to get a nonexistent project
	project, err := registry.Get("nonexistent-project")

	// Based on the implementation, Get returns an error when not found
	if err == nil {
		t.Error("Get() for nonexistent project should return error, got nil")
	}

	if project != nil {
		t.Errorf("Get() for nonexistent project returned %v, want nil", project)
	}
}

func TestProjectRegistry_EmptyRegistry(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	registry := NewProjectRegistry(basePath)

	// List on empty registry
	projects, err := registry.List(models.ProjectFilter{})
	if err != nil {
		t.Fatalf("List() on empty registry error = %v", err)
	}

	if projects == nil {
		t.Error("List() on empty registry returned nil, want empty slice")
	}

	if len(projects) != 0 {
		t.Errorf("List() on empty registry returned %d projects, want 0", len(projects))
	}
}

func TestProjectRegistry_LastUpdatedTimestamp(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	registry := NewProjectRegistry(basePath)

	beforeRegister := time.Now().UTC()

	project := models.Project{
		Name:     "test-project",
		RepoPath: "/path/to/repo",
		Status:   models.ProjectActive,
	}

	// Register the project
	if err := registry.Register(project); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	afterRegister := time.Now().UTC()

	// Get the project and check LastUpdated was set
	retrieved, err := registry.Get("test-project")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if retrieved.LastUpdated.IsZero() {
		t.Error("LastUpdated was not set")
	}

	if retrieved.LastUpdated.Before(beforeRegister) || retrieved.LastUpdated.After(afterRegister) {
		t.Errorf("LastUpdated = %v, want between %v and %v", retrieved.LastUpdated, beforeRegister, afterRegister)
	}
}
