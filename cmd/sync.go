package cmd

import (
	"github.com/spf13/cobra"

	"github.com/mfenderov/claude-sync/internal/logger"
	"github.com/mfenderov/claude-sync/internal/sync"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync configuration (commit + pull + push)",
	Long: `Automatically syncs your Claude Code configuration:
  1. Commits any local changes
  2. Pulls from remote (with rebase)
  3. Pushes to remote

This is the default command when running 'claude-sync' without arguments.`,
	RunE: runSync,
}

func init() {
	rootCmd.AddCommand(syncCmd)

	// Make sync the default command
	rootCmd.RunE = runSync
}

func runSync(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Create adapters to bridge interfaces with real implementations
	log := logger.Default()
	logAdapter := sync.NewLoggerAdapter(log)
	prompterAdapter := sync.NewPrompterAdapter()
	gitAdapter := sync.NewGitAdapter()

	// Create and run the sync service
	service := sync.NewService(gitAdapter, prompterAdapter, logAdapter)
	return service.Run(ctx)
}
