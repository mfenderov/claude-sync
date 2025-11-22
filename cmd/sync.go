package cmd

import (
	"fmt"
	"strings"

	"github.com/mfenderov/claude-sync/internal/git"
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
	fmt.Println()
	fmt.Println(ui.TitleStyle.Render("🎭 Claude Config Sync"))
	fmt.Println()

	// Get Claude directory
	claudeDir, err := git.GetClaudeDir()
	if err != nil {
		fmt.Println(ui.RenderError("✗", err.Error()))
		return err
	}

	// Check if it's a git repo
	if !git.IsGitRepo(claudeDir) {
		msg := "~/.claude is not a git repository"
		fmt.Println(ui.RenderError("✗", msg))
		fmt.Println(ui.RenderMuted("  Run: cd ~/.claude && git init"))
		return fmt.Errorf("%s", msg)
	}

	// Step 1: Check for local changes
	fmt.Println(ui.RenderInfo("⏳", "Checking for local changes..."))
	hasChanges, err := git.HasUncommittedChanges(claudeDir)
	if err != nil {
		fmt.Println(ui.RenderError("✗", "Failed to check changes"))
		return err
	}

	if hasChanges {
		// Get changed files
		changedFiles, err := git.GetChangedFiles(claudeDir)
		if err != nil {
			fmt.Println(ui.RenderError("✗", "Failed to get changed files"))
			return err
		}

		fmt.Println(ui.RenderSuccess("✓", fmt.Sprintf("Found %d changed file(s)", len(changedFiles))))
		for _, file := range changedFiles {
			fmt.Println(ui.ListItemStyle.Render("→ " + file))
		}
		fmt.Println()

		// Step 2: Commit changes
		commitMsg := git.GenerateAutoCommitMessage()
		fmt.Println(ui.RenderInfo("⏳", "Committing changes..."))
		if err := git.CommitChanges(claudeDir, commitMsg); err != nil {
			fmt.Println(ui.RenderError("✗", "Failed to commit"))
			return err
		}
		fmt.Println(ui.RenderSuccess("✓", "Changes committed"))
		fmt.Println(ui.RenderMuted("  " + commitMsg))
		fmt.Println()
	} else {
		fmt.Println(ui.RenderSuccess("✓", "No local changes"))
		fmt.Println()
	}

	// Step 3: Pull with rebase
	fmt.Println(ui.RenderInfo("⏳", "Pulling from remote (with rebase)..."))
	if err := git.PullWithRebase(claudeDir); err != nil {
		fmt.Println(ui.RenderError("✗", "Failed to pull"))
		return err
	}
	fmt.Println(ui.RenderSuccess("✓", "Pulled latest changes"))
	fmt.Println()

	// Step 4: Push
	fmt.Println(ui.RenderInfo("⏳", "Pushing to remote..."))
	if err := git.Push(claudeDir); err != nil {
		fmt.Println(ui.RenderError("✗", "Failed to push"))
		return err
	}
	fmt.Println(ui.RenderSuccess("✓", "Pushed to remote"))
	fmt.Println()

	// Show recent commits
	commits, err := git.GetRecentCommits(claudeDir, 5)
	if err == nil && len(commits) > 0 {
		var commitList strings.Builder
		for _, commit := range commits {
			commitList.WriteString(ui.ListItemStyle.Render(commit) + "\n")
		}

		box := ui.RenderBox("Recent Activity", commitList.String())
		fmt.Println(box)
	}

	fmt.Println(ui.RenderSuccess("✨", "Sync complete!"))
	fmt.Println()

	return nil
}
