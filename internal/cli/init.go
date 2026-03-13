package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/valter-silva-au/ai-dev-brain/internal/core"
	"github.com/valter-silva-au/ai-dev-brain/templates/claude"
)

// NewInitCmd creates the init command with all subcommands
func NewInitCmd() *cobra.Command {
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize workspace",
		Long:  `Initialize a new AI Dev Brain workspace with scaffolding`,
	}

	// Add subcommands
	initCmd.AddCommand(
		newInitWorkspaceCmd(),
		newInitClaudeCmd(),
		newInitProjectCmd(),
	)

	return initCmd
}

// newInitWorkspaceCmd creates the 'init' command for full workspace scaffolding
func newInitWorkspaceCmd() *cobra.Command {
	var (
		name   string
		ai     string
		prefix string
	)

	cmd := &cobra.Command{
		Use:   "workspace [path]",
		Short: "Initialize a new workspace",
		Long:  `Initialize a new AI Dev Brain workspace with full scaffolding`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetPath := "."
			if len(args) > 0 {
				targetPath = args[0]
			}

			// Create absolute path
			absPath, err := filepath.Abs(targetPath)
			if err != nil {
				return fmt.Errorf("failed to resolve path: %w", err)
			}

			// Check if directory exists
			if _, err := os.Stat(absPath); os.IsNotExist(err) {
				if err := os.MkdirAll(absPath, 0o755); err != nil {
					return fmt.Errorf("failed to create directory: %w", err)
				}
			}

			fmt.Printf("Initializing workspace at %s...\n", absPath)

			// Create directory structure
			dirs := []string{
				"tickets",
				"work",
				"sessions",
				".adb",
			}

			for _, dir := range dirs {
				dirPath := filepath.Join(absPath, dir)
				if err := os.MkdirAll(dirPath, 0o755); err != nil {
					return fmt.Errorf("failed to create directory %s: %w", dir, err)
				}
				fmt.Printf("  ✓ Created %s/\n", dir)
			}

			// Create initial backlog.yaml
			backlogPath := filepath.Join(absPath, "backlog.yaml")
			if _, err := os.Stat(backlogPath); os.IsNotExist(err) {
				backlogContent := "tasks: []\n"
				if err := os.WriteFile(backlogPath, []byte(backlogContent), 0o644); err != nil {
					return fmt.Errorf("failed to create backlog.yaml: %w", err)
				}
				fmt.Println("  ✓ Created backlog.yaml")
			}

			// Create .taskrc config file
			taskrcPath := filepath.Join(absPath, ".taskrc")
			if _, err := os.Stat(taskrcPath); os.IsNotExist(err) {
				workspaceName := name
				if workspaceName == "" {
					workspaceName = filepath.Base(absPath)
				}

				taskIDPrefix := prefix
				if taskIDPrefix == "" {
					taskIDPrefix = "TASK"
				}

				aiProvider := ai
				if aiProvider == "" {
					aiProvider = "claude"
				}

				taskrcContent := fmt.Sprintf(`# AI Dev Brain Repository Configuration
# This file configures the workspace for AI-assisted development

name: "%s"
ai_provider: "%s"
task_id_prefix: "%s"

build:
  command: "go build ./..."
  test_command: "go test ./... -count=1"

git:
  worktree_dir: "work"
  default_branch: "main"

hooks:
  enabled: true
  on_task_create: true
  on_status_change: true
`, workspaceName, aiProvider, taskIDPrefix)

				if err := os.WriteFile(taskrcPath, []byte(taskrcContent), 0o644); err != nil {
					return fmt.Errorf("failed to create .taskrc: %w", err)
				}
				fmt.Println("  ✓ Created .taskrc")
			}

			// Create README.md
			readmePath := filepath.Join(absPath, "README.md")
			if _, err := os.Stat(readmePath); os.IsNotExist(err) {
				workspaceName := name
				if workspaceName == "" {
					workspaceName = filepath.Base(absPath)
				}

				readmeContent := "# " + workspaceName + "\n\n" +
					"This workspace is managed by [AI Dev Brain](https://github.com/valter-silva-au/ai-dev-brain).\n\n" +
					"## Quick Start\n\n" +
					"- Create a task: `adb task create <branch-name>`\n" +
					"- Resume a task: `adb task resume <task-id>`\n" +
					"- View tasks: `adb task status`\n\n" +
					"## Structure\n\n" +
					"- `tickets/` - Task-specific context and notes\n" +
					"- `work/` - Git worktrees for task isolation\n" +
					"- `backlog.yaml` - Task backlog\n" +
					"- `.taskrc` - Workspace configuration\n"

				if err := os.WriteFile(readmePath, []byte(readmeContent), 0o644); err != nil {
					return fmt.Errorf("failed to create README.md: %w", err)
				}
				fmt.Println("  ✓ Created README.md")
			}

			fmt.Printf("\n✓ Workspace initialized at %s\n", absPath)
			fmt.Println("\nNext steps:")
			fmt.Printf("  cd %s\n", absPath)
			fmt.Println("  adb task create <branch-name>")

			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Workspace name")
	cmd.Flags().StringVar(&ai, "ai", "claude", "AI provider (claude, gpt)")
	cmd.Flags().StringVar(&prefix, "prefix", "TASK", "Task ID prefix")

	return cmd
}

// newInitClaudeCmd creates the 'init claude' command
func newInitClaudeCmd() *cobra.Command {
	var managed bool

	cmd := &cobra.Command{
		Use:   "claude [path]",
		Short: "Initialize Claude-specific files",
		Long:  `Initialize Claude configuration and context files`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetPath := "."
			if len(args) > 0 {
				targetPath = args[0]
			}

			// Create absolute path
			absPath, err := filepath.Abs(targetPath)
			if err != nil {
				return fmt.Errorf("failed to resolve path: %w", err)
			}

			fmt.Printf("Initializing Claude files at %s...\n", absPath)

			// Create .adb directory if it doesn't exist
			adbDir := filepath.Join(absPath, ".adb")
			if err := os.MkdirAll(adbDir, 0o755); err != nil {
				return fmt.Errorf("failed to create .adb directory: %w", err)
			}

			// Create CLAUDE.md
			claudePath := filepath.Join(absPath, "CLAUDE.md")
			if _, err := os.Stat(claudePath); os.IsNotExist(err) {
				claudeContent := "# Claude Context\n\n" +
					"This workspace uses AI Dev Brain for task management and AI-assisted development.\n\n" +
					"## Usage\n\n" +
					"- Use `adb task create` to create new tasks\n" +
					"- Each task gets isolated in a git worktree\n" +
					"- Task context is maintained in `tickets/TASK-XXXXX/context.md`\n\n" +
					"## Commands\n\n" +
					"- `adb task status` - View all tasks\n" +
					"- `adb sync context` - Regenerate this file\n" +
					"- `adb metrics` - View workspace metrics\n" +
					"- `adb dashboard` - Open TUI dashboard\n"

				if managed {
					claudeContent += "\n## Managed Mode\n\nThis workspace is in managed mode - context files are auto-regenerated.\n"
				}

				if err := os.WriteFile(claudePath, []byte(claudeContent), 0o644); err != nil {
					return fmt.Errorf("failed to create CLAUDE.md: %w", err)
				}
				fmt.Println("  ✓ Created CLAUDE.md")
			}

			// Create claude-user.md
			userContextPath := filepath.Join(adbDir, "claude-user.md")
			if _, err := os.Stat(userContextPath); os.IsNotExist(err) {
				userContent := "# Claude User Context\n\n" +
					"User-specific preferences and context for Claude AI integration.\n\n" +
					"## Preferences\n\n" +
					"- Code style: Follow project conventions\n" +
					"- Testing: Always include tests\n" +
					"- Documentation: Keep inline comments minimal\n"

				if err := os.WriteFile(userContextPath, []byte(userContent), 0o644); err != nil {
					return fmt.Errorf("failed to create claude-user.md: %w", err)
				}
				fmt.Println("  ✓ Created claude-user.md")
			}

			fmt.Println("\n✓ Claude files initialized")

			return nil
		},
	}

	cmd.Flags().BoolVar(&managed, "managed", false, "Enable managed mode (auto-regeneration)")

	return cmd
}

// newInitProjectCmd creates the 'init project' command using ProjectInitializer
func newInitProjectCmd() *cobra.Command {
	var (
		name       string
		ai         string
		prefix     string
		gitInit    bool
		withBMAD   bool
	)

	cmd := &cobra.Command{
		Use:   "project [path]",
		Short: "Initialize a new project with full scaffolding",
		Long: `Initialize a new project with complete workspace scaffolding including:
- Directory structure (tickets, work, sessions, .adb, .claude)
- Configuration files (.taskrc, backlog.yaml)
- Git repository (optional)
- BMAD artifacts (PRD, tech-spec, architecture-doc, quality gates) (optional)
- Claude integration files`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetPath := "."
			if len(args) > 0 {
				targetPath = args[0]
			}

			// Create absolute path
			absPath, err := filepath.Abs(targetPath)
			if err != nil {
				return fmt.Errorf("failed to resolve path: %w", err)
			}

			fmt.Printf("🚀 Initializing project at %s...\n", absPath)

			// Create ProjectInitializer
			initializer := core.NewFileProjectInitializer(claude.FS)

			// Configure options
			options := core.InitOptions{
				Name:         name,
				AIProvider:   ai,
				TaskIDPrefix: prefix,
				GitInit:      gitInit,
				WithBMAD:     withBMAD,
			}

			// Set defaults
			if options.Name == "" {
				options.Name = filepath.Base(absPath)
			}
			if options.AIProvider == "" {
				options.AIProvider = "claude"
			}
			if options.TaskIDPrefix == "" {
				options.TaskIDPrefix = "TASK"
			}

			// Initialize project
			if err := initializer.InitializeProject(absPath, options); err != nil {
				return fmt.Errorf("failed to initialize project: %w", err)
			}

			fmt.Printf("\n✅ Project initialized successfully!\n\n")
			fmt.Println("📁 Created:")
			fmt.Println("   • Directory structure (tickets, work, sessions, .adb, .claude)")
			fmt.Println("   • Configuration files (.taskrc, backlog.yaml)")
			if gitInit {
				fmt.Println("   • Git repository (.git, .gitignore)")
			}
			if withBMAD {
				fmt.Println("   • BMAD artifacts (docs/bmad/*.md)")
			}
			fmt.Println("   • Claude integration files (.claude/)")

			fmt.Printf("\n💡 Next steps:\n")
			if absPath != "." {
				fmt.Printf("   cd %s\n", absPath)
			}
			fmt.Println("   adb task create <branch-name>  # Create your first task")
			fmt.Println("   adb agents                      # See available agents")
			fmt.Println("   adb team dev \"your task\"        # Launch multi-agent team")

			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Project name")
	cmd.Flags().StringVar(&ai, "ai", "claude", "AI provider (claude, gpt)")
	cmd.Flags().StringVar(&prefix, "prefix", "TASK", "Task ID prefix")
	cmd.Flags().BoolVar(&gitInit, "git", false, "Initialize git repository")
	cmd.Flags().BoolVar(&withBMAD, "bmad", false, "Include BMAD artifacts (PRD, tech-spec, etc.)")

	return cmd
}
