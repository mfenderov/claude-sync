package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "claude-sync",
	Short: "🎭 Sync your Claude Code configuration across machines",
	Long: `claude-sync - A beautiful CLI tool for syncing Claude Code configurations.

Keep your Claude Code settings, hooks, plugins, and skills synchronized
across multiple machines with git-based syncing.`,
	Version: "0.1.0",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.SetVersionTemplate(`{{.Version}}
`)
}
