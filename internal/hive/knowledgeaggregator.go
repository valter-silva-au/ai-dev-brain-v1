package hive

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/valter-silva-au/ai-dev-brain/pkg/models"
	"gopkg.in/yaml.v3"
)

// SearchOptions defines filter criteria for knowledge search.
type SearchOptions struct {
	ProjectFilter []string  `yaml:"project_filter,omitempty"`
	TypeFilter    []string  `yaml:"type_filter,omitempty"`
	Since         time.Time `yaml:"since,omitempty"`
	Limit         int       `yaml:"limit,omitempty"`
}

// KnowledgeAggregator aggregates and searches knowledge across multiple projects.
type KnowledgeAggregator interface {
	SearchAcrossProjects(query string, opts SearchOptions) ([]models.HiveKnowledgeResult, error)
	GetDecisionsForTopic(topic string) ([]models.HiveKnowledgeResult, error)
	Index() error
	Load() error
	Save() error
}

// knowledgeAggregatorStore is the internal implementation of KnowledgeAggregator.
type knowledgeAggregatorStore struct {
	basePath   string
	projectReg ProjectRegistry
	entries    []models.HiveKnowledgeResult
	indexedAt  time.Time
}

// knowledgeAggregatorFile represents the YAML file structure for persisting aggregated knowledge.
type knowledgeAggregatorFile struct {
	Version   string                       `yaml:"version"`
	IndexedAt string                       `yaml:"indexed_at"`
	Entries   []models.HiveKnowledgeResult `yaml:"entries"`
}

// knowledgeIndex represents the structure of a project's knowledge index.
type knowledgeIndex struct {
	Version string `yaml:"version"`
	Entries []struct {
		ID         string `yaml:"id"`
		Type       string `yaml:"type"`
		Topic      string `yaml:"topic"`
		Summary    string `yaml:"summary"`
		Detail     string `yaml:"detail"`
		SourceTask string `yaml:"source_task"`
		SourceType string `yaml:"source_type"`
		Date       string `yaml:"date"`
	} `yaml:"entries"`
}

// NewKnowledgeAggregator creates a new KnowledgeAggregator instance.
func NewKnowledgeAggregator(basePath string, projectReg ProjectRegistry) KnowledgeAggregator {
	return &knowledgeAggregatorStore{
		basePath:   basePath,
		projectReg: projectReg,
		entries:    []models.HiveKnowledgeResult{},
	}
}

// Index walks all projects and aggregates their knowledge entries.
func (s *knowledgeAggregatorStore) Index() error {
	// Get all projects from registry
	projects, err := s.projectReg.List(models.ProjectFilter{})
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	s.entries = []models.HiveKnowledgeResult{}

	// Walk through each project
	for _, project := range projects {
		knowledgePath := filepath.Join(project.RepoPath, "docs", "knowledge", "index.yaml")

		// Check if knowledge file exists
		data, err := os.ReadFile(knowledgePath)
		if err != nil {
			if os.IsNotExist(err) {
				// No knowledge file for this project, skip
				continue
			}
			return fmt.Errorf("failed to read knowledge file for project %s: %w", project.Name, err)
		}

		// Parse knowledge index
		var kIndex knowledgeIndex
		if err := yaml.Unmarshal(data, &kIndex); err != nil {
			return fmt.Errorf("failed to parse knowledge file for project %s: %w", project.Name, err)
		}

		// Convert each entry to HiveKnowledgeResult
		for _, entry := range kIndex.Entries {
			// Parse date
			var entryDate time.Time
			if entry.Date != "" {
				entryDate, err = time.Parse("2006-01-02", entry.Date)
				if err != nil {
					// Try RFC3339 format as fallback
					entryDate, err = time.Parse(time.RFC3339, entry.Date)
					if err != nil {
						// If parsing fails, use zero time
						entryDate = time.Time{}
					}
				}
			}

			// Namespace the ID with project name
			namespacedID := fmt.Sprintf("%s::%s", project.Name, entry.ID)

			result := models.HiveKnowledgeResult{
				ID:          namespacedID,
				Project:     project.Name,
				ProjectPath: project.RepoPath,
				LocalID:     entry.ID,
				Type:        entry.Type,
				Topic:       entry.Topic,
				Summary:     entry.Summary,
				Detail:      entry.Detail,
				SourceTask:  entry.SourceTask,
				Date:        entryDate,
				Tags:        []string{}, // Initialize empty tags
			}

			s.entries = append(s.entries, result)
		}
	}

	s.indexedAt = time.Now().UTC()
	return nil
}

// SearchAcrossProjects searches knowledge entries by query string with filtering options.
func (s *knowledgeAggregatorStore) SearchAcrossProjects(query string, opts SearchOptions) ([]models.HiveKnowledgeResult, error) {
	results := []models.HiveKnowledgeResult{}
	queryLower := strings.ToLower(query)

	for _, entry := range s.entries {
		// Apply project filter
		if len(opts.ProjectFilter) > 0 && !contains(opts.ProjectFilter, entry.Project) {
			continue
		}

		// Apply type filter
		if len(opts.TypeFilter) > 0 && !contains(opts.TypeFilter, entry.Type) {
			continue
		}

		// Apply date filter
		if !opts.Since.IsZero() && entry.Date.Before(opts.Since) {
			continue
		}

		// Apply query match (case-insensitive substring match on Summary and Topic)
		summaryLower := strings.ToLower(entry.Summary)
		topicLower := strings.ToLower(entry.Topic)

		if queryLower == "" || strings.Contains(summaryLower, queryLower) || strings.Contains(topicLower, queryLower) {
			results = append(results, entry)

			// Apply limit
			if opts.Limit > 0 && len(results) >= opts.Limit {
				break
			}
		}
	}

	return results, nil
}

// GetDecisionsForTopic returns all decision entries matching the given topic.
func (s *knowledgeAggregatorStore) GetDecisionsForTopic(topic string) ([]models.HiveKnowledgeResult, error) {
	results := []models.HiveKnowledgeResult{}
	topicLower := strings.ToLower(topic)

	for _, entry := range s.entries {
		// Filter by type == "decision"
		if entry.Type != "decision" {
			continue
		}

		// Check if topic contains the query (case-insensitive)
		entryTopicLower := strings.ToLower(entry.Topic)
		if strings.Contains(entryTopicLower, topicLower) {
			results = append(results, entry)
		}
	}

	return results, nil
}

// Load reads the aggregated knowledge index from disk.
func (s *knowledgeAggregatorStore) Load() error {
	indexPath := filepath.Join(s.basePath, "hive-mind", "knowledge", "index.yaml")

	data, err := os.ReadFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist yet, start with empty entries
			s.entries = []models.HiveKnowledgeResult{}
			s.indexedAt = time.Time{}
			return nil
		}
		return fmt.Errorf("failed to read knowledge index: %w", err)
	}

	var fileData knowledgeAggregatorFile
	if err := yaml.Unmarshal(data, &fileData); err != nil {
		return fmt.Errorf("failed to parse knowledge index: %w", err)
	}

	// Parse indexed_at timestamp
	if fileData.IndexedAt != "" {
		indexedAt, err := time.Parse(time.RFC3339, fileData.IndexedAt)
		if err != nil {
			return fmt.Errorf("failed to parse indexed_at timestamp: %w", err)
		}
		s.indexedAt = indexedAt
	}

	s.entries = fileData.Entries
	return nil
}

// Save writes the aggregated knowledge index to disk atomically.
func (s *knowledgeAggregatorStore) Save() error {
	knowledgeDir := filepath.Join(s.basePath, "hive-mind", "knowledge")
	if err := os.MkdirAll(knowledgeDir, 0o755); err != nil {
		return fmt.Errorf("failed to create knowledge directory: %w", err)
	}

	fileData := knowledgeAggregatorFile{
		Version:   "1.0",
		IndexedAt: s.indexedAt.Format(time.RFC3339),
		Entries:   s.entries,
	}

	data, err := yaml.Marshal(&fileData)
	if err != nil {
		return fmt.Errorf("failed to marshal knowledge index: %w", err)
	}

	indexPath := filepath.Join(knowledgeDir, "index.yaml")
	tmpPath := indexPath + ".tmp"

	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	if err := os.Rename(tmpPath, indexPath); err != nil {
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	return nil
}

// contains checks if a string slice contains a given string.
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
