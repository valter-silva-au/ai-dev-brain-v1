package cli

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/valter-silva-au/ai-dev-brain/internal/hive"
	"github.com/valter-silva-au/ai-dev-brain/internal/server"
)

// NewServeCmd creates the serve command
func NewServeCmd() *cobra.Command {
	var (
		port int
		tv   bool
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the web dashboard",
		Long:  `Start the MyImaginationAI Agent Command Center — a live-updating HTMX dashboard showing all agents, tasks, metrics, and chat.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if App == nil {
				return fmt.Errorf("app not initialized")
			}

			// Initialize Hive Mind components
			agentReg := hive.NewAgentRegistry(App.BasePath)
			projectReg := hive.NewProjectRegistry(App.BasePath)
			knowledgeAgg := hive.NewKnowledgeAggregator(App.BasePath, projectReg)
			messageBus := hive.NewMessageBus(App.BasePath)

			// Discover OpenClaw agents
			openclawPath := os.ExpandEnv("$HOME/.openclaw")
			if _, err := agentReg.DiscoverOpenClaw(openclawPath); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Warning: OpenClaw discovery failed: %v\n", err)
			}

			srv := server.NewServer(App, agentReg, projectReg, knowledgeAgg, messageBus)

			addr := fmt.Sprintf("127.0.0.1:%d", port)

			// Open browser in TV mode
			if tv {
				go func() {
					url := fmt.Sprintf("http://%s", addr)
					openBrowser(url)
				}()
			}

			fmt.Printf("MyImaginationAI Agent Command Center\n")
			fmt.Printf("Dashboard: http://%s\n", addr)
			fmt.Printf("Press Ctrl+C to stop\n\n")

			return srv.Start(addr)
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 8400, "Server port")
	cmd.Flags().BoolVar(&tv, "tv", false, "Open browser automatically (TV mode)")

	return cmd
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
