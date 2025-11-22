package cmd

import (
	"fmt"
	"strings"

	"github.com/mfenderov/claude-sync/internal/git"
	"github.com/mfenderov/claude-sync/internal/logger"
	"github.com/mfenderov/claude-sync/internal/ui"
	"github.com/spf13/cobra"
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
	log := logger.Default()
	log.Title("🎭 Claude Config Sync")

	// Get Claude directory
	claudeDir, err := git.GetClaudeDir()
	if err != nil {
		log.Error("✗", err.Error(), err, "directory", "~/.claude")
		return err
	}

	// Check if it's a git repo
	if !git.IsGitRepo(claudeDir) {
		msg := "~/.claude is not a git repository"
		log.Error("✗", msg, fmt.Errorf("not a git repo"), "directory", claudeDir)
		log.Muted("  Run: cd ~/.claude && git init")
		return fmt.Errorf("%s", msg)
	}

	// Step 1: Check for local changes
	log.InfoMsg("⏳", "Checking for local changes...", "directory", claudeDir)
	hasChanges, err := git.HasUncommittedChanges(claudeDir)
	if err != nil {
		log.Error("✗", "Failed to check changes", err, "directory", claudeDir)
		return err
	}

	if hasChanges {
		// Get changed files
		changedFiles, err := git.GetChangedFiles(claudeDir)
		if err != nil {
			log.Error("✗", "Failed to get changed files", err, "directory", claudeDir)
			return err
		}

		log.Success("✓", fmt.Sprintf("Found %d changed file(s)", len(changedFiles)),
			"count", len(changedFiles), "files", changedFiles)
		for _, file := range changedFiles {
			log.ListItem("→ " + file)
		}
		log.Newline()

		// Step 2: Commit changes
		commitMsg := git.GenerateAutoCommitMessage()
		log.InfoMsg("⏳", "Committing changes...", "message", commitMsg)
		if err := git.CommitChanges(claudeDir, commitMsg); err != nil {
			log.Error("✗", "Failed to commit", err, "directory", claudeDir)
			return err
		}
		log.Success("✓", "Changes committed", "message", commitMsg)
		log.Muted("  " + commitMsg)
		log.Newline()
	} else {
		log.Success("✓", "No local changes")
		log.Newline()
	}

	// Step 3: Pull with rebase
	log.InfoMsg("⏳", "Pulling from remote (with rebase)...", "directory", claudeDir)
	if err := git.PullWithRebase(claudeDir); err != nil {
		log.Error("✗", "Failed to pull", err, "directory", claudeDir)
		return err
	}
	log.Success("✓", "Pulled latest changes")
	log.Newline()

	// Step 4: Push
	log.InfoMsg("⏳", "Pushing to remote...", "directory", claudeDir)
	if err := git.Push(claudeDir); err != nil {
		log.Error("✗", "Failed to push", err, "directory", claudeDir)
		return err
	}
	log.Success("✓", "Pushed to remote")
	log.Newline()

	// Show recent commits
	commits, err := git.GetRecentCommits(claudeDir, 5)
	if err == nil && len(commits) > 0 {
		var commitList strings.Builder
		for _, commit := range commits {
			commitList.WriteString(ui.ListItemStyle.Render(commit) + "\n")
		}

		log.Box("Recent Activity", commitList.String())
	}

	log.Success("✨", "Sync complete!")
	log.Newline()

	return nil
}
