# 🥇 Hatcher - Git Worktree Tool

> **Hatcher** "hatches" your worktrees into AI-powered development environments

A powerful command-line tool that simplifies Git worktree management with automatic directory naming, branch detection, and editor integration. Built in Go for cross-platform compatibility.

## 🚀 Features

- **🥚 Automatic worktree creation** with consistent naming (`project-branch-name`)
- **🧠 Smart branch detection** (existing vs new branches)
- **🎯 Editor integration** with Cursor and VS Code support
- **📁 Auto-copy functionality** for development files (`.ai/`, `.cursorrules`, etc.)
- **⚙️ Flexible configuration** with JSON-based file copying rules
- **🔄 Complete worktree lifecycle** management (create, move, remove)
- **🌐 Cross-platform support** (macOS, Windows, Linux)

## 📦 Installation

### Quick Install (Recommended)
```bash
# One-liner installation (no Homebrew tap required)
curl -fsSL https://keishimizu26629.github.io/hatcher/install.sh | bash
```

This installs both `hatcher` and `hch` commands to `/usr/local/bin/`.

### Homebrew
```bash
# Add tap and install
brew tap keishimizu26629/tap
brew install hatcher

# Or install without keeping the tap
brew install keishimizu26629/tap/hatcher
```

### Manual Installation
```bash
# Download latest release
curl -fsSL https://github.com/keishimizu26629/hatcher/releases/latest/download/hatcher-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m) -o hatcher

# Make executable and install
chmod +x hatcher
sudo mv hatcher /usr/local/bin/

# Create hch alias
sudo ln -sf /usr/local/bin/hatcher /usr/local/bin/hch
```

### From Source
```bash
# Prerequisites: Git 2.5+, Go 1.22+
git clone https://github.com/keishimizu26629/hatcher.git
cd hatcher
make install
```

### Uninstall
```bash
# Remove Hatcher completely
curl -fsSL https://keishimizu26629.github.io/hatcher/uninstall.sh | bash
```

## 🚀 Automated Releases

Hatcher uses automated release workflows:

- **Auto-release**: Commits to `main` branch automatically create new releases based on commit messages
- **Semantic versioning**:
  - `feat:` or `✨` → Minor version bump (1.0.0 → 1.1.0)
  - `fix:` or `🔧` → Patch version bump (1.0.0 → 1.0.1)
  - `BREAKING CHANGE:` or `!` → Major version bump (1.0.0 → 2.0.0)
- **Binary distribution**: All platforms built and released automatically
- **Install script updates**: GitHub Pages automatically updated

## 🎯 Quick Start

```bash
# Create a new worktree for feature development
hatcher feature/user-auth

# Move to existing worktree and open in editor
hatcher move main

# Remove worktree and branches when done
hatcher remove -br feature/user-auth

# List all managed worktrees
hatcher list

# Check system configuration
hatcher doctor
```

## 📋 Commands

### Basic Worktree Creation
```bash
hatcher <branch-name>              # Create worktree for branch
hatcher --dry-run feature/test     # Preview what would be created
hatcher --no-copy feature/minimal  # Skip auto-file copying
```

### Move Command (Editor Integration)
```bash
hatcher move <branch-name>         # Open worktree in new editor window
hatcher move -s <branch-name>      # Switch: close current editor, open new
hatcher move -y <branch-name>      # Auto-create if worktree doesn't exist
```

### Remove Command
```bash
hatcher remove <branch-name>       # Remove worktree only
hatcher remove -b <branch-name>    # Remove worktree + local branch
hatcher remove -r <branch-name>    # Remove worktree + remote branch
hatcher remove -br <branch-name>   # Remove worktree + both branches
```

### Utility Commands
```bash
hatcher list                       # List hatcher-managed worktrees
hatcher doctor                     # Validate configuration
```

## 🎨 Directory Structure

```
/your/projects/
├── my-app/                    # Main repository
├── my-app-feature-auth/       # Feature worktree
├── my-app-bugfix-header/      # Bugfix worktree
└── my-app-release-v2/         # Release worktree
```

## 🛠️ Editor Support

| Editor | Detection | Switch Behavior | Notes |
|--------|-----------|----------------|-------|
| **Cursor** | `cursor` command | Quit + reopen | Priority: 1st |
| **VS Code** | `code` command | Quit + reopen | Priority: 2nd |

## ⚙️ Configuration

### Default Auto-Copy Files

By default, Hatcher automatically copies these files/directories to new worktrees:

- `.ai/` - AI assistant configurations (directory with contents)
- `.cursorrules` - Cursor editor rules
- `.clinerules` - Cline rules
- `CLAUDE.md` - Claude Code instructions

### Custom Configuration

Create a configuration file to customize which files are copied:

```json
{
  "version": 2,
  "items": [
    {
      "path": ".ai/",
      "directory": true,
      "recursive": true,
      "rootOnly": true
    },
    {
      "path": ".cursorrules",
      "rootOnly": true,
      "autoDetect": true
    },
    {
      "path": "CLAUDE.md",
      "directory": false,
      "rootOnly": true
    },
    {
      "path": "**/.cursorrules",
      "autoDetect": true
    }
  ]
}
```

**Configuration Options:**
- `path` - File or directory path (supports `**/*.ext` glob patterns)
- `directory` - Explicitly mark as directory (`true`) or file (`false`)
- `recursive` - Copy directory contents recursively
- `rootOnly` - Only copy from project root (not subdirectories)
- `autoDetect` - Auto-detect if path is file or directory

**Configuration Priority:**
1. `.hatcher-auto-copy.json` (project root)
2. `.hatcher/config.json` (project directory)
3. `~/.hatcher/config.yaml` (global)

## 🔧 Development

### Building
```bash
# Install dependencies
go mod download

# Run tests
make test

# Build for current platform
make build

# Build for all platforms
make build-all

# Install locally
make install
```

### Project Structure
```
hatcher/
├── cmd/                    # Command implementations
├── internal/              # Internal packages
│   ├── config/           # Configuration management
│   ├── git/              # Git operations
│   ├── editor/           # Editor integration
│   └── autocopy/         # Auto-copy functionality
├── pkg/                  # Public packages
└── test/                 # Test files and fixtures
```

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch: `hatcher feature/amazing-feature`
3. Commit your changes: `git commit -m 'Add amazing feature'`
4. Push to the branch: `git push origin feature/amazing-feature`
5. Open a Pull Request

## 📝 License

MIT License - see [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Inspired by Git's powerful worktree functionality
- Built for modern development workflows
- Optimized for AI-assisted development environments

## 💡 Recommended Alias

Add this to your shell configuration for quick access:

```bash
# ~/.zshrc or ~/.bashrc
alias hch='hatcher'
```

Then use: `hch feature/new-branch` 🚀

---

**Made with ❤️ for developers who love clean, organized Git workflows**
