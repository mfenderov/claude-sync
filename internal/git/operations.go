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
		return "", &DirectoryError{
			Path: "~",
			Op:   "get home directory",
			Err:  err,
		}
	}
	claudePath := filepath.Join(home, claudeDir)

	if _, err := os.Stat(claudePath); os.IsNotExist(err) {
		return "", &DirectoryError{
			Path: claudePath,
			Op:   "access",
			Err:  err,
		}
	}

	return claudePath, nil
}

// HasUncommittedChanges checks if there are uncommitted changes
func HasUncommittedChanges(ctx context.Context, repoPath string) (bool, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "diff-index", "--quiet", "HEAD", "--")
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return true, nil
		}
		return false, &OperationError{
			Op:   "diff-index",
			Path: repoPath,
			Err:  err,
		}
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
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "add", "-u")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to stage changes: %w\nOutput: %s", err, string(output))
	}

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
		return enhancePullError(err, string(output))
	}
	return nil
}

// enhancePullError provides contextual help for common pull failures
func enhancePullError(err error, output string) error {
	outputLower := strings.ToLower(output)

	// No upstream branch set
	if strings.Contains(outputLower, "no tracking information") ||
		strings.Contains(outputLower, "there is no tracking information") {
		return fmt.Errorf("no upstream branch configured: %w\n\n"+
			"This usually happens after initial setup.\n"+
			"The sync will set upstream automatically.\n\n"+
			"Output: %s", err, output)
	}

	// Reuse push error enhancement for network/auth issues
	return enhancePushError(err, output)
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

// PushWithUpstream pushes to remote and sets upstream tracking
func PushWithUpstream(ctx context.Context, repoPath string) error {
	// Get current branch name
	branch, err := getCurrentBranch(ctx, repoPath)
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "push", "-u", "origin", branch)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return enhancePushError(err, string(output))
	}
	return nil
}

// enhancePushError provides contextual help for common push failures
func enhancePushError(err error, output string) error {
	outputLower := strings.ToLower(output)

	// SSH key issues
	if strings.Contains(outputLower, "permission denied") ||
		strings.Contains(outputLower, "publickey") {
		return fmt.Errorf("SSH authentication failed: %w\n\n"+
			"Common fixes:\n"+
			"  1. Add your SSH key: ssh-add ~/.ssh/id_rsa\n"+
			"  2. Generate a key: ssh-keygen -t ed25519 -C \"your@email.com\"\n"+
			"  3. Add key to GitHub: https://github.com/settings/keys\n"+
			"  4. Test connection: ssh -T git@github.com\n\n"+
			"Output: %s", err, output)
	}

	// Authentication issues (HTTPS)
	if strings.Contains(outputLower, "authentication failed") ||
		strings.Contains(outputLower, "403") {
		return fmt.Errorf("authentication failed: %w\n\n"+
			"Common fixes:\n"+
			"  1. Check repository access permissions\n"+
			"  2. For HTTPS: Update credentials in keychain/credential manager\n"+
			"  3. For SSH: Ensure SSH keys are set up correctly\n"+
			"  4. Verify repository URL: git remote -v\n\n"+
			"Output: %s", err, output)
	}

	// Network issues
	if strings.Contains(outputLower, "could not resolve host") ||
		strings.Contains(outputLower, "connection timed out") ||
		strings.Contains(outputLower, "network") {
		return fmt.Errorf("network error: %w\n\n"+
			"Common fixes:\n"+
			"  1. Check your internet connection\n"+
			"  2. Verify repository URL is correct\n"+
			"  3. Try again in a moment\n"+
			"  4. Check if git hosting service is accessible\n\n"+
			"Output: %s", err, output)
	}

	// Repository doesn't exist
	if strings.Contains(outputLower, "repository not found") ||
		strings.Contains(outputLower, "does not appear to be a git repository") {
		return fmt.Errorf("repository not found: %w\n\n"+
			"Common fixes:\n"+
			"  1. Verify the repository exists on the remote\n"+
			"  2. Check repository URL: git remote -v\n"+
			"  3. Ensure you have access to the repository\n"+
			"  4. Create the repository if it doesn't exist\n\n"+
			"Output: %s", err, output)
	}

	// Generic error with output
	return fmt.Errorf("failed to push: %w\nOutput: %s", err, output)
}

// getCurrentBranch returns the current branch name
func getCurrentBranch(ctx context.Context, repoPath string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "branch", "--show-current")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// GetBranchInfo returns current branch and ahead/behind counts
func GetBranchInfo(ctx context.Context, repoPath string) (branch string, ahead, behind int, err error) {
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "branch", "--show-current")
	output, err := cmd.Output()
	if err != nil {
		return "", 0, 0, fmt.Errorf("failed to get branch: %w", err)
	}
	branch = strings.TrimSpace(string(output))

	cmd = exec.CommandContext(ctx, "git", "-C", repoPath, "rev-list", "--left-right", "--count", "HEAD...@{upstream}")
	output, err = cmd.Output()
	if err != nil {
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
		return &RemoteError{
			URL: remoteURL,
			Op:  "validate",
			Err: fmt.Errorf("%w: %s", err, string(output)),
		}
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
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "add", ".")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to stage files: %w\nOutput: %s", err, string(output))
	}

	cmd = exec.CommandContext(ctx, "git", "-C", repoPath, "commit", "-m", message)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create initial commit: %w\nOutput: %s", err, string(output))
	}

	// Ensure we're on a branch named 'main' (modern default)
	// This normalizes repos that might init with 'master' or no default branch
	if err := ensureDefaultBranch(ctx, repoPath); err != nil {
		// Non-fatal: log but continue if branch rename fails
		return nil
	}

	return nil
}

// ensureDefaultBranch ensures the repository is on 'main' branch
func ensureDefaultBranch(ctx context.Context, repoPath string) error {
	currentBranch, err := getCurrentBranch(ctx, repoPath)
	if err != nil {
		return err
	}

	// If already on 'main', nothing to do
	if currentBranch == "main" {
		return nil
	}

	// Rename current branch to 'main'
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "branch", "-M", "main")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to rename branch to main: %w\nOutput: %s", err, string(output))
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
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "diff", "--name-only", "--diff-filter=U")
	output, err := cmd.Output()
	if err != nil {
		return false, &OperationError{
			Op:   "check conflicts",
			Path: repoPath,
			Err:  err,
		}
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

// CloneRepo clones a remote repository to the specified path
func CloneRepo(ctx context.Context, remoteURL, destPath string) error {
	cmd := exec.CommandContext(ctx, "git", "clone", remoteURL, destPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return enhanceCloneError(err, string(output), remoteURL)
	}
	return nil
}

// enhanceCloneError provides contextual help for common clone failures
func enhanceCloneError(err error, output, remoteURL string) error {
	outputLower := strings.ToLower(output)

	// SSH key issues
	if strings.Contains(outputLower, "permission denied") ||
		strings.Contains(outputLower, "publickey") {
		return fmt.Errorf("SSH authentication failed: %w\n\n"+
			"Common fixes:\n"+
			"  1. Add your SSH key: ssh-add ~/.ssh/id_rsa\n"+
			"  2. Generate a key: ssh-keygen -t ed25519 -C \"your@email.com\"\n"+
			"  3. Add key to GitHub: https://github.com/settings/keys\n"+
			"  4. Test connection: ssh -T git@github.com\n\n"+
			"Output: %s", err, output)
	}

	// Repository not found
	if strings.Contains(outputLower, "repository not found") ||
		strings.Contains(outputLower, "does not appear to be a git repository") ||
		strings.Contains(outputLower, "not found") {
		return fmt.Errorf("repository not found: %w\n\n"+
			"Please check:\n"+
			"  1. The repository URL is correct: %s\n"+
			"  2. The repository exists\n"+
			"  3. You have access to the repository\n\n"+
			"Output: %s", err, remoteURL, output)
	}

	// Empty repository
	if strings.Contains(outputLower, "empty repository") ||
		strings.Contains(outputLower, "warning: you appear to have cloned an empty repository") {
		return fmt.Errorf("repository is empty: %w\n\n"+
			"The repository exists but has no commits.\n"+
			"If this is a new repo, use 'Start fresh' instead.\n\n"+
			"Output: %s", err, output)
	}

	return fmt.Errorf("failed to clone repository: %w\nOutput: %s", err, output)
}

// ClaudeDirPath returns the path to ~/.claude without checking if it exists
func ClaudeDirPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", &DirectoryError{
			Path: "~",
			Op:   "get home directory",
			Err:  err,
		}
	}
	return filepath.Join(home, claudeDir), nil
}

// ClaudeDirExists checks if ~/.claude directory exists
func ClaudeDirExists() (bool, error) {
	path, err := ClaudeDirPath()
	if err != nil {
		return false, err
	}
	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
