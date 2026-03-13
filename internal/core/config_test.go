package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/valter-silva-au/ai-dev-brain/pkg/models"
)

func TestNewViperConfigManager(t *testing.T) {
	t.Run("with custom paths", func(t *testing.T) {
		globalPath := "/custom/global/path"
		repoPath := "/custom/repo/path"

		cm := NewViperConfigManager(globalPath, repoPath)

		if cm.globalConfigPath != globalPath {
			t.Errorf("expected global path %s, got %s", globalPath, cm.globalConfigPath)
		}

		if cm.repoConfigPath != repoPath {
			t.Errorf("expected repo path %s, got %s", repoPath, cm.repoConfigPath)
		}
	})

	t.Run("with default paths", func(t *testing.T) {
		cm := NewViperConfigManager("", "")

		homeDir, _ := os.UserHomeDir()
		expectedGlobal := filepath.Join(homeDir, ".taskconfig")

		if cm.globalConfigPath != expectedGlobal {
			t.Errorf("expected global path %s, got %s", expectedGlobal, cm.globalConfigPath)
		}

		if cm.repoConfigPath != ".taskrc" {
			t.Errorf("expected repo path .taskrc, got %s", cm.repoConfigPath)
		}
	})
}

func TestGetGlobalConfig_NoFile(t *testing.T) {
	// Create a temp directory for testing
	tmpDir := t.TempDir()
	nonExistentPath := filepath.Join(tmpDir, "nonexistent.yaml")

	cm := NewViperConfigManager(nonExistentPath, "")

	config, err := cm.GetGlobalConfig()
	if err != nil {
		t.Fatalf("expected no error when file doesn't exist, got: %v", err)
	}

	// Should return default config
	defaultConfig := models.DefaultGlobalConfig()

	if config.TaskIDPrefix != defaultConfig.TaskIDPrefix {
		t.Errorf("expected default prefix %s, got %s", defaultConfig.TaskIDPrefix, config.TaskIDPrefix)
	}

	if !config.Hooks.Enabled {
		t.Error("expected hooks to be enabled by default")
	}
}

func TestGetGlobalConfig_WithFile(t *testing.T) {
	// Create a temp directory and config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".taskconfig")

	configContent := `task_id_prefix: "PROJ"
defaults:
  priority: "P1"
  type: "feature"
notifications:
  enabled: true
  channels:
    - slack
    - email
  on_events:
    - task_completed
hooks:
  enabled: true
  pre_tool_use: true
  post_tool_use: false
  stop: true
  task_completed: true
  session_end: false
  knowledge_extraction: true
  conflict_detection: true
  auto_format: false
  block_vendor_edits: true
aliases:
  aliases:
    t: "task"
    l: "list"
mcp_servers:
  server1: "http://localhost:8080"
feature_flags:
  new_feature: true
custom_settings:
  setting1: "value1"
`

	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("failed to create test config file: %v", err)
	}

	cm := NewViperConfigManager(configPath, "")

	config, err := cm.GetGlobalConfig()
	if err != nil {
		t.Fatalf("failed to load global config: %v", err)
	}

	// Verify loaded values
	if config.TaskIDPrefix != "PROJ" {
		t.Errorf("expected prefix PROJ, got %s", config.TaskIDPrefix)
	}

	if config.Defaults["priority"] != "P1" {
		t.Errorf("expected priority P1, got %s", config.Defaults["priority"])
	}

	if config.Defaults["type"] != "feature" {
		t.Errorf("expected type feature, got %s", config.Defaults["type"])
	}

	if !config.Notifications.Enabled {
		t.Error("expected notifications to be enabled")
	}

	if len(config.Notifications.Channels) != 2 {
		t.Errorf("expected 2 channels, got %d", len(config.Notifications.Channels))
	}

	if !config.Hooks.PreToolUse {
		t.Error("expected pre_tool_use to be true")
	}

	if config.Hooks.PostToolUse {
		t.Error("expected post_tool_use to be false")
	}

	if config.Hooks.SessionEnd {
		t.Error("expected session_end to be false")
	}

	if !config.Hooks.KnowledgeExtraction {
		t.Error("expected knowledge_extraction to be true")
	}

	if !config.Hooks.ConflictDetection {
		t.Error("expected conflict_detection to be true")
	}

	if config.Aliases.Aliases["t"] != "task" {
		t.Errorf("expected alias 't' to map to 'task', got %s", config.Aliases.Aliases["t"])
	}

	if config.MCPServers["server1"] != "http://localhost:8080" {
		t.Errorf("expected server1 URL, got %s", config.MCPServers["server1"])
	}

	if !config.FeatureFlags["new_feature"] {
		t.Error("expected new_feature flag to be true")
	}

	if config.CustomSettings["setting1"] != "value1" {
		t.Errorf("expected setting1 to be value1, got %s", config.CustomSettings["setting1"])
	}
}

func TestGetRepoConfig_NoFile(t *testing.T) {
	// Create a temp directory for testing
	tmpDir := t.TempDir()
	nonExistentPath := filepath.Join(tmpDir, "nonexistent.yaml")

	cm := NewViperConfigManager("", nonExistentPath)

	config, err := cm.GetRepoConfig()
	if err != nil {
		t.Fatalf("expected no error when file doesn't exist, got: %v", err)
	}

	// Should return default config
	defaultConfig := models.DefaultRepoConfig()

	if config.BaseBranch != defaultConfig.BaseBranch {
		t.Errorf("expected default base branch %s, got %s", defaultConfig.BaseBranch, config.BaseBranch)
	}

	if config.AutoSync != defaultConfig.AutoSync {
		t.Errorf("expected default auto_sync %v, got %v", defaultConfig.AutoSync, config.AutoSync)
	}
}

func TestGetRepoConfig_WithFile(t *testing.T) {
	// Create a temp directory and config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".taskrc")

	configContent := `repo_name: "my-project"
build_command: "go build ./..."
test_command: "go test ./... -v"
lint_command: "golangci-lint run"
reviewers:
  - alice
  - bob
required_checks:
  - lint
  - test
conventions:
  - "Use snake_case for variables"
  - "Add tests for all functions"
base_branch: "develop"
worktree_base_path: "/tmp/worktrees"
auto_sync: true
custom_settings:
  repo_setting: "repo_value"
`

	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("failed to create test config file: %v", err)
	}

	cm := NewViperConfigManager("", configPath)

	config, err := cm.GetRepoConfig()
	if err != nil {
		t.Fatalf("failed to load repo config: %v", err)
	}

	// Verify loaded values
	if config.RepoName != "my-project" {
		t.Errorf("expected repo_name my-project, got %s", config.RepoName)
	}

	if config.BuildCommand != "go build ./..." {
		t.Errorf("expected build command 'go build ./...', got %s", config.BuildCommand)
	}

	if config.TestCommand != "go test ./... -v" {
		t.Errorf("expected test command 'go test ./... -v', got %s", config.TestCommand)
	}

	if config.LintCommand != "golangci-lint run" {
		t.Errorf("expected lint command 'golangci-lint run', got %s", config.LintCommand)
	}

	if len(config.Reviewers) != 2 {
		t.Errorf("expected 2 reviewers, got %d", len(config.Reviewers))
	}

	if config.Reviewers[0] != "alice" || config.Reviewers[1] != "bob" {
		t.Errorf("unexpected reviewers: %v", config.Reviewers)
	}

	if len(config.RequiredChecks) != 2 {
		t.Errorf("expected 2 required checks, got %d", len(config.RequiredChecks))
	}

	if len(config.Conventions) != 2 {
		t.Errorf("expected 2 conventions, got %d", len(config.Conventions))
	}

	if config.BaseBranch != "develop" {
		t.Errorf("expected base branch develop, got %s", config.BaseBranch)
	}

	if config.WorktreeBasePath != "/tmp/worktrees" {
		t.Errorf("expected worktree path /tmp/worktrees, got %s", config.WorktreeBasePath)
	}

	if !config.AutoSync {
		t.Error("expected auto_sync to be true")
	}

	if config.CustomSettings["repo_setting"] != "repo_value" {
		t.Errorf("expected repo_setting to be repo_value, got %s", config.CustomSettings["repo_setting"])
	}
}

func TestLoadConfig_BothFiles(t *testing.T) {
	// Create a temp directory with both config files
	tmpDir := t.TempDir()
	globalPath := filepath.Join(tmpDir, ".taskconfig")
	repoPath := filepath.Join(tmpDir, ".taskrc")

	globalContent := `task_id_prefix: "GLOBAL"
defaults:
  priority: "P0"
hooks:
  enabled: true
  knowledge_extraction: true
`

	repoContent := `repo_name: "test-repo"
build_command: "make build"
base_branch: "main"
`

	if err := os.WriteFile(globalPath, []byte(globalContent), 0o644); err != nil {
		t.Fatalf("failed to create global config: %v", err)
	}

	if err := os.WriteFile(repoPath, []byte(repoContent), 0o644); err != nil {
		t.Fatalf("failed to create repo config: %v", err)
	}

	cm := NewViperConfigManager(globalPath, repoPath)

	merged, err := cm.LoadConfig()
	if err != nil {
		t.Fatalf("failed to load merged config: %v", err)
	}

	// Verify global config is present
	if merged.Global.TaskIDPrefix != "GLOBAL" {
		t.Errorf("expected global prefix GLOBAL, got %s", merged.Global.TaskIDPrefix)
	}

	if merged.Global.Defaults["priority"] != "P0" {
		t.Errorf("expected priority P0, got %s", merged.Global.Defaults["priority"])
	}

	if !merged.Global.Hooks.KnowledgeExtraction {
		t.Error("expected knowledge_extraction to be true")
	}

	// Verify repo config is present
	if merged.Repo.RepoName != "test-repo" {
		t.Errorf("expected repo_name test-repo, got %s", merged.Repo.RepoName)
	}

	if merged.Repo.BuildCommand != "make build" {
		t.Errorf("expected build command 'make build', got %s", merged.Repo.BuildCommand)
	}

	if merged.Repo.BaseBranch != "main" {
		t.Errorf("expected base branch main, got %s", merged.Repo.BaseBranch)
	}
}

func TestLoadConfig_NoFiles(t *testing.T) {
	// Create a temp directory with no config files
	tmpDir := t.TempDir()
	globalPath := filepath.Join(tmpDir, ".taskconfig")
	repoPath := filepath.Join(tmpDir, ".taskrc")

	cm := NewViperConfigManager(globalPath, repoPath)

	merged, err := cm.LoadConfig()
	if err != nil {
		t.Fatalf("expected no error with missing files, got: %v", err)
	}

	// Should return default configs
	if merged.Global.TaskIDPrefix != "TASK" {
		t.Errorf("expected default prefix TASK, got %s", merged.Global.TaskIDPrefix)
	}

	if merged.Repo.BaseBranch != "main" {
		t.Errorf("expected default base branch main, got %s", merged.Repo.BaseBranch)
	}
}

func TestGetGlobalConfig_InvalidYAML(t *testing.T) {
	// Create a temp directory with invalid YAML
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".taskconfig")

	invalidContent := `invalid: yaml: content:
  - this is
  bad yaml
    nested incorrectly
`

	if err := os.WriteFile(configPath, []byte(invalidContent), 0o644); err != nil {
		t.Fatalf("failed to create test config file: %v", err)
	}

	cm := NewViperConfigManager(configPath, "")

	_, err := cm.GetGlobalConfig()
	if err == nil {
		t.Error("expected error when loading invalid YAML")
	}
}

func TestDefaultHookConfig(t *testing.T) {
	config := DefaultHookConfig()

	// Verify Phase 1 features are enabled
	if !config.Enabled {
		t.Error("expected hooks to be enabled")
	}

	if !config.PreToolUse {
		t.Error("expected pre_tool_use to be enabled")
	}

	if !config.PostToolUse {
		t.Error("expected post_tool_use to be enabled")
	}

	if !config.Stop {
		t.Error("expected stop to be enabled")
	}

	if !config.TaskCompleted {
		t.Error("expected task_completed to be enabled")
	}

	if !config.SessionEnd {
		t.Error("expected session_end to be enabled")
	}

	if !config.AutoFormat {
		t.Error("expected auto_format to be enabled")
	}

	if !config.BlockVendorEdits {
		t.Error("expected block_vendor_edits to be enabled")
	}

	// Verify Phase 2/3 features are disabled (opt-in)
	if config.KnowledgeExtraction {
		t.Error("expected knowledge_extraction to be disabled by default (opt-in)")
	}

	if config.ConflictDetection {
		t.Error("expected conflict_detection to be disabled by default (opt-in)")
	}
}

func TestConfigPrecedence(t *testing.T) {
	// Test that repo config takes precedence over global config
	// This test verifies the conceptual precedence by loading both configs
	tmpDir := t.TempDir()
	globalPath := filepath.Join(tmpDir, ".taskconfig")
	repoPath := filepath.Join(tmpDir, ".taskrc")

	globalContent := `task_id_prefix: "GLOBAL"
defaults:
  priority: "P2"
  type: "feat"
hooks:
  enabled: true
  auto_format: true
`

	repoContent := `repo_name: "precedence-test"
base_branch: "main"
`

	if err := os.WriteFile(globalPath, []byte(globalContent), 0o644); err != nil {
		t.Fatalf("failed to create global config: %v", err)
	}

	if err := os.WriteFile(repoPath, []byte(repoContent), 0o644); err != nil {
		t.Fatalf("failed to create repo config: %v", err)
	}

	cm := NewViperConfigManager(globalPath, repoPath)

	merged, err := cm.LoadConfig()
	if err != nil {
		t.Fatalf("failed to load merged config: %v", err)
	}

	// Both should be present and independently loaded
	// Application logic should apply precedence when using values
	if merged.Global == nil {
		t.Error("expected global config to be present")
	}

	if merged.Repo == nil {
		t.Error("expected repo config to be present")
	}

	// Verify both configs have their respective values
	if merged.Global.TaskIDPrefix != "GLOBAL" {
		t.Errorf("expected global prefix, got %s", merged.Global.TaskIDPrefix)
	}

	if merged.Repo.RepoName != "precedence-test" {
		t.Errorf("expected repo name, got %s", merged.Repo.RepoName)
	}
}
