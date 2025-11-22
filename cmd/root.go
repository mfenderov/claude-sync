package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/mfenderov/claude-sync/internal/version"
)

var rootCmd = &cobra.Command{
	Use:   "claude-sync",
	Short: "🎭 Sync your Claude Code configuration across machines",
	Long: `claude-sync - A beautiful CLI tool for syncing Claude Code configurations.

Keep your Claude Code settings, hooks, plugins, and skills synchronized
across multiple machines with git-based syncing.`,
	Version: version.Get().Version,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  "Display detailed version information including commit, build date, and platform",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version.Get().Verbose())
	},
}

// Execute runs the root command and handles errors
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(versionCmd)

	// Custom version template for --version flag
	v := version.Get()
	if v.Commit != "unknown" {
		rootCmd.SetVersionTemplate(fmt.Sprintf("%s (commit: %s)\n", v.String(), v.ShortCommit()))
	} else {
		rootCmd.SetVersionTemplate(v.String() + "\n")
	}
}
