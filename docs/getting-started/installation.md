# Installation

APX can be installed from pre-built binaries (GitHub Releases), Homebrew, or from source.

:::{admonition} Alpha Release
:class: note
APX is currently in **alpha** (`v0.1.0-alpha`). The Homebrew formula and other package manager entries are not yet stable-released. We recommend installing from GitHub Releases or from source for the alpha period.
:::

## GitHub Releases (Recommended for Alpha)

Download the latest binary for your platform from the [GitHub Releases page](https://github.com/infobloxopen/apx/releases).

```bash
# macOS (Apple Silicon)
curl -L https://github.com/infobloxopen/apx/releases/latest/download/apx_Darwin_arm64.tar.gz | tar -xz
chmod +x apx && mv apx /usr/local/bin/

# macOS (Intel)
curl -L https://github.com/infobloxopen/apx/releases/latest/download/apx_Darwin_amd64.tar.gz | tar -xz
chmod +x apx && mv apx /usr/local/bin/

# Linux (x86_64)
curl -L https://github.com/infobloxopen/apx/releases/latest/download/apx_Linux_x86_64.tar.gz | tar -xz
chmod +x apx && sudo mv apx /usr/local/bin/

# Linux (ARM64)
curl -L https://github.com/infobloxopen/apx/releases/latest/download/apx_Linux_arm64.tar.gz | tar -xz
chmod +x apx && sudo mv apx /usr/local/bin/
```

## Homebrew (macOS — Stable Releases Only)

```{admonition} Planned — stable tap pending
:class: note
The Homebrew tap will be enabled when APX reaches a stable release.
```

## Build from Source

For development or latest features:

```bash
git clone https://github.com/infobloxopen/apx.git
cd apx
go build -o apx ./cmd/apx
chmod +x apx && mv apx /usr/local/bin/
```

## Verify Installation

After installation, verify APX is working correctly:

```bash
apx --version
```

You should see output similar to:
```
apx 1.0.0 (abc1234) 2026-01-15
```

## Toolchain Management

APX bundles pinned generators and tools via `apx.lock` for reproducible builds:

```bash
# Download pinned toolchain (respects apx.lock)
apx fetch

# Use container-based execution (alternative)
apx --use-container <command>
```

:::{note}
APX manages its own toolchain to ensure consistent results across environments. The first `apx fetch` will download necessary tools like `buf`, `protoc`, and language-specific generators.
:::

## Shell Completion

APX supports shell completion for bash, zsh, fish, and PowerShell.

### Bash

```bash
# Add to ~/.bashrc
source <(apx completion bash)

# Or install system-wide
apx completion bash | sudo tee /etc/bash_completion.d/apx
```

### Zsh

```bash
# Add to ~/.zshrc
source <(apx completion zsh)

# Or for oh-my-zsh
apx completion zsh > "${fpath[1]}/_apx"
```

### Fish

```bash
apx completion fish | source

# Or install permanently
apx completion fish > ~/.config/fish/completions/apx.fish
```

### PowerShell

```powershell
apx completion powershell | Out-String | Invoke-Expression

# Or add to profile
apx completion powershell >> $PROFILE
```

## Next Steps

Now that APX is installed, you can:

- Try the [Quick Start Guide](quickstart.md)
- Explore [Interactive Initialization](interactive-init.md)
- Check out the [CLI Reference](../cli-reference/index.md)

## Troubleshooting

### Command not found

If you get `command not found: apx`, ensure:

1. The binary is in your `PATH`
2. You have execute permissions: `chmod +x apx`
3. For Go install: `$GOPATH/bin` is in your `PATH`

### Permission denied

If you get permission errors:

```bash
# Make executable
chmod +x apx

# Move to user directory instead of system
mv apx ~/bin/  # ensure ~/bin is in PATH
```

### Go version issues

APX requires Go 1.26+. Check your version:

```bash
go version
```

Update Go if needed from [golang.org](https://golang.org/dl/).

---

**Need help?** Open an issue on [GitHub](https://github.com/infobloxopen/apx/issues) or check existing [discussions](https://github.com/infobloxopen/apx/discussions).