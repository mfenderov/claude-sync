package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mfenderov/claude-sync/internal/git"
	"github.com/mfenderov/claude-sync/internal/logger"
	"github.com/mfenderov/claude-sync/internal/prompts"
	"github.com/mfenderov/claude-sync/internal/ui"
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
	log := logger.Default()
	log.Title("🎭 Claude Config Sync")

	// Check if ~/.claude directory exists
	claudeDirExists, err := git.ClaudeDirExists()
	if err != nil {
		log.Error("✗", "Failed to check Claude directory", err, "directory", "~/.claude")
		return err
	}

	// If ~/.claude doesn't exist, run the first-time setup flow
	if !claudeDirExists {
		claudeDir, pathErr := git.ClaudeDirPath()
		if pathErr != nil {
			log.Error("✗", "Failed to get Claude directory path", pathErr)
			return pathErr
		}
		return runFirstTimeSetup(ctx, log, claudeDir)
	}

	// Get Claude directory (we know it exists now)
	claudeDir, err := git.GetClaudeDir()
	if err != nil {
		log.Error("✗", err.Error(), err, "directory", "~/.claude")
		return err
	}

	// Check if it's a git repo - if not, run initialization flow
	if !git.IsGitRepo(claudeDir) {
		return runInitFlow(ctx, log, claudeDir)
	}

	// Step 1: Commit local changes if any
	if err := commitLocalChanges(ctx, log, claudeDir); err != nil {
		return err
	}

	// Step 2: Pull with rebase
	if err := pullWithRebaseAndHandleConflicts(ctx, log, claudeDir); err != nil {
		return err
	}

	// Step 3: Push to remote
	if err := pushToRemote(ctx, log, claudeDir); err != nil {
		return err
	}

	// Show recent activity
	showRecentActivity(ctx, log, claudeDir)

	log.Success("✨", "Sync complete!")
	log.Newline()

	return nil
}

func commitLocalChanges(ctx context.Context, log *logger.Logger, claudeDir string) error {
	log.InfoMsg("⏳", "Checking for local changes...", "directory", claudeDir)
	hasChanges, err := git.HasUncommittedChanges(ctx, claudeDir)
	if err != nil {
		log.Error("✗", "Failed to check changes", err, "directory", claudeDir)
		return err
	}

	if !hasChanges {
		log.Success("✓", "No local changes")
		log.Newline()
		return nil
	}

	// Get changed files
	changedFiles, err := git.GetChangedFiles(ctx, claudeDir)
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

	// Commit changes
	commitMsg := git.GenerateAutoCommitMessage()
	log.InfoMsg("⏳", "Committing changes...", "message", commitMsg)
	if err := git.CommitChanges(ctx, claudeDir, commitMsg); err != nil {
		log.Error("✗", "Failed to commit", err, "directory", claudeDir)
		return err
	}
	log.Success("✓", "Changes committed", "message", commitMsg)
	log.Muted("  " + commitMsg)
	log.Newline()
	return nil
}

func pullWithRebaseAndHandleConflicts(ctx context.Context, log *logger.Logger, claudeDir string) error {
	var pullErr error
	err := prompts.SpinWhile("Pulling from remote...", func() error {
		pullErr = git.PullWithRebase(ctx, claudeDir)
		return pullErr
	})
	if err != nil {
		return handlePullError(ctx, log, claudeDir, pullErr)
	}
	log.Success("✓", "Pulled latest changes")
	log.Newline()
	return nil
}

func handlePullError(ctx context.Context, log *logger.Logger, claudeDir string, pullErr error) error {
	// Check for conflicts
	hasConflicts, conflictErr := git.HasConflicts(ctx, claudeDir)
	if conflictErr != nil || !hasConflicts {
		log.Error("✗", "Failed to pull", pullErr, "directory", claudeDir)
		return pullErr
	}

	// Handle merge conflicts
	log.Error("✗", "Merge conflicts detected!", pullErr, "directory", claudeDir)
	log.Warning("⚠️", "Conflicts found - aborting sync to keep your config safe")
	log.Muted("  Please resolve conflicts manually and try again:")
	log.Muted("  1. cd ~/.claude")
	log.Muted("  2. Resolve conflicts in affected files")
	log.Muted("  3. git add <resolved-files>")
	log.Muted("  4. git rebase --continue")
	log.Muted("  5. Run claude-sync again")
	log.Newline()

	// Abort the rebase to leave repo in clean state
	if abortErr := git.AbortRebase(ctx, claudeDir); abortErr != nil {
		log.Warning("⚠️", "Failed to abort rebase - manual intervention needed", "error", abortErr)
	} else {
		log.InfoMsg("ℹ️", "Rebase aborted - repository restored to previous state")
	}
	log.Newline()
	return fmt.Errorf("merge conflicts detected - sync aborted")
}

func pushToRemote(ctx context.Context, log *logger.Logger, claudeDir string) error {
	err := prompts.SpinWhile("Pushing to remote...", func() error {
		return git.Push(ctx, claudeDir)
	})
	if err != nil {
		log.Error("✗", "Failed to push", err, "directory", claudeDir)
		return err
	}
	log.Success("✓", "Pushed to remote")
	log.Newline()
	return nil
}

func showRecentActivity(ctx context.Context, log *logger.Logger, claudeDir string) {
	commits, err := git.GetRecentCommits(ctx, claudeDir, 5)
	if err != nil || len(commits) == 0 {
		return
	}

	var commitList strings.Builder
	for _, commit := range commits {
		commitList.WriteString(ui.ListItemStyle.Render(commit) + "\n")
	}
	log.Box("Recent Activity", commitList.String())
}

// runFirstTimeSetup handles setup when ~/.claude doesn't exist at all
func runFirstTimeSetup(ctx context.Context, log *logger.Logger, claudeDir string) error {
	log.Newline()
	log.Title("🎉 First Time Setup")
	log.InfoMsg("📋", "No Claude Code configuration found at ~/.claude")
	log.Muted("  Let's set that up!")
	log.Newline()

	// Ask user what they want to do
	choice, err := prompts.Select("What would you like to do?", []prompts.Option{
		{Label: "📥 Clone existing config (I have a repo already)", Value: "clone"},
		{Label: "🆕 Start fresh (new configuration)", Value: "fresh"},
	})
	if err != nil {
		log.Error("✗", "Failed to read input", err)
		return err
	}

	if choice == "" {
		log.InfoMsg("ℹ️", "Setup cancelled - you can run claude-sync again when ready")
		log.Newline()
		return nil
	}
	log.Newline()

	if choice == "clone" {
		return runCloneFlow(ctx, log, claudeDir)
	}

	// Start fresh - need to create the directory first
	log.InfoMsg("⏳", "Creating ~/.claude directory...")
	if err := createClaudeDir(claudeDir); err != nil {
		log.Error("✗", "Failed to create directory", err, "directory", claudeDir)
		return err
	}
	log.Success("✓", "Directory created")
	log.Newline()

	return runInitFlow(ctx, log, claudeDir)
}

// runCloneFlow handles cloning an existing config repo
func runCloneFlow(ctx context.Context, log *logger.Logger, claudeDir string) error {
	log.InfoMsg("📦", "Enter the URL of your existing Claude config repository")
	log.Muted("  Example: git@github.com:username/claude-config.git")
	log.Newline()

	remoteURL, err := prompts.Input(
		"🔗 Enter your git remote URL:",
		"git@github.com:username/claude-config.git",
	)
	if err != nil {
		log.Error("✗", "Failed to read input", err)
		return err
	}

	if remoteURL == "" {
		log.InfoMsg("ℹ️", "Setup cancelled - you can run claude-sync again when ready")
		log.Newline()
		return nil
	}
	log.Newline()

	// Clone the repository
	err = prompts.SpinWhile("Cloning configuration...", func() error {
		return git.CloneRepo(ctx, remoteURL, claudeDir)
	})
	if err != nil {
		log.Error("✗", "Failed to clone repository", err, "url", remoteURL)
		log.Newline()
		log.Warning("⚠️", "Please make sure:")
		log.Muted("  1. The repository URL is correct")
		log.Muted("  2. The repository exists and has commits")
		log.Muted("  3. You have SSH keys set up (for git@ URLs)")
		log.Newline()
		return err
	}
	log.Success("✓", "Configuration cloned to ~/.claude")
	log.Newline()

	// Success!
	log.Success("🎉", "Setup complete!")
	log.InfoMsg("💡", "Your Claude Code config is ready!")
	log.Muted("  Next steps:")
	log.Muted("  • Run 'claude-sync' anytime to sync changes")
	log.Muted("  • Your config is now synchronized across machines")
	log.Newline()

	return nil
}

// createClaudeDir creates the ~/.claude directory
func createClaudeDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

// runInitFlow handles first-time setup when ~/.claude exists but is not a git repo
func runInitFlow(ctx context.Context, log *logger.Logger, claudeDir string) error {
	log.Newline()
	log.Title("🎉 Git Sync Setup")
	log.InfoMsg("📋", "Claude Code configuration detected!", "directory", claudeDir)
	log.Muted("  Let's set up git sync to keep your config synchronized across machines")
	log.Newline()

	// Step 1: Confirm user wants to proceed
	confirmed, err := prompts.Confirm("🤔 Would you like to set up git sync now?")
	if err != nil {
		log.Error("✗", "Failed to read input", err)
		return err
	}

	if !confirmed {
		log.InfoMsg("ℹ️", "Setup cancelled - you can run claude-sync again when ready")
		log.Newline()
		return nil
	}
	log.Newline()

	// Step 2: Get remote URL
	log.InfoMsg("📦", "Please create a private git repository first")
	log.Muted("  Examples:")
	log.Muted("    • GitHub: https://github.com/new")
	log.Muted("    • GitLab: https://gitlab.com/projects/new")
	log.Muted("    • Bitbucket: https://bitbucket.org/repo/create")
	log.Newline()

	remoteURL, err := prompts.Input(
		"🔗 Enter your git remote URL:",
		"git@github.com:username/claude-config.git",
	)
	if err != nil {
		log.Error("✗", "Failed to read input", err)
		return err
	}

	if remoteURL == "" {
		log.Error("✗", "No remote URL provided - setup cancelled", fmt.Errorf("empty remote URL"))
		log.Newline()
		return fmt.Errorf("setup cancelled")
	}
	log.Newline()

	// Step 3: Validate remote exists
	err = prompts.SpinWhile("Validating remote repository...", func() error {
		return git.ValidateRemote(ctx, remoteURL)
	})
	if err != nil {
		log.Error("✗", "Remote repository not accessible", err, "url", remoteURL)
		log.Newline()
		log.Warning("⚠️", "Please make sure:")
		log.Muted("  1. The repository exists and you have access")
		log.Muted("  2. You have SSH keys set up (for git@ URLs)")
		log.Muted("  3. The URL is correct")
		log.Newline()
		log.InfoMsg("💡", "After creating the repo, run claude-sync again")
		log.Newline()
		return fmt.Errorf("remote repository not accessible")
	}
	log.Success("✓", "Remote repository verified!")
	log.Newline()

	// Step 4: Initialize git repo
	log.InfoMsg("⏳", "Initializing git repository...", "directory", claudeDir)
	if err := git.InitRepo(ctx, claudeDir); err != nil {
		log.Error("✗", "Failed to initialize git repo", err, "directory", claudeDir)
		return err
	}
	log.Success("✓", "Git repository initialized")
	log.Newline()

	// Step 5: Create .gitignore
	log.InfoMsg("⏳", "Creating .gitignore for sensitive files...")
	if err := git.SetupGitignore(claudeDir); err != nil {
		log.Error("✗", "Failed to create .gitignore", err, "directory", claudeDir)
		return err
	}
	log.Success("✓", ".gitignore created")
	log.Muted("  Excluded: credentials.json, *.key, aws-*.sh, .env, etc.")
	log.Newline()

	// Step 6: Initial commit
	log.InfoMsg("⏳", "Creating initial commit...")
	if err := git.InitialCommit(ctx, claudeDir, "Initial Claude Code configuration"); err != nil {
		log.Error("✗", "Failed to create initial commit", err, "directory", claudeDir)
		return err
	}
	log.Success("✓", "Initial commit created")
	log.Newline()

	// Step 7: Add remote
	log.InfoMsg("⏳", "Adding remote repository...", "url", remoteURL)
	if err := git.AddRemote(ctx, claudeDir, "origin", remoteURL); err != nil {
		log.Error("✗", "Failed to add remote", err, "url", remoteURL)
		return err
	}
	log.Success("✓", "Remote added", "name", "origin", "url", remoteURL)
	log.Newline()

	// Step 8: Push to remote with upstream tracking
	err = prompts.SpinWhile("Pushing to remote...", func() error {
		return git.PushWithUpstream(ctx, claudeDir)
	})
	if err != nil {
		log.Error("✗", "Failed to push", err)
		log.Newline()
		log.Warning("⚠️", "Git setup complete, but push failed")
		log.Muted("  Your config is initialized locally. Try:")
		log.Muted("  1. cd ~/.claude")
		log.Muted("  2. git push -u origin main")
		log.Muted("  3. Run claude-sync again")
		log.Newline()
		return err
	}
	log.Success("✓", "Pushed to remote")
	log.Newline()

	// Success!
	log.Success("🎉", "Setup complete!")
	log.InfoMsg("💡", "Your Claude Code config is now synced!")
	log.Muted("  Next steps:")
	log.Muted("  • Make changes to your Claude config")
	log.Muted("  • Run 'claude-sync' to automatically sync")
	log.Muted("  • On other machines: git clone your repo to ~/.claude")
	log.Newline()

	return nil
}
