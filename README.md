# 🎭 claude-sync

[![CI](https://github.com/mfenderov/claude-sync/actions/workflows/ci.yml/badge.svg)](https://github.com/mfenderov/claude-sync/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/mfenderov/claude-sync)](https://goreportcard.com/report/github.com/mfenderov/claude-sync)
[![codecov](https://codecov.io/gh/mfenderov/claude-sync/branch/main/graph/badge.svg)](https://codecov.io/gh/mfenderov/claude-sync)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A CLI tool for syncing Claude Code configurations across machines. Built with [Charm](https://charm.sh/) libraries.

## ✨ Features

- **Zero-Config**: Interactive setup - just run `claude-sync`
- **Smart Sync**: Auto-detects changes, commits, pulls, and pushes
- **Conflict Protection**: Safely aborts on merge conflicts
- **Auto .gitignore**: Excludes sensitive files (credentials, keys, `.env`)

## 🚀 Installation

```bash
go install github.com/mfenderov/claude-sync@latest
```

## 📖 Usage

### First Time

```bash
claude-sync
```

The interactive setup will guide you through connecting to a git repository.

### Daily Sync

```bash
claude-sync          # Commit, pull, push - all in one
claude-sync status   # View repo info, plugins, hooks, skills
```

### On Other Machines

```bash
git clone git@github.com:you/claude-config.git ~/.claude
claude-sync   # Keep it synced
```

## 🛠 Development

```bash
task test     # Run tests
task build    # Build binary
task lint     # Run linter
task ci       # All CI checks
```

## 📝 License

MIT

---

Built with [Charm](https://charm.sh/) • Inspired by [chezmoi](https://chezmoi.io/)
