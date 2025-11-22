#!/usr/bin/env bash
# Copyright 2025 Infoblox Inc.
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Disable colors if NO_COLOR is set or not in terminal
if [[ -n "${NO_COLOR:-}" ]] || [[ ! -t 1 ]]; then
    RED=''
    GREEN=''
    YELLOW=''
    NC=''
fi

# Script to install E2E test dependencies: k3d and kubectl

echo -e "${GREEN}Installing E2E test dependencies...${NC}"

# Detect OS
OS="$(uname -s)"
ARCH="$(uname -m)"

case "$OS" in
    Linux*)
        PLATFORM="linux"
        ;;
    Darwin*)
        PLATFORM="darwin"
        ;;
    MINGW*|MSYS*|CYGWIN*)
        echo -e "${RED}Error: Windows is not supported. Please use WSL2.${NC}"
        exit 1
        ;;
    *)
        echo -e "${RED}Error: Unsupported OS: $OS${NC}"
        exit 1
        ;;
esac

case "$ARCH" in
    x86_64)
        ARCH="amd64"
        ;;
    arm64|aarch64)
        ARCH="arm64"
        ;;
    *)
        echo -e "${RED}Error: Unsupported architecture: $ARCH${NC}"
        exit 1
        ;;
esac

echo "Detected platform: ${PLATFORM}/${ARCH}"

# Install directory
INSTALL_DIR="${HOME}/.local/bin"
mkdir -p "$INSTALL_DIR"

# Add to PATH if not already there
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo -e "${YELLOW}Note: Add $INSTALL_DIR to your PATH${NC}"
    echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
fi

# Install k3d
echo -e "\n${GREEN}Installing k3d...${NC}"
if command -v k3d &> /dev/null; then
    EXISTING_VERSION=$(k3d version | head -n1 | awk '{print $3}')
    echo "k3d is already installed: $EXISTING_VERSION"
    read -p "Reinstall k3d? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Skipping k3d installation"
    else
        curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash
    fi
else
    curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash
fi

# Verify k3d installation
if command -v k3d &> /dev/null; then
    K3D_VERSION=$(k3d version | head -n1 | awk '{print $3}')
    echo -e "${GREEN}✓ k3d installed: $K3D_VERSION${NC}"
else
    echo -e "${RED}✗ k3d installation failed${NC}"
    exit 1
fi

# Install kubectl
echo -e "\n${GREEN}Installing kubectl...${NC}"
if command -v kubectl &> /dev/null; then
    EXISTING_VERSION=$(kubectl version --client --short 2>/dev/null | awk '{print $3}' || kubectl version --client -o json 2>/dev/null | grep -o '"gitVersion":"[^"]*' | cut -d'"' -f4)
    echo "kubectl is already installed: $EXISTING_VERSION"
    read -p "Reinstall kubectl? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Skipping kubectl installation"
    else
        install_kubectl
    fi
else
    install_kubectl
fi

function install_kubectl() {
    # Get latest stable version
    KUBECTL_VERSION=$(curl -L -s https://dl.k8s.io/release/stable.txt)
    
    echo "Installing kubectl $KUBECTL_VERSION for ${PLATFORM}/${ARCH}..."
    
    curl -LO "https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/${PLATFORM}/${ARCH}/kubectl"
    chmod +x kubectl
    mv kubectl "$INSTALL_DIR/kubectl"
}

# Verify kubectl installation
if command -v kubectl &> /dev/null; then
    KUBECTL_VERSION=$(kubectl version --client --short 2>/dev/null | awk '{print $3}' || kubectl version --client -o json 2>/dev/null | grep -o '"gitVersion":"[^"]*' | cut -d'"' -f4)
    echo -e "${GREEN}✓ kubectl installed: $KUBECTL_VERSION${NC}"
else
    echo -e "${RED}✗ kubectl installation failed${NC}"
    exit 1
fi

# Verify Docker is available
echo -e "\n${GREEN}Verifying Docker installation...${NC}"
if command -v docker &> /dev/null; then
    if docker info &> /dev/null; then
        DOCKER_VERSION=$(docker version --format '{{.Server.Version}}' 2>/dev/null || echo "unknown")
        echo -e "${GREEN}✓ Docker is running: $DOCKER_VERSION${NC}"
    else
        echo -e "${RED}✗ Docker is installed but not running${NC}"
        echo "Please start Docker Desktop (macOS/Windows) or Docker Engine (Linux)"
        exit 1
    fi
else
    echo -e "${RED}✗ Docker is not installed${NC}"
    echo "Please install Docker:"
    if [[ "$PLATFORM" == "darwin" ]]; then
        echo "  macOS: https://docs.docker.com/desktop/install/mac-install/"
    else
        echo "  Linux: https://docs.docker.com/engine/install/"
    fi
    exit 1
fi

echo -e "\n${GREEN}✓ All E2E test dependencies installed successfully!${NC}"
echo ""
echo "Installed tools:"
echo "  k3d:     $(command -v k3d)"
echo "  kubectl: $(command -v kubectl)"
echo "  docker:  $(command -v docker)"
echo ""
echo "Run E2E tests with:"
echo "  make test-e2e"
