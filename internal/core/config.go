package core

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"github.com/valter-silva-au/ai-dev-brain/pkg/models"
)

// ConfigurationManager manages configuration loading from multiple sources
type ConfigurationManager interface {
	LoadConfig() (*models.MergedConfig, error)
	GetGlobalConfig() (*models.GlobalConfig, error)
	GetRepoConfig() (*models.RepoConfig, error)
}

// ViperConfigManager implements ConfigurationManager using Viper
type ViperConfigManager struct {
	globalConfigPath string
	repoConfigPath   string
}

// NewViperConfigManager creates a new configuration manager
// If paths are empty, defaults are used:
// - globalConfigPath: ~/.taskconfig
// - repoConfigPath: ./.taskrc
func NewViperConfigManager(globalConfigPath, repoConfigPath string) *ViperConfigManager {
	if globalConfigPath == "" {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			globalConfigPath = filepath.Join(homeDir, ".taskconfig")
		}
	}

	if repoConfigPath == "" {
		repoConfigPath = ".taskrc"
	}

	return &ViperConfigManager{
		globalConfigPath: globalConfigPath,
		repoConfigPath:   repoConfigPath,
	}
}

// GetGlobalConfig loads the global configuration from .taskconfig
func (cm *ViperConfigManager) GetGlobalConfig() (*models.GlobalConfig, error) {
	// Check if global config file exists
	if _, err := os.Stat(cm.globalConfigPath); os.IsNotExist(err) {
		// File doesn't exist, return defaults
		return models.DefaultGlobalConfig(), nil
	}

	// Create a new Viper instance for global config
	v := viper.New()
	v.SetConfigFile(cm.globalConfigPath)
	v.SetConfigType("yaml")

	// Read the config file
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read global config: %w", err)
	}

	// Create empty config struct and unmarshal
	var config models.GlobalConfig
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal global config: %w", err)
	}

	// Apply defaults for any empty fields
	defaults := models.DefaultGlobalConfig()

	if config.TaskIDPrefix == "" {
		config.TaskIDPrefix = defaults.TaskIDPrefix
	}

	if config.Defaults == nil {
		config.Defaults = defaults.Defaults
	}

	if config.MCPServers == nil {
		config.MCPServers = defaults.MCPServers
	}

	if config.FeatureFlags == nil {
		config.FeatureFlags = defaults.FeatureFlags
	}

	if config.CustomSettings == nil {
		config.CustomSettings = defaults.CustomSettings
	}

	if config.Aliases.Aliases == nil {
		config.Aliases.Aliases = defaults.Aliases.Aliases
	}

	return &config, nil
}

// GetRepoConfig loads the per-repository configuration from .taskrc
func (cm *ViperConfigManager) GetRepoConfig() (*models.RepoConfig, error) {
	// Check if repo config file exists
	if _, err := os.Stat(cm.repoConfigPath); os.IsNotExist(err) {
		// File doesn't exist, return defaults
		return models.DefaultRepoConfig(), nil
	}

	// Create a new Viper instance for repo config
	v := viper.New()
	v.SetConfigFile(cm.repoConfigPath)
	v.SetConfigType("yaml")

	// Read the config file
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read repo config: %w", err)
	}

	// Create empty config struct and unmarshal
	var config models.RepoConfig
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal repo config: %w", err)
	}

	// Apply defaults for any empty fields
	defaults := models.DefaultRepoConfig()

	if config.BaseBranch == "" {
		config.BaseBranch = defaults.BaseBranch
	}

	if config.Reviewers == nil {
		config.Reviewers = defaults.Reviewers
	}

	if config.RequiredChecks == nil {
		config.RequiredChecks = defaults.RequiredChecks
	}

	if config.Conventions == nil {
		config.Conventions = defaults.Conventions
	}

	if config.CustomSettings == nil {
		config.CustomSettings = defaults.CustomSettings
	}

	return &config, nil
}

// LoadConfig loads both global and repo configurations with proper precedence
// Precedence: .taskrc (repo) > .taskconfig (global) > defaults
func (cm *ViperConfigManager) LoadConfig() (*models.MergedConfig, error) {
	// Load global config
	globalConfig, err := cm.GetGlobalConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load global config: %w", err)
	}

	// Load repo config
	repoConfig, err := cm.GetRepoConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load repo config: %w", err)
	}

	// Create merged config
	merged := models.NewMergedConfig(globalConfig, repoConfig)

	return merged, nil
}

// DefaultHookConfig returns a HookConfig with Phase 1 features enabled
// This is a convenience re-export from the models package
func DefaultHookConfig() models.HookConfig {
	return models.DefaultHookConfig()
}
