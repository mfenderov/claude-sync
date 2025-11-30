package sync_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mfenderov/claude-sync/internal/sync"
)

// testPrompter simulates user input for E2E tests
type testPrompter struct {
	confirmResponses []bool
	inputResponses   []string
	selectResponses  []string
	confirmIndex     int
	inputIndex       int
	selectIndex      int
}

func (p *testPrompter) Confirm(prompt string) (bool, error) {
	if p.confirmIndex >= len(p.confirmResponses) {
		return false, nil
	}
	resp := p.confirmResponses[p.confirmIndex]
	p.confirmIndex++
	return resp, nil
}

func (p *testPrompter) Input(prompt, placeholder string) (string, error) {
	if p.inputIndex >= len(p.inputResponses) {
		return "", nil
	}
	resp := p.inputResponses[p.inputIndex]
	p.inputIndex++
	return resp, nil
}

func (p *testPrompter) Select(prompt string, options []sync.SelectOption) (string, error) {
	if p.selectIndex >= len(p.selectResponses) {
		return "", nil
	}
	resp := p.selectResponses[p.selectIndex]
	p.selectIndex++
	return resp, nil
}

func (p *testPrompter) SpinWhile(message string, task func() error) error {
	return task()
}

// testLogger captures log output for verification
type testLogger struct {
	messages []string
}

func (l *testLogger) Title(title string) { l.messages = append(l.messages, "TITLE: "+title) }
func (l *testLogger) Success(icon, message string) {
	l.messages = append(l.messages, "SUCCESS: "+message)
}

func (l *testLogger) Error(icon, message string, _ error) {
	l.messages = append(l.messages, "ERROR: "+message)
}

func (l *testLogger) Warning(icon, message string) {
	l.messages = append(l.messages, "WARNING: "+message)
}
func (l *testLogger) Info(icon, message string) { l.messages = append(l.messages, "INFO: "+message) }
func (l *testLogger) Muted(message string)      { l.messages = append(l.messages, "MUTED: "+message) }
func (l *testLogger) ListItem(message string)   { l.messages = append(l.messages, "LIST: "+message) }
func (l *testLogger) Box(title, content string) { l.messages = append(l.messages, "BOX: "+title) }
func (l *testLogger) Newline()                  {}

func (l *testLogger) hasMessage(substr string) bool {
	for _, msg := range l.messages {
		if strings.Contains(msg, substr) {
			return true
		}
	}
	return false
}

// testGitAdapter wraps the real git operations but uses a custom claude dir
type testGitAdapter struct {
	claudeDir string
}

func (g *testGitAdapter) ClaudeDirExists() (bool, error) {
	_, err := os.Stat(g.claudeDir)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

func (g *testGitAdapter) ClaudeDirPath() (string, error) {
	return g.claudeDir, nil
}

func (g *testGitAdapter) GetClaudeDir() (string, error) {
	if _, err := os.Stat(g.claudeDir); os.IsNotExist(err) {
		return "", err
	}
	return g.claudeDir, nil
}

func (g *testGitAdapter) CreateClaudeDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func (g *testGitAdapter) RemoveClaudeDir() error {
	return os.RemoveAll(g.claudeDir)
}

func (g *testGitAdapter) IsGitRepo(path string) bool {
	gitDir := filepath.Join(path, ".git")
	info, err := os.Stat(gitDir)
	return err == nil && info.IsDir()
}

func (g *testGitAdapter) InitRepo(ctx context.Context, path string) error {
	cmd := exec.CommandContext(ctx, "git", "init", path)
	return cmd.Run()
}

func (g *testGitAdapter) CloneRepo(ctx context.Context, remoteURL, destPath string) error {
	cmd := exec.CommandContext(ctx, "git", "clone", remoteURL, destPath)
	return cmd.Run()
}

func (g *testGitAdapter) SetupGitignore(path string) error {
	content := "credentials.json\n*.key\n.env\n"
	return os.WriteFile(filepath.Join(path, ".gitignore"), []byte(content), 0o644)
}

func (g *testGitAdapter) InitialCommit(ctx context.Context, path, message string) error {
	if err := runGit(ctx, path, "add", "-A"); err != nil {
		return err
	}
	return runGit(ctx, path, "commit", "-m", message)
}

func (g *testGitAdapter) ValidateRemote(ctx context.Context, remoteURL string) error {
	cmd := exec.CommandContext(ctx, "git", "ls-remote", remoteURL)
	return cmd.Run()
}

func (g *testGitAdapter) RemoteHasCommits(ctx context.Context, remoteURL string) (bool, error) {
	cmd := exec.CommandContext(ctx, "git", "ls-remote", "--heads", remoteURL)
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(output)) != "", nil
}

func (g *testGitAdapter) AddRemote(ctx context.Context, path, name, url string) error {
	return runGit(ctx, path, "remote", "add", name, url)
}

func (g *testGitAdapter) Fetch(ctx context.Context, path string) error {
	return runGit(ctx, path, "fetch", "origin")
}

func (g *testGitAdapter) GetChangedFiles(ctx context.Context, path string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", path, "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	if len(strings.TrimSpace(string(output))) == 0 {
		return nil, nil
	}
	var files []string
	for _, line := range strings.Split(string(output), "\n") {
		if len(line) > 3 {
			files = append(files, strings.TrimSpace(line[3:]))
		}
	}
	return files, nil
}

func (g *testGitAdapter) CommitChanges(ctx context.Context, path, message string) error {
	if err := runGit(ctx, path, "add", "-A"); err != nil {
		return err
	}
	return runGit(ctx, path, "commit", "-m", message)
}

func (g *testGitAdapter) PullWithRebase(ctx context.Context, path string) error {
	return runGit(ctx, path, "pull", "--rebase")
}

func (g *testGitAdapter) PullAllowUnrelatedHistories(ctx context.Context, path string) error {
	return runGit(ctx, path, "pull", "--no-rebase", "origin", "main", "--allow-unrelated-histories")
}

func (g *testGitAdapter) Push(ctx context.Context, path string) error {
	return runGit(ctx, path, "push")
}

func (g *testGitAdapter) PushWithUpstream(ctx context.Context, path string) error {
	return runGit(ctx, path, "push", "-u", "origin", "main")
}

func (g *testGitAdapter) GetBranchInfo(ctx context.Context, path string) (string, int, int, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", path, "branch", "--show-current")
	output, err := cmd.Output()
	if err != nil {
		return "", 0, 0, err
	}
	return strings.TrimSpace(string(output)), 0, 0, nil
}

func (g *testGitAdapter) GetRecentCommits(ctx context.Context, path string, count int) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", path, "log", "--oneline", "-n", "5")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return strings.Split(strings.TrimSpace(string(output)), "\n"), nil
}

func (g *testGitAdapter) HasConflicts(ctx context.Context, path string) (bool, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", path, "diff", "--name-only", "--diff-filter=U")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(output)) != "", nil
}

func (g *testGitAdapter) AbortRebase(ctx context.Context, path string) error {
	return runGit(ctx, path, "rebase", "--abort")
}

func (g *testGitAdapter) GenerateAutoCommitMessage() string {
	return "Auto-sync: " + time.Now().Format("2006-01-02 15:04")
}

func runGit(ctx context.Context, path string, args ...string) error {
	fullArgs := append([]string{"-C", path}, args...)
	cmd := exec.CommandContext(ctx, "git", fullArgs...)
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test User",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test User",
		"GIT_COMMITTER_EMAIL=test@example.com",
		"GIT_CONFIG_COUNT=1",
		"GIT_CONFIG_KEY_0=init.defaultBranch",
		"GIT_CONFIG_VALUE_0=main",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &gitError{cmd: "git " + strings.Join(args, " "), output: string(output), err: err}
	}
	return nil
}

type gitError struct {
	err    error
	cmd    string
	output string
}

func (e *gitError) Error() string {
	return e.cmd + ": " + e.err.Error() + "\nOutput: " + e.output
}

// createBareRepo creates a bare git repository for testing
func createBareRepo(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("Failed to create bare repo dir: %v", err)
	}
	cmd := exec.Command("git", "init", "--bare", "--initial-branch=main", path)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init bare repo: %v", err)
	}
}

// createBareRepoWithCommits creates a bare repo with existing commits
func createBareRepoWithCommits(t *testing.T, bareRepoPath string) {
	t.Helper()
	ctx := context.Background()
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "work")

	// Create bare repo
	createBareRepo(t, bareRepoPath)

	// Create a working repo, add content, push to bare
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatalf("Failed to create work dir: %v", err)
	}

	if err := runGit(ctx, ".", "clone", bareRepoPath, workDir); err != nil {
		// Bare repo is empty, init instead
		if err := runGit(ctx, ".", "init", workDir); err != nil {
			t.Fatalf("Failed to init work repo: %v", err)
		}
		if err := runGit(ctx, workDir, "remote", "add", "origin", bareRepoPath); err != nil {
			t.Fatalf("Failed to add remote: %v", err)
		}
	}

	// Create a file and commit
	testFile := filepath.Join(workDir, "remote-config.json")
	if err := os.WriteFile(testFile, []byte(`{"from": "remote"}`), 0o644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	if err := runGit(ctx, workDir, "add", "-A"); err != nil {
		t.Fatalf("Failed to git add: %v", err)
	}

	if err := runGit(ctx, workDir, "commit", "-m", "Remote initial commit"); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	if err := runGit(ctx, workDir, "push", "-u", "origin", "main"); err != nil {
		// Try master if main fails
		if err := runGit(ctx, workDir, "push", "-u", "origin", "master"); err != nil {
			t.Fatalf("Failed to push: %v", err)
		}
	}
}

// TestE2E_NormalSync tests the happy path sync when everything is set up
func TestE2E_NormalSync(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	bareRepoDir := filepath.Join(tmpDir, "remote.git")

	// Setup: Create bare repo and initialized local repo
	createBareRepo(t, bareRepoDir)

	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatalf("Failed to create claude dir: %v", err)
	}

	gitAdapter := &testGitAdapter{claudeDir: claudeDir}

	if err := gitAdapter.InitRepo(ctx, claudeDir); err != nil {
		t.Fatalf("Failed to init repo: %v", err)
	}

	if err := gitAdapter.SetupGitignore(claudeDir); err != nil {
		t.Fatalf("Failed to setup gitignore: %v", err)
	}

	testFile := filepath.Join(claudeDir, "config.json")
	if err := os.WriteFile(testFile, []byte(`{"test": true}`), 0o644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	if err := gitAdapter.InitialCommit(ctx, claudeDir, "Initial commit"); err != nil {
		t.Fatalf("Failed to create initial commit: %v", err)
	}

	if err := gitAdapter.AddRemote(ctx, claudeDir, "origin", bareRepoDir); err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}

	if err := gitAdapter.PushWithUpstream(ctx, claudeDir); err != nil {
		t.Fatalf("Failed to push: %v", err)
	}

	// Make a change
	if err := os.WriteFile(testFile, []byte(`{"test": true, "updated": true}`), 0o644); err != nil {
		t.Fatalf("Failed to update test file: %v", err)
	}

	// Run sync service
	logger := &testLogger{}
	prompter := &testPrompter{}
	service := sync.NewService(gitAdapter, prompter, logger)

	if err := service.Run(ctx); err != nil {
		t.Fatalf("Service.Run failed: %v", err)
	}

	// Verify
	if !logger.hasMessage("Sync complete") {
		t.Error("Expected 'Sync complete' message")
	}

	// Verify file was committed and pushed
	changedFiles, err := gitAdapter.GetChangedFiles(ctx, claudeDir)
	if err != nil {
		t.Fatalf("Failed to get changed files: %v", err)
	}
	if len(changedFiles) != 0 {
		t.Errorf("Expected no changed files after sync, got %v", changedFiles)
	}
}

// TestE2E_FirstTimeSetup_Clone tests cloning an existing repo when ~/.claude doesn't exist
func TestE2E_FirstTimeSetup_Clone(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	bareRepoDir := filepath.Join(tmpDir, "remote.git")

	// Setup: Create bare repo WITH commits (simulate existing config repo)
	createBareRepoWithCommits(t, bareRepoDir)

	// claudeDir does NOT exist yet - this is the "first time" scenario

	gitAdapter := &testGitAdapter{claudeDir: claudeDir}
	logger := &testLogger{}
	prompter := &testPrompter{
		selectResponses: []string{"clone"},     // Choose "clone existing config"
		inputResponses:  []string{bareRepoDir}, // Enter remote URL
	}

	service := sync.NewService(gitAdapter, prompter, logger)

	if err := service.Run(ctx); err != nil {
		t.Fatalf("Service.Run failed: %v", err)
	}

	// Verify: claudeDir should now exist and be a git repo
	if !gitAdapter.IsGitRepo(claudeDir) {
		t.Error("Expected claudeDir to be a git repo after clone")
	}

	// Verify: Should have the remote file
	remoteFile := filepath.Join(claudeDir, "remote-config.json")
	if _, err := os.Stat(remoteFile); os.IsNotExist(err) {
		t.Error("Expected remote-config.json to exist after clone")
	}

	if !logger.hasMessage("Setup complete") {
		t.Error("Expected 'Setup complete' message")
	}
}

// TestE2E_FirstTimeSetup_Fresh tests fresh start with empty remote
func TestE2E_FirstTimeSetup_Fresh(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	bareRepoDir := filepath.Join(tmpDir, "remote.git")

	// Setup: Create EMPTY bare repo
	createBareRepo(t, bareRepoDir)

	// claudeDir does NOT exist yet

	gitAdapter := &testGitAdapter{claudeDir: claudeDir}
	logger := &testLogger{}
	prompter := &testPrompter{
		selectResponses:  []string{"fresh"},     // Choose "start fresh"
		confirmResponses: []bool{true},          // Confirm setup
		inputResponses:   []string{bareRepoDir}, // Enter remote URL
	}

	service := sync.NewService(gitAdapter, prompter, logger)

	if err := service.Run(ctx); err != nil {
		t.Fatalf("Service.Run failed: %v", err)
	}

	// Verify: claudeDir should exist and be a git repo
	if !gitAdapter.IsGitRepo(claudeDir) {
		t.Error("Expected claudeDir to be a git repo after fresh setup")
	}

	// Verify: Should have .gitignore
	gitignoreFile := filepath.Join(claudeDir, ".gitignore")
	if _, err := os.Stat(gitignoreFile); os.IsNotExist(err) {
		t.Error("Expected .gitignore to exist after fresh setup")
	}

	// Verify: Remote should have commits now
	hasCommits, err := gitAdapter.RemoteHasCommits(ctx, bareRepoDir)
	if err != nil {
		t.Fatalf("Failed to check remote: %v", err)
	}
	if !hasCommits {
		t.Error("Expected remote to have commits after push")
	}

	if !logger.hasMessage("Setup complete") {
		t.Error("Expected 'Setup complete' message")
	}
}

// TestE2E_InitFlow_RemoteHasCommits_Replace tests the critical bug fix:
// When local ~/.claude exists (not git repo) and remote already has commits,
// user chooses to replace local with remote
func TestE2E_InitFlow_RemoteHasCommits_Replace(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	bareRepoDir := filepath.Join(tmpDir, "remote.git")

	// Setup: Create bare repo WITH commits
	createBareRepoWithCommits(t, bareRepoDir)

	// Setup: Create local claudeDir with LOCAL content (NOT a git repo)
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatalf("Failed to create claude dir: %v", err)
	}
	localFile := filepath.Join(claudeDir, "local-config.json")
	if err := os.WriteFile(localFile, []byte(`{"from": "local"}`), 0o644); err != nil {
		t.Fatalf("Failed to write local file: %v", err)
	}

	gitAdapter := &testGitAdapter{claudeDir: claudeDir}
	logger := &testLogger{}
	prompter := &testPrompter{
		confirmResponses: []bool{true},          // Confirm setup
		inputResponses:   []string{bareRepoDir}, // Enter remote URL
		selectResponses:  []string{"replace"},   // Choose "replace local with remote"
	}

	service := sync.NewService(gitAdapter, prompter, logger)

	if err := service.Run(ctx); err != nil {
		t.Fatalf("Service.Run failed: %v", err)
	}

	// Verify: claudeDir should be a git repo
	if !gitAdapter.IsGitRepo(claudeDir) {
		t.Error("Expected claudeDir to be a git repo after replace")
	}

	// Verify: Should have REMOTE content, NOT local
	remoteFile := filepath.Join(claudeDir, "remote-config.json")
	if _, err := os.Stat(remoteFile); os.IsNotExist(err) {
		t.Error("Expected remote-config.json to exist after replace")
	}

	// Verify: Local file should be GONE (replaced)
	if _, err := os.Stat(localFile); !os.IsNotExist(err) {
		t.Error("Expected local-config.json to be GONE after replace")
	}

	if !logger.hasMessage("Setup complete") {
		t.Error("Expected 'Setup complete' message")
	}
}

// TestE2E_InitFlow_RemoteHasCommits_Merge tests the critical bug fix:
// When local ~/.claude exists (not git repo) and remote already has commits,
// user chooses to merge histories (keep both)
func TestE2E_InitFlow_RemoteHasCommits_Merge(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	bareRepoDir := filepath.Join(tmpDir, "remote.git")

	// Setup: Create bare repo WITH commits
	createBareRepoWithCommits(t, bareRepoDir)

	// Setup: Create local claudeDir with LOCAL content (NOT a git repo)
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatalf("Failed to create claude dir: %v", err)
	}
	localFile := filepath.Join(claudeDir, "local-config.json")
	if err := os.WriteFile(localFile, []byte(`{"from": "local"}`), 0o644); err != nil {
		t.Fatalf("Failed to write local file: %v", err)
	}

	gitAdapter := &testGitAdapter{claudeDir: claudeDir}
	logger := &testLogger{}
	prompter := &testPrompter{
		confirmResponses: []bool{true},          // Confirm setup
		inputResponses:   []string{bareRepoDir}, // Enter remote URL
		selectResponses:  []string{"merge"},     // Choose "merge histories"
	}

	service := sync.NewService(gitAdapter, prompter, logger)

	if err := service.Run(ctx); err != nil {
		t.Fatalf("Service.Run failed: %v", err)
	}

	// Verify: claudeDir should be a git repo
	if !gitAdapter.IsGitRepo(claudeDir) {
		t.Error("Expected claudeDir to be a git repo after merge")
	}

	// Verify: Should have BOTH remote AND local content
	remoteFile := filepath.Join(claudeDir, "remote-config.json")
	if _, err := os.Stat(remoteFile); os.IsNotExist(err) {
		t.Error("Expected remote-config.json to exist after merge")
	}

	if _, err := os.Stat(localFile); os.IsNotExist(err) {
		t.Error("Expected local-config.json to STILL exist after merge")
	}

	// Verify logs
	if !logger.hasMessage("Histories merged") {
		t.Error("Expected 'Histories merged' message")
	}

	if !logger.hasMessage("Setup complete") {
		t.Error("Expected 'Setup complete' message")
	}

	// Verify: Check git log has merge commit
	commits, err := gitAdapter.GetRecentCommits(ctx, claudeDir, 5)
	if err != nil {
		t.Fatalf("Failed to get commits: %v", err)
	}
	if len(commits) < 2 {
		t.Errorf("Expected at least 2 commits after merge, got %d", len(commits))
	}
}

// TestE2E_InitFlow_UserCancels tests that cancellation works correctly
func TestE2E_InitFlow_UserCancels(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")

	// Setup: Create local claudeDir (NOT a git repo)
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatalf("Failed to create claude dir: %v", err)
	}
	localFile := filepath.Join(claudeDir, "local-config.json")
	if err := os.WriteFile(localFile, []byte(`{"from": "local"}`), 0o644); err != nil {
		t.Fatalf("Failed to write local file: %v", err)
	}

	gitAdapter := &testGitAdapter{claudeDir: claudeDir}
	logger := &testLogger{}
	prompter := &testPrompter{
		confirmResponses: []bool{false}, // User declines setup
	}

	service := sync.NewService(gitAdapter, prompter, logger)

	// Should NOT error - just cancel gracefully
	if err := service.Run(ctx); err != nil {
		t.Fatalf("Service.Run should not error on cancel: %v", err)
	}

	// Verify: claudeDir should NOT be a git repo
	if gitAdapter.IsGitRepo(claudeDir) {
		t.Error("Expected claudeDir to NOT be a git repo after cancel")
	}

	// Verify: Local file should still exist
	if _, err := os.Stat(localFile); os.IsNotExist(err) {
		t.Error("Expected local-config.json to still exist after cancel")
	}

	if !logger.hasMessage("cancelled") {
		t.Error("Expected 'cancelled' message")
	}
}
