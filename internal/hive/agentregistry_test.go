package hive

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/valter-silva-au/ai-dev-brain/pkg/models"
)

func TestAgentRegistry_RegisterAndGet(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	registry := NewAgentRegistry(basePath)

	// Create a test agent
	agent := models.Agent{
		Name:          "test-agent",
		Type:          models.AgentClaudeCode,
		Model:         "claude-sonnet-4",
		Capabilities:  []string{"coding", "testing"},
		Role:          "developer",
		Status:        models.AgentIdle,
		HomeProject:   "test-project",
		ActiveTask:    "TASK-001",
		SessionCount:  5,
		MemoryPath:    "/path/to/memory",
		WorkspacePath: "/path/to/workspace",
	}

	// Register the agent
	err := registry.Register(agent)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Get by name
	retrieved, err := registry.Get("test-agent")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if retrieved == nil {
		t.Fatal("Get() returned nil")
	}

	// Verify all fields
	if retrieved.Name != agent.Name {
		t.Errorf("Name = %v, want %v", retrieved.Name, agent.Name)
	}
	if retrieved.Type != agent.Type {
		t.Errorf("Type = %v, want %v", retrieved.Type, agent.Type)
	}
	if retrieved.Model != agent.Model {
		t.Errorf("Model = %v, want %v", retrieved.Model, agent.Model)
	}
	if retrieved.Role != agent.Role {
		t.Errorf("Role = %v, want %v", retrieved.Role, agent.Role)
	}
	if retrieved.Status != agent.Status {
		t.Errorf("Status = %v, want %v", retrieved.Status, agent.Status)
	}
	if len(retrieved.Capabilities) != len(agent.Capabilities) {
		t.Errorf("Capabilities length = %v, want %v", len(retrieved.Capabilities), len(agent.Capabilities))
	}
	if retrieved.HomeProject != agent.HomeProject {
		t.Errorf("HomeProject = %v, want %v", retrieved.HomeProject, agent.HomeProject)
	}
	if retrieved.WorkspacePath != agent.WorkspacePath {
		t.Errorf("WorkspacePath = %v, want %v", retrieved.WorkspacePath, agent.WorkspacePath)
	}
}

func TestAgentRegistry_RegisterUpdate(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	registry := NewAgentRegistry(basePath)

	// Register initial agent
	agent1 := models.Agent{
		Name:   "test-agent",
		Type:   models.AgentClaudeCode,
		Model:  "claude-sonnet-4",
		Role:   "developer",
		Status: models.AgentIdle,
	}

	err := registry.Register(agent1)
	if err != nil {
		t.Fatalf("Register() initial error = %v", err)
	}

	// Register same agent with updated fields
	agent2 := models.Agent{
		Name:   "test-agent", // Same name
		Type:   models.AgentOpenClaw,
		Model:  "claude-opus-4",
		Role:   "architect",
		Status: models.AgentBusy,
	}

	err = registry.Register(agent2)
	if err != nil {
		t.Fatalf("Register() update error = %v", err)
	}

	// List all agents to verify no duplicates
	allAgents, err := registry.List(models.AgentFilter{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(allAgents) != 1 {
		t.Errorf("List() returned %d agents, want 1 (should update, not duplicate)", len(allAgents))
	}

	// Verify updated values
	retrieved, err := registry.Get("test-agent")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if retrieved.Type != models.AgentOpenClaw {
		t.Errorf("Type = %v, want %v", retrieved.Type, models.AgentOpenClaw)
	}
	if retrieved.Model != "claude-opus-4" {
		t.Errorf("Model = %v, want 'claude-opus-4'", retrieved.Model)
	}
	if retrieved.Role != "architect" {
		t.Errorf("Role = %v, want 'architect'", retrieved.Role)
	}
	if retrieved.Status != models.AgentBusy {
		t.Errorf("Status = %v, want %v", retrieved.Status, models.AgentBusy)
	}
}

func TestAgentRegistry_GetCaseInsensitive(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	registry := NewAgentRegistry(basePath)

	// Register agent with capital 'N'
	agent := models.Agent{
		Name:   "Nexus",
		Type:   models.AgentClaudeCode,
		Model:  "claude-sonnet-4",
		Status: models.AgentIdle,
	}

	err := registry.Register(agent)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Get with lowercase 'n'
	retrieved, err := registry.Get("nexus")
	if err != nil {
		t.Fatalf("Get() with lowercase error = %v", err)
	}
	if retrieved == nil {
		t.Fatal("Get() with lowercase returned nil")
	}

	if retrieved.Name != "Nexus" {
		t.Errorf("Name = %v, want 'Nexus'", retrieved.Name)
	}
}

func TestAgentRegistry_ListAll(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	registry := NewAgentRegistry(basePath)

	// Register 3 agents
	agents := []models.Agent{
		{
			Name:   "agent-1",
			Type:   models.AgentClaudeCode,
			Model:  "claude-sonnet-4",
			Status: models.AgentIdle,
		},
		{
			Name:   "agent-2",
			Type:   models.AgentOpenClaw,
			Model:  "claude-opus-4",
			Status: models.AgentBusy,
		},
		{
			Name:   "agent-3",
			Type:   models.AgentClaudeCode,
			Model:  "claude-sonnet-4",
			Status: models.AgentOffline,
		},
	}

	for _, a := range agents {
		if err := registry.Register(a); err != nil {
			t.Fatalf("Register() error = %v", err)
		}
	}

	// List with empty filter
	allAgents, err := registry.List(models.AgentFilter{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(allAgents) != 3 {
		t.Errorf("List() returned %d agents, want 3", len(allAgents))
	}

	// Verify all agent names are present
	names := make(map[string]bool)
	for _, a := range allAgents {
		names[a.Name] = true
	}

	for _, expected := range []string{"agent-1", "agent-2", "agent-3"} {
		if !names[expected] {
			t.Errorf("List() missing agent %s", expected)
		}
	}
}

func TestAgentRegistry_ListFilterByType(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	registry := NewAgentRegistry(basePath)

	// Register agents with different types
	agents := []models.Agent{
		{
			Name:   "claude-agent-1",
			Type:   models.AgentClaudeCode,
			Model:  "claude-sonnet-4",
			Status: models.AgentIdle,
		},
		{
			Name:   "openclaw-agent-1",
			Type:   models.AgentOpenClaw,
			Model:  "claude-opus-4",
			Status: models.AgentIdle,
		},
		{
			Name:   "openclaw-agent-2",
			Type:   models.AgentOpenClaw,
			Model:  "claude-sonnet-4",
			Status: models.AgentBusy,
		},
		{
			Name:   "claude-agent-2",
			Type:   models.AgentClaudeCode,
			Model:  "claude-sonnet-4",
			Status: models.AgentIdle,
		},
	}

	for _, a := range agents {
		if err := registry.Register(a); err != nil {
			t.Fatalf("Register() error = %v", err)
		}
	}

	// Filter by openclaw type
	filter := models.AgentFilter{
		Type: models.AgentOpenClaw,
	}

	openclawAgents, err := registry.List(filter)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(openclawAgents) != 2 {
		t.Errorf("List() returned %d openclaw agents, want 2", len(openclawAgents))
	}

	// Verify all returned agents are openclaw type
	for _, a := range openclawAgents {
		if a.Type != models.AgentOpenClaw {
			t.Errorf("List() returned agent with type %v, want %v", a.Type, models.AgentOpenClaw)
		}
	}
}

func TestAgentRegistry_ListFilterByCapabilities(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	registry := NewAgentRegistry(basePath)

	// Register agents with different capabilities
	agents := []models.Agent{
		{
			Name:         "agent-1",
			Type:         models.AgentClaudeCode,
			Model:        "claude-sonnet-4",
			Capabilities: []string{"coding", "testing"},
			Status:       models.AgentIdle,
		},
		{
			Name:         "agent-2",
			Type:         models.AgentOpenClaw,
			Model:        "claude-opus-4",
			Capabilities: []string{"architecture", "design"},
			Status:       models.AgentIdle,
		},
		{
			Name:         "agent-3",
			Type:         models.AgentClaudeCode,
			Model:        "claude-sonnet-4",
			Capabilities: []string{"coding", "documentation"},
			Status:       models.AgentIdle,
		},
	}

	for _, a := range agents {
		if err := registry.Register(a); err != nil {
			t.Fatalf("Register() error = %v", err)
		}
	}

	// Filter by "coding" capability
	filter := models.AgentFilter{
		Capabilities: []string{"coding"},
	}

	codingAgents, err := registry.List(filter)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(codingAgents) != 2 {
		t.Errorf("List() returned %d agents with 'coding' capability, want 2", len(codingAgents))
	}

	// Verify returned agents have the coding capability
	for _, a := range codingAgents {
		hasCoding := false
		for _, cap := range a.Capabilities {
			if cap == "coding" {
				hasCoding = true
				break
			}
		}
		if !hasCoding {
			t.Errorf("List() returned agent %s without 'coding' capability", a.Name)
		}
	}
}

func TestAgentRegistry_ListFilterByStatus(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	registry := NewAgentRegistry(basePath)

	// Register agents with different statuses
	agents := []models.Agent{
		{
			Name:   "idle-agent-1",
			Type:   models.AgentClaudeCode,
			Model:  "claude-sonnet-4",
			Status: models.AgentIdle,
		},
		{
			Name:   "busy-agent",
			Type:   models.AgentOpenClaw,
			Model:  "claude-opus-4",
			Status: models.AgentBusy,
		},
		{
			Name:   "idle-agent-2",
			Type:   models.AgentClaudeCode,
			Model:  "claude-sonnet-4",
			Status: models.AgentIdle,
		},
		{
			Name:   "offline-agent",
			Type:   models.AgentOpenClaw,
			Model:  "claude-sonnet-4",
			Status: models.AgentOffline,
		},
	}

	for _, a := range agents {
		if err := registry.Register(a); err != nil {
			t.Fatalf("Register() error = %v", err)
		}
	}

	// Filter by idle status
	filter := models.AgentFilter{
		Status: models.AgentIdle,
	}

	idleAgents, err := registry.List(filter)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(idleAgents) != 2 {
		t.Errorf("List() returned %d idle agents, want 2", len(idleAgents))
	}

	// Verify all returned agents are idle
	for _, a := range idleAgents {
		if a.Status != models.AgentIdle {
			t.Errorf("List() returned agent with status %v, want %v", a.Status, models.AgentIdle)
		}
	}
}

func TestAgentRegistry_SaveAndLoad(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()

	// Create first registry and register agents
	registry1 := NewAgentRegistry(basePath)

	agents := []models.Agent{
		{
			Name:          "agent-1",
			Type:          models.AgentClaudeCode,
			Model:         "claude-sonnet-4",
			Capabilities:  []string{"coding"},
			Role:          "developer",
			Status:        models.AgentIdle,
			HomeProject:   "project-1",
			WorkspacePath: "/path/to/workspace1",
		},
		{
			Name:          "agent-2",
			Type:          models.AgentOpenClaw,
			Model:         "claude-opus-4",
			Capabilities:  []string{"architecture"},
			Role:          "architect",
			Status:        models.AgentBusy,
			HomeProject:   "project-2",
			WorkspacePath: "/path/to/workspace2",
		},
	}

	for _, a := range agents {
		if err := registry1.Register(a); err != nil {
			t.Fatalf("Register() error = %v", err)
		}
	}

	// Save to disk
	if err := registry1.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Create new registry at same path and load
	registry2 := NewAgentRegistry(basePath)
	if err := registry2.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// List all agents from the loaded registry
	loadedAgents, err := registry2.List(models.AgentFilter{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(loadedAgents) != len(agents) {
		t.Errorf("Load() returned %d agents, want %d", len(loadedAgents), len(agents))
	}

	// Verify agents match
	agentMap := make(map[string]models.Agent)
	for _, a := range loadedAgents {
		agentMap[a.Name] = a
	}

	for _, original := range agents {
		loaded, exists := agentMap[original.Name]
		if !exists {
			t.Errorf("Load() missing agent %s", original.Name)
			continue
		}

		if loaded.Type != original.Type {
			t.Errorf("Agent %s: Type = %v, want %v", original.Name, loaded.Type, original.Type)
		}
		if loaded.Model != original.Model {
			t.Errorf("Agent %s: Model = %v, want %v", original.Name, loaded.Model, original.Model)
		}
		if loaded.Role != original.Role {
			t.Errorf("Agent %s: Role = %v, want %v", original.Name, loaded.Role, original.Role)
		}
		if loaded.Status != original.Status {
			t.Errorf("Agent %s: Status = %v, want %v", original.Name, loaded.Status, original.Status)
		}
	}
}

func TestAgentRegistry_GetNotFound(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	registry := NewAgentRegistry(basePath)

	// Try to get a nonexistent agent
	agent, err := registry.Get("nonexistent-agent")

	// Based on the implementation, Get returns an error when not found
	if err == nil {
		t.Error("Get() for nonexistent agent should return error, got nil")
	}

	if agent != nil {
		t.Errorf("Get() for nonexistent agent returned %v, want nil", agent)
	}
}

func TestAgentRegistry_DiscoverOpenClaw(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	// Create minimal openclaw.json
	openclawConfig := map[string]interface{}{
		"agents": map[string]interface{}{
			"defaults": map[string]interface{}{
				"model": map[string]interface{}{
					"primary": "amazon-bedrock/us.anthropic.claude-opus-4-6-v1",
				},
			},
			"list": []map[string]interface{}{
				{
					"id":        "coder",
					"name":      "coder",
					"workspace": "/tmp/ws-coder",
				},
				{
					"id":        "producer",
					"name":      "producer",
					"workspace": "/tmp/ws-producer",
				},
			},
		},
	}

	configData, err := json.MarshalIndent(openclawConfig, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal openclaw config: %v", err)
	}

	configPath := filepath.Join(tempDir, "openclaw.json")
	if err := os.WriteFile(configPath, configData, 0o644); err != nil {
		t.Fatalf("Failed to write openclaw.json: %v", err)
	}

	// Create registry and discover agents
	basePath := t.TempDir()
	registry := NewAgentRegistry(basePath)

	discoveredAgents, err := registry.DiscoverOpenClaw(tempDir)
	if err != nil {
		t.Fatalf("DiscoverOpenClaw() error = %v", err)
	}

	// Verify 2 agents were discovered
	if len(discoveredAgents) != 2 {
		t.Errorf("DiscoverOpenClaw() returned %d agents, want 2", len(discoveredAgents))
	}

	// Verify agent properties
	for _, agent := range discoveredAgents {
		if agent.Type != models.AgentOpenClaw {
			t.Errorf("Agent %s: Type = %v, want %v", agent.Name, agent.Type, models.AgentOpenClaw)
		}
		if agent.Model != "amazon-bedrock/us.anthropic.claude-opus-4-6-v1" {
			t.Errorf("Agent %s: Model = %v, want 'amazon-bedrock/us.anthropic.claude-opus-4-6-v1'", agent.Name, agent.Model)
		}
		if agent.Status != models.AgentIdle {
			t.Errorf("Agent %s: Status = %v, want %v", agent.Name, agent.Status, models.AgentIdle)
		}
		if agent.Name != "coder" && agent.Name != "producer" {
			t.Errorf("Agent %s: unexpected name, want 'coder' or 'producer'", agent.Name)
		}
	}

	// Verify workspace paths are set correctly
	agentMap := make(map[string]models.Agent)
	for _, a := range discoveredAgents {
		agentMap[a.Name] = a
	}

	if coder, exists := agentMap["coder"]; exists {
		if coder.WorkspacePath != "/tmp/ws-coder" {
			t.Errorf("Agent coder: WorkspacePath = %v, want '/tmp/ws-coder'", coder.WorkspacePath)
		}
	} else {
		t.Error("DiscoverOpenClaw() missing 'coder' agent")
	}

	if producer, exists := agentMap["producer"]; exists {
		if producer.WorkspacePath != "/tmp/ws-producer" {
			t.Errorf("Agent producer: WorkspacePath = %v, want '/tmp/ws-producer'", producer.WorkspacePath)
		}
	} else {
		t.Error("DiscoverOpenClaw() missing 'producer' agent")
	}
}

func TestAgentRegistry_DiscoverOpenClawMissingFile(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	// Create registry and attempt to discover from nonexistent path
	basePath := t.TempDir()
	registry := NewAgentRegistry(basePath)

	// Call DiscoverOpenClaw on path without openclaw.json
	discoveredAgents, err := registry.DiscoverOpenClaw(filepath.Join(tempDir, "nonexistent"))

	// Should return an error
	if err == nil {
		t.Error("DiscoverOpenClaw() with missing file should return error, got nil")
	}

	// Should not return agents
	if discoveredAgents != nil && len(discoveredAgents) > 0 {
		t.Errorf("DiscoverOpenClaw() with missing file returned %d agents, want 0 or nil", len(discoveredAgents))
	}
}

func TestAgentRegistry_LastSeenTimestamp(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	registry := NewAgentRegistry(basePath)

	beforeRegister := time.Now().UTC()

	agent := models.Agent{
		Name:   "test-agent",
		Type:   models.AgentClaudeCode,
		Model:  "claude-sonnet-4",
		Status: models.AgentIdle,
	}

	// Register the agent
	if err := registry.Register(agent); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	afterRegister := time.Now().UTC()

	// Get the agent and check LastSeen was set
	retrieved, err := registry.Get("test-agent")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if retrieved.LastSeen.IsZero() {
		t.Error("LastSeen was not set")
	}

	if retrieved.LastSeen.Before(beforeRegister) || retrieved.LastSeen.After(afterRegister) {
		t.Errorf("LastSeen = %v, want between %v and %v", retrieved.LastSeen, beforeRegister, afterRegister)
	}
}
