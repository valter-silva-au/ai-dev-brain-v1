package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/valter-silva-au/ai-dev-brain/internal/core"
)

// NewSyncCmd creates the sync command with all subcommands
func NewSyncCmd() *cobra.Command {
	syncCmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync and regenerate context files",
		Long:  `Commands for regenerating context files for AI agents`,
	}

	// Add subcommands
	syncCmd.AddCommand(
		newSyncContextCmd(),
		newSyncTaskContextCmd(),
		newSyncReposCmd(),
		newSyncClaudeUserCmd(),
		newSyncAllCmd(),
	)

	return syncCmd
}

// newSyncContextCmd creates the 'sync context' command
func newSyncContextCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "context",
		Short: "Regenerate CLAUDE.md",
		Long:  `Regenerate the main CLAUDE.md context file from backlog and task data`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if App == nil {
				return fmt.Errorf("app not initialized")
			}

			// Create context generator
			contextGen := core.NewContextGenerator(
				App.BasePath+"/backlog.yaml",
				App.BasePath+"/tickets",
				App.BasePath,
				App.TemplateManager,
			)

			fmt.Println("Regenerating CLAUDE.md...")
			if err := contextGen.GenerateContext(); err != nil {
				return fmt.Errorf("failed to regenerate context: %w", err)
			}

			fmt.Println("✓ CLAUDE.md regenerated")
			return nil
		},
	}

	return cmd
}

// newSyncTaskContextCmd creates the 'sync task-context' command
func newSyncTaskContextCmd() *cobra.Command {
	var hookMode bool

	cmd := &cobra.Command{
		Use:   "task-context <task-id>",
		Short: "Regenerate task-specific context",
		Long:  `Regenerate context.md for a specific task`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if App == nil {
				return fmt.Errorf("app not initialized")
			}

			taskID := args[0]

			// Create context generator
			contextGen := core.NewContextGenerator(
				App.BasePath+"/backlog.yaml",
				App.BasePath+"/tickets",
				App.BasePath,
				App.TemplateManager,
			)

			fmt.Printf("Regenerating context for %s...\n", taskID)
			if err := contextGen.GenerateTaskContext(taskID, hookMode); err != nil {
				return fmt.Errorf("failed to regenerate task context: %w", err)
			}

			fmt.Printf("✓ Task context regenerated for %s\n", taskID)
			return nil
		},
	}

	cmd.Flags().BoolVar(&hookMode, "hook-mode", false, "Run in hook mode (append timestamp only)")

	return cmd
}

// newSyncReposCmd creates the 'sync repos' command
func newSyncReposCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repos",
		Short: "Regenerate repository context",
		Long:  `Regenerate repository structure context`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if App == nil {
				return fmt.Errorf("app not initialized")
			}

			// Create context generator
			contextGen := core.NewContextGenerator(
				App.BasePath+"/backlog.yaml",
				App.BasePath+"/tickets",
				App.BasePath,
				App.TemplateManager,
			)

			fmt.Println("Regenerating repository context...")
			if err := contextGen.GenerateRepoContext(); err != nil {
				return fmt.Errorf("failed to regenerate repo context: %w", err)
			}

			fmt.Println("✓ Repository context regenerated")
			return nil
		},
	}

	return cmd
}

// newSyncClaudeUserCmd creates the 'sync claude-user' command
func newSyncClaudeUserCmd() *cobra.Command {
	var (
		dryRun bool
		mcp    bool
	)

	cmd := &cobra.Command{
		Use:   "claude-user",
		Short: "Regenerate Claude user context",
		Long:  `Regenerate Claude-specific user context files`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if App == nil {
				return fmt.Errorf("app not initialized")
			}

			// Create context generator
			contextGen := core.NewContextGenerator(
				App.BasePath+"/backlog.yaml",
				App.BasePath+"/tickets",
				App.BasePath,
				App.TemplateManager,
			)

			fmt.Println("Regenerating Claude user context...")
			if err := contextGen.GenerateClaudeUserContext(dryRun, mcp); err != nil {
				return fmt.Errorf("failed to regenerate Claude user context: %w", err)
			}

			if !dryRun {
				fmt.Println("✓ Claude user context regenerated")
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without writing")
	cmd.Flags().BoolVar(&mcp, "mcp", false, "Include MCP integration context")

	return cmd
}

// newSyncAllCmd creates the 'sync all' command
func newSyncAllCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "all",
		Short: "Regenerate all context files",
		Long:  `Regenerate all context files (CLAUDE.md, repo context, user context)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if App == nil {
				return fmt.Errorf("app not initialized")
			}

			// Create context generator
			contextGen := core.NewContextGenerator(
				App.BasePath+"/backlog.yaml",
				App.BasePath+"/tickets",
				App.BasePath,
				App.TemplateManager,
			)

			fmt.Println("Regenerating all context files...")
			if err := contextGen.GenerateAll(); err != nil {
				return fmt.Errorf("failed to regenerate context files: %w", err)
			}

			fmt.Println("✓ All context files regenerated")
			return nil
		},
	}

	return cmd
}
