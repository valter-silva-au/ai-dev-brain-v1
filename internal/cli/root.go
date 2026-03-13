package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewRootCmd creates the root command for the ADB CLI
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "adb",
		Short: "AI Dev Brain - Task management and workflow automation",
		Long: `AI Dev Brain (adb) is a task management and workflow automation tool
that integrates with git worktrees, Claude Code, and terminal environments.`,
		Version: fmt.Sprintf("%s (commit: %s, built: %s)", Version, Commit, Date),
		SilenceUsage: true,
	}

	// Add subcommands
	rootCmd.AddCommand(NewTaskCmd())
	rootCmd.AddCommand(NewSessionCmd())
	rootCmd.AddCommand(NewSyncCmd())
	rootCmd.AddCommand(NewInitCmd())
	rootCmd.AddCommand(NewExecCmd())
	rootCmd.AddCommand(NewRunCmd())
	rootCmd.AddCommand(NewMetricsCmd())
	rootCmd.AddCommand(NewAlertsCmd())
	rootCmd.AddCommand(NewDashboardCmd())
	rootCmd.AddCommand(NewHookCmd())
	rootCmd.AddCommand(NewVersionCmd())

	return rootCmd
}
