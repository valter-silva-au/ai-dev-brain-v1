package models

// NotificationConfig holds notification settings
type NotificationConfig struct {
	Enabled  bool     `yaml:"enabled"`
	Channels []string `yaml:"channels,omitempty"`
	OnEvents []string `yaml:"on_events,omitempty"`
}

// TeamRoutingConfig holds team routing settings
type TeamRoutingConfig struct {
	Enabled      bool              `yaml:"enabled"`
	DefaultTeam  string            `yaml:"default_team,omitempty"`
	TeamPatterns map[string]string `yaml:"team_patterns,omitempty"` // pattern -> team mapping
}

// HookConfig holds hook execution settings
type HookConfig struct {
	Enabled                 bool     `yaml:"enabled"`
	PreToolUse              bool     `yaml:"pre_tool_use"`
	PostToolUse             bool     `yaml:"post_tool_use"`
	Stop                    bool     `yaml:"stop"`
	TaskCompleted           bool     `yaml:"task_completed"`
	SessionEnd              bool     `yaml:"session_end"`
	KnowledgeExtraction     bool     `yaml:"knowledge_extraction"`
	ConflictDetection       bool     `yaml:"conflict_detection"`
	AutoFormat              bool     `yaml:"auto_format"`
	BlockVendorEdits        bool     `yaml:"block_vendor_edits"`
	AllowedVendorPatterns   []string `yaml:"allowed_vendor_patterns,omitempty"`
	CustomPreToolUseScript  string   `yaml:"custom_pre_tool_use_script,omitempty"`
	CustomPostToolUseScript string   `yaml:"custom_post_tool_use_script,omitempty"`
}

// CLIAliasConfig holds CLI alias definitions
type CLIAliasConfig struct {
	Aliases map[string]string `yaml:"aliases,omitempty"` // alias -> command mapping
}

// GlobalConfig represents the global .taskconfig configuration
type GlobalConfig struct {
	TaskIDPrefix   string              `yaml:"task_id_prefix"`
	BasePath       string              `yaml:"base_path,omitempty"`
	Defaults       map[string]string   `yaml:"defaults,omitempty"`
	Notifications  NotificationConfig  `yaml:"notifications"`
	TeamRouting    TeamRoutingConfig   `yaml:"team_routing"`
	Hooks          HookConfig          `yaml:"hooks"`
	Aliases        CLIAliasConfig      `yaml:"aliases"`
	MCPServers     map[string]string   `yaml:"mcp_servers,omitempty"` // name -> URL mapping
	FeatureFlags   map[string]bool     `yaml:"feature_flags,omitempty"`
	CustomSettings map[string]string   `yaml:"custom_settings,omitempty"`
}

// RepoConfig represents the per-repository .taskrc configuration
type RepoConfig struct {
	RepoName          string            `yaml:"repo_name,omitempty"`
	BuildCommand      string            `yaml:"build_command,omitempty"`
	TestCommand       string            `yaml:"test_command,omitempty"`
	LintCommand       string            `yaml:"lint_command,omitempty"`
	Reviewers         []string          `yaml:"reviewers,omitempty"`
	RequiredChecks    []string          `yaml:"required_checks,omitempty"`
	Conventions       []string          `yaml:"conventions,omitempty"`
	BaseBranch        string            `yaml:"base_branch,omitempty"`
	WorktreeBasePath  string            `yaml:"worktree_base_path,omitempty"`
	AutoSync          bool              `yaml:"auto_sync"`
	CustomSettings    map[string]string `yaml:"custom_settings,omitempty"`
}

// MergedConfig represents the combined configuration from global and repo configs
type MergedConfig struct {
	Global *GlobalConfig `yaml:"global"`
	Repo   *RepoConfig   `yaml:"repo"`
}

// DefaultGlobalConfig returns a GlobalConfig with sensible defaults
func DefaultGlobalConfig() *GlobalConfig {
	return &GlobalConfig{
		TaskIDPrefix: "TASK",
		Defaults: map[string]string{
			"priority": "P2",
			"type":     "feat",
		},
		Notifications: NotificationConfig{
			Enabled:  false,
			Channels: []string{},
			OnEvents: []string{},
		},
		TeamRouting: TeamRoutingConfig{
			Enabled:      false,
			TeamPatterns: make(map[string]string),
		},
		Hooks: DefaultHookConfig(),
		Aliases: CLIAliasConfig{
			Aliases: make(map[string]string),
		},
		MCPServers:   make(map[string]string),
		FeatureFlags: make(map[string]bool),
		CustomSettings: make(map[string]string),
	}
}

// DefaultHookConfig returns a HookConfig with Phase 1 features enabled
func DefaultHookConfig() HookConfig {
	return HookConfig{
		Enabled:              true,
		PreToolUse:           true,
		PostToolUse:          true,
		Stop:                 true,
		TaskCompleted:        true,
		SessionEnd:           true,
		KnowledgeExtraction:  false, // Phase 2/3 - opt-in
		ConflictDetection:    false, // Phase 2/3 - opt-in
		AutoFormat:           true,
		BlockVendorEdits:     true,
		AllowedVendorPatterns: []string{},
	}
}

// DefaultRepoConfig returns a RepoConfig with sensible defaults
func DefaultRepoConfig() *RepoConfig {
	return &RepoConfig{
		BaseBranch: "main",
		AutoSync:   false,
		Reviewers:  []string{},
		RequiredChecks: []string{},
		Conventions: []string{},
		CustomSettings: make(map[string]string),
	}
}

// NewMergedConfig creates a new MergedConfig with optional global and repo configs
func NewMergedConfig(global *GlobalConfig, repo *RepoConfig) *MergedConfig {
	if global == nil {
		global = DefaultGlobalConfig()
	}
	if repo == nil {
		repo = DefaultRepoConfig()
	}
	return &MergedConfig{
		Global: global,
		Repo:   repo,
	}
}
