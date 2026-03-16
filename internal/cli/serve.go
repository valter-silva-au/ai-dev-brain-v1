package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/valter-silva-au/ai-dev-brain/internal/hive"
	"github.com/valter-silva-au/ai-dev-brain/internal/server"
)

const defaultPort = 8400

// NewServeCmd creates the serve command with start/stop/restart/status subcommands
func NewServeCmd() *cobra.Command {
	var (
		port int
		tv   bool
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the web dashboard",
		Long: `Start the MyImaginationAI Agent Command Center — a live-updating HTMX dashboard.

Run in foreground:  adb serve [--port 8400] [--tv]
Run as daemon:      adb serve start [--port 8400] [--tv]
Stop daemon:        adb serve stop
Restart daemon:     adb serve restart [--port 8400] [--tv]
Check status:       adb serve status`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// No subcommand — run in foreground
			return runForeground(port, tv)
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", defaultPort, "Server port")
	cmd.Flags().BoolVar(&tv, "tv", false, "Open browser automatically (TV mode)")

	// Subcommands
	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start dashboard as a background daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			return daemonStart(port, tv)
		},
	}
	startCmd.Flags().IntVarP(&port, "port", "p", defaultPort, "Server port")
	startCmd.Flags().BoolVar(&tv, "tv", false, "Open browser automatically")

	stopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop the dashboard daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			return daemonStop()
		},
	}

	restartCmd := &cobra.Command{
		Use:   "restart",
		Short: "Restart the dashboard daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = daemonStop()
			return daemonStart(port, tv)
		},
	}
	restartCmd.Flags().IntVarP(&port, "port", "p", defaultPort, "Server port")
	restartCmd.Flags().BoolVar(&tv, "tv", false, "Open browser automatically")

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Check if the dashboard daemon is running",
		RunE: func(cmd *cobra.Command, args []string) error {
			return daemonStatus()
		},
	}

	cmd.AddCommand(startCmd, stopCmd, restartCmd, statusCmd)
	return cmd
}

// pidFilePath returns the path to the PID file
func pidFilePath() string {
	base := os.Getenv("ADB_HOME")
	if base == "" {
		base = "."
	}
	return filepath.Join(base, ".adb_serve.pid")
}

// runForeground starts the server in the current process (blocking)
func runForeground(port int, tv bool) error {
	if App == nil {
		return fmt.Errorf("app not initialized")
	}

	agentReg := hive.NewAgentRegistry(App.BasePath)
	projectReg := hive.NewProjectRegistry(App.BasePath)
	knowledgeAgg := hive.NewKnowledgeAggregator(App.BasePath, projectReg)
	messageBus := hive.NewMessageBus(App.BasePath)

	openclawPath := os.ExpandEnv("$HOME/.openclaw")
	if _, err := agentReg.DiscoverOpenClaw(openclawPath); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: OpenClaw discovery failed: %v\n", err)
	}

	srv := server.NewServer(App, agentReg, projectReg, knowledgeAgg, messageBus)
	addr := fmt.Sprintf("127.0.0.1:%d", port)

	if tv {
		go openBrowser(fmt.Sprintf("http://%s", addr))
	}

	fmt.Printf("MyImaginationAI Agent Command Center\n")
	fmt.Printf("Dashboard: http://%s\n", addr)
	fmt.Printf("Press Ctrl+C to stop\n\n")

	return srv.Start(addr)
}

// daemonStart launches the server as a background process
func daemonStart(port int, tv bool) error {
	// Check if already running
	if pid, running := readPID(); running {
		return fmt.Errorf("dashboard already running (PID %d). Use 'adb serve restart' to restart", pid)
	}

	// Find the adb binary
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot find adb binary: %w", err)
	}

	// Build args for the foreground serve command
	args := []string{"serve", "--port", strconv.Itoa(port)}
	if tv {
		args = append(args, "--tv")
	}

	// Start as detached background process
	cmd := exec.Command(exe, args...)
	cmd.Env = os.Environ()
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	pid := cmd.Process.Pid

	// Write PID file
	if err := os.WriteFile(pidFilePath(), []byte(strconv.Itoa(pid)), 0o644); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	// Detach — don't wait for the child
	_ = cmd.Process.Release()

	fmt.Printf("✓ Dashboard started (PID %d)\n", pid)
	fmt.Printf("  http://127.0.0.1:%d\n", port)
	fmt.Printf("  Stop with: adb serve stop\n")

	if tv {
		openBrowser(fmt.Sprintf("http://127.0.0.1:%d", port))
	}

	return nil
}

// daemonStop kills the background server
func daemonStop() error {
	pid, running := readPID()
	if !running {
		fmt.Println("Dashboard is not running.")
		return nil
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("cannot find process %d: %w", pid, err)
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		// Process might already be dead
		_ = os.Remove(pidFilePath())
		fmt.Printf("✓ Dashboard stopped (PID %d was not running)\n", pid)
		return nil
	}

	_ = os.Remove(pidFilePath())
	fmt.Printf("✓ Dashboard stopped (PID %d)\n", pid)
	return nil
}

// daemonStatus checks if the daemon is running
func daemonStatus() error {
	pid, running := readPID()
	if running {
		fmt.Printf("✓ Dashboard running (PID %d)\n", pid)
		fmt.Printf("  http://127.0.0.1:%d\n", defaultPort)
	} else if pid > 0 {
		fmt.Printf("✗ Dashboard not running (stale PID %d)\n", pid)
		_ = os.Remove(pidFilePath())
	} else {
		fmt.Println("✗ Dashboard not running")
	}
	return nil
}

// readPID reads the PID file and checks if the process is alive
func readPID() (int, bool) {
	data, err := os.ReadFile(pidFilePath())
	if err != nil {
		return 0, false
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, false
	}

	// Check if process is alive
	process, err := os.FindProcess(pid)
	if err != nil {
		return pid, false
	}

	// On Unix, FindProcess always succeeds — use Signal(0) to check
	err = process.Signal(syscall.Signal(0))
	return pid, err == nil
}

// openBrowser opens the default browser
func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		return
	}
	_ = cmd.Start()
}
