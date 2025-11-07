#!/bin/bash
# Test the API endpoints
set -e

JELLYFIN_URL="http://localhost:8096"
JELLYFIN_USER="${JELLYFIN_USER:-test}"
JELLYFIN_PASS="${JELLYFIN_PASS:-test}"

echo "=========================================="
echo "Testing Plugin API"
echo "=========================================="
echo ""

# Authenticate and get token
echo "Authenticating as user: $JELLYFIN_USER"
AUTH_RESPONSE=$(curl -s -X POST "$JELLYFIN_URL/Users/AuthenticateByName" \
    -H "Content-Type: application/json" \
    -H "X-Emby-Authorization: MediaBrowser Client=\"TestScript\", Device=\"TestDevice\", DeviceId=\"test-device-001\", Version=\"1.0.0\"" \
    -d "{\"Username\":\"$JELLYFIN_USER\",\"Pw\":\"$JELLYFIN_PASS\"}")

TOKEN=$(echo "$AUTH_RESPONSE" | jq -r '.AccessToken')
USER_ID=$(echo "$AUTH_RESPONSE" | jq -r '.User.Id')

if [ "$TOKEN" = "null" ] || [ -z "$TOKEN" ]; then
    echo "✗ Authentication failed!"
    echo "Make sure user exists (default: test/test)"
    echo "Response: $AUTH_RESPONSE"
    exit 1
fi

echo "✓ Authenticated successfully"
echo "Token: ${TOKEN:0:20}..."
echo "User ID: $USER_ID"
echo ""

# Test 1: Status endpoint
echo "Test 1: Checking plugin status..."
echo "Request: GET $JELLYFIN_URL/api/prunarr/status"
echo ""
STATUS=$(curl -s "$JELLYFIN_URL/api/prunarr/status" -H "X-Emby-Token: $TOKEN")
echo "Response:"
echo "$STATUS" | jq . 2>/dev/null || echo "$STATUS"
echo ""

if echo "$STATUS" | grep -qi "version"; then
    echo "✓ Status endpoint working"
else
    echo "✗ Status endpoint failed"
    echo "Make sure plugin is installed and configured"
    exit 1
fi

# Test 2: Add movie
echo ""
echo "Test 2: Adding test movie (creating symlink)..."
echo "Request: POST $JELLYFIN_URL/api/prunarr/symlinks/add"
echo ""
ADD_RESPONSE=$(curl -s -X POST "$JELLYFIN_URL/api/prunarr/symlinks/add" \
    -H "Content-Type: application/json" \
    -H "X-Emby-Token: $TOKEN" \
    -d '{
        "items": [
            {
                "sourcePath": "/media/movies/Test Movie (2024)/Test Movie (2024).mkv",
                "targetDirectory": "/data/leaving-soon"
            }
        ]
    }')
echo "Response:"
echo "$ADD_RESPONSE" | jq . 2>/dev/null || echo "$ADD_RESPONSE"
echo ""

if echo "$ADD_RESPONSE" | grep -qi "success"; then
    echo "✓ Symlink created successfully"
else
    echo "✗ Failed to create symlink"
    echo "Check Jellyfin logs: docker logs jellyfin-test"
    exit 1
fi

# Test 3: Check symlink
echo ""
echo "Test 3: Verifying symlink creation..."
if [ -L "leaving-soon-data/Test Movie (2024).mkv" ]; then
    echo "✓ Symlink created on host"
    ls -lah leaving-soon-data/
else
    echo "✗ Symlink not found on host"
    echo "Host directory contents:"
    ls -lah leaving-soon-data/ || echo "Directory empty"
fi

echo ""
echo "Checking inside container..."
CONTAINER_LS=$(docker exec jellyfin-test ls -lah /data/leaving-soon/ 2>/dev/null || echo "Failed to list")
echo "$CONTAINER_LS"

if echo "$CONTAINER_LS" | grep -q "Test Movie"; then
    echo "✓ Symlink exists inside container"
else
    echo "✗ Symlink not found inside container"
fi

# Final verification steps
echo ""
echo "=========================================="
echo "Test 4: List Symlinks"
echo "=========================================="
echo ""

LIST_RESPONSE=$(curl -s "$JELLYFIN_URL/api/prunarr/symlinks/list?directory=/data/leaving-soon" -H "X-Emby-Token: $TOKEN")
echo "Response:"
echo "$LIST_RESPONSE" | jq . 2>/dev/null || echo "$LIST_RESPONSE"
echo ""

SYMLINK_COUNT=$(echo "$LIST_RESPONSE" | jq -r '.Count' 2>/dev/null || echo "0")
if [ "$SYMLINK_COUNT" -gt 0 ]; then
    echo "✓ Found $SYMLINK_COUNT symlink(s)"
else
    echo "⚠ No symlinks found"
fi

echo ""
echo "=========================================="
echo "Test 5: Verifying Movie in Jellyfin Library"
echo "=========================================="
echo ""

# Get library ID
LIBRARY_ID=$(curl -s "$JELLYFIN_URL/Library/VirtualFolders" -H "X-Emby-Token: $TOKEN" | jq -r '.[] | select(.Name == "Leaving Soon") | .ItemId')
echo "Leaving Soon Library ID: $LIBRARY_ID"

if [ -z "$LIBRARY_ID" ] || [ "$LIBRARY_ID" = "null" ]; then
    echo "⚠ 'Leaving Soon' library not found!"
    echo "Note: Prunarr needs to create this library. Plugin only manages symlinks."
    echo ""
    echo "To manually create the library:"
    echo "1. Go to Dashboard → Libraries"
    echo "2. Add Media Library → Name: 'Leaving Soon'"
    echo "3. Add folder: /data/leaving-soon"
    echo "4. Content type: Movies"
    echo "5. Save and scan"
else
    # Get items in library
    ITEMS=$(curl -s "$JELLYFIN_URL/Users/$USER_ID/Items?ParentId=$LIBRARY_ID&Recursive=true" -H "X-Emby-Token: $TOKEN")
    ITEM_COUNT=$(echo "$ITEMS" | jq -r '.TotalRecordCount')
    
    echo "Items in Leaving Soon library: $ITEM_COUNT"
    echo ""
    echo "Movies:"
    echo "$ITEMS" | jq '.Items[] | {Name, Type, Id}'
    echo ""
    
    if [ "$ITEM_COUNT" -gt 0 ]; then
        echo "✓ Movie(s) found in Jellyfin library!"
    else
        echo "⚠ No movies found in library yet"
        echo "Triggering library scan..."
        curl -s -X POST "$JELLYFIN_URL/Library/Refresh" -H "X-Emby-Token: $TOKEN"
        echo "Wait a few seconds and refresh the Jellyfin UI"
    fi
fi

echo ""
echo "=========================================="
echo "Manual Verification"
echo "=========================================="
echo ""
echo "1. Open http://localhost:8096"
echo "2. Login with: $JELLYFIN_USER / $JELLYFIN_PASS"
echo "3. Check 'Leaving Soon' library in Home"
echo "4. Verify 'Test Movie' appears and is playable"
echo ""
echo "=========================================="
echo "Additional Tests"
echo "=========================================="
echo ""
echo "Test removal:"
echo "  ./test-remove.sh"
echo ""
echo "View Jellyfin logs:"
echo "  docker logs jellyfin-test -f"
echo ""
echo "Clean up:"
echo "  docker-compose -f docker-compose.test.yml down"
echo "  rm -rf test-media jellyfin-config jellyfin-cache leaving-soon-data"
echo ""
