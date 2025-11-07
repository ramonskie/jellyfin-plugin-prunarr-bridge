#!/bin/bash
# Check and install prerequisites
set -e

echo "=========================================="
echo "Checking Prerequisites"
echo "=========================================="
echo ""

# Check Docker
if command -v docker &> /dev/null; then
    echo "✓ Docker is installed ($(docker --version))"
else
    echo "✗ Docker is NOT installed"
    echo "  Install from: https://docs.docker.com/get-docker/"
    exit 1
fi

# Check Docker Compose
if command -v docker-compose &> /dev/null; then
    echo "✓ Docker Compose is installed ($(docker-compose --version))"
else
    echo "✗ Docker Compose is NOT installed"
    echo "  Install from: https://docs.docker.com/compose/install/"
    exit 1
fi

# Check mise (required for .NET SDK)
if ! command -v mise &> /dev/null; then
    echo "✗ mise is NOT installed"
    echo ""
    echo "mise is required to manage .NET SDK installation"
    echo "Install mise from: https://mise.jit.su"
    exit 1
fi

echo "✓ mise is installed ($(mise --version | head -1))"

# Check if .NET SDK is installed via mise
if mise which dotnet &> /dev/null 2>&1; then
    DOTNET_VERSION=$(mise exec -- dotnet --version)
    echo "✓ .NET SDK is installed via mise ($DOTNET_VERSION)"
else
    echo "⚠ .NET SDK is NOT installed via mise"
    echo "Installing .NET SDK via mise..."
    mise install
    
    if mise which dotnet &> /dev/null 2>&1; then
        DOTNET_VERSION=$(mise exec -- dotnet --version)
        echo "✓ .NET SDK installed successfully via mise ($DOTNET_VERSION)"
    else
        echo "✗ .NET SDK installation via mise failed"
        exit 1
    fi
fi

# Check jq (optional, for pretty JSON output)
if command -v jq &> /dev/null; then
    echo "✓ jq is installed (for pretty JSON output)"
else
    echo "⚠ jq is NOT installed (optional, for pretty JSON output)"
    echo "  Install with: sudo apt install jq (or your package manager)"
fi

# Check if port 8096 is available
if ss -tuln | grep -q ":8096 "; then
    echo "⚠ Port 8096 is already in use"
    echo "  Another Jellyfin instance might be running"
    echo "  Stop it before running tests"
else
    echo "✓ Port 8096 is available"
fi

echo ""
echo "=========================================="
echo "All prerequisites are ready!"
echo "=========================================="
echo ""
echo "You can now run: ./quick-test.sh"
echo ""
