#!/bin/bash
set -e

echo "=========================================="
echo "Jellyfin Plugin Test Script"
echo "=========================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
JELLYFIN_URL="http://localhost:8096"
TEST_MOVIE_DIR="./test-media/movies/Test Movie (2024)"
TEST_MOVIE_FILE="$TEST_MOVIE_DIR/Test Movie (2024).mkv"
LEAVING_SOON_DIR="./leaving-soon-data"

print_status() {
    echo -e "${GREEN}[✓]${NC} $1"
}

print_error() {
    echo -e "${RED}[✗]${NC} $1"
}

print_info() {
    echo -e "${YELLOW}[i]${NC} $1"
}

# Step 1: Create test media
echo "Step 1: Creating test media..."
mkdir -p "$TEST_MOVIE_DIR"
if [ ! -f "$TEST_MOVIE_FILE" ]; then
    # Create a small dummy video file (1MB)
    dd if=/dev/zero of="$TEST_MOVIE_FILE" bs=1M count=1 2>/dev/null
    print_status "Created test movie file: $TEST_MOVIE_FILE"
else
    print_info "Test movie already exists"
fi

# Step 2: Create directories
echo ""
echo "Step 2: Setting up directories..."
mkdir -p jellyfin-config/plugins/OxiCleanarr
mkdir -p jellyfin-cache
mkdir -p "$LEAVING_SOON_DIR"
print_status "Directories created"

# Step 3: Build the plugin
echo ""
echo "Step 3: Building the plugin..."
cd Jellyfin.Plugin.OxiCleanarr
if ! dotnet build --configuration Release; then
    print_error "Failed to build plugin"
    exit 1
fi
print_status "Plugin built successfully"
cd ..

# Step 4: Install the plugin
echo ""
echo "Step 4: Installing plugin to Jellyfin..."
# Find the built DLL (it may be in net7.0 or net8.0 directory)
PLUGIN_DLL=$(find Jellyfin.Plugin.OxiCleanarr/bin/Release -name "Jellyfin.Plugin.OxiCleanarr.dll" | head -1)
if [ -z "$PLUGIN_DLL" ]; then
    print_error "Could not find built plugin DLL"
    exit 1
fi
cp "$PLUGIN_DLL" jellyfin-config/plugins/OxiCleanarr/
print_status "Plugin installed to: jellyfin-config/plugins/OxiCleanarr/"

# Step 5: Update docker-compose with correct paths
echo ""
echo "Step 5: Updating docker-compose.yml..."
cat > docker-compose.test.yml << EOF
version: '3.8'

services:
  jellyfin:
    image: jellyfin/jellyfin:latest
    container_name: jellyfin-test
    environment:
      - PUID=1000
      - PGID=1000
      - TZ=UTC
    volumes:
      - ./jellyfin-config:/config
      - ./jellyfin-cache:/cache
      # Test media
      - ./test-media/movies:/media/movies:ro
      # Shared volume for "Leaving Soon" symlinks
      - ./leaving-soon-data:/data/leaving-soon
    ports:
      - "8096:8096"
    restart: unless-stopped

volumes:
  leaving-soon-data:
EOF
print_status "docker-compose.test.yml created"

# Step 6: Start Jellyfin
echo ""
echo "Step 6: Starting Jellyfin..."
docker-compose -f docker-compose.test.yml down 2>/dev/null || true
docker-compose -f docker-compose.test.yml up -d
print_status "Jellyfin container started"

# Wait for Jellyfin to start
echo ""
print_info "Waiting for Jellyfin to start (this may take 30-60 seconds)..."
for i in {1..30}; do
    if curl -s "$JELLYFIN_URL/health" >/dev/null 2>&1; then
        print_status "Jellyfin is ready!"
        break
    fi
    echo -n "."
    sleep 2
done
echo ""

# Step 7: Check if Jellyfin is accessible
echo ""
echo "Step 7: Checking Jellyfin accessibility..."
if ! curl -s "$JELLYFIN_URL/health" >/dev/null 2>&1; then
    print_error "Jellyfin is not accessible at $JELLYFIN_URL"
    print_info "Please wait a bit longer and check Docker logs: docker logs jellyfin-test"
    exit 1
fi
print_status "Jellyfin is accessible"

# Step 8: Check if plugin is loaded
echo ""
echo "Step 8: Checking if plugin is loaded..."
print_info "Please open $JELLYFIN_URL in your browser and:"
print_info "1. Complete the initial setup wizard (if first time)"
print_info "2. Login as administrator"
print_info "3. Go to Dashboard → Plugins"
print_info "4. Verify 'OxiCleanarr Bridge' plugin is listed"
print_info "5. Configure the plugin:"
print_info "   - No configuration needed! Plugin is now stateless."
echo ""
read -p "Press Enter once you've completed the plugin configuration..."

# Step 9: Test the API
echo ""
echo "Step 9: Testing the plugin API..."
echo ""
print_info "Testing status endpoint..."
STATUS_RESPONSE=$(curl -s "$JELLYFIN_URL/api/oxicleanarr/status" || echo "FAILED")
echo "$STATUS_RESPONSE"
if echo "$STATUS_RESPONSE" | grep -q "version"; then
    print_status "Status endpoint working"
else
    print_error "Status endpoint failed"
fi

# Step 10: Add test movie to Leaving Soon
echo ""
echo "Step 10: Adding test movie to 'Leaving Soon' library..."
ADD_RESPONSE=$(curl -s -X POST "$JELLYFIN_URL/api/oxicleanarr/symlinks/add" \
    -H "Content-Type: application/json" \
    -d "{
        \"items\": [
            {
                \"sourcePath\": \"/media/movies/Test Movie (2024)/Test Movie (2024).mkv\",
                \"targetDirectory\": \"/data/leaving-soon\"
            }
        ]
    }" || echo "FAILED")

echo "$ADD_RESPONSE"
if echo "$ADD_RESPONSE" | grep -q "success"; then
    print_status "Movie added successfully!"
else
    print_error "Failed to add movie"
    print_info "Response: $ADD_RESPONSE"
fi

# Step 11: Check if symlink was created
echo ""
echo "Step 11: Verifying symlink creation..."
if [ -L "$LEAVING_SOON_DIR/Test Movie (2024).mkv" ]; then
    print_status "Symlink created in: $LEAVING_SOON_DIR/"
    ls -lah "$LEAVING_SOON_DIR/"
else
    print_error "Symlink not found in $LEAVING_SOON_DIR/"
    print_info "Directory contents:"
    ls -lah "$LEAVING_SOON_DIR/" || echo "Directory is empty or doesn't exist"
fi

# Step 12: Test directory management API
echo ""
echo "Step 12: Testing directory management API..."
echo ""

# Test 12a: Create directory
print_info "Testing directory creation..."
CREATE_DIR_RESPONSE=$(curl -s -X POST "$JELLYFIN_URL/api/oxicleanarr/directories/create" \
    -H "Content-Type: application/json" \
    -d "{
        \"directory\": \"/data/test-directory\"
    }" || echo "FAILED")

echo "$CREATE_DIR_RESPONSE"
if echo "$CREATE_DIR_RESPONSE" | grep -q '"Success":true'; then
    print_status "Directory creation endpoint working"
    
    # Verify directory exists in container
    if docker exec jellyfin-test test -d /data/test-directory; then
        print_status "Directory exists in container"
    else
        print_error "Directory not found in container"
    fi
else
    print_error "Directory creation failed"
fi

# Test 12b: Create directory again (idempotence test)
echo ""
print_info "Testing directory creation idempotence..."
CREATE_DIR_RESPONSE2=$(curl -s -X POST "$JELLYFIN_URL/api/oxicleanarr/directories/create" \
    -H "Content-Type: application/json" \
    -d "{
        \"directory\": \"/data/test-directory\"
    }" || echo "FAILED")

if echo "$CREATE_DIR_RESPONSE2" | grep -q '"Created":false'; then
    print_status "Correctly reported directory already exists"
else
    print_error "Should have reported Created: false"
fi

# Test 12c: Remove empty directory
echo ""
print_info "Testing empty directory removal..."
REMOVE_DIR_RESPONSE=$(curl -s -X DELETE "$JELLYFIN_URL/api/oxicleanarr/directories/remove" \
    -H "Content-Type: application/json" \
    -d "{
        \"directory\": \"/data/test-directory\",
        \"force\": false
    }" || echo "FAILED")

echo "$REMOVE_DIR_RESPONSE"
if echo "$REMOVE_DIR_RESPONSE" | grep -q '"Success":true'; then
    print_status "Directory removal endpoint working"
    
    # Verify directory was removed
    if ! docker exec jellyfin-test test -d /data/test-directory; then
        print_status "Directory successfully removed from container"
    else
        print_error "Directory still exists in container"
    fi
else
    print_error "Directory removal failed"
fi

# Test 12d: Test force removal with non-empty directory
echo ""
print_info "Testing force removal of non-empty directory..."

# Create directory and add a file
docker exec jellyfin-test mkdir -p /data/test-nonempty
docker exec jellyfin-test touch /data/test-nonempty/test-file.txt

# Try to remove without force (should fail)
REMOVE_NO_FORCE=$(curl -s -X DELETE "$JELLYFIN_URL/api/oxicleanarr/directories/remove" \
    -H "Content-Type: application/json" \
    -d "{
        \"directory\": \"/data/test-nonempty\",
        \"force\": false
    }" || echo "FAILED")

if echo "$REMOVE_NO_FORCE" | grep -q "error"; then
    print_status "Correctly rejected removal of non-empty directory"
else
    print_error "Should have rejected removal of non-empty directory"
fi

# Remove with force (should succeed)
REMOVE_WITH_FORCE=$(curl -s -X DELETE "$JELLYFIN_URL/api/oxicleanarr/directories/remove" \
    -H "Content-Type: application/json" \
    -d "{
        \"directory\": \"/data/test-nonempty\",
        \"force\": true
    }" || echo "FAILED")

if echo "$REMOVE_WITH_FORCE" | grep -q '"Success":true'; then
    print_status "Force removal succeeded"
    
    if ! docker exec jellyfin-test test -d /data/test-nonempty; then
        print_status "Non-empty directory successfully removed"
    else
        print_error "Directory still exists after force removal"
    fi
else
    print_error "Force removal failed"
fi

# Step 13: Manual verification
echo ""
echo "=========================================="
echo "Test Complete - Manual Verification"
echo "=========================================="
echo ""
print_info "Please verify the following in Jellyfin web UI ($JELLYFIN_URL):"
echo ""
echo "1. Check if 'Leaving Soon' library exists:"
echo "   → Go to Home → Libraries"
echo "   → You should see a 'Leaving Soon' library"
echo ""
echo "2. Check if the test movie appears:"
echo "   → Open the 'Leaving Soon' library"
echo "   → Look for 'Test Movie (2024)'"
echo ""
echo "3. Check Jellyfin logs for any errors:"
echo "   → Dashboard → Logs"
echo ""
echo "=========================================="
echo "Useful Commands:"
echo "=========================================="
echo ""
echo "View Jellyfin logs:"
echo "  docker logs jellyfin-test -f"
echo ""
echo "Check symlinks inside container:"
echo "  docker exec jellyfin-test ls -lah /data/leaving-soon/"
echo ""
echo "Test API manually:"
echo "  curl $JELLYFIN_URL/api/oxicleanarr/status"
echo ""
echo "Clean up test environment:"
echo "  docker-compose -f docker-compose.test.yml down"
echo "  rm -rf test-media jellyfin-config jellyfin-cache leaving-soon-data"
echo ""
