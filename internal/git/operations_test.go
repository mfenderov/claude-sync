package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var _ = context.Background

func createTestRepo(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()

	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to configure git email: %v", err)
	}

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to configure git name: %v", err)
	}

	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("initial content"), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to add files: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create initial commit: %v", err)
	}

	return tmpDir
}

func TestIsGitRepo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		setup    func(*testing.T) string
		name     string
		expected bool
	}{
		{
			name: "valid git repository",
			setup: func(t *testing.T) string {
				t.Helper()
				return createTestRepo(t)
			},
			expected: true,
		},
		{
			name: "not a git repository",
			setup: func(t *testing.T) string {
				t.Helper()
				return t.TempDir()
			},
			expected: false,
		},
		{
			name: "non-existent directory",
			setup: func(t *testing.T) string {
				t.Helper()
				return "/path/that/does/not/exist"
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := tt.setup(t)

			result := IsGitRepo(dir)
			if result != tt.expected {
				t.Errorf("IsGitRepo() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestHasUncommittedChanges(t *testing.T) {
	t.Parallel()

	tests := []struct {
		setup    func(string)
		name     string
		expected bool
	}{
		{
			name: "no changes",
			setup: func(dir string) {
			},
			expected: false,
		},
		{
			name: "modified file",
			setup: func(dir string) {
				testFile := filepath.Join(dir, "test.txt")
				_ = os.WriteFile(testFile, []byte("modified content"), 0o644)
			},
			expected: true,
		},
		{
			name: "new untracked file",
			setup: func(dir string) {
				newFile := filepath.Join(dir, "new.txt")
				_ = os.WriteFile(newFile, []byte("new content"), 0o644)
			},
			expected: true, // Untracked files should be detected for syncing
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := createTestRepo(t)

			tt.setup(dir)

			hasChanges, err := HasUncommittedChanges(t.Context(), dir)
			if err != nil {
				t.Fatalf("HasUncommittedChanges() error = %v", err)
			}

			if hasChanges != tt.expected {
				t.Errorf("HasUncommittedChanges() = %v, want %v", hasChanges, tt.expected)
			}
		})
	}
}

func TestGetChangedFiles(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	dir := createTestRepo(t)

	files, err := GetChangedFiles(ctx, dir)
	if err != nil {
		t.Fatalf("GetChangedFiles() error = %v", err)
	}
	if len(files) != 0 {
		t.Errorf("GetChangedFiles() = %v, want empty slice", files)
	}

	testFile := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(testFile, []byte("modified"), 0o644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	files, err = GetChangedFiles(ctx, dir)
	if err != nil {
		t.Fatalf("GetChangedFiles() error = %v", err)
	}
	if len(files) != 1 {
		t.Errorf("GetChangedFiles() returned %d files, want 1", len(files))
	}
	if len(files) == 1 && files[0] != "test.txt" {
		t.Errorf("GetChangedFiles() = %v, want [test.txt]", files)
	}
}

func TestCommitChanges(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	dir := createTestRepo(t)

	testFile := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(testFile, []byte("modified"), 0o644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	commitMsg := "Test commit"
	err := CommitChanges(ctx, dir, commitMsg)
	if err != nil {
		t.Fatalf("CommitChanges() error = %v", err)
	}

	hasChanges, err := HasUncommittedChanges(ctx, dir)
	if err != nil {
		t.Fatalf("HasUncommittedChanges() error = %v", err)
	}
	if hasChanges {
		t.Error("CommitChanges() did not commit all changes")
	}

	cmd := exec.Command("git", "log", "-1", "--pretty=%s")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to get last commit message: %v", err)
	}

	actualMsg := string(output)
	if actualMsg[:len(commitMsg)] != commitMsg {
		t.Errorf("Commit message = %q, want %q", actualMsg, commitMsg)
	}
}

func TestGetBranchInfo(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	dir := createTestRepo(t)

	branch, ahead, behind, err := GetBranchInfo(ctx, dir)
	if err != nil {
		t.Fatalf("GetBranchInfo() error = %v", err)
	}

	if branch != "main" && branch != "master" {
		t.Errorf("GetBranchInfo() branch = %q, want main or master", branch)
	}

	if ahead != 0 || behind != 0 {
		t.Errorf("GetBranchInfo() ahead=%d, behind=%d, want 0, 0", ahead, behind)
	}
}

func TestGetRecentCommits(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	dir := createTestRepo(t)

	for i := 0; i < 3; i++ {
		testFile := filepath.Join(dir, "test.txt")
		content := []byte("content " + string(rune('A'+i)))
		if err := os.WriteFile(testFile, content, 0o644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		cmd := exec.Command("git", "add", ".")
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to add files: %v", err)
		}

		cmd = exec.Command("git", "commit", "-m", "Commit "+string(rune('A'+i)))
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to commit: %v", err)
		}
	}

	commits, err := GetRecentCommits(ctx, dir, 2)
	if err != nil {
		t.Fatalf("GetRecentCommits() error = %v", err)
	}

	if len(commits) != 2 {
		t.Errorf("GetRecentCommits(2) returned %d commits, want 2", len(commits))
	}
}

func TestGenerateAutoCommitMessage(t *testing.T) {
	t.Parallel()

	msg := GenerateAutoCommitMessage()

	if len(msg) == 0 {
		t.Error("GenerateAutoCommitMessage() returned empty string")
	}

	if len(msg) < 20 {
		t.Errorf("GenerateAutoCommitMessage() = %q, seems too short", msg)
	}
}

func TestGetClaudeDir(t *testing.T) {
	originalHome := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", originalHome) }()

	_, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("Cannot get home directory: %v", err)
	}

	tmpHome := t.TempDir()

	claudeDir := filepath.Join(tmpHome, ".claude")
	if err := os.Mkdir(claudeDir, 0o755); err != nil {
		t.Fatalf("Failed to create .claude dir: %v", err)
	}

	if err := os.Setenv("HOME", tmpHome); err != nil {
		t.Fatalf("Failed to set HOME: %v", err)
	}

	dir, err := GetClaudeDir()
	if err != nil {
		t.Errorf("GetClaudeDir() error = %v", err)
	}

	if dir != claudeDir {
		t.Errorf("GetClaudeDir() = %q, want %q", dir, claudeDir)
	}

	if err := os.RemoveAll(claudeDir); err != nil {
		t.Fatalf("Failed to remove .claude dir: %v", err)
	}
	_, err = GetClaudeDir()
	if err == nil {
		t.Error("GetClaudeDir() should error when .claude doesn't exist")
	}
}

func TestEnsureDefaultBranch(t *testing.T) {
	t.Parallel()

	repoPath := createTestRepo(t)
	ctx := context.Background()

	// Test 1: Branch should be renamed to 'main' if on 'master'
	cmd := exec.Command("git", "branch", "-M", "master")
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to rename to master: %v", err)
	}

	if err := ensureDefaultBranch(ctx, repoPath); err != nil {
		t.Fatalf("ensureDefaultBranch failed: %v", err)
	}

	branch, err := getCurrentBranch(ctx, repoPath)
	if err != nil {
		t.Fatalf("Failed to get current branch: %v", err)
	}

	if branch != "main" {
		t.Errorf("Expected branch 'main', got '%s'", branch)
	}
}

func TestGetCurrentBranch(t *testing.T) {
	t.Parallel()

	repoPath := createTestRepo(t)
	ctx := context.Background()

	branch, err := getCurrentBranch(ctx, repoPath)
	if err != nil {
		t.Fatalf("Failed to get current branch: %v", err)
	}

	// Should be either main or master depending on git config
	if branch != "main" && branch != "master" {
		t.Errorf("Expected branch 'main' or 'master', got '%s'", branch)
	}
}

func TestEnhancePushError(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		output        string
		expectedInMsg string
	}{
		{
			name:          "SSH permission denied",
			output:        "Permission denied (publickey)",
			expectedInMsg: "SSH authentication failed",
		},
		{
			name:          "Authentication failed",
			output:        "Authentication failed for",
			expectedInMsg: "authentication failed",
		},
		{
			name:          "Network timeout",
			output:        "connection timed out",
			expectedInMsg: "network error",
		},
		{
			name:          "Repository not found",
			output:        "repository not found",
			expectedInMsg: "repository not found",
		},
		{
			name:          "Generic error",
			output:        "some random error",
			expectedInMsg: "failed to push",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := enhancePushError(context.DeadlineExceeded, tc.output)
			errMsg := err.Error()

			if !strings.Contains(errMsg, tc.expectedInMsg) {
				t.Errorf("Expected error message to contain '%s', got: %s", tc.expectedInMsg, errMsg)
			}

			// Verify that helpful guidance is included
			if !strings.Contains(errMsg, "Common fixes") && tc.name != "Generic error" {
				t.Errorf("Expected error message to contain 'Common fixes', got: %s", errMsg)
			}
		})
	}
}

func TestEnhancePullError(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		output        string
		expectedInMsg string
	}{
		{
			name:          "No upstream configured",
			output:        "There is no tracking information for the current branch",
			expectedInMsg: "no upstream branch configured",
		},
		{
			name:          "SSH permission denied during pull",
			output:        "Permission denied (publickey)",
			expectedInMsg: "SSH authentication failed",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := enhancePullError(context.DeadlineExceeded, tc.output)
			errMsg := err.Error()

			if !strings.Contains(errMsg, tc.expectedInMsg) {
				t.Errorf("Expected error message to contain '%s', got: %s", tc.expectedInMsg, errMsg)
			}
		})
	}
}
