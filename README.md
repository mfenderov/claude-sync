# 🎭 claude-sync

A beautiful CLI tool for syncing Claude Code configurations across machines.

Built with ❤️ using Go and [Charm](https://charm.sh/) libraries.

## ✨ Features

- **Automatic Sync**: One command to commit, pull (with rebase), and push
- **Beautiful TUI**: Gorgeous terminal UI powered by Lipgloss
- **Smart Commits**: Auto-generated commit messages with timestamps
- **Status Dashboard**: View your config status with plugins, hooks, and skills
- **Git Integration**: Built-in git operations for seamless syncing

## 🚀 Installation

### From Source

```bash
go install github.com/mfenderov/claude-sync@latest
```

### Build Locally

```bash
git clone https://github.com/mfenderov/claude-sync.git
cd claude-sync
go build -o claude-sync
go install
```

## 📖 Usage

### Sync Configuration

The main command automatically syncs your Claude Code config:

```bash
claude-sync
# or
claude-sync sync
```

This will:
1. ✅ Detect and commit local changes
2. ✅ Pull from remote (with rebase)
3. ✅ Push to remote
4. ✅ Show recent activity

### View Status

See your current configuration status:

```bash
claude-sync status
```

Displays:
- Repository and branch info
- Modified files
- Enabled plugins
- Installed hooks
- Available skills

## 🎨 What It Looks Like

```
🎭 Claude Config Sync

⏳ Checking for local changes...
✓ Found 2 changed file(s)
  → settings.json
  → hooks/status-line.sh

⏳ Committing changes...
✓ Changes committed
  Auto-sync from your-machine at 2025-11-22 10:43:09

⏳ Pulling from remote (with rebase)...
✓ Pulled latest changes

⏳ Pushing to remote...
✓ Pushed to remote

╭──────────────────────────────────────────────────╮
│  Recent Activity                                 │
├──────────────────────────────────────────────────┤
│  abc123  Update AWS region      2 minutes ago    │
│  def456  Add new hook           1 hour ago       │
╰──────────────────────────────────────────────────╯

✨ Sync complete!
```

## 🏗 Architecture

### Project Structure

```
claude-sync/
├── cmd/              # Cobra commands
│   ├── root.go       # Root command
│   ├── sync.go       # Sync command
│   └── status.go     # Status command
├── internal/
│   ├── git/          # Git operations
│   ├── claude/       # Claude config parsing
│   └── ui/           # Lipgloss styling
└── main.go           # Entry point
```

### Tech Stack

- **[Cobra](https://github.com/spf13/cobra)** - CLI framework
- **[Lipgloss](https://github.com/charmbracelet/lipgloss)** - Terminal styling
- **[Bubbles](https://github.com/charmbracelet/bubbles)** - TUI components
- **[Charm Log](https://github.com/charmbracelet/log)** - Beautiful logging

## 🔧 Requirements

- Go 1.21+
- Git
- Claude Code with config in `~/.claude`

## 📋 Prerequisites

Your `~/.claude` directory must be a git repository:

```bash
cd ~/.claude
git init
git remote add origin git@github.com:yourusername/claude-config.git
```

## 🎯 Use Cases

### Daily Workflow
```bash
# Morning on Machine A: make changes
vim ~/.claude/settings.json

# Sync to GitHub
claude-sync

# Switch to Machine B
claude-sync  # Automatically pulls latest changes
```

### Check Status Before Syncing
```bash
claude-sync status  # See what will be synced
claude-sync          # Sync it!
```

## 🔮 Future Features

- [ ] Interactive mode with menu navigation
- [ ] Diff viewer for uncommitted changes
- [ ] Backup/restore functionality
- [ ] Profile management (work/personal)
- [ ] Selective sync (hooks only, plugins only, etc.)
- [ ] Config validation
- [ ] Remote config templates

## 🤝 Contributing

This is currently a personal project, but suggestions and contributions are welcome!

## 📝 License

MIT License - feel free to use and modify as needed.

## 🙏 Credits

- Built with [Charm](https://charm.sh/) libraries
- Inspired by dotfile managers like `chezmoi` and `dotbot`

---

Made with ❤️ for the Claude Code community
