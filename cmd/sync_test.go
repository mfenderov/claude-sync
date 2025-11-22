package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func createTestClaudeRepo(t *testing.T) string {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "claude-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Configure git
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	cmd.Run()

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	cmd.Run()

	// Create test files
	os.WriteFile(filepath.Join(tmpDir, "settings.json"), []byte(`{"enabledPlugins":{}}`), 0644)
	os.MkdirAll(filepath.Join(tmpDir, "hooks"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "skills"), 0755)

	// Initial commit
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = tmpDir
	cmd.Run()

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tmpDir
	cmd.Run()

	return tmpDir
}

func TestSyncCommand(t *testing.T) {
	// This test verifies the sync command can be created and has correct properties
	if syncCmd == nil {
		t.Fatal("syncCmd is nil")
	}

	if syncCmd.Use != "sync" {
		t.Errorf("syncCmd.Use = %q, want %q", syncCmd.Use, "sync")
	}

	if syncCmd.RunE == nil {
		t.Error("syncCmd.RunE is nil, should be set")
	}
}

func TestStatusCommand(t *testing.T) {
	// This test verifies the status command can be created and has correct properties
	if statusCmd == nil {
		t.Fatal("statusCmd is nil")
	}

	if statusCmd.Use != "status" {
		t.Errorf("statusCmd.Use = %q, want %q", statusCmd.Use, "status")
	}

	if statusCmd.RunE == nil {
		t.Error("statusCmd.RunE is nil, should be set")
	}
}
