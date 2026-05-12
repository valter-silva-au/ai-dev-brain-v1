package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/valter-silva-au/ai-dev-brain/internal/integration"
	"github.com/valter-silva-au/ai-dev-brain/internal/scheduler"
)

// NewSchedulerCmd creates the `adb scheduler` command group.
func NewSchedulerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scheduler",
		Short: "Run recurring maintenance jobs in the background",
		Long: `adb has a small built-in scheduler that runs recurring jobs:

  repos-pull     fetch + fast-forward every repo under <workspace>/repos
  alerts-tick    evaluate alert conditions and log transitions
  events-rotate  size-check the event and scheduler logs, rotate if large

Start:    adb scheduler start
Stop:     adb scheduler stop
List:     adb scheduler list
Status:   adb scheduler status

Or run foreground (what the detached daemon invokes):
          adb scheduler run`,
	}
	cmd.AddCommand(
		newSchedulerStartCmd(),
		newSchedulerStopCmd(),
		newSchedulerRestartCmd(),
		newSchedulerStatusCmd(),
		newSchedulerRunCmd(),
		newSchedulerListCmd(),
	)
	return cmd
}

// ---- paths ----

func schedulerPIDPath() string {
	base := schedulerBase()
	return filepath.Join(base, ".adb_scheduler.pid")
}

func schedulerLogPath() string {
	base := schedulerBase()
	return filepath.Join(base, ".adb_scheduler.log")
}

func schedulerStatePath() string {
	base := schedulerBase()
	return filepath.Join(base, ".adb_scheduler_state.yaml")
}

func schedulerBase() string {
	if App != nil && App.BasePath != "" {
		return App.BasePath
	}
	if p := os.Getenv("ADB_HOME"); p != "" {
		return p
	}
	return "."
}

// ---- subcommands ----

func newSchedulerStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start the scheduler as a background daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			return schedulerDaemonStart()
		},
	}
}

func newSchedulerStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the scheduler daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			return schedulerDaemonStop()
		},
	}
}

func newSchedulerRestartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restart",
		Short: "Restart the scheduler daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = schedulerDaemonStop()
			return schedulerDaemonStart()
		},
	}
}

func newSchedulerStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Report whether the scheduler daemon is running",
		RunE: func(cmd *cobra.Command, args []string) error {
			pid, alive := readPIDFile(schedulerPIDPath())
			switch {
			case alive:
				fmt.Printf("✓ Scheduler running (PID %d)\n", pid)
				fmt.Printf("  log:   %s\n", schedulerLogPath())
				fmt.Printf("  state: %s\n", schedulerStatePath())
			case pid > 0:
				fmt.Printf("✗ Scheduler not running (stale PID %d)\n", pid)
				_ = os.Remove(schedulerPIDPath())
			default:
				fmt.Println("✗ Scheduler not running")
			}
			return nil
		},
	}
}

func newSchedulerRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "Run the scheduler in the foreground (used by the daemon)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return schedulerRunForeground()
		},
	}
}

func newSchedulerListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List scheduler jobs and their last-run state",
		RunE: func(cmd *cobra.Command, args []string) error {
			states, err := scheduler.LoadStates(schedulerStatePath())
			if err != nil {
				return fmt.Errorf("load state: %w", err)
			}
			jobs := scheduler.DefaultJobs(scheduler.Deps{})
			byName := make(map[string]scheduler.State)
			for _, s := range states {
				byName[s.Name] = s
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "JOB\tINTERVAL\tRUNS\tFAILURES\tSKIPPED\tLAST_RUN\tLAST_DURATION\tLAST_ERROR")
			for _, j := range jobs {
				s := byName[j.Name]
				lastRun := "-"
				if !s.LastStart.IsZero() {
					lastRun = s.LastStart.Local().Format(time.RFC3339)
				}
				fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%d\t%s\t%s\t%s\n",
					j.Name,
					j.DefaultInterval,
					s.Runs,
					s.Failures,
					s.Skipped,
					lastRun,
					s.LastDuration,
					truncateText(s.LastError, 60),
				)
			}
			return w.Flush()
		},
	}
}

func truncateText(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

// ---- daemon lifecycle (mirrors serve.go) ----

func schedulerDaemonStart() error {
	if pid, alive := readPIDFile(schedulerPIDPath()); alive {
		return fmt.Errorf("scheduler already running (PID %d). Use 'adb scheduler restart'", pid)
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot find adb binary: %w", err)
	}

	cmd := exec.Command(exe, "scheduler", "run")
	cmd.Env = os.Environ()
	cmd.Stdout = nil
	cmd.Stderr = nil
	detachProcess(cmd)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start scheduler daemon: %w", err)
	}
	pid := cmd.Process.Pid
	if err := os.WriteFile(schedulerPIDPath(), []byte(strconv.Itoa(pid)), 0o644); err != nil {
		return fmt.Errorf("write PID file: %w", err)
	}
	_ = cmd.Process.Release()

	fmt.Printf("✓ Scheduler started (PID %d)\n", pid)
	fmt.Printf("  log:   %s\n", schedulerLogPath())
	fmt.Printf("  stop:  adb scheduler stop\n")
	return nil
}

func schedulerDaemonStop() error {
	pid, alive := readPIDFile(schedulerPIDPath())
	if !alive {
		if pid > 0 {
			_ = os.Remove(schedulerPIDPath())
		}
		fmt.Println("Scheduler is not running.")
		return nil
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		_ = os.Remove(schedulerPIDPath())
		return nil
	}
	if err := stopProcess(p); err != nil {
		_ = os.Remove(schedulerPIDPath())
		fmt.Printf("✓ Scheduler stopped (PID %d was not running)\n", pid)
		return nil
	}
	_ = os.Remove(schedulerPIDPath())
	fmt.Printf("✓ Scheduler stopped (PID %d)\n", pid)
	return nil
}

// readPIDFile reads a PID file and checks whether the process is alive.
// Extracted so serve.go and scheduler.go don't need to share a private
// helper.
func readPIDFile(path string) (int, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, false
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return pid, false
	}
	return pid, processAlive(p)
}

// ---- foreground loop ----

func schedulerRunForeground() error {
	if App == nil {
		return fmt.Errorf("app not initialized")
	}

	// Open (or create) the log file and duplicate output to it.
	if err := os.MkdirAll(filepath.Dir(schedulerLogPath()), 0o755); err != nil {
		return fmt.Errorf("prep log dir: %w", err)
	}
	logFile, err := os.OpenFile(schedulerLogPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	defer logFile.Close()

	logger := io.MultiWriter(os.Stdout, logFile)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	deps := scheduler.Deps{
		BasePath: App.BasePath,
		Logger:   logger,
		PullRepos: func(ctx context.Context) (string, error) {
			summary, err := integration.PullAllRepos(App.BasePath, integration.PullOpts{})
			if err != nil {
				return "", err
			}
			return summary.Format(), nil
		},
		EvaluateAlerts: func(ctx context.Context) (int, string, error) {
			if App.AlertEvaluator == nil {
				return 0, "", nil
			}
			alerts, err := App.AlertEvaluator.EvaluateAll()
			if err != nil {
				return 0, "", err
			}
			var buf bytes.Buffer
			for _, a := range alerts {
				fmt.Fprintf(&buf, "      [%s] %s\n", a.Severity, a.Message)
			}
			return len(alerts), buf.String(), nil
		},
		LogFiles: []string{
			filepath.Join(App.BasePath, ".events.jsonl"),
			schedulerLogPath(),
		},
	}

	jobs := scheduler.DefaultJobs(deps)

	fmt.Fprintf(logger, "adb scheduler starting with %d jobs\n", len(jobs))
	return scheduler.Run(ctx, scheduler.RunOptions{
		Jobs:       jobs,
		StateFile:  schedulerStatePath(),
		Logger:     logger,
		RunOnStart: false, // avoid a pull storm at daemon startup
	})
}
