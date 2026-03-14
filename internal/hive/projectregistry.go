package hive

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/valter-silva-au/ai-dev-brain/pkg/models"
	"gopkg.in/yaml.v3"
)

// ProjectRegistry manages the registration and retrieval of projects in the Hive Mind system.
type ProjectRegistry interface {
	Register(project models.Project) error
	Get(repoPathOrName string) (*models.Project, error)
	List(filter models.ProjectFilter) ([]models.Project, error)
	Load() error
	Save() error
}

// projectRegistryStore is the internal implementation of ProjectRegistry.
type projectRegistryStore struct {
	basePath string
	projects []models.Project
}

// projectRegistryFile represents the YAML file structure for persisting project data.
type projectRegistryFile struct {
	Version  string           `yaml:"version"`
	Projects []models.Project `yaml:"projects"`
}

// NewProjectRegistry creates a new ProjectRegistry instance.
func NewProjectRegistry(basePath string) ProjectRegistry {
	return &projectRegistryStore{
		basePath: basePath,
		projects: []models.Project{},
	}
}

// Load reads the project registry from disk.
func (s *projectRegistryStore) Load() error {
	indexPath := filepath.Join(s.basePath, "projects", "index.yaml")

	data, err := os.ReadFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist yet, start with empty registry
			s.projects = []models.Project{}
			return nil
		}
		return fmt.Errorf("failed to read project registry: %w", err)
	}

	var fileData projectRegistryFile
	if err := yaml.Unmarshal(data, &fileData); err != nil {
		return fmt.Errorf("failed to parse project registry: %w", err)
	}

	s.projects = fileData.Projects
	return nil
}

// Save writes the project registry to disk atomically.
func (s *projectRegistryStore) Save() error {
	projectsDir := filepath.Join(s.basePath, "projects")
	if err := os.MkdirAll(projectsDir, 0o755); err != nil {
		return fmt.Errorf("failed to create projects directory: %w", err)
	}

	fileData := projectRegistryFile{
		Version:  "1.0",
		Projects: s.projects,
	}

	data, err := yaml.Marshal(&fileData)
	if err != nil {
		return fmt.Errorf("failed to marshal project registry: %w", err)
	}

	indexPath := filepath.Join(projectsDir, "index.yaml")
	tmpPath := indexPath + ".tmp"

	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	if err := os.Rename(tmpPath, indexPath); err != nil {
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	return nil
}

// Register adds or updates a project in the registry.
func (s *projectRegistryStore) Register(project models.Project) error {
	// Update LastUpdated timestamp
	project.LastUpdated = time.Now().UTC()

	// Check if project already exists (match by RepoPath)
	for i, p := range s.projects {
		if p.RepoPath == project.RepoPath {
			// Update existing project
			s.projects[i] = project
			return nil
		}
	}

	// Add new project
	s.projects = append(s.projects, project)
	return nil
}

// Get retrieves a project by name or repo path.
func (s *projectRegistryStore) Get(repoPathOrName string) (*models.Project, error) {
	for i, p := range s.projects {
		if p.Name == repoPathOrName || p.RepoPath == repoPathOrName {
			return &s.projects[i], nil
		}
	}
	return nil, fmt.Errorf("project not found: %s", repoPathOrName)
}

// List returns projects matching the given filter criteria.
func (s *projectRegistryStore) List(filter models.ProjectFilter) ([]models.Project, error) {
	var result []models.Project

	for _, p := range s.projects {
		if matchesFilter(p, filter) {
			result = append(result, p)
		}
	}

	return result, nil
}

// matchesFilter checks if a project matches the given filter criteria.
func matchesFilter(project models.Project, filter models.ProjectFilter) bool {
	// If Status is specified, it must match
	if filter.Status != "" && project.Status != filter.Status {
		return false
	}

	// If Tags are specified, at least one must match
	if len(filter.Tags) > 0 {
		if !hasAnyMatch(project.Tags, filter.Tags) {
			return false
		}
	}

	// If TechStack is specified, at least one must match
	if len(filter.TechStack) > 0 {
		if !hasAnyMatch(project.TechStack, filter.TechStack) {
			return false
		}
	}

	return true
}

// hasAnyMatch checks if any element from the first slice exists in the second slice.
func hasAnyMatch(projectItems []string, filterItems []string) bool {
	for _, filterItem := range filterItems {
		for _, projectItem := range projectItems {
			if projectItem == filterItem {
				return true
			}
		}
	}
	return false
}
