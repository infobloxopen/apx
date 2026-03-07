#!/usr/bin/env bash
# Copyright 2025 Infoblox Inc.
# SPDX-License-Identifier: Apache-2.0
#
# Standalone installer for the apx CLI.
# Downloads a pre-built binary from GitHub Releases, verifies its SHA-256
# checksum, and installs it to a configurable directory.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/infobloxopen/apx/main/install.sh | bash
#   curl -fsSL ... | VERSION=1.2.3 bash
#   curl -fsSL ... | INSTALL_DIR=/usr/local/bin bash
#
# Environment variables:
#   VERSION      - Pin a specific release (default: latest)
#   INSTALL_DIR  - Where to place the binary (default: ~/.local/bin)
#   GITHUB_TOKEN - Optional token for GitHub API (avoids rate limits)
#   NO_COLOR     - Disable colored output when set

set -euo pipefail

# ── Globals ──────────────────────────────────────────────────────────────────

REPO_OWNER="infobloxopen"
REPO_NAME="apx"
BINARY_NAME="apx"
GITHUB_BASE="https://github.com/${REPO_OWNER}/${REPO_NAME}"
API_BASE="https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}"

# ── Colors ───────────────────────────────────────────────────────────────────

setup_colors() {
    if [[ -n "${NO_COLOR:-}" ]] || [[ ! -t 1 ]]; then
        RED=""
        GREEN=""
        YELLOW=""
        BLUE=""
        BOLD=""
        RESET=""
    else
        RED='\033[0;31m'
        GREEN='\033[0;32m'
        YELLOW='\033[1;33m'
        BLUE='\033[0;34m'
        BOLD='\033[1m'
        RESET='\033[0m'
    fi
}

# ── Logging ──────────────────────────────────────────────────────────────────

info()  { printf "%b\n" "${BLUE}info${RESET}  $*"; }
ok()    { printf "%b\n" "${GREEN}  ok${RESET}  $*"; }
warn()  { printf "%b\n" "${YELLOW}warn${RESET}  $*" >&2; }
error() { printf "%b\n" "${RED}error${RESET} $*" >&2; }
die()   { error "$@"; exit 1; }

# ── Prerequisite checks ─────────────────────────────────────────────────────

need_cmd() {
    if ! command -v "$1" &>/dev/null; then
        die "Required command '${BOLD}$1${RESET}' not found. Please install it and retry."
    fi
}

check_prerequisites() {
    need_cmd curl
    need_cmd tar
    need_cmd uname

    # At least one SHA-256 tool must be available.
    if command -v sha256sum &>/dev/null; then
        SHA_CMD="sha256sum"
    elif command -v shasum &>/dev/null; then
        SHA_CMD="shasum -a 256"
    else
        die "Neither ${BOLD}sha256sum${RESET} nor ${BOLD}shasum${RESET} found. Cannot verify checksums."
    fi
}

# ── Platform detection ───────────────────────────────────────────────────────

detect_os() {
    local os
    os="$(uname -s)"
    case "$os" in
        Linux*)          echo "linux"  ;;
        Darwin*)         echo "darwin" ;;
        MINGW*|MSYS*|CYGWIN*)
            echo "windows" ;;
        *)
            die "Unsupported operating system: ${BOLD}$os${RESET}" ;;
    esac
}

detect_arch() {
    local arch
    arch="$(uname -m)"
    case "$arch" in
        x86_64|amd64)    echo "amd64" ;;
        arm64|aarch64)   echo "arm64" ;;
        *)
            die "Unsupported architecture: ${BOLD}$arch${RESET}" ;;
    esac
}

# ── Version resolution ───────────────────────────────────────────────────────

# Resolve the latest release tag from GitHub.
# Strategy 1: redirect-based (no API rate limit).
# Strategy 2: GitHub REST API (fallback).
resolve_latest_version() {
    local version=""
    local auth_header=()

    if [[ -n "${GITHUB_TOKEN:-}" ]]; then
        auth_header=(-H "Authorization: token ${GITHUB_TOKEN}")
    fi

    # Strategy 1 – follow redirect on /releases/latest and grab tag from URL.
    version=$(curl -fsSI "${auth_header[@]+"${auth_header[@]}"}" \
        "${GITHUB_BASE}/releases/latest" 2>/dev/null \
        | grep -i '^location:' \
        | sed -E 's|.*/tag/v?||;s/[[:space:]]//g') || true

    if [[ -n "$version" ]]; then
        echo "$version"
        return
    fi

    # Strategy 2 – GitHub API.
    version=$(curl -fsSL "${auth_header[@]+"${auth_header[@]}"}" \
        "${API_BASE}/releases/latest" 2>/dev/null \
        | grep '"tag_name":' \
        | sed -E 's/.*"v?([^"]+)".*/\1/') || true

    if [[ -n "$version" ]]; then
        echo "$version"
        return
    fi

    die "Could not determine the latest release version.\n" \
        "      Set ${BOLD}VERSION${RESET} explicitly or provide ${BOLD}GITHUB_TOKEN${RESET} to avoid rate limits."
}

# ── Download helpers ─────────────────────────────────────────────────────────

download() {
    local url="$1" dest="$2"
    local auth_header=()
    if [[ -n "${GITHUB_TOKEN:-}" ]]; then
        auth_header=(-H "Authorization: token ${GITHUB_TOKEN}")
    fi
    local http_code
    http_code=$(curl -fsSL "${auth_header[@]+"${auth_header[@]}"}" \
        -w "%{http_code}" -o "$dest" "$url") || true

    if [[ "$http_code" -lt 200 || "$http_code" -ge 300 ]]; then
        die "Download failed (HTTP $http_code): $url"
    fi
}

# ── Checksum verification ───────────────────────────────────────────────────

verify_checksum() {
    local archive="$1" checksums_file="$2"
    local archive_name
    archive_name="$(basename "$archive")"

    local expected
    expected=$(grep "${archive_name}" "$checksums_file" | awk '{print $1}')

    if [[ -z "$expected" ]]; then
        die "No checksum found for ${BOLD}${archive_name}${RESET} in checksums.txt"
    fi

    local actual
    actual=$($SHA_CMD "$archive" | awk '{print $1}')

    if [[ "$expected" != "$actual" ]]; then
        error "Checksum mismatch for ${BOLD}${archive_name}${RESET}"
        error "  expected: $expected"
        error "  actual:   $actual"
        die "The downloaded file may be corrupted or tampered with."
    fi
}

# ── PATH helper ──────────────────────────────────────────────────────────────

ensure_in_path() {
    local dir="$1"
    if [[ ":${PATH}:" == *":${dir}:"* ]]; then
        return
    fi

    warn "${BOLD}${dir}${RESET} is not in your PATH."

    local shell_name
    shell_name="$(basename "${SHELL:-/bin/sh}")"
    local rc_file=""
    # shellcheck disable=SC2088 # Tildes are intentional — shown to user as typed paths
    case "$shell_name" in
        bash) rc_file="~/.bashrc" ;;
        zsh)  rc_file="~/.zshrc"  ;;
        fish) rc_file="~/.config/fish/config.fish" ;;
    esac

    if [[ -n "$rc_file" ]]; then
        warn "Add it by running:"
        if [[ "$shell_name" == "fish" ]]; then
            warn "  fish_add_path ${dir}"
        else
            warn "  echo 'export PATH=\"${dir}:\$PATH\"' >> ${rc_file}"
        fi
    else
        warn "Add ${dir} to your shell's configuration file."
    fi
}

# ── Main ─────────────────────────────────────────────────────────────────────

main() {
    setup_colors

    printf "%b\n" "${BOLD}apx installer${RESET}"
    echo ""

    check_prerequisites

    # ── Platform ──
    local os arch
    os="$(detect_os)"
    arch="$(detect_arch)"
    info "Platform: ${BOLD}${os}/${arch}${RESET}"

    # ── Version ──
    local version="${VERSION:-}"
    if [[ -z "$version" ]]; then
        info "Resolving latest version…"
        version="$(resolve_latest_version)"
    fi
    # Strip leading 'v' if present for archive naming.
    version="${version#v}"
    info "Version:  ${BOLD}v${version}${RESET}"

    # ── Archive naming ──
    local ext="tar.gz"
    if [[ "$os" == "windows" ]]; then
        ext="zip"
    fi
    local archive_name="${BINARY_NAME}_${version}_${os}_${arch}.${ext}"
    local release_tag="v${version}"
    local base_url="${GITHUB_BASE}/releases/download/${release_tag}"

    # ── Temp directory ──
    local tmpdir
    tmpdir="$(mktemp -d)"
    trap 'rm -rf "$tmpdir"' EXIT

    # ── Download archive + checksums ──
    info "Downloading ${BOLD}${archive_name}${RESET}…"
    download "${base_url}/${archive_name}" "${tmpdir}/${archive_name}"

    info "Downloading ${BOLD}checksums.txt${RESET}…"
    download "${base_url}/checksums.txt" "${tmpdir}/checksums.txt"

    # ── Verify integrity ──
    info "Verifying checksum…"
    verify_checksum "${tmpdir}/${archive_name}" "${tmpdir}/checksums.txt"
    ok   "Checksum verified"

    # ── Extract ──
    info "Extracting…"
    if [[ "$ext" == "tar.gz" ]]; then
        tar -xzf "${tmpdir}/${archive_name}" -C "$tmpdir"
    else
        # Windows zip — requires unzip.
        need_cmd unzip
        unzip -oq "${tmpdir}/${archive_name}" -d "$tmpdir"
    fi

    if [[ ! -f "${tmpdir}/${BINARY_NAME}" ]]; then
        die "Expected binary '${BINARY_NAME}' not found in archive."
    fi

    # ── Install ──
    local install_dir="${INSTALL_DIR:-${HOME}/.local/bin}"
    mkdir -p "$install_dir"

    local dest="${install_dir}/${BINARY_NAME}"

    # If the target requires elevated privileges, try sudo.
    if [[ -w "$install_dir" ]]; then
        mv "${tmpdir}/${BINARY_NAME}" "$dest"
        chmod +x "$dest"
    else
        warn "${BOLD}${install_dir}${RESET} is not writable — using sudo."
        sudo mv "${tmpdir}/${BINARY_NAME}" "$dest"
        sudo chmod +x "$dest"
    fi

    ok   "Installed ${BOLD}${BINARY_NAME}${RESET} to ${BOLD}${dest}${RESET}"

    # ── PATH check ──
    ensure_in_path "$install_dir"

    # ── Verify installation ──
    echo ""
    if command -v "$BINARY_NAME" &>/dev/null; then
        info "Verify:   $("$BINARY_NAME" --version 2>/dev/null || echo "${BINARY_NAME} v${version}")"
    else
        warn "Run '${BINARY_NAME} --version' after updating your PATH to verify."
    fi

    echo ""
    printf "%b\n" "${GREEN}${BOLD}apx v${version}${RESET}${GREEN} installed successfully.${RESET}"
}

# Wrap everything in main so the entire script is parsed before execution.
# This is critical for `curl | bash` usage.
main "$@"
