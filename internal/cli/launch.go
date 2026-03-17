package cli

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/valter-silva-au/ai-dev-brain/internal/integration"
)

// launchWorkflow launches the workflow for a task:
// 1. Renames terminal tab (macOS/iTerm2 compatible)
// 2. Updates terminal state
// 3. Launches Claude Code in the worktree
// 4. Drops user into an interactive shell
func launchWorkflow(taskID, worktreePath string, resume bool) error {
	// Update terminal tab title
	if err := renameTerminalTab(taskID); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to rename terminal tab: %v\n", err)
	}

	// Update terminal state via App
	if App != nil && App.TerminalStateWriter != nil {
		termState := integration.TerminalState{
			WorktreePath: worktreePath,
			TaskID:       taskID,
			Status:       "active",
			LastUpdated:  time.Now().UTC().Format(time.RFC3339),
		}
		if err := App.TerminalStateWriter.WriteState(termState); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to update terminal state: %v\n", err)
		}
	}

	// Launch Claude Code in worktree
	fmt.Printf("Opening Claude Code in %s...\n", worktreePath)
	if err := launchClaudeCode(worktreePath, resume); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to launch Claude Code: %v\n", err)

		// Drop into interactive shell as fallback
		fmt.Println("\nDropping into interactive shell...")
		fmt.Printf("Working directory: %s\n", worktreePath)
		fmt.Println("Type 'exit' to return to the main shell.")

		return launchInteractiveShell(taskID, worktreePath)
	}

	return nil
}

// renameTerminalTab renames the terminal tab using escape sequences
func renameTerminalTab(taskID string) error {
	title := fmt.Sprintf("ADB: %s", taskID)

	// Use ANSI escape sequences to set terminal title
	// This works for most modern terminals (iTerm2, Terminal.app, gnome-terminal, etc.)
	fmt.Printf("\033]0;%s\007", title)

	return nil
}

// launchClaudeCode launches Claude Code in the specified directory.
// If resume is true, it passes --continue to resume the most recent conversation.
func launchClaudeCode(path string, resume bool) error {
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return fmt.Errorf("claude CLI not found in PATH: %w", err)
	}

	args := []string{}
	if resume {
		args = append(args, "--continue")
	}

	cmd := exec.Command(claudePath, args...)
	cmd.Dir = path
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("claude exited with error: %w", err)
	}

	return nil
}

// launchInteractiveShell launches an interactive shell in the specified directory
func launchInteractiveShell(taskID, path string) error {
	// Determine shell
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash" // Default to bash
	}

	// Create command
	cmd := exec.Command(shell)
	cmd.Dir = path
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set environment variables
	env := os.Environ()
	env = append(env, fmt.Sprintf("ADB_TASK_ID=%s", taskID))
	env = append(env, fmt.Sprintf("ADB_WORKTREE_PATH=%s", path))
	cmd.Env = env

	// Run shell
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("shell exited with error: %w", err)
	}

	return nil
}
