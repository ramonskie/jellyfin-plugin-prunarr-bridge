# Jellyfin Prunarr Bridge Plugin - API Documentation

## Overview

The Jellyfin Prunarr Bridge Plugin provides a minimal, focused API for managing symlinks. The plugin is **completely stateless and configuration-free** - all paths are provided via API requests.

## Design Philosophy

**The plugin does ONE thing well: Manage symlinks as instructed**

- ✅ Create symlinks at specified locations
- ✅ Remove symlinks
- ✅ List symlinks in specified directories
- ✅ Health check

**The plugin does NOT:**
- ❌ Store any configuration
- ❌ Remember any paths
- ❌ Create/manage Jellyfin libraries
- ❌ Trigger library scans
- ❌ Manage media metadata
- ❌ Handle business logic

**Prunarr is in complete control** - it knows all paths, manages all state, and orchestrates all operations.

## Authentication

All endpoints except `/status` require authentication using Jellyfin's built-in authentication system.

### Using API Keys (Recommended for Prunarr)

```bash
# Create API key in Jellyfin: Dashboard → API Keys
curl -H "X-Emby-Token: your-api-key-here" http://localhost:8096/api/prunarr/...
```

### Using Session Tokens (For Testing)

```bash
# 1. Authenticate with username/password
curl -X POST http://localhost:8096/Users/AuthenticateByName \
  -H "Content-Type: application/json" \
  -H "X-Emby-Authorization: MediaBrowser Client=\"YourApp\", Device=\"YourDevice\", DeviceId=\"device-id\", Version=\"1.0.0\"" \
  -d '{"Username":"your-user","Pw":"your-pass"}'

# 2. Use returned AccessToken
curl -H "X-Emby-Token: session-token" http://localhost:8096/api/prunarr/...
```

## Endpoints

### GET /api/prunarr/status

Get plugin version.

**Authentication:** None required (public endpoint)

**Response:**
```json
{
  "Version": "2.0.0.0"
}
```

**Example:**
```bash
curl http://localhost:8096/api/prunarr/status
```

---

### POST /api/prunarr/symlinks/add

Create symlinks for media files.

**Authentication:** Required

**Request Body:**
```json
{
  "items": [
    {
      "sourcePath": "/media/movies/Movie Name (2024)/movie.mkv",
      "targetDirectory": "/data/leaving-soon"
    }
  ]
}
```

**Response:**
```json
{
  "Success": true,
  "CreatedSymlinks": [
    "/data/leaving-soon/movie.mkv"
  ],
  "Errors": []
}
```

**Response Codes:**
- `200 OK` - Symlinks created successfully
- `400 Bad Request` - Invalid request (no items provided)
- `401 Unauthorized` - Authentication required

**Example:**
```bash
curl -X POST http://localhost:8096/api/prunarr/symlinks/add \
  -H "Content-Type: application/json" \
  -H "X-Emby-Token: your-token" \
  -d '{
    "items": [
      {
        "sourcePath": "/media/movies/Movie1/movie1.mkv",
        "targetDirectory": "/data/leaving-soon"
      },
      {
        "sourcePath": "/media/movies/Movie2/movie2.mkv",
        "targetDirectory": "/data/leaving-soon-tv"
      }
    ]
  }'
```

**Notes:**
- **targetDirectory** is required for each item - Prunarr specifies where to create the symlink
- If symlink already exists, it will be replaced
- Target directory is created automatically if it doesn't exist
- Each item is processed independently - partial success is possible
- Check `Errors` array for any failures

---

### POST /api/prunarr/symlinks/remove

Remove symlinks.

**Authentication:** Required

**Request Body:**
```json
{
  "symlinkPaths": [
    "/data/leaving-soon/movie.mkv"
  ]
}
```

**Response:**
```json
{
  "Success": true,
  "RemovedSymlinks": [
    "/data/leaving-soon/movie.mkv"
  ],
  "Errors": []
}
```

**Response Codes:**
- `200 OK` - Symlinks removed successfully
- `400 Bad Request` - Invalid request (no paths provided)
- `401 Unauthorized` - Authentication required

**Example:**
```bash
curl -X POST http://localhost:8096/api/prunarr/symlinks/remove \
  -H "Content-Type: application/json" \
  -H "X-Emby-Token: your-token" \
  -d '{
    "symlinkPaths": [
      "/data/leaving-soon/movie1.mkv",
      "/data/leaving-soon/movie2.mkv"
    ]
  }'
```

**Notes:**
- If symlink doesn't exist, operation continues without error
- Each path is processed independently
- Check `Errors` array for any failures

---

### GET /api/prunarr/symlinks/list

List all symlinks in a specified directory.

**Authentication:** Required

**Query Parameters:**
- `directory` (required) - The directory path to list symlinks from

**Response:**
```json
{
  "Symlinks": [
    {
      "Path": "/data/leaving-soon/movie.mkv",
      "Target": "/media/movies/Movie Name (2024)/movie.mkv",
      "Name": "movie.mkv"
    }
  ],
  "Count": 1
}
```

**Response Codes:**
- `200 OK` - Symlinks listed successfully
- `400 Bad Request` - Directory parameter missing
- `401 Unauthorized` - Authentication required
- `500 Internal Server Error` - Failed to list symlinks

**Example:**
```bash
curl "http://localhost:8096/api/prunarr/symlinks/list?directory=/data/leaving-soon" \
  -H "X-Emby-Token: your-token"
```

**Notes:**
- **directory** parameter is required - Prunarr specifies which directory to list
- Only returns actual symlinks (not regular files)
- Returns empty array if directory doesn't exist or is empty
- Useful for Prunarr to verify current state

---

## Configuration

**The plugin requires NO configuration!** It is completely stateless.

All paths are provided via API requests. Prunarr is in complete control of where symlinks are created.

---

## Prunarr Integration Guide

### Recommended Workflow

```python
# 1. Create symlink via plugin
response = requests.post(
    "http://jellyfin:8096/api/prunarr/symlinks/add",
    headers={"X-Emby-Token": api_key},
    json={"items": [{
        "sourcePath": movie_path,
        "targetDirectory": "/data/leaving-soon"
    }]}
)

if not response.json()["Success"]:
    handle_error()

# 2. Ensure library exists (Prunarr's responsibility)
if not library_exists("Leaving Soon"):
    create_library("Leaving Soon", "/data/leaving-soon")

# 3. Trigger library scan (Prunarr's responsibility)
scan_library("Leaving Soon")
```

### Creating the Library (One-Time Setup)

Prunarr should create the "Leaving Soon" library on first use:

```bash
# Via Jellyfin API
POST /Library/VirtualFolders
{
  "name": "Leaving Soon",
  "collectionType": "movies",
  "paths": ["/data/leaving-soon"]
}
```

### Triggering Library Scan

After adding/removing symlinks, Prunarr should trigger a scan:

```bash
# Scan specific library
POST /Library/Refresh
```

---

## Migration from v1.0.2

### Breaking Changes

**Removed Endpoints:**
- `POST /api/prunarr/leaving-soon/clear` - Use `/symlinks/remove` for each item instead

**Renamed Endpoints:**
- `POST /api/prunarr/leaving-soon/add` → `POST /api/prunarr/symlinks/add`
- `POST /api/prunarr/leaving-soon/remove` → `POST /api/prunarr/symlinks/remove`

**Removed Configuration:**
- `AutoCreateVirtualFolder` - Prunarr must create library
- `VirtualFolderName` - Prunarr manages library name
- `SymlinkBasePath` - Target directory now provided in each API request

**Removed Behavior:**
- Plugin no longer creates or manages Jellyfin libraries
- Plugin no longer triggers library scans
- Prunarr is now responsible for library lifecycle

### Migration Steps

1. Update API endpoint URLs in Prunarr
2. Implement library management in Prunarr
3. Implement library scanning in Prunarr after batch operations
4. Remove old configuration options

---

## Error Handling

All endpoints return errors in a consistent format:

**Success with Partial Errors:**
```json
{
  "Success": true,
  "CreatedSymlinks": ["/data/leaving-soon/movie1.mkv"],
  "Errors": [
    "/media/movies/movie2.mkv: File not found"
  ]
}
```

**Complete Failure:**
```json
{
  "error": "Plugin configuration not available"
}
```

**Best Practices:**
- Always check `Success` field
- Check `Errors` array for partial failures
- Log all errors for debugging
- Implement retry logic for transient failures

---

## Testing

See test scripts for examples:
- `tests/test-api.sh` - Tests add and list operations
- `tests/test-remove.sh` - Tests remove and list operations

Run tests:
```bash
# Full test suite
./tests/test-api.sh
./tests/test-remove.sh
```

---

## Technical Details

### Symlink Management

- Uses .NET's `File.CreateSymbolicLink()` (requires .NET 6+)
- Symlinks are created with same filename as source
- Existing symlinks are replaced automatically
- Only actual symlinks are listed (regular files ignored)

### Thread Safety

- All operations are thread-safe
- Multiple concurrent requests are handled correctly
- File operations are atomic where possible

### Platform Support

- **Linux:** Full support ✅
- **Windows:** Requires admin privileges for symlink creation
- **macOS:** Full support ✅

---

## Support

For issues or questions:
- Check logs: `docker logs jellyfin-test -f`
- Verify plugin is loaded: `GET /api/prunarr/status`
- Check authentication: Ensure valid API key or session token
- Review filesystem permissions: Plugin runs as Jellyfin user
