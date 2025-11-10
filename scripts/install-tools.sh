#!/bin/bash
# APX Tool Installation Script

set -e

echo "Installing APX dependencies..."

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Install Go tools
echo "Installing Go-based tools..."

# Install buf
if ! command_exists buf; then
    echo "Installing buf..."
    go install github.com/bufbuild/buf/cmd/buf@latest
else
    echo "✓ buf is already installed"
fi

# Install oasdiff
if ! command_exists oasdiff; then
    echo "Installing oasdiff..."
    go install github.com/Tufin/oasdiff@latest
else
    echo "✓ oasdiff is already installed"
fi

# Install protoc-gen-go
if ! command_exists protoc-gen-go; then
    echo "Installing protoc-gen-go..."
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
else
    echo "✓ protoc-gen-go is already installed"
fi

# Install protoc-gen-go-grpc
if ! command_exists protoc-gen-go-grpc; then
    echo "Installing protoc-gen-go-grpc..."
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
else
    echo "✓ protoc-gen-go-grpc is already installed"
fi

# Install Node.js tools
if command_exists npm; then
    echo "Installing Node.js-based tools..."
    
    # Install spectral
    if ! command_exists spectral; then
        echo "Installing spectral..."
        npm install -g @stoplight/spectral-cli
    else
        echo "✓ spectral is already installed"
    fi
else
    echo "⚠ npm not found. Please install Node.js to use spectral."
    echo "  Visit: https://nodejs.org/"
fi

# Check for protoc
if ! command_exists protoc; then
    echo "⚠ protoc not found. Please install Protocol Buffers compiler."
    echo "  Visit: https://github.com/protocolbuffers/protobuf/releases"
    echo "  Or use your system package manager:"
    echo "    macOS: brew install protobuf"
    echo "    Ubuntu/Debian: apt-get install protobuf-compiler"
    echo "    CentOS/RHEL: yum install protobuf-compiler"
else
    echo "✓ protoc is already installed"
fi

# Check for git
if ! command_exists git; then
    echo "⚠ git not found. Please install Git."
    echo "  Visit: https://git-scm.com/downloads"
else
    echo "✓ git is already installed"
fi

echo ""
echo "✅ Tool installation complete!"
echo ""
echo "Installed tools:"
command_exists buf && echo "  ✓ buf $(buf --version 2>/dev/null | head -n1)"
command_exists oasdiff && echo "  ✓ oasdiff $(oasdiff version 2>/dev/null || echo 'installed')"
command_exists protoc-gen-go && echo "  ✓ protoc-gen-go installed"
command_exists protoc-gen-go-grpc && echo "  ✓ protoc-gen-go-grpc installed"
command_exists spectral && echo "  ✓ spectral $(spectral --version 2>/dev/null || echo 'installed')"
command_exists protoc && echo "  ✓ protoc $(protoc --version 2>/dev/null || echo 'installed')"
command_exists git && echo "  ✓ git $(git --version 2>/dev/null || echo 'installed')"

echo ""
echo "You can now use 'apx' to manage your API schemas!"
echo "Run 'apx --help' to get started."