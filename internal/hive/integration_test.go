package hive

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/valter-silva-au/ai-dev-brain/pkg/models"
	"gopkg.in/yaml.v3"
)

// TestIntegration_FullWorkflow tests the complete Hive Mind workflow from end to end.
func TestIntegration_FullWorkflow(t *testing.T) {
	t.Parallel()

	// Create temp dir as basePath
	basePath := t.TempDir()

	// Create 2 fake project directories
	project1Path := filepath.Join(basePath, "projects", "project-alpha")
	project2Path := filepath.Join(basePath, "projects", "project-beta")

	// Setup project-alpha with knowledge entries
	setupProjectWithKnowledge(t, project1Path, []knowledgeEntry{
		{
			ID:         "alpha-001",
			Type:       "decision",
			Topic:      "API Design",
			Summary:    "REST API structure for project alpha",
			Detail:     "Using REST principles with JSON payloads",
			SourceTask: "ALPHA-001",
			SourceType: "manual",
			Date:       "2026-03-01",
		},
		{
			ID:         "alpha-002",
			Type:       "pattern",
			Topic:      "Error Handling",
			Summary:    "Error handling patterns in alpha",
			Detail:     "Wrap all errors with context",
			SourceTask: "ALPHA-002",
			SourceType: "manual",
			Date:       "2026-03-02",
		},
		{
			ID:         "alpha-003",
			Type:       "lesson",
			Topic:      "Testing",
			Summary:    "Testing lessons learned",
			Detail:     "Use t.TempDir() for all file-based tests",
			SourceTask: "ALPHA-003",
			SourceType: "manual",
			Date:       "2026-03-03",
		},
	})

	// Setup project-beta with knowledge entries
	setupProjectWithKnowledge(t, project2Path, []knowledgeEntry{
		{
			ID:         "beta-001",
			Type:       "decision",
			Topic:      "Database Choice",
			Summary:    "Selected PostgreSQL for project beta",
			Detail:     "PostgreSQL chosen for ACID compliance",
			SourceTask: "BETA-001",
			SourceType: "manual",
			Date:       "2026-03-04",
		},
		{
			ID:         "beta-002",
			Type:       "pattern",
			Topic:      "Logging",
			Summary:    "Structured logging pattern",
			Detail:     "Using structured logging with JSON format",
			SourceTask: "BETA-002",
			SourceType: "manual",
			Date:       "2026-03-05",
		},
	})

	// Create fake openclaw directory with 2 agents
	openclawPath := filepath.Join(basePath, "openclaw")
	setupOpenClawConfig(t, openclawPath, []openclawAgent{
		{
			ID:        "agent-001",
			Name:      "agent-alpha",
			Workspace: filepath.Join(basePath, "workspace", "agent-alpha"),
			AgentDir:  filepath.Join(openclawPath, "agents", "agent-alpha"),
		},
		{
			ID:        "agent-002",
			Name:      "agent-beta",
			Workspace: filepath.Join(basePath, "workspace", "agent-beta"),
			AgentDir:  filepath.Join(openclawPath, "agents", "agent-beta"),
		},
	})

	// Create SOUL.md files for agents
	createSOULFile(t, filepath.Join(openclawPath, "agents", "agent-alpha"), "API development, code review")
	createSOULFile(t, filepath.Join(openclawPath, "agents", "agent-beta"), "Testing, documentation")

	// Step 1: Create ProjectRegistry and register both projects
	projectReg := NewProjectRegistry(basePath)

	project1 := models.Project{
		Name:        "project-alpha",
		RepoPath:    project1Path,
		Purpose:     "Alpha project for testing",
		TechStack:   []string{"go"},
		Status:      models.ProjectActive,
		Tags:        []string{"test"},
		DefaultAI:   "claude",
		LastUpdated: time.Now().UTC(),
	}

	project2 := models.Project{
		Name:        "project-beta",
		RepoPath:    project2Path,
		Purpose:     "Beta project for testing",
		TechStack:   []string{"go", "postgres"},
		Status:      models.ProjectActive,
		Tags:        []string{"test"},
		DefaultAI:   "claude",
		LastUpdated: time.Now().UTC(),
	}

	if err := projectReg.Register(project1); err != nil {
		t.Fatalf("Failed to register project1: %v", err)
	}
	if err := projectReg.Register(project2); err != nil {
		t.Fatalf("Failed to register project2: %v", err)
	}

	// Step 2: Create KnowledgeAggregator and index
	knowledgeAgg := NewKnowledgeAggregator(basePath, projectReg)
	if err := knowledgeAgg.Index(); err != nil {
		t.Fatalf("Failed to index knowledge: %v", err)
	}

	// Step 3: Verify search returns entries from both projects
	results, err := knowledgeAgg.SearchAcrossProjects("", SearchOptions{})
	if err != nil {
		t.Fatalf("SearchAcrossProjects() error = %v", err)
	}
	if len(results) != 5 {
		t.Errorf("SearchAcrossProjects() returned %d results, want 5", len(results))
	}

	// Verify entries from both projects are present
	projectsFound := make(map[string]int)
	for _, result := range results {
		projectsFound[result.Project]++
	}
	if projectsFound["project-alpha"] != 3 {
		t.Errorf("Found %d entries from project-alpha, want 3", projectsFound["project-alpha"])
	}
	if projectsFound["project-beta"] != 2 {
		t.Errorf("Found %d entries from project-beta, want 2", projectsFound["project-beta"])
	}

	// Search for specific topic
	decisionResults, err := knowledgeAgg.GetDecisionsForTopic("API")
	if err != nil {
		t.Fatalf("GetDecisionsForTopic() error = %v", err)
	}
	if len(decisionResults) != 1 {
		t.Errorf("GetDecisionsForTopic('API') returned %d results, want 1", len(decisionResults))
	}

	// Step 4: Create AgentRegistry and discover agents
	agentReg := NewAgentRegistry(basePath)
	discoveredAgents, err := agentReg.DiscoverOpenClaw(openclawPath)
	if err != nil {
		t.Fatalf("DiscoverOpenClaw() error = %v", err)
	}
	if len(discoveredAgents) != 2 {
		t.Errorf("DiscoverOpenClaw() found %d agents, want 2", len(discoveredAgents))
	}

	// Step 5: Register discovered agents
	for _, agent := range discoveredAgents {
		if err := agentReg.Register(agent); err != nil {
			t.Fatalf("Failed to register agent %s: %v", agent.Name, err)
		}
	}

	// Step 6: Verify list agents returns both
	allAgents, err := agentReg.List(models.AgentFilter{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(allAgents) != 2 {
		t.Errorf("List() returned %d agents, want 2", len(allAgents))
	}

	// Verify agent names
	agentNames := make(map[string]bool)
	for _, agent := range allAgents {
		agentNames[agent.Name] = true
	}
	if !agentNames["agent-alpha"] {
		t.Error("agent-alpha not found in agent list")
	}
	if !agentNames["agent-beta"] {
		t.Error("agent-beta not found in agent list")
	}

	// Step 7: Create MessageBus and publish message
	messageBus := NewMessageBus(basePath)

	msg := models.HiveMessage{
		From:           "agent-alpha",
		To:             "agent-beta",
		Subject:        "Test Message",
		Content:        "Hello from agent-alpha to agent-beta",
		Type:           models.HiveMessageRequest,
		Priority:       "normal",
		ConversationID: "conv-001",
	}

	if err := messageBus.Publish(msg); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	// Step 8: Subscribe for recipient and verify message received
	messages, err := messageBus.Subscribe("agent-beta")
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}
	if len(messages) != 1 {
		t.Errorf("Subscribe() returned %d messages, want 1", len(messages))
	}
	if len(messages) > 0 {
		if messages[0].From != "agent-alpha" {
			t.Errorf("Message from = %s, want agent-alpha", messages[0].From)
		}
		if messages[0].To != "agent-beta" {
			t.Errorf("Message to = %s, want agent-beta", messages[0].To)
		}
		if messages[0].Subject != "Test Message" {
			t.Errorf("Message subject = %s, want 'Test Message'", messages[0].Subject)
		}
	}

	// Step 9: Mark processed and verify inbox empty
	if len(messages) > 0 {
		if err := messageBus.MarkProcessed(messages[0].ID); err != nil {
			t.Fatalf("MarkProcessed() error = %v", err)
		}
	}

	// Verify inbox is now empty
	messagesAfter, err := messageBus.Subscribe("agent-beta")
	if err != nil {
		t.Fatalf("Subscribe() after processing error = %v", err)
	}
	if len(messagesAfter) != 0 {
		t.Errorf("Subscribe() after processing returned %d messages, want 0", len(messagesAfter))
	}
}

// TestIntegration_SaveLoadRoundTrip tests that all data survives save/load cycles.
func TestIntegration_SaveLoadRoundTrip(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()

	// Setup projects with knowledge
	project1Path := filepath.Join(basePath, "projects", "project-gamma")
	setupProjectWithKnowledge(t, project1Path, []knowledgeEntry{
		{
			ID:         "gamma-001",
			Type:       "decision",
			Topic:      "Architecture",
			Summary:    "Microservices architecture",
			Detail:     "Chose microservices for scalability",
			SourceTask: "GAMMA-001",
			SourceType: "manual",
			Date:       "2026-03-10",
		},
	})

	// Create and register project
	projectReg1 := NewProjectRegistry(basePath)
	project := models.Project{
		Name:        "project-gamma",
		RepoPath:    project1Path,
		Purpose:     "Testing save/load",
		TechStack:   []string{"go", "docker"},
		Status:      models.ProjectActive,
		Tags:        []string{"microservices"},
		DefaultAI:   "claude",
		LastUpdated: time.Now().UTC(),
	}
	if err := projectReg1.Register(project); err != nil {
		t.Fatalf("Failed to register project: %v", err)
	}
	if err := projectReg1.Save(); err != nil {
		t.Fatalf("Failed to save project registry: %v", err)
	}

	// Index knowledge
	knowledgeAgg1 := NewKnowledgeAggregator(basePath, projectReg1)
	if err := knowledgeAgg1.Index(); err != nil {
		t.Fatalf("Failed to index knowledge: %v", err)
	}
	if err := knowledgeAgg1.Save(); err != nil {
		t.Fatalf("Failed to save knowledge aggregator: %v", err)
	}

	// Register agents
	agentReg1 := NewAgentRegistry(basePath)
	agent1 := models.Agent{
		Name:         "agent-gamma",
		Type:         models.AgentOpenClaw,
		Model:        "claude-sonnet-4",
		Capabilities: []string{"coding", "review"},
		Status:       models.AgentIdle,
		LastSeen:     time.Now().UTC(),
	}
	if err := agentReg1.Register(agent1); err != nil {
		t.Fatalf("Failed to register agent: %v", err)
	}
	if err := agentReg1.Save(); err != nil {
		t.Fatalf("Failed to save agent registry: %v", err)
	}

	// Create fresh instances at same path
	projectReg2 := NewProjectRegistry(basePath)
	if err := projectReg2.Load(); err != nil {
		t.Fatalf("Failed to load project registry: %v", err)
	}

	knowledgeAgg2 := NewKnowledgeAggregator(basePath, projectReg2)
	if err := knowledgeAgg2.Load(); err != nil {
		t.Fatalf("Failed to load knowledge aggregator: %v", err)
	}

	agentReg2 := NewAgentRegistry(basePath)
	if err := agentReg2.Load(); err != nil {
		t.Fatalf("Failed to load agent registry: %v", err)
	}

	// Verify projects survived the round trip
	loadedProject, err := projectReg2.Get("project-gamma")
	if err != nil {
		t.Fatalf("Failed to get loaded project: %v", err)
	}
	if loadedProject.Name != project.Name {
		t.Errorf("Loaded project name = %s, want %s", loadedProject.Name, project.Name)
	}
	if loadedProject.RepoPath != project.RepoPath {
		t.Errorf("Loaded project repo path = %s, want %s", loadedProject.RepoPath, project.RepoPath)
	}
	if len(loadedProject.TechStack) != len(project.TechStack) {
		t.Errorf("Loaded project tech stack length = %d, want %d", len(loadedProject.TechStack), len(project.TechStack))
	}

	// Verify knowledge survived the round trip
	knowledgeResults, err := knowledgeAgg2.SearchAcrossProjects("", SearchOptions{})
	if err != nil {
		t.Fatalf("Failed to search knowledge: %v", err)
	}
	if len(knowledgeResults) != 1 {
		t.Errorf("Loaded knowledge entries = %d, want 1", len(knowledgeResults))
	}
	if len(knowledgeResults) > 0 {
		if knowledgeResults[0].Topic != "Architecture" {
			t.Errorf("Loaded knowledge topic = %s, want 'Architecture'", knowledgeResults[0].Topic)
		}
	}

	// Verify agents survived the round trip
	loadedAgent, err := agentReg2.Get("agent-gamma")
	if err != nil {
		t.Fatalf("Failed to get loaded agent: %v", err)
	}
	if loadedAgent.Name != agent1.Name {
		t.Errorf("Loaded agent name = %s, want %s", loadedAgent.Name, agent1.Name)
	}
	if loadedAgent.Type != agent1.Type {
		t.Errorf("Loaded agent type = %s, want %s", loadedAgent.Type, agent1.Type)
	}
	if len(loadedAgent.Capabilities) != len(agent1.Capabilities) {
		t.Errorf("Loaded agent capabilities length = %d, want %d", len(loadedAgent.Capabilities), len(agent1.Capabilities))
	}
}

// TestIntegration_EmptyEcosystem tests that all components handle empty state gracefully.
func TestIntegration_EmptyEcosystem(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()

	// Create all components with empty basePath
	projectReg := NewProjectRegistry(basePath)
	knowledgeAgg := NewKnowledgeAggregator(basePath, projectReg)
	agentReg := NewAgentRegistry(basePath)
	messageBus := NewMessageBus(basePath)

	// Test ProjectRegistry operations on empty state
	projects, err := projectReg.List(models.ProjectFilter{})
	if err != nil {
		t.Errorf("ProjectRegistry.List() on empty state error = %v, want nil", err)
	}
	if len(projects) != 0 {
		t.Errorf("ProjectRegistry.List() on empty state returned %d projects, want 0", len(projects))
	}

	_, err = projectReg.Get("nonexistent")
	if err == nil {
		t.Error("ProjectRegistry.Get() on nonexistent project returned nil error, want error")
	}

	// Test KnowledgeAggregator operations on empty state
	err = knowledgeAgg.Index()
	if err != nil {
		t.Errorf("KnowledgeAggregator.Index() on empty state error = %v, want nil", err)
	}

	results, err := knowledgeAgg.SearchAcrossProjects("test", SearchOptions{})
	if err != nil {
		t.Errorf("KnowledgeAggregator.SearchAcrossProjects() on empty state error = %v, want nil", err)
	}
	if len(results) != 0 {
		t.Errorf("KnowledgeAggregator.SearchAcrossProjects() on empty state returned %d results, want 0", len(results))
	}

	decisions, err := knowledgeAgg.GetDecisionsForTopic("test")
	if err != nil {
		t.Errorf("KnowledgeAggregator.GetDecisionsForTopic() on empty state error = %v, want nil", err)
	}
	if len(decisions) != 0 {
		t.Errorf("KnowledgeAggregator.GetDecisionsForTopic() on empty state returned %d results, want 0", len(decisions))
	}

	// Test AgentRegistry operations on empty state
	agents, err := agentReg.List(models.AgentFilter{})
	if err != nil {
		t.Errorf("AgentRegistry.List() on empty state error = %v, want nil", err)
	}
	if len(agents) != 0 {
		t.Errorf("AgentRegistry.List() on empty state returned %d agents, want 0", len(agents))
	}

	_, err = agentReg.Get("nonexistent")
	if err == nil {
		t.Error("AgentRegistry.Get() on nonexistent agent returned nil error, want error")
	}

	// Test MessageBus operations on empty state
	messages, err := messageBus.Subscribe("nonexistent-agent")
	if err != nil {
		t.Errorf("MessageBus.Subscribe() on empty inbox error = %v, want nil", err)
	}
	if len(messages) != 0 {
		t.Errorf("MessageBus.Subscribe() on empty inbox returned %d messages, want 0", len(messages))
	}

	conversation, err := messageBus.GetConversation("nonexistent-conversation")
	if err != nil {
		t.Errorf("MessageBus.GetConversation() on empty state error = %v, want nil", err)
	}
	if len(conversation) != 0 {
		t.Errorf("MessageBus.GetConversation() on empty state returned %d messages, want 0", len(conversation))
	}

	// Test Load operations on empty state (no files exist yet)
	err = projectReg.Load()
	if err != nil {
		t.Errorf("ProjectRegistry.Load() on empty state error = %v, want nil", err)
	}

	err = knowledgeAgg.Load()
	if err != nil {
		t.Errorf("KnowledgeAggregator.Load() on empty state error = %v, want nil", err)
	}

	err = agentReg.Load()
	if err != nil {
		t.Errorf("AgentRegistry.Load() on empty state error = %v, want nil", err)
	}

	// Verify no panics by calling save operations
	err = projectReg.Save()
	if err != nil {
		t.Errorf("ProjectRegistry.Save() on empty state error = %v, want nil", err)
	}

	err = knowledgeAgg.Save()
	if err != nil {
		t.Errorf("KnowledgeAggregator.Save() on empty state error = %v, want nil", err)
	}

	err = agentReg.Save()
	if err != nil {
		t.Errorf("AgentRegistry.Save() on empty state error = %v, want nil", err)
	}
}

// Helper types and functions
// Note: knowledgeEntry type is defined in knowledgeaggregator_test.go

type openclawAgent struct {
	ID        string
	Name      string
	Workspace string
	AgentDir  string
}

// setupProjectWithKnowledge creates a fake project directory with knowledge entries.
func setupProjectWithKnowledge(t *testing.T, projectPath string, entries []knowledgeEntry) {
	t.Helper()

	// Create directory structure
	knowledgePath := filepath.Join(projectPath, "docs", "knowledge")
	if err := os.MkdirAll(knowledgePath, 0o755); err != nil {
		t.Fatalf("Failed to create knowledge directory: %v", err)
	}

	// Create knowledge index
	type knowledgeIndexEntry struct {
		ID         string `yaml:"id"`
		Type       string `yaml:"type"`
		Topic      string `yaml:"topic"`
		Summary    string `yaml:"summary"`
		Detail     string `yaml:"detail"`
		SourceTask string `yaml:"source_task"`
		SourceType string `yaml:"source_type"`
		Date       string `yaml:"date"`
	}

	type knowledgeIndexFile struct {
		Version string                    `yaml:"version"`
		Entries []knowledgeIndexEntry     `yaml:"entries"`
	}

	var indexEntries []knowledgeIndexEntry
	for _, entry := range entries {
		indexEntries = append(indexEntries, knowledgeIndexEntry{
			ID:         entry.ID,
			Type:       entry.Type,
			Topic:      entry.Topic,
			Summary:    entry.Summary,
			Detail:     entry.Detail,
			SourceTask: entry.SourceTask,
			SourceType: entry.SourceType,
			Date:       entry.Date,
		})
	}

	indexFile := knowledgeIndexFile{
		Version: "1.0",
		Entries: indexEntries,
	}

	data, err := yaml.Marshal(&indexFile)
	if err != nil {
		t.Fatalf("Failed to marshal knowledge index: %v", err)
	}

	indexPath := filepath.Join(knowledgePath, "index.yaml")
	if err := os.WriteFile(indexPath, data, 0o644); err != nil {
		t.Fatalf("Failed to write knowledge index: %v", err)
	}
}

// setupOpenClawConfig creates a fake openclaw.json configuration.
func setupOpenClawConfig(t *testing.T, openclawPath string, agents []openclawAgent) {
	t.Helper()

	if err := os.MkdirAll(openclawPath, 0o755); err != nil {
		t.Fatalf("Failed to create openclaw directory: %v", err)
	}

	type agentEntry struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		Workspace string `json:"workspace"`
		AgentDir  string `json:"agentDir"`
	}

	type config struct {
		Agents struct {
			Defaults struct {
				Model struct {
					Primary string `json:"primary"`
				} `json:"model"`
			} `json:"defaults"`
			List []agentEntry `json:"list"`
		} `json:"agents"`
	}

	var cfg config
	cfg.Agents.Defaults.Model.Primary = "claude-sonnet-4"

	for _, agent := range agents {
		cfg.Agents.List = append(cfg.Agents.List, agentEntry{
			ID:        agent.ID,
			Name:      agent.Name,
			Workspace: agent.Workspace,
			AgentDir:  agent.AgentDir,
		})
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal openclaw config: %v", err)
	}

	configPath := filepath.Join(openclawPath, "openclaw.json")
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		t.Fatalf("Failed to write openclaw config: %v", err)
	}
}

// createSOULFile creates a fake SOUL.md file for an agent.
func createSOULFile(t *testing.T, agentDir string, capabilities string) {
	t.Helper()

	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("Failed to create agent directory: %v", err)
	}

	soulContent := `# Agent SOUL

## Objective
` + capabilities + `

## Core Behaviors
- Follow best practices
- Write clean code
`

	soulPath := filepath.Join(agentDir, "SOUL.md")
	if err := os.WriteFile(soulPath, []byte(soulContent), 0o644); err != nil {
		t.Fatalf("Failed to write SOUL.md: %v", err)
	}
}
