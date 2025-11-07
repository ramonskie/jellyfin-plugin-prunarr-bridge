# Quick Start: Testing the Plugin

This is a simplified test workflow to verify the Jellyfin plugin works correctly.

## Prerequisites

- Docker and Docker Compose
- mise (or .NET 9.0 SDK)
- jq (for JSON parsing)

## Quick Test (3 Steps)

### Step 1: Build and Start
```bash
./quick-test.sh
```

This will:
- Build the plugin
- Create a test movie file
- Start Jellyfin with the plugin installed

### Step 2: Initial Jellyfin Setup

1. Open http://localhost:8096
2. Complete the initial setup wizard:
   - Select your language
   - Create a user account:
     - **Username**: `test`
     - **Password**: `test`
   - **IMPORTANT**: When adding libraries, create a "Leaving Soon" library:
     - Click "Add Media Library"
     - Name: `Leaving Soon`
     - Content type: `Movies` (or `Mixed`)
     - Folder paths: Click `+` and enter `/data/leaving-soon`
     - Click OK
   - Finish setup

**Note**: The plugin **no longer** auto-creates libraries. You must manually create the "Leaving Soon" library during setup, or Prunarr must create it via the Jellyfin API.

**Note**: The test scripts default to username `test` and password `test`. You can override these:
```bash
export JELLYFIN_USER=myuser
export JELLYFIN_PASS=mypass
```

### Step 3: Test the API
```bash
./test-api.sh
```

This will:
- Authenticate with Jellyfin
- Test the plugin status endpoint
- Add a test movie symlink to "Leaving Soon"
- Verify symlink creation on the filesystem
- List all symlinks
- **Note**: You may need to manually trigger a library scan in Jellyfin for the movie to appear
- Guide you through manual verification in the UI

### Optional: Test Removal
```bash
./test-remove.sh
```

This tests:
- Removing individual movie symlinks
- Listing symlinks to verify removal

## What You Should See

### ✅ Success Indicators

1. **In Terminal**:
   ```
   ✓ Authenticated successfully
   ✓ Status endpoint working
   ✓ Movie added successfully
   ✓ Symlink created on host
   ✓ Symlink exists inside container
   ✓ Movie(s) found in Jellyfin library!
   ```

2. **In Jellyfin UI** (http://localhost:8096):
   - Login with `test` / `test`
   - "Leaving Soon" library appears in Home (if you created it during setup)
   - "Test Movie" appears in that library after scanning
   - Movie metadata is loaded (it's a short test video)
   - **Note**: If the movie doesn't appear, manually trigger a scan:
     - Dashboard → Libraries → Scan All Libraries

### ❌ If Something Fails

**Authentication Issues**:
```bash
# If you created a different user during setup
export JELLYFIN_USER=yourusername
export JELLYFIN_PASS=yourpassword
./test-api.sh
```

**Plugin Not Loading**:
```bash
# View Jellyfin logs
docker logs jellyfin-test -f

# Check if plugin is loaded
curl http://localhost:8096/api/prunarr/status

# Check plugin installation
docker exec jellyfin-test ls -lah /config/plugins/PrunarrBridge/
```

**Symlink Issues**:
```bash
# Check symlinks inside container
docker exec jellyfin-test ls -lah /data/leaving-soon/

# Check host symlinks
ls -lah leaving-soon-data/
```

## Configuration

**The plugin requires NO configuration!** (v2.0+)

The plugin is completely stateless. All paths (target directories) are provided via API requests. Prunarr is in complete control of where symlinks are created.

**Removed in v2.0** (breaking changes):
- ~~Symlink Base Path~~ - Target directory now provided in each API request
- ~~Virtual Folder Name~~ - Libraries must be created manually or by Prunarr
- ~~Auto Create Virtual Folder~~ - Plugin no longer manages libraries

The plugin uses Jellyfin's built-in authentication. Any authenticated Jellyfin user can call the API endpoints using their API token.

## Cleanup

When done:
```bash
docker-compose -f docker-compose.test.yml down
rm -rf test-media jellyfin-config jellyfin-cache leaving-soon-data
```

## Full Documentation

For detailed troubleshooting and advanced testing, see [TEST_GUIDE.md](TEST_GUIDE.md).
