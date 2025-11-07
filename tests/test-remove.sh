#!/bin/bash
# Test removing movies from Leaving Soon
set -e

JELLYFIN_URL="http://localhost:8096"
JELLYFIN_USER="${JELLYFIN_USER:-test}"
JELLYFIN_PASS="${JELLYFIN_PASS:-test}"

echo "=========================================="
echo "Testing Movie Removal"
echo "=========================================="
echo ""

# Authenticate
echo "Authenticating as user: $JELLYFIN_USER"
AUTH_RESPONSE=$(curl -s -X POST "$JELLYFIN_URL/Users/AuthenticateByName" \
    -H "Content-Type: application/json" \
    -H "X-Emby-Authorization: MediaBrowser Client=\"TestScript\", Device=\"TestDevice\", DeviceId=\"test-device-001\", Version=\"1.0.0\"" \
    -d "{\"Username\":\"$JELLYFIN_USER\",\"Pw\":\"$JELLYFIN_PASS\"}")

TOKEN=$(echo "$AUTH_RESPONSE" | jq -r '.AccessToken')
USER_ID=$(echo "$AUTH_RESPONSE" | jq -r '.User.Id')

if [ "$TOKEN" = "null" ] || [ -z "$TOKEN" ]; then
    echo "✗ Authentication failed!"
    exit 1
fi

echo "✓ Authenticated successfully"
echo ""

echo "Test: Removing test movie (deleting symlink)..."
echo "Request: POST $JELLYFIN_URL/api/oxicleanarr/symlinks/remove"
echo ""
REMOVE_RESPONSE=$(curl -s -X POST "$JELLYFIN_URL/api/oxicleanarr/symlinks/remove" \
    -H "Content-Type: application/json" \
    -H "X-Emby-Token: $TOKEN" \
    -d '{
        "symlinkPaths": [
            "/data/leaving-soon/Test Movie (2024).mkv"
        ]
    }')
echo "Response:"
echo "$REMOVE_RESPONSE" | jq . 2>/dev/null || echo "$REMOVE_RESPONSE"
echo ""

if echo "$REMOVE_RESPONSE" | grep -qi "success"; then
    echo "✓ Removal request successful"
else
    echo "✗ Removal failed"
    exit 1
fi

echo ""
echo "Verifying symlink is removed..."
if [ -L "leaving-soon-data/Test Movie (2024).mkv" ]; then
    echo "✗ Symlink still exists on host"
    ls -lah leaving-soon-data/
else
    echo "✓ Symlink removed from host"
fi

echo ""
echo "=========================================="
echo "Test: List Symlinks"
echo "=========================================="
echo ""

# Add the movie back first
echo "Adding movie back..."
curl -s -X POST "$JELLYFIN_URL/api/oxicleanarr/symlinks/add" \
    -H "Content-Type: application/json" \
    -H "X-Emby-Token: $TOKEN" \
    -d '{
        "items": [
            {
                "sourcePath": "/media/movies/Test Movie (2024)/Test Movie (2024).mkv",
                "targetDirectory": "/data/leaving-soon"
            }
        ]
    }' > /dev/null
echo "✓ Movie added back"

echo ""
echo "Listing all symlinks..."
LIST_RESPONSE=$(curl -s "$JELLYFIN_URL/api/oxicleanarr/symlinks/list?directory=/data/leaving-soon" -H "X-Emby-Token: $TOKEN")
echo "Response:"
echo "$LIST_RESPONSE" | jq . 2>/dev/null || echo "$LIST_RESPONSE"
echo ""

SYMLINK_COUNT=$(echo "$LIST_RESPONSE" | jq -r '.Count' 2>/dev/null || echo "0")
if [ "$SYMLINK_COUNT" -gt 0 ]; then
    echo "✓ Found $SYMLINK_COUNT symlink(s)"
else
    echo "✗ No symlinks found"
fi

echo ""
echo "=========================================="
echo "All Tests Complete!"
echo "=========================================="
echo ""
