# Contract: Shell Installer Script

**File**: `install.sh` (repo root)  
**Type**: Existing file

## Interface

### Invocation

```bash
# Install latest version
curl -fsSL https://raw.githubusercontent.com/infobloxopen/apx/main/install.sh | bash

# Install specific version
curl -fsSL https://raw.githubusercontent.com/infobloxopen/apx/main/install.sh | VERSION=1.2.3 bash

# Install to custom directory
curl -fsSL https://raw.githubusercontent.com/infobloxopen/apx/main/install.sh | INSTALL_DIR=/usr/local/bin bash
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `VERSION` | (latest release) | Specific version to install (without `v` prefix: `1.2.3`; or with: `v1.2.3`) |
| `INSTALL_DIR` | `~/.local/bin` | Target directory for the binary |
| `GITHUB_TOKEN` | (none) | Optional: for private repos or API rate limits |

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error (network, permissions, unknown platform) |

### Supported Platforms

| OS | Architecture | Archive Name |
|----|-------------|-------------|
| Linux | amd64 | `apx_VERSION_linux_amd64.tar.gz` |
| Linux | arm64 | `apx_VERSION_linux_arm64.tar.gz` |
| macOS | amd64 | `apx_VERSION_darwin_amd64.tar.gz` |
| macOS | arm64 | `apx_VERSION_darwin_arm64.tar.gz` |

Windows is not supported (no bash). Windows users should use Scoop or download from GitHub Releases.

### Output

```
Detected OS: darwin, Arch: arm64
Installing apx v1.2.3...
Downloading https://github.com/infobloxopen/apx/releases/download/v1.2.3/apx_1.2.3_darwin_arm64.tar.gz
Verifying checksum...
Installing to /Users/dev/.local/bin/apx
apx v1.2.3 installed successfully!

Add to PATH (if not already):
  export PATH="$HOME/.local/bin:$PATH"
```

### Behavior Requirements

1. **OS/Arch Detection**: Uses `uname -s` (OS) and `uname -m` (arch), mapped to Go naming:
   - `Linux` → `linux`, `Darwin` → `darwin`
   - `x86_64` → `amd64`, `aarch64`/`arm64` → `arm64`

2. **Version Resolution**: If `VERSION` not set, queries `https://api.github.com/repos/infobloxopen/apx/releases/latest` and parses `.tag_name` from redirected URL

3. **Download**: Fetches archive from `https://github.com/infobloxopen/apx/releases/download/v${VERSION}/apx_${VERSION}_${OS}_${ARCH}.tar.gz`

4. **Checksum Verification**: Downloads `checksums.txt` from the same release, extracts the expected SHA256 for the downloaded archive, and verifies with `sha256sum` or `shasum -a 256`

5. **Installation**: Extracts binary from archive, moves to `INSTALL_DIR`, sets executable permissions

6. **PATH Guidance**: If `INSTALL_DIR` is not in `$PATH`, prints a message showing how to add it

7. **Idempotent**: Re-running overwrites existing binary cleanly

8. **Non-interactive**: No prompts, no `read` calls, safe for `curl | bash` and CI

9. **curl | bash Safety**: Entire script wrapped in `main() { ... }; main "$@"` to prevent partial execution if download is interrupted
