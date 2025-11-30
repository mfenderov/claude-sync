package sync

import (
	"context"

	"github.com/mfenderov/claude-sync/internal/git"
	"github.com/mfenderov/claude-sync/internal/logger"
	"github.com/mfenderov/claude-sync/internal/prompts"
)

// LoggerAdapter adapts logger.Logger to the Logger interface.
type LoggerAdapter struct {
	log *logger.Logger
}

// NewLoggerAdapter creates a new LoggerAdapter.
func NewLoggerAdapter(log *logger.Logger) *LoggerAdapter {
	return &LoggerAdapter{log: log}
}

func (l *LoggerAdapter) Title(title string)                  { l.log.Title(title) }
func (l *LoggerAdapter) Success(icon, message string)        { l.log.Success(icon, message) }
func (l *LoggerAdapter) Error(icon, message string, _ error) { l.log.Error(icon, message, nil) }
func (l *LoggerAdapter) Warning(icon, message string)        { l.log.Warning(icon, message) }
func (l *LoggerAdapter) Info(icon, message string)           { l.log.InfoMsg(icon, message) }
func (l *LoggerAdapter) Muted(message string)                { l.log.Muted(message) }
func (l *LoggerAdapter) ListItem(message string)             { l.log.ListItem(message) }
func (l *LoggerAdapter) Box(title, content string)           { l.log.Box(title, content) }
func (l *LoggerAdapter) Newline()                            { l.log.Newline() }

// PrompterAdapter adapts the prompts package to the Prompter interface.
type PrompterAdapter struct{}

// NewPrompterAdapter creates a new PrompterAdapter.
func NewPrompterAdapter() *PrompterAdapter {
	return &PrompterAdapter{}
}

func (p *PrompterAdapter) Confirm(prompt string) (bool, error) {
	return prompts.Confirm(prompt)
}

func (p *PrompterAdapter) Input(prompt, placeholder string) (string, error) {
	return prompts.Input(prompt, placeholder)
}

func (p *PrompterAdapter) Select(prompt string, options []SelectOption) (string, error) {
	// Convert SelectOption to prompts.Option
	promptOptions := make([]prompts.Option, len(options))
	for i, opt := range options {
		promptOptions[i] = prompts.Option{
			Label: opt.Label,
			Value: opt.Value,
		}
	}
	return prompts.Select(prompt, promptOptions)
}

func (p *PrompterAdapter) SpinWhile(message string, task func() error) error {
	return prompts.SpinWhile(message, task)
}

// GitAdapter adapts the git package to the GitOperator interface.
type GitAdapter struct{}

// NewGitAdapter creates a new GitAdapter.
func NewGitAdapter() *GitAdapter {
	return &GitAdapter{}
}

// Directory operations
func (g *GitAdapter) ClaudeDirExists() (bool, error)    { return git.ClaudeDirExists() }
func (g *GitAdapter) ClaudeDirPath() (string, error)    { return git.ClaudeDirPath() }
func (g *GitAdapter) GetClaudeDir() (string, error)     { return git.GetClaudeDir() }
func (g *GitAdapter) CreateClaudeDir(path string) error { return git.CreateClaudeDir(path) }
func (g *GitAdapter) RemoveClaudeDir() error            { return git.RemoveClaudeDir() }
func (g *GitAdapter) IsGitRepo(path string) bool        { return git.IsGitRepo(path) }

// Repository operations
func (g *GitAdapter) InitRepo(ctx context.Context, path string) error {
	return git.InitRepo(ctx, path)
}

func (g *GitAdapter) CloneRepo(ctx context.Context, remoteURL, destPath string) error {
	return git.CloneRepo(ctx, remoteURL, destPath)
}

func (g *GitAdapter) SetupGitignore(path string) error {
	return git.SetupGitignore(path)
}

func (g *GitAdapter) InitialCommit(ctx context.Context, path, message string) error {
	return git.InitialCommit(ctx, path, message)
}

// Remote operations
func (g *GitAdapter) ValidateRemote(ctx context.Context, remoteURL string) error {
	return git.ValidateRemote(ctx, remoteURL)
}

func (g *GitAdapter) RemoteHasCommits(ctx context.Context, remoteURL string) (bool, error) {
	return git.RemoteHasCommits(ctx, remoteURL)
}

func (g *GitAdapter) AddRemote(ctx context.Context, path, name, url string) error {
	return git.AddRemote(ctx, path, name, url)
}

func (g *GitAdapter) Fetch(ctx context.Context, path string) error {
	return git.Fetch(ctx, path)
}

// Sync operations
func (g *GitAdapter) HasUncommittedChanges(ctx context.Context, path string) (bool, error) {
	return git.HasUncommittedChanges(ctx, path)
}

func (g *GitAdapter) GetChangedFiles(ctx context.Context, path string) ([]string, error) {
	return git.GetChangedFiles(ctx, path)
}

func (g *GitAdapter) CommitChanges(ctx context.Context, path, message string) error {
	return git.CommitChanges(ctx, path, message)
}

func (g *GitAdapter) PullWithRebase(ctx context.Context, path string) error {
	return git.PullWithRebase(ctx, path)
}

func (g *GitAdapter) PullAllowUnrelatedHistories(ctx context.Context, path string) error {
	return git.PullAllowUnrelatedHistories(ctx, path)
}

func (g *GitAdapter) Push(ctx context.Context, path string) error {
	return git.Push(ctx, path)
}

func (g *GitAdapter) PushWithUpstream(ctx context.Context, path string) error {
	return git.PushWithUpstream(ctx, path)
}

// Info operations
func (g *GitAdapter) GetBranchInfo(ctx context.Context, path string) (branch string, ahead, behind int, err error) {
	return git.GetBranchInfo(ctx, path)
}

func (g *GitAdapter) GetRecentCommits(ctx context.Context, path string, count int) ([]string, error) {
	return git.GetRecentCommits(ctx, path, count)
}

func (g *GitAdapter) HasConflicts(ctx context.Context, path string) (bool, error) {
	return git.HasConflicts(ctx, path)
}

func (g *GitAdapter) AbortRebase(ctx context.Context, path string) error {
	return git.AbortRebase(ctx, path)
}

func (g *GitAdapter) GenerateAutoCommitMessage() string {
	return git.GenerateAutoCommitMessage()
}
