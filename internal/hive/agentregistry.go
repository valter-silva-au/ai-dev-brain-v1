package hive

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/valter-silva-au/ai-dev-brain/pkg/models"
	"gopkg.in/yaml.v3"
)

// AgentRegistry manages the registration and retrieval of agents in the Hive Mind system.
type AgentRegistry interface {
	Register(agent models.Agent) error
	Get(name string) (*models.Agent, error)
	List(filter models.AgentFilter) ([]models.Agent, error)
	DiscoverOpenClaw(openclawPath string) ([]models.Agent, error)
	Load() error
	Save() error
}

// agentRegistryStore is the internal implementation of AgentRegistry.
type agentRegistryStore struct {
	basePath string
	agents   []models.Agent
}

// agentRegistryFile represents the YAML file structure for persisting agent data.
type agentRegistryFile struct {
	Version string         `yaml:"version"`
	Agents  []models.Agent `yaml:"agents"`
}

// openclawConfig represents the structure of openclaw.json file.
type openclawConfig struct {
	Agents struct {
		Defaults struct {
			Model struct {
				Primary string `json:"primary"`
			} `json:"model"`
		} `json:"defaults"`
		List []struct {
			ID        string `json:"id"`
			Name      string `json:"name"`
			Workspace string `json:"workspace"`
			AgentDir  string `json:"agentDir"`
		} `json:"list"`
	} `json:"agents"`
}

// NewAgentRegistry creates a new AgentRegistry instance.
func NewAgentRegistry(basePath string) AgentRegistry {
	return &agentRegistryStore{
		basePath: basePath,
		agents:   []models.Agent{},
	}
}

// Load reads the agent registry from disk.
func (s *agentRegistryStore) Load() error {
	indexPath := filepath.Join(s.basePath, "agents", "index.yaml")

	data, err := os.ReadFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist yet, start with empty registry
			s.agents = []models.Agent{}
			return nil
		}
		return fmt.Errorf("failed to read agent registry: %w", err)
	}

	var fileData agentRegistryFile
	if err := yaml.Unmarshal(data, &fileData); err != nil {
		return fmt.Errorf("failed to parse agent registry: %w", err)
	}

	s.agents = fileData.Agents
	return nil
}

// Save writes the agent registry to disk atomically.
func (s *agentRegistryStore) Save() error {
	agentsDir := filepath.Join(s.basePath, "agents")
	if err := os.MkdirAll(agentsDir, 0o755); err != nil {
		return fmt.Errorf("failed to create agents directory: %w", err)
	}

	fileData := agentRegistryFile{
		Version: "1.0",
		Agents:  s.agents,
	}

	data, err := yaml.Marshal(&fileData)
	if err != nil {
		return fmt.Errorf("failed to marshal agent registry: %w", err)
	}

	indexPath := filepath.Join(agentsDir, "index.yaml")
	tmpPath := indexPath + ".tmp"

	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	if err := os.Rename(tmpPath, indexPath); err != nil {
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	return nil
}

// Register adds or updates an agent in the registry.
func (s *agentRegistryStore) Register(agent models.Agent) error {
	// Update LastSeen timestamp
	agent.LastSeen = time.Now().UTC()

	// Check if agent already exists (match by Name)
	for i, a := range s.agents {
		if a.Name == agent.Name {
			// Update existing agent
			s.agents[i] = agent
			return nil
		}
	}

	// Add new agent
	s.agents = append(s.agents, agent)
	return nil
}

// Get retrieves an agent by name (case-insensitive).
func (s *agentRegistryStore) Get(name string) (*models.Agent, error) {
	nameLower := strings.ToLower(name)
	for i, a := range s.agents {
		if strings.ToLower(a.Name) == nameLower {
			return &s.agents[i], nil
		}
	}
	return nil, fmt.Errorf("agent not found: %s", name)
}

// List returns agents matching the given filter criteria.
func (s *agentRegistryStore) List(filter models.AgentFilter) ([]models.Agent, error) {
	result := []models.Agent{}

	for _, a := range s.agents {
		if matchesAgentFilter(a, filter) {
			result = append(result, a)
		}
	}

	return result, nil
}

// matchesAgentFilter checks if an agent matches the given filter criteria.
func matchesAgentFilter(agent models.Agent, filter models.AgentFilter) bool {
	// If Type is specified, it must match
	if filter.Type != "" && agent.Type != filter.Type {
		return false
	}

	// If Status is specified, it must match
	if filter.Status != "" && agent.Status != filter.Status {
		return false
	}

	// If Capabilities are specified, at least one must match
	if len(filter.Capabilities) > 0 {
		if !hasAnyMatch(agent.Capabilities, filter.Capabilities) {
			return false
		}
	}

	return true
}

// DiscoverOpenClaw reads openclaw.json and discovers agents without auto-registering them.
func (s *agentRegistryStore) DiscoverOpenClaw(openclawPath string) ([]models.Agent, error) {
	configPath := filepath.Join(openclawPath, "openclaw.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read openclaw.json: %w", err)
	}

	var config openclawConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse openclaw.json: %w", err)
	}

	var discoveredAgents []models.Agent
	defaultModel := config.Agents.Defaults.Model.Primary
	if defaultModel == "" {
		defaultModel = "claude-sonnet-4" // fallback default
	}

	for _, agentEntry := range config.Agents.List {
		agent := models.Agent{
			Name:          agentEntry.Name,
			Type:          models.AgentOpenClaw,
			Model:         defaultModel,
			Status:        models.AgentIdle,
			WorkspacePath: agentEntry.Workspace,
		}

		// Try to read SOUL.md for capabilities
		if agentEntry.AgentDir != "" {
			soulPath := filepath.Join(agentEntry.AgentDir, "SOUL.md")
			if capabilities := extractCapabilitiesFromSOUL(soulPath); len(capabilities) > 0 {
				agent.Capabilities = capabilities
			}
		}

		discoveredAgents = append(discoveredAgents, agent)
	}

	return discoveredAgents, nil
}

// extractCapabilitiesFromSOUL reads SOUL.md and extracts capabilities from the first line after "## Objective" or "## Core Behaviors".
func extractCapabilitiesFromSOUL(soulPath string) []string {
	data, err := os.ReadFile(soulPath)
	if err != nil {
		return nil
	}

	lines := strings.Split(string(data), "\n")
	inObjective := false
	inCoreBehaviors := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for section headers
		if strings.HasPrefix(trimmed, "## Objective") {
			inObjective = true
			inCoreBehaviors = false
			continue
		}
		if strings.HasPrefix(trimmed, "## Core Behaviors") {
			inCoreBehaviors = true
			inObjective = false
			continue
		}

		// If we hit another section header, stop
		if strings.HasPrefix(trimmed, "## ") {
			inObjective = false
			inCoreBehaviors = false
			continue
		}

		// If we're in a target section and find a non-empty line, extract capabilities
		if (inObjective || inCoreBehaviors) && trimmed != "" {
			// Simple extraction: split by commas or use the whole line as a single capability
			if strings.Contains(trimmed, ",") {
				parts := strings.Split(trimmed, ",")
				capabilities := []string{}
				for _, part := range parts {
					cap := strings.TrimSpace(part)
					if cap != "" {
						capabilities = append(capabilities, cap)
					}
				}
				return capabilities
			}
			return []string{trimmed}
		}
	}

	return nil
}
