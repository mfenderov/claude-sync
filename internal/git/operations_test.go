package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// Helper function to create a test git repository
func createTestRepo(t *testing.T) string {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "claude-sync-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Configure git user
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("Failed to configure git email: %v", err)
	}

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("Failed to configure git name: %v", err)
	}

	// Create initial commit
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("initial content"), 0o644); err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("Failed to add files: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create initial commit: %v", err)
	}

	return tmpDir
}

func TestIsGitRepo(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() string
		expected bool
	}{
		{
			name: "valid git repository",
			setup: func() string {
				return createTestRepo(t)
			},
			expected: true,
		},
		{
			name: "not a git repository",
			setup: func() string {
				tmpDir, _ := os.MkdirTemp("", "claude-sync-test-*")
				return tmpDir
			},
			expected: false,
		},
		{
			name: "non-existent directory",
			setup: func() string {
				return "/path/that/does/not/exist"
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.setup()
			defer func() { _ = os.RemoveAll(dir) }()

			result := IsGitRepo(dir)
			if result != tt.expected {
				t.Errorf("IsGitRepo() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestHasUncommittedChanges(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(string)
		expected bool
	}{
		{
			name: "no changes",
			setup: func(dir string) {
				// No changes needed
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
			expected: false, // Untracked files don't count as uncommitted changes
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := createTestRepo(t)
			defer func() { _ = os.RemoveAll(dir) }()

			tt.setup(dir)

			hasChanges, err := HasUncommittedChanges(dir)
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
	dir := createTestRepo(t)
	defer func() { _ = os.RemoveAll(dir) }()

	// Test with no changes
	files, err := GetChangedFiles(dir)
	if err != nil {
		t.Fatalf("GetChangedFiles() error = %v", err)
	}
	if len(files) != 0 {
		t.Errorf("GetChangedFiles() = %v, want empty slice", files)
	}

	// Modify a file
	testFile := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(testFile, []byte("modified"), 0o644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	files, err = GetChangedFiles(dir)
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
	dir := createTestRepo(t)
	defer func() { _ = os.RemoveAll(dir) }()

	// Modify a file
	testFile := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(testFile, []byte("modified"), 0o644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Commit the changes
	commitMsg := "Test commit"
	err := CommitChanges(dir, commitMsg)
	if err != nil {
		t.Fatalf("CommitChanges() error = %v", err)
	}

	// Verify no uncommitted changes remain
	hasChanges, err := HasUncommittedChanges(dir)
	if err != nil {
		t.Fatalf("HasUncommittedChanges() error = %v", err)
	}
	if hasChanges {
		t.Error("CommitChanges() did not commit all changes")
	}

	// Verify commit message
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
	dir := createTestRepo(t)
	defer func() { _ = os.RemoveAll(dir) }()

	branch, ahead, behind, err := GetBranchInfo(dir)
	if err != nil {
		t.Fatalf("GetBranchInfo() error = %v", err)
	}

	// Should be on main or master branch
	if branch != "main" && branch != "master" {
		t.Errorf("GetBranchInfo() branch = %q, want main or master", branch)
	}

	// No upstream, so ahead/behind should be 0
	if ahead != 0 || behind != 0 {
		t.Errorf("GetBranchInfo() ahead=%d, behind=%d, want 0, 0", ahead, behind)
	}
}

func TestGetRecentCommits(t *testing.T) {
	dir := createTestRepo(t)
	defer func() { _ = os.RemoveAll(dir) }()

	// Add more commits
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

	commits, err := GetRecentCommits(dir, 2)
	if err != nil {
		t.Fatalf("GetRecentCommits() error = %v", err)
	}

	if len(commits) != 2 {
		t.Errorf("GetRecentCommits(2) returned %d commits, want 2", len(commits))
	}
}

func TestGenerateAutoCommitMessage(t *testing.T) {
	msg := GenerateAutoCommitMessage()

	// Should contain "Auto-sync"
	if len(msg) == 0 {
		t.Error("GenerateAutoCommitMessage() returned empty string")
	}

	// Should contain timestamp format
	if len(msg) < 20 {
		t.Errorf("GenerateAutoCommitMessage() = %q, seems too short", msg)
	}
}

func TestGetClaudeDir(t *testing.T) {
	// Save original home
	originalHome := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", originalHome) }()

	// Test with valid home
	_, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("Cannot get home directory: %v", err)
	}

	// Create a temporary .claude directory for testing
	tmpHome, err := os.MkdirTemp("", "claude-sync-home-*")
	if err != nil {
		t.Fatalf("Failed to create temp home: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpHome) }()

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

	// Test with non-existent .claude directory
	if err := os.RemoveAll(claudeDir); err != nil {
		t.Fatalf("Failed to remove .claude dir: %v", err)
	}
	_, err = GetClaudeDir()
	if err == nil {
		t.Error("GetClaudeDir() should error when .claude doesn't exist")
	}
}
