package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/valter-silva-au/ai-dev-brain/internal/core"
	"github.com/valter-silva-au/ai-dev-brain/pkg/models"
)

// NewTaskCmd creates the task command with all subcommands
func NewTaskCmd() *cobra.Command {
	taskCmd := &cobra.Command{
		Use:   "task",
		Short: "Manage task lifecycle",
		Long:  `Commands for creating, resuming, archiving, and managing tasks`,
	}

	// Add subcommands
	taskCmd.AddCommand(
		newTaskCreateCmd(),
		newTaskResumeCmd(),
		newTaskArchiveCmd(),
		newTaskUnarchiveCmd(),
		newTaskCleanupCmd(),
		newTaskStatusCmd(),
		newTaskPriorityCmd(),
		newTaskUpdateCmd(),
	)

	return taskCmd
}

// newTaskCreateCmd creates the 'task create' command
func newTaskCreateCmd() *cobra.Command {
	var (
		taskType    string
		repo        string
		priority    string
		owner       string
		tags        []string
		description string
		acceptance  []string
	)

	cmd := &cobra.Command{
		Use:   "create <branch>",
		Short: "Create a new task",
		Long:  `Create a new task with worktree and branch isolation`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if App == nil {
				return fmt.Errorf("app not initialized")
			}

			branch := args[0]

			// Validate task type
			var tt models.TaskType
			switch taskType {
			case "feat":
				tt = models.TaskTypeFeat
			case "bug":
				tt = models.TaskTypeBug
			case "spike":
				tt = models.TaskTypeSpike
			case "refactor":
				tt = models.TaskTypeRefactor
			default:
				return fmt.Errorf("invalid task type: %s (must be feat, bug, spike, or refactor)", taskType)
			}

			// Validate priority
			var p models.Priority
			switch priority {
			case "P0":
				p = models.PriorityP0
			case "P1":
				p = models.PriorityP1
			case "P2":
				p = models.PriorityP2
			case "P3":
				p = models.PriorityP3
			default:
				return fmt.Errorf("invalid priority: %s (must be P0, P1, P2, or P3)", priority)
			}

			// Create task options
			opts := core.CreateTaskOpts{
				Title:              fmt.Sprintf("[%s] %s", taskType, branch),
				Description:        description,
				AcceptanceCriteria: acceptance,
				TaskType:           tt,
				Priority:           p,
				Owner:              owner,
				Tags:               tags,
				Repo:               repo,
			}

			task, err := App.TaskManager.Create(opts)
			if err != nil {
				return fmt.Errorf("failed to create task: %w", err)
			}

			fmt.Printf("✓ Task %s created\n", task.ID)
			fmt.Printf("  Branch: %s\n", task.Branch)
			fmt.Printf("  Worktree: %s\n", task.WorktreePath)
			fmt.Printf("  Ticket: %s\n", task.TicketPath)

			// Launch workflow if worktree was created
			if task.WorktreePath != "" {
				fmt.Println("\nLaunching workflow...")
				return launchWorkflow(task.ID, task.WorktreePath)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&taskType, "type", "feat", "Task type (feat, bug, spike, refactor)")
	cmd.Flags().StringVar(&repo, "repo", "", "Repository name")
	cmd.Flags().StringVar(&priority, "priority", "P2", "Priority (P0, P1, P2, P3)")
	cmd.Flags().StringVar(&owner, "owner", "", "Task owner")
	cmd.Flags().StringSliceVar(&tags, "tags", []string{}, "Task tags (comma-separated)")
	cmd.Flags().StringVar(&description, "description", "", "Task description")
	cmd.Flags().StringSliceVar(&acceptance, "acceptance", []string{}, "Acceptance criteria (comma-separated)")

	return cmd
}

// newTaskResumeCmd creates the 'task resume' command
func newTaskResumeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resume <task-id>",
		Short: "Resume a task",
		Long:  `Resume a task, promoting it from backlog to in_progress and launching the workflow`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if App == nil {
				return fmt.Errorf("app not initialized")
			}

			taskID := args[0]

			task, err := App.TaskManager.Resume(taskID)
			if err != nil {
				return fmt.Errorf("failed to resume task: %w", err)
			}

			fmt.Printf("✓ Task %s resumed\n", task.ID)
			fmt.Printf("  Status: %s\n", task.Status)
			fmt.Printf("  Worktree: %s\n", task.WorktreePath)

			// Launch workflow if worktree exists
			if task.WorktreePath != "" {
				fmt.Println("\nLaunching workflow...")
				return launchWorkflow(task.ID, task.WorktreePath)
			}

			return nil
		},
	}

	return cmd
}

// newTaskArchiveCmd creates the 'task archive' command
func newTaskArchiveCmd() *cobra.Command {
	var (
		force        bool
		keepWorktree bool
	)

	cmd := &cobra.Command{
		Use:   "archive <task-id>",
		Short: "Archive a task",
		Long:  `Archive a task by moving it to _archived/ and removing its worktree`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if App == nil {
				return fmt.Errorf("app not initialized")
			}

			taskID := args[0]

			// Check if worktree should be kept
			if keepWorktree {
				// Load task to get worktree path
				task, err := App.BacklogManager.GetTask(taskID)
				if err != nil {
					return fmt.Errorf("failed to load task: %w", err)
				}

				// Store worktree path
				worktreePath := task.WorktreePath

				// Archive without removing worktree (we'll manually skip it)
				// For now, just print a warning since the TaskManager.Archive always removes worktree
				if worktreePath != "" {
					fmt.Printf("Warning: --keep-worktree is not yet implemented. Worktree will be removed.\n")
				}
			}

			if err := App.TaskManager.Archive(taskID); err != nil {
				if !force {
					return fmt.Errorf("failed to archive task: %w", err)
				}
				fmt.Fprintf(os.Stderr, "Warning: archive completed with errors: %v\n", err)
			}

			fmt.Printf("✓ Task %s archived\n", taskID)
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Force archive even if errors occur")
	cmd.Flags().BoolVar(&keepWorktree, "keep-worktree", false, "Keep worktree after archiving")

	return cmd
}

// newTaskUnarchiveCmd creates the 'task unarchive' command
func newTaskUnarchiveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unarchive <task-id>",
		Short: "Unarchive a task",
		Long:  `Unarchive a task by moving it back from _archived/ to active tickets`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if App == nil {
				return fmt.Errorf("app not initialized")
			}

			taskID := args[0]

			if err := App.TaskManager.Unarchive(taskID); err != nil {
				return fmt.Errorf("failed to unarchive task: %w", err)
			}

			fmt.Printf("✓ Task %s unarchived\n", taskID)
			return nil
		},
	}

	return cmd
}

// newTaskCleanupCmd creates the 'task cleanup' command
func newTaskCleanupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cleanup <task-id>",
		Short: "Clean up a task's worktree",
		Long:  `Remove only the worktree for a task, leaving ticket data intact`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if App == nil {
				return fmt.Errorf("app not initialized")
			}

			taskID := args[0]

			if err := App.TaskManager.Cleanup(taskID); err != nil {
				return fmt.Errorf("failed to cleanup task: %w", err)
			}

			fmt.Printf("✓ Task %s worktree cleaned up\n", taskID)
			return nil
		},
	}

	return cmd
}

// newTaskStatusCmd creates the 'task status' command
func newTaskStatusCmd() *cobra.Command {
	var filterStatus string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "List tasks by status",
		Long:  `List all tasks, optionally filtered by status`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if App == nil {
				return fmt.Errorf("app not initialized")
			}

			backlog, err := App.BacklogManager.Load()
			if err != nil {
				return fmt.Errorf("failed to load backlog: %w", err)
			}

			// Filter tasks by status if specified
			tasks := backlog.Tasks
			if filterStatus != "" {
				var filtered []models.Task
				for _, task := range tasks {
					if string(task.Status) == filterStatus {
						filtered = append(filtered, task)
					}
				}
				tasks = filtered
			}

			if len(tasks) == 0 {
				fmt.Println("No tasks found")
				return nil
			}

			// Group by status
			byStatus := make(map[models.TaskStatus][]models.Task)
			for _, task := range tasks {
				byStatus[task.Status] = append(byStatus[task.Status], task)
			}

			// Print grouped by status
			statuses := []models.TaskStatus{
				models.TaskStatusInProgress,
				models.TaskStatusReview,
				models.TaskStatusBlocked,
				models.TaskStatusBacklog,
				models.TaskStatusDone,
				models.TaskStatusArchived,
			}

			for _, status := range statuses {
				tasks := byStatus[status]
				if len(tasks) == 0 {
					continue
				}

				fmt.Printf("\n%s (%d):\n", strings.ToUpper(string(status)), len(tasks))
				for _, task := range tasks {
					fmt.Printf("  %s: [%s] %s [%s] (owner: %s)\n",
						task.ID, task.Type, task.Title, task.Priority, task.Owner)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&filterStatus, "filter", "", "Filter by status (backlog, in_progress, blocked, review, done, archived)")

	return cmd
}

// newTaskPriorityCmd creates the 'task priority' command
func newTaskPriorityCmd() *cobra.Command {
	var newPriority string

	cmd := &cobra.Command{
		Use:   "priority <task-id>...",
		Short: "Update task priority",
		Long:  `Update the priority of one or more tasks`,
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if App == nil {
				return fmt.Errorf("app not initialized")
			}

			// Validate priority
			var p models.Priority
			switch newPriority {
			case "P0":
				p = models.PriorityP0
			case "P1":
				p = models.PriorityP1
			case "P2":
				p = models.PriorityP2
			case "P3":
				p = models.PriorityP3
			default:
				return fmt.Errorf("invalid priority: %s (must be P0, P1, P2, or P3)", newPriority)
			}

			// Update each task
			for _, taskID := range args {
				if err := App.TaskManager.UpdatePriority(taskID, p); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to update %s: %v\n", taskID, err)
					continue
				}
				fmt.Printf("✓ Task %s priority updated to %s\n", taskID, newPriority)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&newPriority, "priority", "", "New priority (P0, P1, P2, P3)")
	cmd.MarkFlagRequired("priority")

	return cmd
}

// newTaskUpdateCmd creates the 'task update' command
func newTaskUpdateCmd() *cobra.Command {
	var (
		status   string
		priority string
		owner    string
	)

	cmd := &cobra.Command{
		Use:   "update <task-id>",
		Short: "Update task properties",
		Long:  `Update various properties of a task`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if App == nil {
				return fmt.Errorf("app not initialized")
			}

			taskID := args[0]

			// Load task
			task, err := App.BacklogManager.GetTask(taskID)
			if err != nil {
				return fmt.Errorf("failed to load task: %w", err)
			}

			updated := false

			// Update status if provided
			if status != "" {
				var s models.TaskStatus
				switch status {
				case "backlog":
					s = models.TaskStatusBacklog
				case "in_progress":
					s = models.TaskStatusInProgress
				case "blocked":
					s = models.TaskStatusBlocked
				case "review":
					s = models.TaskStatusReview
				case "done":
					s = models.TaskStatusDone
				case "archived":
					s = models.TaskStatusArchived
				default:
					return fmt.Errorf("invalid status: %s", status)
				}

				if err := App.TaskManager.UpdateStatus(taskID, s); err != nil {
					return fmt.Errorf("failed to update status: %w", err)
				}
				fmt.Printf("✓ Status updated to %s\n", status)
				updated = true
			}

			// Update priority if provided
			if priority != "" {
				var p models.Priority
				switch priority {
				case "P0":
					p = models.PriorityP0
				case "P1":
					p = models.PriorityP1
				case "P2":
					p = models.PriorityP2
				case "P3":
					p = models.PriorityP3
				default:
					return fmt.Errorf("invalid priority: %s", priority)
				}

				if err := App.TaskManager.UpdatePriority(taskID, p); err != nil {
					return fmt.Errorf("failed to update priority: %w", err)
				}
				fmt.Printf("✓ Priority updated to %s\n", priority)
				updated = true
			}

			// Update owner if provided
			if owner != "" {
				task.Owner = owner
				task.UpdateTimestamp()
				if err := App.BacklogManager.UpdateTask(*task); err != nil {
					return fmt.Errorf("failed to update owner: %w", err)
				}
				fmt.Printf("✓ Owner updated to %s\n", owner)
				updated = true
			}

			if !updated {
				fmt.Println("No updates specified. Use --status, --priority, or --owner flags.")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&status, "status", "", "New status")
	cmd.Flags().StringVar(&priority, "priority", "", "New priority")
	cmd.Flags().StringVar(&owner, "owner", "", "New owner")

	return cmd
}
