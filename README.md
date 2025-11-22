# 🎭 claude-sync

[![CI](https://github.com/mfenderov/claude-sync/actions/workflows/ci.yml/badge.svg)](https://github.com/mfenderov/claude-sync/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/mfenderov/claude-sync)](https://goreportcard.com/report/github.com/mfenderov/claude-sync)
[![codecov](https://codecov.io/gh/mfenderov/claude-sync/branch/main/graph/badge.svg)](https://codecov.io/gh/mfenderov/claude-sync)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/github/go-mod/go-version/mfenderov/claude-sync)](https://go.dev/)

A beautiful CLI tool for syncing Claude Code configurations across machines.

Built with ❤️ using Go and [Charm](https://charm.sh/) libraries.

## ✨ Features

- **Zero-Config Setup**: Interactive first-time setup - just run `claude-sync`!
- **Smart Sync**: Automatically detects changes, commits, pulls, and pushes
- **Beautiful TUI**: Gorgeous terminal UI with emojis and colors powered by Lipgloss
- **Conflict Protection**: Safely detects and aborts on merge conflicts to protect your config
- **Auto .gitignore**: Automatically excludes sensitive files (credentials, keys, AWS scripts)
- **Remote Validation**: Checks if your git repository exists before setup
- **Status Dashboard**: View your config status with plugins, hooks, and skills

## 🚀 Quick Start

### Installation

```bash
go install github.com/mfenderov/claude-sync@latest
```

### First Time Setup

Just run `claude-sync` and it will guide you through an interactive setup:

```bash
claude-sync
```

That's it! The tool will:
1. 🔍 Detect your Claude Code configuration
2. 🎯 Prompt you to create a private git repository
3. ✨ Initialize git, create .gitignore, and push to remote
4. 🎉 Your config is now synced!

No manual git setup required - it just works!

## 📖 Usage

### Daily Workflow

After the initial setup, just run `claude-sync` whenever you want to sync:

```bash
claude-sync
```

This automatically:
1. ✅ Detects and commits local changes
2. ✅ Pulls from remote (with rebase)
3. ✅ Pushes to remote
4. ✅ Shows recent activity
5. ⚠️ Safely aborts on conflicts to protect your config

### View Status

Check your configuration status:

```bash
claude-sync status
```

Displays:
- Repository and branch info
- Modified files
- Enabled plugins
- Installed hooks
- Available skills

### On Other Machines

1. Clone your config repo:
```bash
git clone git@github.com:yourusername/claude-config.git ~/.claude
```

2. Run `claude-sync` to keep it synced:
```bash
claude-sync
```

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
- A git hosting service account (GitHub, GitLab, Bitbucket, etc.)

## 🎯 Use Cases

### First Time Setup
```bash
# Just run it - interactive setup walks you through everything!
claude-sync

# It will:
# - Detect your Claude Code config
# - Ask for your git repository URL
# - Validate the repository exists
# - Initialize git with smart .gitignore
# - Push to remote
```

### Daily Workflow on Machine A
```bash
# Make changes to your Claude config
vim ~/.claude/settings.json

# Sync to GitHub - one command!
claude-sync
```

### Sync on Machine B
```bash
# First time: clone the repo
git clone git@github.com:yourusername/claude-config.git ~/.claude

# Then just sync whenever you want
claude-sync  # Automatically pulls latest changes
```

### Check Status Anytime
```bash
claude-sync status  # See repo info, plugins, hooks, skills
```

## 🔮 Future Features

- [ ] Diff viewer for uncommitted changes
- [ ] Backup/restore functionality
- [ ] Profile management (work/personal)
- [ ] Selective sync (hooks only, plugins only, etc.)
- [ ] Config validation before sync
- [ ] Remote config templates/presets
- [ ] Automatic conflict resolution for specific file types
- [ ] Sync history and rollback

## 🛠 Development

This project uses [Task](https://taskfile.dev/) for build automation.

### Prerequisites

- Go 1.21 or later
- Git
- Task (optional, but recommended)

### Setup

```bash
# Clone the repository
git clone https://github.com/mfenderov/claude-sync.git
cd claude-sync

# Install development tools
task install-tools

# Run tests
task test

# Build
task build

# Run all checks
task check
```

### Available Tasks

```bash
task --list              # Show all available tasks
task test                # Run tests
task test-coverage       # Run tests with coverage report
task lint                # Run linter
task fmt                 # Format code
task build               # Build binary
task install             # Install to $GOPATH/bin
task ci                  # Run all CI checks locally
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
task test-coverage

# Run tests in verbose mode
task test-verbose
```

## 🤝 Contributing

This is currently a personal project, but suggestions and contributions are welcome!

### How to Contribute

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run tests (`task test`)
5. Run linter (`task lint`)
6. Commit your changes (`git commit -m 'Add amazing feature'`)
7. Push to the branch (`git push origin feature/amazing-feature`)
8. Open a Pull Request

## 📝 License

MIT License - feel free to use and modify as needed.

## 🙏 Credits

- Built with [Charm](https://charm.sh/) libraries
- Inspired by dotfile managers like `chezmoi` and `dotbot`

---

Made with ❤️ for the Claude Code community
