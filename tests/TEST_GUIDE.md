# Testing Guide: Jellyfin Plugin for "Leaving Soon" Library

This guide will walk you through testing the Jellyfin plugin to verify it can actually move movies into a "Leaving Soon" library.

## Prerequisites

Before running the test:

1. **Docker and Docker Compose** installed
2. **.NET SDK 7.0+** installed (for building the plugin)
3. **No other Jellyfin instance** running on port 8096
4. **Enough disk space** for Jellyfin config and cache (~500MB)

Check prerequisites:
```bash
docker --version
docker-compose --version
dotnet --version
```

## Quick Test (Automated)

We've created an automated test script that will:
- Build the plugin
- Create test media files
- Start Jellyfin with the plugin
- Test the API endpoints
- Verify symlink creation

### Run the Test

```bash
./test-plugin.sh
```

The script will pause for manual steps when needed. Follow the on-screen instructions.

## Manual Test Steps

If you prefer to test manually, follow these steps:

### Step 1: Build the Plugin

```bash
cd Jellyfin.Plugin.PrunarrBridge
dotnet build --configuration Release
cd ..
```

### Step 2: Install the Plugin

```bash
# Create plugin directory
mkdir -p jellyfin-config/plugins/PrunarrBridge

# Find and copy the built DLL
cp Jellyfin.Plugin.PrunarrBridge/bin/Release/net*/Jellyfin.Plugin.PrunarrBridge.dll \
   jellyfin-config/plugins/PrunarrBridge/
```

### Step 3: Create Test Media

```bash
# Create a test movie
mkdir -p test-media/movies/"Test Movie (2024)"
dd if=/dev/zero of="test-media/movies/Test Movie (2024)/Test Movie (2024).mkv" bs=1M count=1
```

### Step 4: Start Jellyfin

```bash
docker-compose -f docker-compose.test.yml up -d
```

Wait for Jellyfin to start (30-60 seconds), then check:
```bash
curl http://localhost:8096/health
```

### Step 5: Complete Jellyfin Setup

1. Open http://localhost:8096 in your browser
2. Complete the initial setup wizard:
   - Select language
   - Create admin user
   - Skip library setup for now (we'll add "Leaving Soon" via plugin)
   - Finish setup

3. Login as administrator

### Step 6: Configure the Plugin

1. Go to **Dashboard** → **Plugins**
2. Find **Prunarr Bridge** in the list
3. Click on it to configure
4. Set the following:
   - **Symlink Base Path**: `/data/leaving-soon`
   - **Virtual Folder Name**: `Leaving Soon`
   - **Auto Create Virtual Folder**: `true`
5. Click **Save**
6. Restart Jellyfin if needed:
   ```bash
   docker restart jellyfin-test
   ```

### Step 7: Test the Status Endpoint

```bash
curl http://localhost:8096/api/prunarr/status
```

Expected output:
```json
{
  "version": "1.0.0.0",
  "symlinkBasePath": "/data/leaving-soon",
  "virtualFolderName": "Leaving Soon",
  "autoCreateVirtualFolder": true
}
```

### Step 8: Add a Movie to "Leaving Soon"

```bash
curl -X POST http://localhost:8096/api/prunarr/leaving-soon/add \
  -H "Content-Type: application/json" \
  -d '{
    "items": [
      {
        "sourcePath": "/media/movies/Test Movie (2024)/Test Movie (2024).mkv"
      }
    ]
  }'
```

Expected output:
```json
{
  "success": true,
  "createdSymlinks": [
    "/data/leaving-soon/Test Movie (2024).mkv"
  ],
  "errors": []
}
```

### Step 9: Verify Symlink Creation

Check if the symlink was created on the host:
```bash
ls -lah leaving-soon-data/
```

Check inside the container:
```bash
docker exec jellyfin-test ls -lah /data/leaving-soon/
```

You should see a symlink pointing to the original movie file.

### Step 10: Verify in Jellyfin UI

1. Go to Jellyfin home page
2. Check if **"Leaving Soon"** library exists in the libraries list
3. Click on the library
4. Trigger a library scan if the movie doesn't appear:
   - Dashboard → Libraries → Scan All Libraries
5. The test movie should appear in the "Leaving Soon" library

### Step 11: Test Removal

Remove the movie from "Leaving Soon":
```bash
curl -X POST http://localhost:8096/api/prunarr/leaving-soon/remove \
  -H "Content-Type: application/json" \
  -d '{
    "symlinkPaths": [
      "/data/leaving-soon/Test Movie (2024).mkv"
    ]
  }'
```

Verify the symlink is removed:
```bash
ls -lah leaving-soon-data/
```

### Step 12: Test Clear All

Add the movie back, then test clearing all:
```bash
# Add again
curl -X POST http://localhost:8096/api/prunarr/leaving-soon/add \
  -H "Content-Type: application/json" \
  -d '{
    "items": [
      {
        "sourcePath": "/media/movies/Test Movie (2024)/Test Movie (2024).mkv"
      }
    ]
  }'

# Clear all
curl -X POST http://localhost:8096/api/prunarr/leaving-soon/clear
```

## Troubleshooting

### Plugin Not Loading

**Symptom**: Plugin doesn't appear in Jellyfin's plugin list

**Solutions**:
1. Check if DLL is in correct location:
   ```bash
   ls -lah jellyfin-config/plugins/PrunarrBridge/
   ```
2. Check Jellyfin logs:
   ```bash
   docker logs jellyfin-test | grep -i plugin
   docker logs jellyfin-test | grep -i prunarr
   ```
3. Restart Jellyfin:
   ```bash
   docker restart jellyfin-test
   ```
4. Check .NET version compatibility - plugin requires .NET 7.0+

### Symlink Not Created

**Symptom**: API returns success but no symlink in directory

**Solutions**:
1. Check if source file exists in container:
   ```bash
   docker exec jellyfin-test ls -lah /media/movies/Test\ Movie\ \(2024\)/
   ```
2. Check permissions on leaving-soon directory:
   ```bash
   docker exec jellyfin-test ls -lah /data/
   ```
3. Check Jellyfin logs for errors:
   ```bash
   docker logs jellyfin-test -f
   ```
4. Verify the source path matches exactly (case-sensitive)

### Movie Not Appearing in Library

**Symptom**: Symlink exists but movie doesn't show in Jellyfin

**Solutions**:
1. Manually trigger library scan:
   - Dashboard → Libraries → Scan All Libraries
2. Check if "Leaving Soon" library was created:
   - Dashboard → Libraries
3. Check library paths:
   - Dashboard → Libraries → Leaving Soon → Manage Library
   - Verify `/data/leaving-soon` is in the paths
4. Check file permissions:
   ```bash
   docker exec jellyfin-test ls -lah /data/leaving-soon/
   ```

### API Endpoints Return 404

**Symptom**: API calls return 404 Not Found

**Solutions**:
1. Verify plugin is loaded:
   - Check Dashboard → Plugins
2. Check the correct URL format:
   - ✅ Correct: `http://localhost:8096/api/prunarr/status`
   - ❌ Wrong: `http://localhost:8096/prunarr/status`
3. Restart Jellyfin to reload plugin routes:
   ```bash
   docker restart jellyfin-test
   ```

### Permission Denied Errors

**Symptom**: Symlink creation fails with permission errors

**Solutions**:
1. Check volume ownership:
   ```bash
   ls -lahn leaving-soon-data/
   ```
2. Fix ownership if needed:
   ```bash
   sudo chown -R 1000:1000 leaving-soon-data/
   ```
3. Restart Jellyfin:
   ```bash
   docker restart jellyfin-test
   ```

## Viewing Logs

### Plugin Logs

```bash
# View all logs
docker logs jellyfin-test -f

# Filter for plugin logs
docker logs jellyfin-test 2>&1 | grep -i prunarr

# Filter for errors
docker logs jellyfin-test 2>&1 | grep -i error
```

### Inside Container Inspection

```bash
# Shell into container
docker exec -it jellyfin-test /bin/bash

# Inside container:
ls -lah /config/plugins/
ls -lah /data/leaving-soon/
ls -lah /media/movies/
```

## What to Look For in Successful Test

### ✅ Success Indicators

1. **Plugin loads**: Appears in Dashboard → Plugins
2. **API responds**: Status endpoint returns plugin info
3. **Symlinks created**: Files appear in `leaving-soon-data/`
4. **Library created**: "Leaving Soon" appears in libraries list
5. **Movies visible**: Test movie appears in "Leaving Soon" library
6. **Metadata works**: Movie shows poster, title, etc. (inherited from original)
7. **Playback works**: Can play the movie from "Leaving Soon" library

### ❌ Failure Indicators

1. Plugin not in plugins list
2. API endpoints return 404
3. No symlinks created
4. Library not created automatically
5. Movies not visible in library
6. Permission denied errors in logs
7. Symlinks created but not readable

## Testing with Real Media

Once the test with dummy files works, test with real media:

1. Update docker-compose.test.yml to mount your real media:
   ```yaml
   volumes:
     - /path/to/your/real/movies:/media/movies:ro
   ```

2. Restart Jellyfin:
   ```bash
   docker-compose -f docker-compose.test.yml down
   docker-compose -f docker-compose.test.yml up -d
   ```

3. Add a real movie:
   ```bash
   curl -X POST http://localhost:8096/api/prunarr/leaving-soon/add \
     -H "Content-Type: application/json" \
     -d '{
       "items": [
         {
           "sourcePath": "/media/movies/Your Movie (2024)/Your Movie (2024).mkv"
         }
       ]
     }'
   ```

4. Verify in Jellyfin UI that:
   - Movie appears with correct metadata
   - Poster/artwork is shown
   - Movie is playable
   - Movie info (cast, plot, etc.) is available

## Cleanup

When done testing:

```bash
# Stop containers
docker-compose -f docker-compose.test.yml down

# Remove test data (optional)
rm -rf test-media jellyfin-config jellyfin-cache leaving-soon-data

# Remove built plugin (optional)
rm -rf Jellyfin.Plugin.PrunarrBridge/bin Jellyfin.Plugin.PrunarrBridge/obj
```

## Next Steps

After successful testing:

1. **Document Results**: Note any issues or successes
2. **Test Edge Cases**:
   - Movies with special characters in filenames
   - Movies in subdirectories
   - Large files
   - Different video formats
3. **Performance Testing**:
   - Add multiple movies at once
   - Test library scan time
   - Test removal performance
4. **Integration Testing**:
   - Test with actual Prunarr instance
   - Test automatic cleanup
   - Test date-based expiration

## Expected Behavior

When everything works correctly:

1. **Adding a movie**:
   - Symlink is created in `/data/leaving-soon/`
   - Library scan is triggered automatically (if configured)
   - Movie appears in "Leaving Soon" library within seconds
   - Movie has all metadata from original

2. **Removing a movie**:
   - Symlink is deleted
   - Movie disappears from "Leaving Soon" library (after library scan)
   - Original movie remains untouched

3. **Clearing all**:
   - All symlinks are removed
   - "Leaving Soon" library becomes empty
   - Original movies remain untouched

## Support

If you encounter issues:

1. Check this troubleshooting guide
2. Review Jellyfin logs: `docker logs jellyfin-test`
3. Check plugin configuration in Jellyfin UI
4. Verify Docker volume mounts are correct
5. Test API endpoints with curl commands above
