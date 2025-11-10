# Installation

APX can be installed in several ways. Choose the method that works best for your environment.

## Homebrew (Recommended)

```bash
# Homebrew (example)
brew install <org>/tap/apx
```

## Download from GitHub Releases

Pre-built binaries are available for major platforms:

```bash
# Download from GitHub Releases and place on PATH
curl -L https://github.com/infobloxopen/apx/releases/latest/download/apx_$(uname -s)_$(uname -m).tar.gz | tar -xz
chmod +x apx && mv apx /usr/local/bin/apx
```

### Platform-specific Downloads

#### macOS

```bash
# Intel
curl -L https://github.com/infobloxopen/apx/releases/latest/download/apx_darwin_amd64.tar.gz | tar -xz
chmod +x apx && mv apx /usr/local/bin/

# Apple Silicon
curl -L https://github.com/infobloxopen/apx/releases/latest/download/apx_darwin_arm64.tar.gz | tar -xz
chmod +x apx && mv apx /usr/local/bin/
```

#### Linux

```bash
# x86_64
curl -L https://github.com/infobloxopen/apx/releases/latest/download/apx_linux_amd64.tar.gz | tar -xz
chmod +x apx && sudo mv apx /usr/local/bin/

# ARM64
curl -L https://github.com/infobloxopen/apx/releases/latest/download/apx_linux_arm64.tar.gz | tar -xz
chmod +x apx && sudo mv apx /usr/local/bin/
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
apx version 1.0.0 (commit: abc1234)
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

APX requires Go 1.21+. Check your version:

```bash
go version
```

Update Go if needed from [golang.org](https://golang.org/dl/).

---

**Need help?** Open an issue on [GitHub](https://github.com/infobloxopen/apx/issues) or check existing [discussions](https://github.com/infobloxopen/apx/discussions).