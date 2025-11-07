#!/bin/bash
# Quick test script - Just the essentials
set -e

echo "=========================================="
echo "Quick Plugin Test"
echo "=========================================="
echo ""

# Build plugin
echo "1. Building plugin..."
cd Jellyfin.Plugin.PrunarrBridge
mise exec -- dotnet build --configuration Release
cd ..
echo "✓ Plugin built"

# Install plugin
echo ""
echo "2. Installing plugin..."
mkdir -p jellyfin-config/plugins/PrunarrBridge
PLUGIN_DLL=$(find Jellyfin.Plugin.PrunarrBridge/bin/Release -name "Jellyfin.Plugin.PrunarrBridge.dll" | head -1)
cp "$PLUGIN_DLL" jellyfin-config/plugins/PrunarrBridge/
echo "✓ Plugin installed"

# Create test data
echo ""
echo "3. Creating test movie..."
mkdir -p "test-media/movies/Test Movie (2024)"
# Create a valid video file
if command -v ffmpeg &> /dev/null; then
    ffmpeg -f lavfi -i testsrc=duration=5:size=640x480:rate=1 -f lavfi -i sine=frequency=1000:duration=5 -y "test-media/movies/Test Movie (2024)/Test Movie (2024).mkv" > /dev/null 2>&1
    echo "✓ Test video created with ffmpeg"
else
    dd if=/dev/zero of="test-media/movies/Test Movie (2024)/Test Movie (2024).mkv" bs=1M count=1 2>/dev/null
    echo "✓ Test file created (ffmpeg not available - file may not play)"
fi

# Create directories
echo ""
echo "4. Setting up directories..."
mkdir -p jellyfin-cache leaving-soon-data
echo "✓ Directories ready"

# Start Jellyfin
echo ""
echo "5. Starting Jellyfin (this will take ~30 seconds)..."
docker-compose -f docker-compose.test.yml down 2>/dev/null || true
docker-compose -f docker-compose.test.yml up -d

# Wait for Jellyfin
echo ""
echo "Waiting for Jellyfin to start..."
for i in {1..30}; do
    if curl -s http://localhost:8096/health >/dev/null 2>&1; then
        echo "✓ Jellyfin is ready!"
        break
    fi
    echo -n "."
    sleep 2
done
echo ""

echo ""
echo "=========================================="
echo "Next Steps"
echo "=========================================="
echo ""
echo "Setup Jellyfin (first time only):"
echo "1. Open http://localhost:8096 in your browser"
echo "2. Complete setup wizard with these credentials:"
echo "   Username: test"
echo "   Password: test"
echo "3. Skip library setup (plugin will create it)"
echo ""
echo "Test the plugin:"
echo "  export JELLYFIN_USER=test"
echo "  export JELLYFIN_PASS=test"
echo "  ./test-api.sh"
echo ""
echo "Or to run tests with default credentials (test/test):"
echo "  ./test-api.sh"
echo ""
