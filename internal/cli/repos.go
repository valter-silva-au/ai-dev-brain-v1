package cli

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/valter-silva-au/ai-dev-brain/internal/integration"
)

// NewReposCmd creates the `adb repos` command group.
func NewReposCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repos",
		Short: "Manage cloned repositories under <workspace>/repos",
		Long: `Operations across every git repository under <workspace>/repos.

Used in one-shot form directly (` + "`adb repos pull`" + `) or scheduled via
the adb scheduler (see ` + "`adb scheduler`" + `).`,
	}
	cmd.AddCommand(newReposPullCmd())
	return cmd
}

func newReposPullCmd() *cobra.Command {
	var (
		fetchOnly bool
		timeout   time.Duration
		root      string
	)
	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Fetch and fast-forward all repos under <workspace>/repos",
		Long: `Walks <workspace>/repos recursively for git repositories and:

  - runs 'git fetch --all --prune' on each,
  - runs 'git pull --ff-only' only if the working tree is clean, HEAD is
    on the upstream default branch, and an upstream is configured.

Dirty, non-default-branch, or unconfigured-upstream repos are recorded as
skipped rather than treated as errors.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if App == nil {
				return fmt.Errorf("app not initialized")
			}
			opts := integration.PullOpts{
				ReposRoot:      root,
				PerRepoTimeout: timeout,
				FetchOnly:      fetchOnly,
			}
			if opts.ReposRoot == "" {
				opts.ReposRoot = filepath.Join(App.BasePath, "repos")
			}
			fmt.Printf("Scanning %s...\n", opts.ReposRoot)
			summary, err := integration.PullAllRepos(App.BasePath, opts)
			if err != nil {
				return err
			}
			fmt.Println(summary.Format())
			for _, r := range summary.Repos {
				status := r.Action
				if r.Err != nil {
					status = fmt.Sprintf("%s: %v", r.Action, r.Err)
				}
				fmt.Printf("  %-20s %s\n", status, r.Path)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&fetchOnly, "fetch-only", false, "Only fetch — never fast-forward")
	cmd.Flags().DurationVar(&timeout, "timeout", 60*time.Second, "Per-repo timeout for each git command")
	cmd.Flags().StringVar(&root, "root", "", "Override repos root (default: <workspace>/repos)")
	return cmd
}
