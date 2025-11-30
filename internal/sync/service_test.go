package sync

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
)

func TestService_Run_NormalSync(t *testing.T) {
	t.Parallel()

	// Setup mocks
	git := NewMockGitOperator(t)
	prompter := NewMockPrompter(t)
	logger := NewMockLogger(t)

	claudeDir := "/home/user/.claude"

	// Setup expectations for normal sync flow
	git.EXPECT().ClaudeDirExists().Return(true, nil)
	git.EXPECT().GetClaudeDir().Return(claudeDir, nil)
	git.EXPECT().IsGitRepo(claudeDir).Return(true)
	git.EXPECT().GetChangedFiles(mock.Anything, claudeDir).Return([]string{"settings.json"}, nil)
	git.EXPECT().GenerateAutoCommitMessage().Return("Auto-sync: 2024-01-01")
	git.EXPECT().CommitChanges(mock.Anything, claudeDir, "Auto-sync: 2024-01-01").Return(nil)
	git.EXPECT().PullWithRebase(mock.Anything, claudeDir).Return(nil)
	git.EXPECT().Push(mock.Anything, claudeDir).Return(nil)
	git.EXPECT().GetRecentCommits(mock.Anything, claudeDir, 5).Return([]string{"abc123 Previous commit"}, nil)
	git.EXPECT().GetBranchInfo(mock.Anything, claudeDir).Return("main", 0, 0, nil).Maybe()

	// Logger expectations (allow any calls)
	logger.EXPECT().Title(mock.Anything).Maybe()
	logger.EXPECT().Success(mock.Anything, mock.Anything).Maybe()
	logger.EXPECT().Info(mock.Anything, mock.Anything).Maybe()
	logger.EXPECT().Muted(mock.Anything).Maybe()
	logger.EXPECT().ListItem(mock.Anything).Maybe()
	logger.EXPECT().Box(mock.Anything, mock.Anything).Maybe()
	logger.EXPECT().Newline().Maybe()

	// Prompter expectations
	prompter.EXPECT().SpinWhile(mock.Anything, mock.Anything).RunAndReturn(func(msg string, task func() error) error {
		return task()
	}).Maybe()

	service := NewService(git, prompter, logger)
	ctx := context.Background()

	err := service.Run(ctx)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestService_Run_NoChanges(t *testing.T) {
	t.Parallel()

	git := NewMockGitOperator(t)
	prompter := NewMockPrompter(t)
	logger := NewMockLogger(t)

	claudeDir := "/home/user/.claude"

	// Setup expectations
	git.EXPECT().ClaudeDirExists().Return(true, nil)
	git.EXPECT().GetClaudeDir().Return(claudeDir, nil)
	git.EXPECT().IsGitRepo(claudeDir).Return(true)
	git.EXPECT().GetChangedFiles(mock.Anything, claudeDir).Return([]string{}, nil) // No changes
	git.EXPECT().PullWithRebase(mock.Anything, claudeDir).Return(nil)
	git.EXPECT().Push(mock.Anything, claudeDir).Return(nil)
	git.EXPECT().GetRecentCommits(mock.Anything, claudeDir, 5).Return([]string{}, nil)

	// Logger expectations
	logger.EXPECT().Title(mock.Anything).Maybe()
	logger.EXPECT().Success(mock.Anything, mock.Anything).Maybe()
	logger.EXPECT().Newline().Maybe()
	logger.EXPECT().Box(mock.Anything, mock.Anything).Maybe()

	// Prompter expectations
	prompter.EXPECT().SpinWhile(mock.Anything, mock.Anything).RunAndReturn(func(msg string, task func() error) error {
		return task()
	}).Maybe()

	service := NewService(git, prompter, logger)
	ctx := context.Background()

	err := service.Run(ctx)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// CommitChanges should NOT have been called (verified by mock expectations)
}

func TestService_InitFlow_EmptyRemote_FreshInit(t *testing.T) {
	t.Parallel()

	git := NewMockGitOperator(t)
	prompter := NewMockPrompter(t)
	logger := NewMockLogger(t)

	claudeDir := "/home/user/.claude"
	remoteURL := "git@github.com:user/claude-config.git"

	// Setup expectations for init flow with empty remote
	git.EXPECT().ClaudeDirExists().Return(true, nil)
	git.EXPECT().GetClaudeDir().Return(claudeDir, nil)
	git.EXPECT().IsGitRepo(claudeDir).Return(false) // Not a git repo yet
	git.EXPECT().ValidateRemote(mock.Anything, remoteURL).Return(nil)
	git.EXPECT().RemoteHasCommits(mock.Anything, remoteURL).Return(false, nil) // Empty remote
	git.EXPECT().InitRepo(mock.Anything, claudeDir).Return(nil)
	git.EXPECT().SetupGitignore(claudeDir).Return(nil)
	git.EXPECT().InitialCommit(mock.Anything, claudeDir, "Initial Claude Code configuration").Return(nil)
	git.EXPECT().AddRemote(mock.Anything, claudeDir, "origin", remoteURL).Return(nil)
	git.EXPECT().PushWithUpstream(mock.Anything, claudeDir).Return(nil)

	// Prompter expectations
	prompter.EXPECT().Confirm("🤔 Would you like to set up git sync now?").Return(true, nil)
	prompter.EXPECT().Input(mock.Anything, mock.Anything).Return(remoteURL, nil)
	prompter.EXPECT().SpinWhile(mock.Anything, mock.Anything).RunAndReturn(func(msg string, task func() error) error {
		return task()
	}).Maybe()

	// Logger expectations
	logger.EXPECT().Title(mock.Anything).Maybe()
	logger.EXPECT().Success(mock.Anything, mock.Anything).Maybe()
	logger.EXPECT().Info(mock.Anything, mock.Anything).Maybe()
	logger.EXPECT().Muted(mock.Anything).Maybe()
	logger.EXPECT().Newline().Maybe()

	service := NewService(git, prompter, logger)
	ctx := context.Background()

	err := service.Run(ctx)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestService_InitFlow_RemoteHasCommits_UserChoosesReplace(t *testing.T) {
	t.Parallel()

	git := NewMockGitOperator(t)
	prompter := NewMockPrompter(t)
	logger := NewMockLogger(t)

	claudeDir := "/home/user/.claude"
	remoteURL := "git@github.com:user/claude-config.git"

	// Setup expectations
	git.EXPECT().ClaudeDirExists().Return(true, nil)
	git.EXPECT().GetClaudeDir().Return(claudeDir, nil)
	git.EXPECT().IsGitRepo(claudeDir).Return(false)
	git.EXPECT().ValidateRemote(mock.Anything, remoteURL).Return(nil)
	git.EXPECT().RemoteHasCommits(mock.Anything, remoteURL).Return(true, nil) // Remote has commits!
	git.EXPECT().RemoveClaudeDir().Return(nil)
	git.EXPECT().CloneRepo(mock.Anything, remoteURL, claudeDir).Return(nil)

	// Prompter expectations
	prompter.EXPECT().Confirm("🤔 Would you like to set up git sync now?").Return(true, nil)
	prompter.EXPECT().Input(mock.Anything, mock.Anything).Return(remoteURL, nil)
	prompter.EXPECT().Select("How would you like to proceed?", mock.Anything).Return("replace", nil)
	prompter.EXPECT().SpinWhile(mock.Anything, mock.Anything).RunAndReturn(func(msg string, task func() error) error {
		return task()
	}).Maybe()

	// Logger expectations
	logger.EXPECT().Title(mock.Anything).Maybe()
	logger.EXPECT().Success(mock.Anything, mock.Anything).Maybe()
	logger.EXPECT().Info(mock.Anything, mock.Anything).Maybe()
	logger.EXPECT().Warning(mock.Anything, mock.Anything).Maybe()
	logger.EXPECT().Muted(mock.Anything).Maybe()
	logger.EXPECT().Newline().Maybe()

	service := NewService(git, prompter, logger)
	ctx := context.Background()

	err := service.Run(ctx)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestService_InitFlow_RemoteHasCommits_UserChoosesMerge(t *testing.T) {
	t.Parallel()

	git := NewMockGitOperator(t)
	prompter := NewMockPrompter(t)
	logger := NewMockLogger(t)

	claudeDir := "/home/user/.claude"
	remoteURL := "git@github.com:user/claude-config.git"

	// Setup expectations for merge flow
	git.EXPECT().ClaudeDirExists().Return(true, nil)
	git.EXPECT().GetClaudeDir().Return(claudeDir, nil)
	git.EXPECT().IsGitRepo(claudeDir).Return(false)
	git.EXPECT().ValidateRemote(mock.Anything, remoteURL).Return(nil)
	git.EXPECT().RemoteHasCommits(mock.Anything, remoteURL).Return(true, nil)
	git.EXPECT().InitRepo(mock.Anything, claudeDir).Return(nil)
	git.EXPECT().SetupGitignore(claudeDir).Return(nil)
	git.EXPECT().InitialCommit(mock.Anything, claudeDir, "Local Claude Code configuration").Return(nil)
	git.EXPECT().AddRemote(mock.Anything, claudeDir, "origin", remoteURL).Return(nil)
	git.EXPECT().Fetch(mock.Anything, claudeDir).Return(nil)
	git.EXPECT().PullAllowUnrelatedHistories(mock.Anything, claudeDir).Return(nil)
	git.EXPECT().PushWithUpstream(mock.Anything, claudeDir).Return(nil)

	// Prompter expectations
	prompter.EXPECT().Confirm("🤔 Would you like to set up git sync now?").Return(true, nil)
	prompter.EXPECT().Input(mock.Anything, mock.Anything).Return(remoteURL, nil)
	prompter.EXPECT().Select("How would you like to proceed?", mock.Anything).Return("merge", nil)
	prompter.EXPECT().SpinWhile(mock.Anything, mock.Anything).RunAndReturn(func(msg string, task func() error) error {
		return task()
	}).Maybe()

	// Logger expectations
	logger.EXPECT().Title(mock.Anything).Maybe()
	logger.EXPECT().Success(mock.Anything, mock.Anything).Maybe()
	logger.EXPECT().Info(mock.Anything, mock.Anything).Maybe()
	logger.EXPECT().Warning(mock.Anything, mock.Anything).Maybe()
	logger.EXPECT().Muted(mock.Anything).Maybe()
	logger.EXPECT().Newline().Maybe()

	service := NewService(git, prompter, logger)
	ctx := context.Background()

	err := service.Run(ctx)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestService_InitFlow_RemoteHasCommits_UserCancels(t *testing.T) {
	t.Parallel()

	git := NewMockGitOperator(t)
	prompter := NewMockPrompter(t)
	logger := NewMockLogger(t)

	claudeDir := "/home/user/.claude"
	remoteURL := "git@github.com:user/claude-config.git"

	// Setup expectations
	git.EXPECT().ClaudeDirExists().Return(true, nil)
	git.EXPECT().GetClaudeDir().Return(claudeDir, nil)
	git.EXPECT().IsGitRepo(claudeDir).Return(false)
	git.EXPECT().ValidateRemote(mock.Anything, remoteURL).Return(nil)
	git.EXPECT().RemoteHasCommits(mock.Anything, remoteURL).Return(true, nil)

	// Prompter expectations - user cancels
	prompter.EXPECT().Confirm("🤔 Would you like to set up git sync now?").Return(true, nil)
	prompter.EXPECT().Input(mock.Anything, mock.Anything).Return(remoteURL, nil)
	prompter.EXPECT().Select("How would you like to proceed?", mock.Anything).Return("cancel", nil)
	prompter.EXPECT().SpinWhile(mock.Anything, mock.Anything).RunAndReturn(func(msg string, task func() error) error {
		return task()
	}).Maybe()

	// Logger expectations
	logger.EXPECT().Title(mock.Anything).Maybe()
	logger.EXPECT().Success(mock.Anything, mock.Anything).Maybe()
	logger.EXPECT().Info(mock.Anything, mock.Anything).Maybe()
	logger.EXPECT().Warning(mock.Anything, mock.Anything).Maybe()
	logger.EXPECT().Muted(mock.Anything).Maybe()
	logger.EXPECT().Newline().Maybe()

	service := NewService(git, prompter, logger)
	ctx := context.Background()

	err := service.Run(ctx)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestService_FirstTimeSetup_Clone(t *testing.T) {
	t.Parallel()

	git := NewMockGitOperator(t)
	prompter := NewMockPrompter(t)
	logger := NewMockLogger(t)

	claudeDir := "/home/user/.claude"
	remoteURL := "git@github.com:user/claude-config.git"

	// Setup expectations - directory doesn't exist
	git.EXPECT().ClaudeDirExists().Return(false, nil)
	git.EXPECT().ClaudeDirPath().Return(claudeDir, nil)
	git.EXPECT().CloneRepo(mock.Anything, remoteURL, claudeDir).Return(nil)

	// Prompter expectations
	prompter.EXPECT().Select("What would you like to do?", mock.Anything).Return("clone", nil)
	prompter.EXPECT().Input(mock.Anything, mock.Anything).Return(remoteURL, nil)
	prompter.EXPECT().SpinWhile(mock.Anything, mock.Anything).RunAndReturn(func(msg string, task func() error) error {
		return task()
	}).Maybe()

	// Logger expectations
	logger.EXPECT().Title(mock.Anything).Maybe()
	logger.EXPECT().Success(mock.Anything, mock.Anything).Maybe()
	logger.EXPECT().Info(mock.Anything, mock.Anything).Maybe()
	logger.EXPECT().Muted(mock.Anything).Maybe()
	logger.EXPECT().Newline().Maybe()

	service := NewService(git, prompter, logger)
	ctx := context.Background()

	err := service.Run(ctx)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestService_UserDeclinesSetup(t *testing.T) {
	t.Parallel()

	git := NewMockGitOperator(t)
	prompter := NewMockPrompter(t)
	logger := NewMockLogger(t)

	claudeDir := "/home/user/.claude"

	// Setup expectations
	git.EXPECT().ClaudeDirExists().Return(true, nil)
	git.EXPECT().GetClaudeDir().Return(claudeDir, nil)
	git.EXPECT().IsGitRepo(claudeDir).Return(false)

	// User declines setup
	prompter.EXPECT().Confirm("🤔 Would you like to set up git sync now?").Return(false, nil)

	// Logger expectations
	logger.EXPECT().Title(mock.Anything).Maybe()
	logger.EXPECT().Info(mock.Anything, mock.Anything).Maybe()
	logger.EXPECT().Muted(mock.Anything).Maybe()
	logger.EXPECT().Newline().Maybe()

	service := NewService(git, prompter, logger)
	ctx := context.Background()

	err := service.Run(ctx)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}
