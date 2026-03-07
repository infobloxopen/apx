# Contract: Documentation Updates

**Files**: `README.md`, `docs/getting-started/installation.md`  
**Type**: Modify existing

## README.md — Installation Section

The existing Installation section (around line 101) currently lists:
1. Homebrew
2. Download Binary
3. Build from Source

Update to:

```markdown
## Installation

### Homebrew (macOS / Linux)

```bash
brew install infobloxopen/tap/apx
```

### Scoop (Windows)

```powershell
scoop bucket add infobloxopen https://github.com/infobloxopen/scoop-bucket
scoop install infobloxopen/apx
```

### Shell Installer (macOS / Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/infobloxopen/apx/main/install.sh | bash
```

### Download Binary

Download pre-built binaries from the [GitHub Releases](https://github.com/infobloxopen/apx/releases) page.

### Build from Source

```bash
go install github.com/infobloxopen/apx/cmd/apx@latest
```
```

## docs/getting-started/installation.md

Add sections for Scoop and Shell Installer between the existing Homebrew and GitHub Releases sections. Update the overview paragraph to mention all installation methods.

### New: Scoop Section

```markdown
## Scoop (Windows)

[Scoop](https://scoop.sh) is a command-line installer for Windows.

### Install

1. Add the APX bucket:

   ```powershell
   scoop bucket add infobloxopen https://github.com/infobloxopen/scoop-bucket
   ```

2. Install APX:

   ```powershell
   scoop install infobloxopen/apx
   ```

### Update

```powershell
scoop update apx
```

### Uninstall

```powershell
scoop uninstall apx
```
```

### New: Shell Installer Section

```markdown
## Shell Installer (macOS / Linux)

For quick installation without a package manager, use the one-line installer:

```bash
curl -fsSL https://raw.githubusercontent.com/infobloxopen/apx/main/install.sh | bash
```

### Options

Install a specific version:

```bash
curl -fsSL https://raw.githubusercontent.com/infobloxopen/apx/main/install.sh | VERSION=1.2.3 bash
```

Install to a custom directory:

```bash
curl -fsSL https://raw.githubusercontent.com/infobloxopen/apx/main/install.sh | INSTALL_DIR=/usr/local/bin bash
```

The installer automatically detects your operating system and architecture, downloads the appropriate binary, and verifies its checksum.
```
