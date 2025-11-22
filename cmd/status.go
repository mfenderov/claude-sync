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

	// Get branch info
	branch, ahead, behind, err := git.GetBranchInfo(claudeDir)
	if err != nil {
		fmt.Println(ui.RenderError("✗", "Failed to get branch info"))
		return err
	}

	// Get remote URL
	remoteURL := getRemoteURL(claudeDir)

	// Build repository section
	var repoInfo strings.Builder
	repoInfo.WriteString(ui.InfoStyle.Render("Repository: ") + remoteURL + "\n")

	branchInfo := fmt.Sprintf("Branch:     %s", branch)
	if ahead > 0 || behind > 0 {
		branchInfo += fmt.Sprintf(" ↑%d ↓%d", ahead, behind)
		if ahead > 0 {
			branchInfo += ui.WarningStyle.Render(fmt.Sprintf(" (%d ahead", ahead))
			if behind > 0 {
				branchInfo += ", "
			}
		}
		if behind > 0 {
			if ahead == 0 {
				branchInfo += ui.WarningStyle.Render(fmt.Sprintf(" (%d behind", behind))
			} else {
				branchInfo += ui.WarningStyle.Render(fmt.Sprintf("%d behind", behind))
			}
		}
		branchInfo += ui.WarningStyle.Render(")")
	}
	repoInfo.WriteString(ui.InfoStyle.Render(branchInfo))

	fmt.Println(ui.BoxStyle.Render(repoInfo.String()))
	log.Info("Repository status", "branch", branch, "ahead", ahead, "behind", behind, "remote", remoteURL)

	// Check for uncommitted changes
	hasChanges, _ := git.HasUncommittedChanges(claudeDir)
	if hasChanges {
		changedFiles, _ := git.GetChangedFiles(claudeDir)
		var changeInfo strings.Builder
		changeInfo.WriteString(ui.WarningStyle.Render(fmt.Sprintf("📝 Modified Files (%d)", len(changedFiles))) + "\n\n")
		for _, file := range changedFiles {
			changeInfo.WriteString(ui.ListItemStyle.Render("• " + file) + "\n")
		}
		fmt.Println(ui.BoxStyle.Render(changeInfo.String()))
		log.Warn("Uncommitted changes detected", "count", len(changedFiles), "files", changedFiles)
	}

	// Show plugins
	plugins := getEnabledPlugins(claudeDir)
	if len(plugins) > 0 {
		var pluginInfo strings.Builder
		pluginInfo.WriteString(ui.SuccessStyle.Render(fmt.Sprintf("📦 Plugins (%d)", len(plugins))) + "\n\n")
		for _, plugin := range plugins {
			pluginInfo.WriteString(ui.ListItemStyle.Render(ui.SuccessStyle.Render("✓")+" "+plugin) + "\n")
		}
		fmt.Println(ui.BoxStyle.Render(pluginInfo.String()))
		log.Info("Enabled plugins", "count", len(plugins), "plugins", plugins)
	}

	// Show hooks
	hooks := getHooks(claudeDir)
	if len(hooks) > 0 {
		var hookInfo strings.Builder
		hookInfo.WriteString(ui.InfoStyle.Render(fmt.Sprintf("🪝 Hooks (%d)", len(hooks))) + "\n\n")
		for _, hook := range hooks {
			hookInfo.WriteString(ui.ListItemStyle.Render(ui.SuccessStyle.Render("✓")+" "+hook) + "\n")
		}
		fmt.Println(ui.BoxStyle.Render(hookInfo.String()))
		log.Info("Installed hooks", "count", len(hooks), "hooks", hooks)
	}

	// Show skills
	skills := getSkills(claudeDir)
	if len(skills) > 0 {
		var skillInfo strings.Builder
		skillInfo.WriteString(ui.InfoStyle.Render(fmt.Sprintf("🎯 Skills (%d)", len(skills))) + "\n\n")
		for _, skill := range skills {
			skillInfo.WriteString(ui.ListItemStyle.Render(ui.SuccessStyle.Render("✓")+" "+skill) + "\n")
		}
		fmt.Println(ui.BoxStyle.Render(skillInfo.String()))
		log.Info("Loaded skills", "count", len(skills), "skills", skills)
	}

	log.Newline()
	return nil
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
