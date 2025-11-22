package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mfenderov/claude-sync/internal/git"
)

func TestE2ESync(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tmpDir := t.TempDir()
	fakeClaudeDir := filepath.Join(tmpDir, "fake-claude")
	bareRepoDir := filepath.Join(tmpDir, "remote.git")

	if err := os.Mkdir(fakeClaudeDir, 0o755); err != nil {
		t.Fatalf("Failed to create fake .claude dir: %v", err)
	}

	if err := os.Mkdir(bareRepoDir, 0o755); err != nil {
		t.Fatalf("Failed to create bare repo dir: %v", err)
	}

	cmd := exec.CommandContext(ctx, "git", "init", "--bare", bareRepoDir)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init bare repo: %v", err)
	}

	if err := git.InitRepo(ctx, fakeClaudeDir); err != nil {
		t.Fatalf("Failed to init local repo: %v", err)
	}

	cmd = exec.CommandContext(ctx, "git", "-C", fakeClaudeDir, "config", "user.email", "test@example.com")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to configure git email: %v", err)
	}

	cmd = exec.CommandContext(ctx, "git", "-C", fakeClaudeDir, "config", "user.name", "Test User")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to configure git name: %v", err)
	}

	testFile := filepath.Join(fakeClaudeDir, "config.json")
	if err := os.WriteFile(testFile, []byte(`{"test": true}`), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	if err := git.SetupGitignore(fakeClaudeDir); err != nil {
		t.Fatalf("Failed to setup gitignore: %v", err)
	}

	if err := git.InitialCommit(ctx, fakeClaudeDir, "Initial commit"); err != nil {
		t.Fatalf("Failed to create initial commit: %v", err)
	}

	if err := git.AddRemote(ctx, fakeClaudeDir, "origin", bareRepoDir); err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}

	cmd = exec.CommandContext(ctx, "git", "-C", fakeClaudeDir, "push", "-u", "origin", "main")
	if err := cmd.Run(); err != nil {
		cmd = exec.CommandContext(ctx, "git", "-C", fakeClaudeDir, "push", "-u", "origin", "master")
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("Failed to push to remote: %v\nOutput: %s", err, string(output))
		}
	}

	if err := os.WriteFile(testFile, []byte(`{"test": true, "updated": true}`), 0o644); err != nil {
		t.Fatalf("Failed to update test file: %v", err)
	}

	hasChanges, err := git.HasUncommittedChanges(ctx, fakeClaudeDir)
	if err != nil {
		t.Fatalf("Failed to check uncommitted changes: %v", err)
	}
	if !hasChanges {
		t.Error("Expected uncommitted changes after modifying file")
	}

	if err := git.CommitChanges(ctx, fakeClaudeDir, "Update config"); err != nil {
		t.Fatalf("Failed to commit changes: %v", err)
	}

	hasChanges, err = git.HasUncommittedChanges(ctx, fakeClaudeDir)
	if err != nil {
		t.Fatalf("Failed to check uncommitted changes: %v", err)
	}
	if hasChanges {
		t.Error("Expected no uncommitted changes after commit")
	}

	if err := git.Push(ctx, fakeClaudeDir); err != nil {
		t.Fatalf("Failed to push changes: %v", err)
	}

	branch, ahead, behind, err := git.GetBranchInfo(ctx, fakeClaudeDir)
	if err != nil {
		t.Fatalf("Failed to get branch info: %v", err)
	}

	if branch != "main" && branch != "master" {
		t.Errorf("Expected branch main or master, got %s", branch)
	}

	if ahead != 0 {
		t.Errorf("Expected 0 commits ahead after push, got %d", ahead)
	}

	commits, err := git.GetRecentCommits(ctx, fakeClaudeDir, 5)
	if err != nil {
		t.Fatalf("Failed to get recent commits: %v", err)
	}

	if len(commits) < 2 {
		t.Errorf("Expected at least 2 commits, got %d", len(commits))
	}

	foundUpdateCommit := false
	for _, commit := range commits {
		if strings.Contains(commit, "Update config") {
			foundUpdateCommit = true
			break
		}
	}
	if !foundUpdateCommit {
		t.Error("Expected to find 'Update config' commit in recent commits")
	}

	changedFiles, err := git.GetChangedFiles(ctx, fakeClaudeDir)
	if err != nil {
		t.Fatalf("Failed to get changed files: %v", err)
	}
	if len(changedFiles) != 0 {
		t.Errorf("Expected no changed files, got %v", changedFiles)
	}

	t.Logf("E2E test completed successfully!")
	t.Logf("  Branch: %s", branch)
	t.Logf("  Ahead: %d, Behind: %d", ahead, behind)
	t.Logf("  Recent commits: %d", len(commits))
}
