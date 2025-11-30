package sync

import (
	"context"
	"fmt"
	"strings"
)

// Service handles the sync business logic with injected dependencies.
// This allows for testing without actual TUI or git operations.
type Service struct {
	git      GitOperator
	prompter Prompter
	logger   Logger
}

// NewService creates a new sync service with the given dependencies.
func NewService(git GitOperator, prompter Prompter, logger Logger) *Service {
	return &Service{
		git:      git,
		prompter: prompter,
		logger:   logger,
	}
}

// Run executes the main sync flow.
func (s *Service) Run(ctx context.Context) error {
	s.logger.Title("🎭 Claude Config Sync")

	// Check if ~/.claude directory exists
	claudeDirExists, err := s.git.ClaudeDirExists()
	if err != nil {
		s.logger.Error("✗", "Failed to check Claude directory", err)
		return err
	}

	// If ~/.claude doesn't exist, run the first-time setup flow
	if !claudeDirExists {
		claudeDir, pathErr := s.git.ClaudeDirPath()
		if pathErr != nil {
			s.logger.Error("✗", "Failed to get Claude directory path", pathErr)
			return pathErr
		}
		return s.runFirstTimeSetup(ctx, claudeDir)
	}

	// Get Claude directory (we know it exists now)
	claudeDir, err := s.git.GetClaudeDir()
	if err != nil {
		s.logger.Error("✗", err.Error(), err)
		return err
	}

	// Check if it's a git repo - if not, run initialization flow
	if !s.git.IsGitRepo(claudeDir) {
		return s.runInitFlow(ctx, claudeDir)
	}

	// Normal sync: commit, pull, push
	if err := s.commitLocalChanges(ctx, claudeDir); err != nil {
		return err
	}

	if err := s.pullWithRebaseAndHandleConflicts(ctx, claudeDir); err != nil {
		return err
	}

	if err := s.pushToRemote(ctx, claudeDir); err != nil {
		return err
	}

	s.showRecentActivity(ctx, claudeDir)

	s.logger.Success("✨", "Sync complete!")
	s.logger.Newline()

	return nil
}

// commitLocalChanges commits any uncommitted changes.
func (s *Service) commitLocalChanges(ctx context.Context, claudeDir string) error {
	changedFiles, err := s.git.GetChangedFiles(ctx, claudeDir)
	if err != nil {
		s.logger.Error("✗", "Failed to check for changes", err)
		return err
	}

	if len(changedFiles) == 0 {
		s.logger.Success("✓", "No local changes")
		s.logger.Newline()
		return nil
	}

	s.logger.Success("✓", fmt.Sprintf("Found %d changed file(s)", len(changedFiles)))
	for _, file := range changedFiles {
		s.logger.ListItem("→ " + file)
	}
	s.logger.Newline()

	commitMsg := s.git.GenerateAutoCommitMessage()
	s.logger.Info("⏳", "Committing changes...")
	if err := s.git.CommitChanges(ctx, claudeDir, commitMsg); err != nil {
		s.logger.Error("✗", "Failed to commit", err)
		return err
	}
	s.logger.Success("✓", "Changes committed")
	s.logger.Muted("  " + commitMsg)
	s.logger.Newline()
	return nil
}

// pullWithRebaseAndHandleConflicts pulls from remote and handles conflicts.
func (s *Service) pullWithRebaseAndHandleConflicts(ctx context.Context, claudeDir string) error {
	var pullErr error
	err := s.prompter.SpinWhile("Pulling from remote...", func() error {
		pullErr = s.git.PullWithRebase(ctx, claudeDir)
		return pullErr
	})
	if err != nil {
		return s.handlePullError(ctx, claudeDir, pullErr)
	}
	s.logger.Success("✓", "Pulled latest changes")
	s.logger.Newline()
	return nil
}

// handlePullError handles errors during pull operations.
func (s *Service) handlePullError(ctx context.Context, claudeDir string, pullErr error) error {
	hasConflicts, conflictErr := s.git.HasConflicts(ctx, claudeDir)
	if conflictErr != nil || !hasConflicts {
		s.logger.Error("✗", "Failed to pull", pullErr)
		return pullErr
	}

	s.logger.Error("✗", "Merge conflicts detected!", pullErr)
	s.logger.Warning("⚠️", "Conflicts found - aborting sync to keep your config safe")
	s.logger.Muted("  Please resolve conflicts manually and try again:")
	s.logger.Muted("  1. cd ~/.claude")
	s.logger.Muted("  2. Resolve conflicts in affected files")
	s.logger.Muted("  3. git add <resolved-files>")
	s.logger.Muted("  4. git rebase --continue")
	s.logger.Muted("  5. Run claude-sync again")
	s.logger.Newline()

	if abortErr := s.git.AbortRebase(ctx, claudeDir); abortErr != nil {
		s.logger.Warning("⚠️", "Failed to abort rebase - manual intervention needed")
	} else {
		s.logger.Info("ℹ️", "Rebase aborted - repository restored to previous state")
	}
	s.logger.Newline()
	return fmt.Errorf("merge conflicts detected - sync aborted")
}

// pushToRemote pushes changes to remote.
func (s *Service) pushToRemote(ctx context.Context, claudeDir string) error {
	err := s.prompter.SpinWhile("Pushing to remote...", func() error {
		return s.git.Push(ctx, claudeDir)
	})
	if err != nil {
		s.logger.Error("✗", "Failed to push", err)
		return err
	}
	s.logger.Success("✓", "Pushed to remote")
	s.logger.Newline()
	return nil
}

// showRecentActivity displays recent commits.
func (s *Service) showRecentActivity(ctx context.Context, claudeDir string) {
	commits, err := s.git.GetRecentCommits(ctx, claudeDir, 5)
	if err != nil || len(commits) == 0 {
		return
	}

	var commitList strings.Builder
	for _, commit := range commits {
		commitList.WriteString("  " + commit + "\n")
	}
	s.logger.Box("Recent Activity", commitList.String())
}

// runFirstTimeSetup handles setup when ~/.claude doesn't exist at all.
func (s *Service) runFirstTimeSetup(ctx context.Context, claudeDir string) error {
	s.logger.Newline()
	s.logger.Title("🎉 First Time Setup")
	s.logger.Info("📋", "No Claude Code configuration found at ~/.claude")
	s.logger.Muted("  Let's set that up!")
	s.logger.Newline()

	choice, err := s.prompter.Select("What would you like to do?", []SelectOption{
		{Label: "📥 Clone existing config (I have a repo already)", Value: "clone"},
		{Label: "🆕 Start fresh (new configuration)", Value: "fresh"},
	})
	if err != nil {
		s.logger.Error("✗", "Failed to read input", err)
		return err
	}

	if choice == "" {
		s.logger.Info("ℹ️", "Setup cancelled - you can run claude-sync again when ready")
		s.logger.Newline()
		return nil
	}
	s.logger.Newline()

	if choice == "clone" {
		return s.runCloneFlow(ctx, claudeDir, "")
	}

	// Start fresh - need to create the directory first
	s.logger.Info("⏳", "Creating ~/.claude directory...")
	if err := s.git.CreateClaudeDir(claudeDir); err != nil {
		s.logger.Error("✗", "Failed to create directory", err)
		return err
	}
	s.logger.Success("✓", "Directory created")
	s.logger.Newline()

	return s.runInitFlow(ctx, claudeDir)
}

// runCloneFlow handles cloning an existing config repo.
func (s *Service) runCloneFlow(ctx context.Context, claudeDir, remoteURL string) error {
	if remoteURL == "" {
		s.logger.Info("📦", "Enter the URL of your existing Claude config repository")
		s.logger.Muted("  Example: git@github.com:username/claude-config.git")
		s.logger.Newline()

		var err error
		remoteURL, err = s.prompter.Input(
			"🔗 Enter your git remote URL:",
			"git@github.com:username/claude-config.git",
		)
		if err != nil {
			s.logger.Error("✗", "Failed to read input", err)
			return err
		}

		if remoteURL == "" {
			s.logger.Info("ℹ️", "Setup cancelled - you can run claude-sync again when ready")
			s.logger.Newline()
			return nil
		}
		s.logger.Newline()
	}

	err := s.prompter.SpinWhile("Cloning configuration...", func() error {
		return s.git.CloneRepo(ctx, remoteURL, claudeDir)
	})
	if err != nil {
		s.logger.Error("✗", "Failed to clone repository", err)
		s.logger.Newline()
		s.logger.Warning("⚠️", "Please make sure:")
		s.logger.Muted("  1. The repository URL is correct")
		s.logger.Muted("  2. The repository exists and has commits")
		s.logger.Muted("  3. You have SSH keys set up (for git@ URLs)")
		s.logger.Newline()
		return err
	}
	s.logger.Success("✓", "Configuration cloned to ~/.claude")
	s.logger.Newline()

	s.logger.Success("🎉", "Setup complete!")
	s.logger.Info("💡", "Your Claude Code config is ready!")
	s.logger.Muted("  Next steps:")
	s.logger.Muted("  • Run 'claude-sync' anytime to sync changes")
	s.logger.Muted("  • Your config is now synchronized across machines")
	s.logger.Newline()

	return nil
}

// runInitFlow handles first-time setup when ~/.claude exists but is not a git repo.
func (s *Service) runInitFlow(ctx context.Context, claudeDir string) error {
	s.logger.Newline()
	s.logger.Title("🎉 Git Sync Setup")
	s.logger.Info("📋", "Claude Code configuration detected!")
	s.logger.Muted("  Let's set up git sync to keep your config synchronized across machines")
	s.logger.Newline()

	confirmed, err := s.prompter.Confirm("🤔 Would you like to set up git sync now?")
	if err != nil {
		s.logger.Error("✗", "Failed to read input", err)
		return err
	}

	if !confirmed {
		s.logger.Info("ℹ️", "Setup cancelled - you can run claude-sync again when ready")
		s.logger.Newline()
		return nil
	}
	s.logger.Newline()

	// Get remote URL
	s.logger.Info("📦", "Please create a private git repository first")
	s.logger.Muted("  Examples:")
	s.logger.Muted("    • GitHub: https://github.com/new")
	s.logger.Muted("    • GitLab: https://gitlab.com/projects/new")
	s.logger.Muted("    • Bitbucket: https://bitbucket.org/repo/create")
	s.logger.Newline()

	remoteURL, err := s.prompter.Input(
		"🔗 Enter your git remote URL:",
		"git@github.com:username/claude-config.git",
	)
	if err != nil {
		s.logger.Error("✗", "Failed to read input", err)
		return err
	}

	if remoteURL == "" {
		s.logger.Error("✗", "No remote URL provided - setup cancelled", fmt.Errorf("empty remote URL"))
		s.logger.Newline()
		return fmt.Errorf("setup cancelled")
	}
	s.logger.Newline()

	// Validate remote exists
	err = s.prompter.SpinWhile("Validating remote repository...", func() error {
		return s.git.ValidateRemote(ctx, remoteURL)
	})
	if err != nil {
		s.logger.Error("✗", "Remote repository not accessible", err)
		s.logger.Newline()
		s.logger.Warning("⚠️", "Please make sure:")
		s.logger.Muted("  1. The repository exists and you have access")
		s.logger.Muted("  2. You have SSH keys set up (for git@ URLs)")
		s.logger.Muted("  3. The URL is correct")
		s.logger.Newline()
		s.logger.Info("💡", "After creating the repo, run claude-sync again")
		s.logger.Newline()
		return fmt.Errorf("remote repository not accessible")
	}
	s.logger.Success("✓", "Remote repository verified!")
	s.logger.Newline()

	// Check if remote has existing commits - THIS IS THE NEW LOGIC
	return s.handleRemoteState(ctx, claudeDir, remoteURL)
}

// handleRemoteState checks the remote and determines the appropriate action.
func (s *Service) handleRemoteState(ctx context.Context, claudeDir, remoteURL string) error {
	hasCommits, err := s.git.RemoteHasCommits(ctx, remoteURL)
	if err != nil {
		s.logger.Warning("⚠️", "Could not check remote history, proceeding with fresh init")
		return s.executeFreshInit(ctx, claudeDir, remoteURL)
	}

	if hasCommits {
		return s.handleRemoteWithCommits(ctx, claudeDir, remoteURL)
	}

	return s.executeFreshInit(ctx, claudeDir, remoteURL)
}

// handleRemoteWithCommits prompts user when remote already has commits.
func (s *Service) handleRemoteWithCommits(ctx context.Context, claudeDir, remoteURL string) error {
	s.logger.Warning("⚠️", "Remote repository already has commits!")
	s.logger.Muted("  Your local ~/.claude has different content than the remote.")
	s.logger.Newline()

	choice, err := s.prompter.Select("How would you like to proceed?", []SelectOption{
		{Label: "📥 Use remote config (replace local)", Value: "replace"},
		{Label: "🔀 Merge histories (keep both)", Value: "merge"},
		{Label: "❌ Cancel", Value: "cancel"},
	})
	if err != nil {
		s.logger.Error("✗", "Failed to read input", err)
		return err
	}

	switch choice {
	case "replace":
		return s.executeReplaceWithRemote(ctx, claudeDir, remoteURL)
	case "merge":
		return s.executeMergeHistories(ctx, claudeDir, remoteURL)
	case "cancel", "":
		s.logger.Info("ℹ️", "Setup cancelled")
		s.logger.Newline()
		return nil
	}

	return nil
}

// executeFreshInit initializes a new git repo and pushes to empty remote.
func (s *Service) executeFreshInit(ctx context.Context, claudeDir, remoteURL string) error {
	s.logger.Info("⏳", "Initializing git repository...")
	if err := s.git.InitRepo(ctx, claudeDir); err != nil {
		s.logger.Error("✗", "Failed to initialize git repo", err)
		return err
	}
	s.logger.Success("✓", "Git repository initialized")
	s.logger.Newline()

	s.logger.Info("⏳", "Creating .gitignore for sensitive files...")
	if err := s.git.SetupGitignore(claudeDir); err != nil {
		s.logger.Error("✗", "Failed to create .gitignore", err)
		return err
	}
	s.logger.Success("✓", ".gitignore created")
	s.logger.Muted("  Excluded: credentials.json, *.key, aws-*.sh, .env, etc.")
	s.logger.Newline()

	s.logger.Info("⏳", "Creating initial commit...")
	if err := s.git.InitialCommit(ctx, claudeDir, "Initial Claude Code configuration"); err != nil {
		s.logger.Error("✗", "Failed to create initial commit", err)
		return err
	}
	s.logger.Success("✓", "Initial commit created")
	s.logger.Newline()

	s.logger.Info("⏳", "Adding remote repository...")
	if err := s.git.AddRemote(ctx, claudeDir, "origin", remoteURL); err != nil {
		s.logger.Error("✗", "Failed to add remote", err)
		return err
	}
	s.logger.Success("✓", "Remote added")
	s.logger.Newline()

	err := s.prompter.SpinWhile("Pushing to remote...", func() error {
		return s.git.PushWithUpstream(ctx, claudeDir)
	})
	if err != nil {
		s.logger.Error("✗", "Failed to push", err)
		s.logger.Newline()
		s.logger.Warning("⚠️", "Git setup complete, but push failed")
		s.logger.Muted("  Your config is initialized locally. Try:")
		s.logger.Muted("  1. cd ~/.claude")
		s.logger.Muted("  2. git push -u origin main")
		s.logger.Muted("  3. Run claude-sync again")
		s.logger.Newline()
		return err
	}
	s.logger.Success("✓", "Pushed to remote")
	s.logger.Newline()

	s.logger.Success("🎉", "Setup complete!")
	s.logger.Info("💡", "Your Claude Code config is now synced!")
	s.logger.Muted("  Next steps:")
	s.logger.Muted("  • Make changes to your Claude config")
	s.logger.Muted("  • Run 'claude-sync' to automatically sync")
	s.logger.Muted("  • On other machines: git clone your repo to ~/.claude")
	s.logger.Newline()

	return nil
}

// executeReplaceWithRemote removes local ~/.claude and clones remote.
func (s *Service) executeReplaceWithRemote(ctx context.Context, claudeDir, remoteURL string) error {
	s.logger.Info("⏳", "Removing local configuration...")
	if err := s.git.RemoveClaudeDir(); err != nil {
		s.logger.Error("✗", "Failed to remove local config", err)
		return err
	}
	s.logger.Success("✓", "Local config removed")
	s.logger.Newline()

	return s.runCloneFlow(ctx, claudeDir, remoteURL)
}

// executeMergeHistories initializes local repo and merges with remote history.
func (s *Service) executeMergeHistories(ctx context.Context, claudeDir, remoteURL string) error {
	s.logger.Info("⏳", "Initializing local git repository...")
	if err := s.git.InitRepo(ctx, claudeDir); err != nil {
		s.logger.Error("✗", "Failed to initialize git repo", err)
		return err
	}
	s.logger.Success("✓", "Git repository initialized")
	s.logger.Newline()

	s.logger.Info("⏳", "Creating .gitignore for sensitive files...")
	if err := s.git.SetupGitignore(claudeDir); err != nil {
		s.logger.Error("✗", "Failed to create .gitignore", err)
		return err
	}
	s.logger.Success("✓", ".gitignore created")
	s.logger.Muted("  Excluded: credentials.json, *.key, aws-*.sh, .env, etc.")
	s.logger.Newline()

	s.logger.Info("⏳", "Committing local configuration...")
	if err := s.git.InitialCommit(ctx, claudeDir, "Local Claude Code configuration"); err != nil {
		s.logger.Error("✗", "Failed to create local commit", err)
		return err
	}
	s.logger.Success("✓", "Local files committed")
	s.logger.Newline()

	s.logger.Info("⏳", "Adding remote repository...")
	if err := s.git.AddRemote(ctx, claudeDir, "origin", remoteURL); err != nil {
		s.logger.Error("✗", "Failed to add remote", err)
		return err
	}
	s.logger.Success("✓", "Remote added")
	s.logger.Newline()

	err := s.prompter.SpinWhile("Fetching remote history...", func() error {
		return s.git.Fetch(ctx, claudeDir)
	})
	if err != nil {
		s.logger.Error("✗", "Failed to fetch remote", err)
		return err
	}
	s.logger.Success("✓", "Remote history fetched")
	s.logger.Newline()

	err = s.prompter.SpinWhile("Merging histories...", func() error {
		return s.git.PullAllowUnrelatedHistories(ctx, claudeDir)
	})
	if err != nil {
		s.logger.Error("✗", "Failed to merge histories", err)
		s.logger.Newline()
		s.logger.Warning("⚠️", "This can happen if there are conflicting files.")
		s.logger.Muted("  Resolve conflicts manually in ~/.claude, then run claude-sync again.")
		s.logger.Newline()
		return err
	}
	s.logger.Success("✓", "Histories merged successfully")
	s.logger.Newline()

	err = s.prompter.SpinWhile("Pushing merged config...", func() error {
		return s.git.PushWithUpstream(ctx, claudeDir)
	})
	if err != nil {
		s.logger.Error("✗", "Failed to push", err)
		return err
	}
	s.logger.Success("✓", "Pushed to remote")
	s.logger.Newline()

	s.logger.Success("🎉", "Setup complete!")
	s.logger.Info("💡", "Histories merged successfully!")
	s.logger.Muted("  Your local and remote configurations have been combined.")
	s.logger.Muted("  Run 'claude-sync' anytime to keep them synchronized.")
	s.logger.Newline()

	return nil
}
