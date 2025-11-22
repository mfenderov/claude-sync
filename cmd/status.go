package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mfenderov/claude-sync/internal/git"
	"github.com/mfenderov/claude-sync/internal/logger"
	"github.com/mfenderov/claude-sync/internal/ui"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show configuration status",
	Long:  `Display the current status of your Claude Code configuration including git status, plugins, and hooks.`,
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	log := logger.Default()
	log.Title("📊 Configuration Status")

	// Get Claude directory
	claudeDir, err := git.GetClaudeDir()
	if err != nil {
		log.Error("✗", err.Error(), err, "directory", "~/.claude")
		return err
	}

	// Check if it's a git repo
	if !git.IsGitRepo(claudeDir) {
		msg := "~/.claude is not a git repository"
		log.Error("✗", msg, fmt.Errorf("not a git repo"), "directory", claudeDir)
		return fmt.Errorf("%s", msg)
	}

	// Get and display branch info
	branch, ahead, behind, err := git.GetBranchInfo(claudeDir)
	if err != nil {
		fmt.Println(ui.RenderError("✗", "Failed to get branch info"))
		return err
	}
	displayRepositoryInfo(claudeDir, branch, ahead, behind, log)

	// Display modified files if any
	displayModifiedFiles(claudeDir, log)

	// Display plugins, hooks, and skills
	displayPlugins(claudeDir, log)
	displayHooks(claudeDir, log)
	displaySkills(claudeDir, log)

	log.Newline()
	return nil
}

func displayRepositoryInfo(claudeDir, branch string, ahead, behind int, log *logger.Logger) {
	remoteURL := getRemoteURL(claudeDir)

	var repoInfo strings.Builder
	repoInfo.WriteString(ui.InfoStyle.Render("Repository: "))
	repoInfo.WriteString(remoteURL)
	repoInfo.WriteString("\n")

	branchInfo := formatBranchInfo(branch, ahead, behind)
	repoInfo.WriteString(ui.InfoStyle.Render(branchInfo))

	fmt.Println(ui.BoxStyle.Render(repoInfo.String()))
	log.Info("Repository status", "branch", branch, "ahead", ahead, "behind", behind, "remote", remoteURL)
}

func formatBranchInfo(branch string, ahead, behind int) string {
	branchInfo := fmt.Sprintf("Branch:     %s", branch)
	if ahead == 0 && behind == 0 {
		return branchInfo
	}

	branchInfo += fmt.Sprintf(" ↑%d ↓%d", ahead, behind)
	var status []string
	if ahead > 0 {
		status = append(status, fmt.Sprintf("%d ahead", ahead))
	}
	if behind > 0 {
		status = append(status, fmt.Sprintf("%d behind", behind))
	}
	branchInfo += ui.WarningStyle.Render(fmt.Sprintf(" (%s)", strings.Join(status, ", ")))
	return branchInfo
}

func displayModifiedFiles(claudeDir string, log *logger.Logger) {
	hasChanges, err := git.HasUncommittedChanges(claudeDir)
	if err != nil {
		log.Warning("⚠️", "Could not check for uncommitted changes", "error", err)
		return
	}

	if !hasChanges {
		return
	}

	changedFiles, err := git.GetChangedFiles(claudeDir)
	if err != nil {
		log.Warning("⚠️", "Could not get changed files", "error", err)
		return
	}

	var changeInfo strings.Builder
	changeInfo.WriteString(ui.WarningStyle.Render(fmt.Sprintf("📝 Modified Files (%d)", len(changedFiles))))
	changeInfo.WriteString("\n\n")
	for _, file := range changedFiles {
		changeInfo.WriteString(ui.ListItemStyle.Render("• " + file))
		changeInfo.WriteString("\n")
	}
	fmt.Println(ui.BoxStyle.Render(changeInfo.String()))
	log.Warn("Uncommitted changes detected", "count", len(changedFiles), "files", changedFiles)
}

func displayPlugins(claudeDir string, log *logger.Logger) {
	plugins := getEnabledPlugins(claudeDir)
	if len(plugins) == 0 {
		return
	}

	var pluginInfo strings.Builder
	pluginInfo.WriteString(ui.SuccessStyle.Render(fmt.Sprintf("📦 Plugins (%d)", len(plugins))))
	pluginInfo.WriteString("\n\n")
	for _, plugin := range plugins {
		pluginInfo.WriteString(ui.ListItemStyle.Render(ui.SuccessStyle.Render("✓") + " " + plugin))
		pluginInfo.WriteString("\n")
	}
	fmt.Println(ui.BoxStyle.Render(pluginInfo.String()))
	log.Info("Enabled plugins", "count", len(plugins), "plugins", plugins)
}

func displayHooks(claudeDir string, log *logger.Logger) {
	hooks := getHooks(claudeDir)
	if len(hooks) == 0 {
		return
	}

	var hookInfo strings.Builder
	hookInfo.WriteString(ui.InfoStyle.Render(fmt.Sprintf("🪝 Hooks (%d)", len(hooks))))
	hookInfo.WriteString("\n\n")
	for _, hook := range hooks {
		hookInfo.WriteString(ui.ListItemStyle.Render(ui.SuccessStyle.Render("✓") + " " + hook))
		hookInfo.WriteString("\n")
	}
	fmt.Println(ui.BoxStyle.Render(hookInfo.String()))
	log.Info("Installed hooks", "count", len(hooks), "hooks", hooks)
}

func displaySkills(claudeDir string, log *logger.Logger) {
	skills := getSkills(claudeDir)
	if len(skills) == 0 {
		return
	}

	var skillInfo strings.Builder
	skillInfo.WriteString(ui.InfoStyle.Render(fmt.Sprintf("🎯 Skills (%d)", len(skills))))
	skillInfo.WriteString("\n\n")
	for _, skill := range skills {
		skillInfo.WriteString(ui.ListItemStyle.Render(ui.SuccessStyle.Render("✓") + " " + skill))
		skillInfo.WriteString("\n")
	}
	fmt.Println(ui.BoxStyle.Render(skillInfo.String()))
	log.Info("Loaded skills", "count", len(skills), "skills", skills)
}

func getRemoteURL(repoPath string) string {
	cmd := "git"
	args := []string{"-C", repoPath, "config", "--get", "remote.origin.url"}
	output, err := executeCommand(cmd, args...)
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(output)
}

func getEnabledPlugins(claudeDir string) []string {
	settingsPath := filepath.Join(claudeDir, "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return []string{}
	}

	var settings struct {
		EnabledPlugins map[string]bool `json:"enabledPlugins"`
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		return []string{}
	}

	var plugins []string
	for plugin, enabled := range settings.EnabledPlugins {
		if enabled {
			plugins = append(plugins, plugin)
		}
	}
	return plugins
}

func getHooks(claudeDir string) []string {
	hooksDir := filepath.Join(claudeDir, "hooks")
	entries, err := os.ReadDir(hooksDir)
	if err != nil {
		return []string{}
	}

	var hooks []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sh") {
			hooks = append(hooks, entry.Name())
		}
	}
	return hooks
}

func getSkills(claudeDir string) []string {
	skillsDir := filepath.Join(claudeDir, "skills")
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return []string{}
	}

	var skills []string
	for _, entry := range entries {
		if entry.IsDir() {
			skills = append(skills, entry.Name())
		}
	}
	return skills
}

func executeCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}
