package models

// NotificationConfig holds notification settings
type NotificationConfig struct {
	Enabled  bool     `mapstructure:"enabled" yaml:"enabled"`
	Channels []string `mapstructure:"channels" yaml:"channels,omitempty"`
	OnEvents []string `mapstructure:"on_events" yaml:"on_events,omitempty"`
}

// TeamRoutingConfig holds team routing settings
type TeamRoutingConfig struct {
	Enabled      bool              `mapstructure:"enabled" yaml:"enabled"`
	DefaultTeam  string            `mapstructure:"default_team" yaml:"default_team,omitempty"`
	TeamPatterns map[string]string `mapstructure:"team_patterns" yaml:"team_patterns,omitempty"` // pattern -> team mapping
}

// HookConfig holds hook execution settings
type HookConfig struct {
	Enabled                 bool     `mapstructure:"enabled" yaml:"enabled"`
	PreToolUse              bool     `mapstructure:"pre_tool_use" yaml:"pre_tool_use"`
	PostToolUse             bool     `mapstructure:"post_tool_use" yaml:"post_tool_use"`
	Stop                    bool     `mapstructure:"stop" yaml:"stop"`
	TaskCompleted           bool     `mapstructure:"task_completed" yaml:"task_completed"`
	SessionEnd              bool     `mapstructure:"session_end" yaml:"session_end"`
	KnowledgeExtraction     bool     `mapstructure:"knowledge_extraction" yaml:"knowledge_extraction"`
	ConflictDetection       bool     `mapstructure:"conflict_detection" yaml:"conflict_detection"`
	AutoFormat              bool     `mapstructure:"auto_format" yaml:"auto_format"`
	BlockVendorEdits        bool     `mapstructure:"block_vendor_edits" yaml:"block_vendor_edits"`
	AllowedVendorPatterns   []string `mapstructure:"allowed_vendor_patterns" yaml:"allowed_vendor_patterns,omitempty"`
	CustomPreToolUseScript  string   `mapstructure:"custom_pre_tool_use_script" yaml:"custom_pre_tool_use_script,omitempty"`
	CustomPostToolUseScript string   `mapstructure:"custom_post_tool_use_script" yaml:"custom_post_tool_use_script,omitempty"`
}

// CLIAliasConfig holds CLI alias definitions
type CLIAliasConfig struct {
	Aliases map[string]string `mapstructure:"aliases" yaml:"aliases,omitempty"` // alias -> command mapping
}

// GlobalConfig represents the global .taskconfig configuration
type GlobalConfig struct {
	TaskIDPrefix   string             `mapstructure:"task_id_prefix" yaml:"task_id_prefix"`
	BasePath       string             `mapstructure:"base_path" yaml:"base_path,omitempty"`
	Defaults       map[string]string  `mapstructure:"defaults" yaml:"defaults,omitempty"`
	Notifications  NotificationConfig `mapstructure:"notifications" yaml:"notifications"`
	TeamRouting    TeamRoutingConfig  `mapstructure:"team_routing" yaml:"team_routing"`
	Hooks          HookConfig         `mapstructure:"hooks" yaml:"hooks"`
	Aliases        CLIAliasConfig     `mapstructure:"aliases" yaml:"aliases"`
	MCPServers     map[string]string  `mapstructure:"mcp_servers" yaml:"mcp_servers,omitempty"` // name -> URL mapping
	FeatureFlags   map[string]bool    `mapstructure:"feature_flags" yaml:"feature_flags,omitempty"`
	CustomSettings map[string]string  `mapstructure:"custom_settings" yaml:"custom_settings,omitempty"`
}

// RepoConfig represents the per-repository .taskrc configuration
type RepoConfig struct {
	RepoName         string            `mapstructure:"repo_name" yaml:"repo_name,omitempty"`
	BuildCommand     string            `mapstructure:"build_command" yaml:"build_command,omitempty"`
	TestCommand      string            `mapstructure:"test_command" yaml:"test_command,omitempty"`
	LintCommand      string            `mapstructure:"lint_command" yaml:"lint_command,omitempty"`
	Reviewers        []string          `mapstructure:"reviewers" yaml:"reviewers,omitempty"`
	RequiredChecks   []string          `mapstructure:"required_checks" yaml:"required_checks,omitempty"`
	Conventions      []string          `mapstructure:"conventions" yaml:"conventions,omitempty"`
	BaseBranch       string            `mapstructure:"base_branch" yaml:"base_branch,omitempty"`
	WorktreeBasePath string            `mapstructure:"worktree_base_path" yaml:"worktree_base_path,omitempty"`
	AutoSync         bool              `mapstructure:"auto_sync" yaml:"auto_sync"`
	CustomSettings   map[string]string `mapstructure:"custom_settings" yaml:"custom_settings,omitempty"`
}

// MergedConfig represents the combined configuration from global and repo configs
type MergedConfig struct {
	Global *GlobalConfig `mapstructure:"global" yaml:"global"`
	Repo   *RepoConfig   `mapstructure:"repo" yaml:"repo"`
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
		MCPServers:     make(map[string]string),
		FeatureFlags:   make(map[string]bool),
		CustomSettings: make(map[string]string),
	}
}

// DefaultHookConfig returns a HookConfig with Phase 1 features enabled
func DefaultHookConfig() HookConfig {
	return HookConfig{
		Enabled:               true,
		PreToolUse:            true,
		PostToolUse:           true,
		Stop:                  true,
		TaskCompleted:         true,
		SessionEnd:            true,
		KnowledgeExtraction:   false, // Phase 2/3 - opt-in
		ConflictDetection:     false, // Phase 2/3 - opt-in
		AutoFormat:            true,
		BlockVendorEdits:      true,
		AllowedVendorPatterns: []string{},
	}
}

// DefaultRepoConfig returns a RepoConfig with sensible defaults
func DefaultRepoConfig() *RepoConfig {
	return &RepoConfig{
		BaseBranch:     "main",
		AutoSync:       false,
		Reviewers:      []string{},
		RequiredChecks: []string{},
		Conventions:    []string{},
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
