// Package git provides operations for git repository management.
//
// This package wraps common git operations like init, commit, pull, push,
// and remote validation. It's designed specifically for managing Claude Code
// configuration repositories.
//
// All functions accept a context.Context as their first parameter for
// cancellation support. Git operations can be interrupted gracefully by
// cancelling the context (e.g., via Ctrl+C).
//
// Thread Safety: Functions in this package are NOT thread-safe.
// Callers must ensure that only one operation is performed on a
// given repository at a time.
package git

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const claudeDir = ".claude"

// GetClaudeDir returns the path to ~/.claude
func GetClaudeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	claudePath := filepath.Join(home, claudeDir)

	// Check if directory exists
	if _, err := os.Stat(claudePath); os.IsNotExist(err) {
		return "", fmt.Errorf("~/.claude directory not found")
	}

	return claudePath, nil
}

// HasUncommittedChanges checks if there are uncommitted changes
func HasUncommittedChanges(ctx context.Context, repoPath string) (bool, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "diff-index", "--quiet", "HEAD", "--")
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Exit code 1 means there are changes
			if exitErr.ExitCode() == 1 {
				return true, nil
			}
		}
		return false, fmt.Errorf("failed to check git status: %w", err)
	}
	return false, nil
}

// GetChangedFiles returns list of modified files
func GetChangedFiles(ctx context.Context, repoPath string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "diff", "--name-only", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get changed files: %w", err)
	}

	files := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(files) == 1 && files[0] == "" {
		return []string{}, nil
	}
	return files, nil
}

// CommitChanges commits all tracked changes
func CommitChanges(ctx context.Context, repoPath string, message string) error {
	// Stage all tracked files
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "add", "-u")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to stage changes: %w\nOutput: %s", err, string(output))
	}

	// Commit
	cmd = exec.CommandContext(ctx, "git", "-C", repoPath, "commit", "-m", message)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to commit: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// PullWithRebase pulls from remote with rebase
func PullWithRebase(ctx context.Context, repoPath string) error {
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "pull", "--rebase")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to pull: %w\nOutput: %s", err, string(output))
	}
	return nil
}

// Push pushes to remote
func Push(ctx context.Context, repoPath string) error {
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "push")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to push: %w\nOutput: %s", err, string(output))
	}
	return nil
}

// GetBranchInfo returns current branch and ahead/behind counts
func GetBranchInfo(ctx context.Context, repoPath string) (branch string, ahead, behind int, err error) {
	// Get branch name
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "branch", "--show-current")
	output, err := cmd.Output()
	if err != nil {
		return "", 0, 0, fmt.Errorf("failed to get branch: %w", err)
	}
	branch = strings.TrimSpace(string(output))

	// Get ahead/behind counts
	cmd = exec.CommandContext(ctx, "git", "-C", repoPath, "rev-list", "--left-right", "--count", "HEAD...@{upstream}")
	output, err = cmd.Output()
	if err != nil {
		// No upstream or other error, return branch only
		return branch, 0, 0, nil
	}

	parts := strings.Fields(strings.TrimSpace(string(output)))
	if len(parts) == 2 {
		if _, err := fmt.Sscanf(parts[0], "%d", &ahead); err != nil {
			return branch, 0, 0, fmt.Errorf("failed to parse ahead count: %w", err)
		}
		if _, err := fmt.Sscanf(parts[1], "%d", &behind); err != nil {
			return branch, 0, 0, fmt.Errorf("failed to parse behind count: %w", err)
		}
	}

	return branch, ahead, behind, nil
}

// GetRecentCommits returns recent commit messages
func GetRecentCommits(ctx context.Context, repoPath string, count int) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "log", fmt.Sprintf("-%d", count), "--pretty=format:%h %s")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get commits: %w", err)
	}

	commits := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(commits) == 1 && commits[0] == "" {
		return []string{}, nil
	}
	return commits, nil
}

// GenerateAutoCommitMessage creates a timestamp-based commit message
func GenerateAutoCommitMessage() string {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	return fmt.Sprintf("Auto-sync from %s at %s", hostname, timestamp)
}

// IsGitRepo checks if the directory is a git repository
func IsGitRepo(repoPath string) bool {
	gitDir := filepath.Join(repoPath, ".git")
	info, err := os.Stat(gitDir)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// InitRepo initializes a new git repository
func InitRepo(ctx context.Context, repoPath string) error {
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "init")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to initialize git repo: %w\nOutput: %s", err, string(output))
	}
	return nil
}

// ValidateRemote checks if a remote repository exists and is accessible
func ValidateRemote(ctx context.Context, remoteURL string) error {
	cmd := exec.CommandContext(ctx, "git", "ls-remote", remoteURL)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("remote repository not accessible: %w\nOutput: %s", err, string(output))
	}
	return nil
}

// AddRemote adds a remote repository
func AddRemote(ctx context.Context, repoPath, name, url string) error {
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "remote", "add", name, url)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to add remote: %w\nOutput: %s", err, string(output))
	}
	return nil
}

// InitialCommit creates the initial commit with all files
func InitialCommit(ctx context.Context, repoPath, message string) error {
	// Stage all files
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "add", ".")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to stage files: %w\nOutput: %s", err, string(output))
	}

	// Commit
	cmd = exec.CommandContext(ctx, "git", "-C", repoPath, "commit", "-m", message)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create initial commit: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// SetupGitignore creates a .gitignore file with sensible defaults
func SetupGitignore(repoPath string) error {
	gitignoreContent := `# Credentials and secrets
credentials.json
*.key
*.pem
*.p12
*-key.json
service-account*.json

# AWS scripts (may contain credentials)
aws-*.sh

# Environment files
.env
.env.*

# IDE and editor files
.vscode/
.idea/
*.swp
*.swo
*~

# OS files
.DS_Store
Thumbs.db

# Logs
*.log
`

	gitignorePath := filepath.Join(repoPath, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0o644); err != nil {
		return fmt.Errorf("failed to create .gitignore: %w", err)
	}

	return nil
}

// HasConflicts checks if there are merge conflicts
func HasConflicts(ctx context.Context, repoPath string) (bool, error) {
	// Check for unmerged paths
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "diff", "--name-only", "--diff-filter=U")
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to check for conflicts: %w", err)
	}

	conflicts := strings.TrimSpace(string(output))
	return conflicts != "", nil
}

// AbortRebase aborts an ongoing rebase
func AbortRebase(ctx context.Context, repoPath string) error {
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "rebase", "--abort")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to abort rebase: %w\nOutput: %s", err, string(output))
	}
	return nil
}
