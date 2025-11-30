// Package sync provides the business logic for syncing Claude configurations.
// It uses dependency injection for testability, separating business logic
// from TUI interactions and git operations.
package sync

import "context"

// Prompter defines the interface for user interaction.
// This allows the business logic to be tested with mock implementations.
type Prompter interface {
	// Confirm asks for yes/no confirmation
	Confirm(prompt string) (bool, error)

	// Input asks for text input with a placeholder
	Input(prompt, placeholder string) (string, error)

	// Select presents options and returns the selected value
	Select(prompt string, options []SelectOption) (string, error)

	// SpinWhile shows a spinner while executing a task
	SpinWhile(message string, task func() error) error
}

// SelectOption represents a choice in a select prompt
type SelectOption struct {
	Label string
	Value string
}

// Logger defines the interface for output messages.
// This allows the business logic to be tested without console output.
type Logger interface {
	Title(title string)
	Success(icon, message string)
	Error(icon, message string, err error)
	Warning(icon, message string)
	Info(icon, message string)
	Muted(message string)
	ListItem(message string)
	Box(title, content string)
	Newline()
}

// GitOperator defines the interface for git operations.
// This allows the business logic to be tested with mock git operations.
type GitOperator interface {
	// Directory operations
	ClaudeDirExists() (bool, error)
	ClaudeDirPath() (string, error)
	GetClaudeDir() (string, error)
	CreateClaudeDir(path string) error
	RemoveClaudeDir() error
	IsGitRepo(path string) bool

	// Repository operations
	InitRepo(ctx context.Context, path string) error
	CloneRepo(ctx context.Context, remoteURL, destPath string) error
	SetupGitignore(path string) error
	InitialCommit(ctx context.Context, path, message string) error

	// Remote operations
	ValidateRemote(ctx context.Context, remoteURL string) error
	RemoteHasCommits(ctx context.Context, remoteURL string) (bool, error)
	AddRemote(ctx context.Context, path, name, url string) error
	Fetch(ctx context.Context, path string) error

	// Sync operations
	HasUncommittedChanges(ctx context.Context, path string) (bool, error)
	GetChangedFiles(ctx context.Context, path string) ([]string, error)
	CommitChanges(ctx context.Context, path, message string) error
	PullWithRebase(ctx context.Context, path string) error
	PullAllowUnrelatedHistories(ctx context.Context, path string) error
	Push(ctx context.Context, path string) error
	PushWithUpstream(ctx context.Context, path string) error

	// Info operations
	GetBranchInfo(ctx context.Context, path string) (branch string, ahead, behind int, err error)
	GetRecentCommits(ctx context.Context, path string, count int) ([]string, error)
	HasConflicts(ctx context.Context, path string) (bool, error)
	AbortRebase(ctx context.Context, path string) error
	GenerateAutoCommitMessage() string
}
