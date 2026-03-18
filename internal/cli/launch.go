package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/valter-silva-au/ai-dev-brain/internal/integration"
)

// taskLaunchInfo carries task metadata through the launch workflow
type taskLaunchInfo struct {
	TaskID       string `json:"task_id"`
	TaskType     string `json:"task_type"`
	Priority     string `json:"priority"`
	Status       string `json:"status"`
	WorktreePath string `json:"worktree_path"`
	Branch       string `json:"branch"`
	Resume       bool   `json:"resume"`
	Timestamp    string `json:"timestamp"`
}

// launchWorkflow launches the workflow for a task.
// In VS Code, it delegates to the extension for styled terminal creation.
// In plain terminals, it launches Claude Code directly.
func launchWorkflow(info taskLaunchInfo) error {
	// Update terminal state
	if App != nil && App.TerminalStateWriter != nil {
		termState := integration.TerminalState{
			WorktreePath: info.WorktreePath,
			TaskID:       info.TaskID,
			Status:       "active",
			LastUpdated:  time.Now().UTC().Format(time.RFC3339),
		}
		if err := App.TerminalStateWriter.WriteState(termState); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to update terminal state: %v\n", err)
		}
	}

	// VS Code: write launch file for the extension to pick up
	if os.Getenv("TERM_PROGRAM") == "vscode" {
		return launchViaVSCode(info)
	}

	// Plain terminal: rename tab and launch directly
	title := fmt.Sprintf("%s %s %s", info.TaskID, info.TaskType, info.Priority)
	fmt.Printf("\033]0;%s\007", title)

	fmt.Printf("Opening Claude Code in %s...\n", info.WorktreePath)
	if err := launchClaudeCode(info.WorktreePath, info.Resume); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to launch Claude Code: %v\n", err)

		fmt.Println("\nDropping into interactive shell...")
		fmt.Printf("Working directory: %s\n", info.WorktreePath)
		fmt.Println("Type 'exit' to return to the main shell.")

		return launchInteractiveShell(info.TaskID, info.WorktreePath)
	}

	return nil
}

// launchViaVSCode writes a launch request for the VS Code extension
func launchViaVSCode(info taskLaunchInfo) error {
	info.Timestamp = time.Now().UTC().Format(time.RFC3339)

	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal launch info: %w", err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	launchFile := filepath.Join(homeDir, ".adb_terminal_launch.json")
	if err := os.WriteFile(launchFile, data, 0o644); err != nil {
		return fmt.Errorf("failed to write launch file: %w", err)
	}

	fmt.Printf("Launching styled terminal for %s %s %s...\n", info.TaskID, info.TaskType, info.Priority)
	return nil
}

// launchClaudeCode launches Claude Code in the specified directory.
func launchClaudeCode(path string, resume bool) error {
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return fmt.Errorf("claude CLI not found in PATH: %w", err)
	}

	args := []string{"--dangerously-skip-permissions"}
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
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}

	cmd := exec.Command(shell)
	cmd.Dir = path
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	env := os.Environ()
	env = append(env, fmt.Sprintf("ADB_TASK_ID=%s", taskID))
	env = append(env, fmt.Sprintf("ADB_WORKTREE_PATH=%s", path))
	cmd.Env = env

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("shell exited with error: %w", err)
	}

	return nil
}
