# Installation Guide: Go Sidecar Service for Prunarr

This guide explains how to install and configure the Go sidecar service for managing Jellyfin's "Leaving Soon" virtual folder.

## Overview

The Go sidecar approach runs as a separate container/service alongside Jellyfin. It communicates with Jellyfin via REST API and manages symlinks in a shared volume.

## Prerequisites

- Docker and Docker Compose (recommended) OR Go 1.21+ for building from source
- Jellyfin server running and accessible
- Access to Jellyfin's API key
- Prunarr configured and running

## Quick Start with Docker Compose

### 1. Clone/Copy the Service Files

Ensure you have the `jellyfin-sidecar/` directory with all files.

### 2. Create Configuration File

Copy the example configuration:

```bash
cp jellyfin-sidecar/config.example.json jellyfin-sidecar/config.json
```

Edit `jellyfin-sidecar/config.json`:

```json
{
  "server": {
    "host": "0.0.0.0",
    "port": 8090
  },
  "jellyfin": {
    "url": "http://jellyfin:8096",
    "api_key": "YOUR_JELLYFIN_API_KEY_HERE"
  },
  "symlink": {
    "base_path": "/data/leaving-soon",
    "virtual_folder_name": "Leaving Soon",
    "collection_type": "mixed"
  },
  "security": {
    "api_key": "YOUR_PRUNARR_API_KEY_HERE"
  }
}
```

**Configuration Options:**

- `server.host`: Listen address (use `0.0.0.0` for Docker)
- `server.port`: HTTP port for the API (default: 8090)
- `jellyfin.url`: Jellyfin server URL (use container name in Docker)
- `jellyfin.api_key`: Jellyfin API key (get from Dashboard → API Keys)
- `symlink.base_path`: Directory for symlinks (must match volume mount)
- `symlink.virtual_folder_name`: Name for the Virtual Folder (default: "Leaving Soon")
- `symlink.collection_type`: `movies`, `tvshows`, or `mixed`
- `security.api_key`: API key for Prunarr to authenticate with this service

### 3. Update docker-compose.yml

Use the provided `docker-compose.yml` and update the volume paths:

```yaml
services:
  jellyfin:
    image: jellyfin/jellyfin:latest
    volumes:
      - ./jellyfin-config:/config
      - /path/to/your/movies:/media/movies:ro
      - /path/to/your/tvshows:/media/tvshows:ro
      - leaving-soon-data:/data/leaving-soon
    ports:
      - "8096:8096"

  jellyfin-sidecar:
    build: ./jellyfin-sidecar
    volumes:
      - ./jellyfin-sidecar/config.json:/etc/jellyfin-sidecar/config.json:ro
      - leaving-soon-data:/data/leaving-soon
      - /path/to/your/movies:/media/movies:ro
      - /path/to/your/tvshows:/media/tvshows:ro
    ports:
      - "8090:8090"
    depends_on:
      - jellyfin

volumes:
  leaving-soon-data:
```

**Important Volume Mappings:**

1. **Shared volume** (`leaving-soon-data`): Must be mounted to both Jellyfin and sidecar
2. **Source media**: Mount read-only to sidecar (same paths as Jellyfin sees them)
3. **Config file**: Mount to `/etc/jellyfin-sidecar/config.json`

### 4. Start the Services

```bash
docker-compose up -d
```

Check the logs:

```bash
docker-compose logs -f jellyfin-sidecar
```

You should see:
```
Starting Jellyfin Sidecar Service v1.0.0
Jellyfin URL: http://jellyfin:8096
Symlink Base Path: /data/leaving-soon
Virtual Folder Name: Leaving Soon
Starting server on 0.0.0.0:8090
```

## Building from Source

If you prefer to build and run without Docker:

### 1. Install Go

```bash
# Ubuntu/Debian
sudo apt-get install golang-1.21

# Or download from https://go.dev/dl/
```

### 2. Build the Service

```bash
cd jellyfin-sidecar
go build -o jellyfin-sidecar ./cmd/main.go
```

### 3. Create Configuration

```bash
mkdir -p /etc/jellyfin-sidecar
cp config.example.json /etc/jellyfin-sidecar/config.json
# Edit the config file with your settings
nano /etc/jellyfin-sidecar/config.json
```

### 4. Run the Service

```bash
./jellyfin-sidecar -config /etc/jellyfin-sidecar/config.json
```

### 5. (Optional) Create a systemd Service

Create `/etc/systemd/system/jellyfin-sidecar.service`:

```ini
[Unit]
Description=Jellyfin Sidecar Service
After=network.target jellyfin.service

[Service]
Type=simple
User=jellyfin
ExecStart=/usr/local/bin/jellyfin-sidecar -config /etc/jellyfin-sidecar/config.json
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable jellyfin-sidecar
sudo systemctl start jellyfin-sidecar
```

## Getting Jellyfin API Key

1. Log into Jellyfin as an administrator
2. Navigate to **Dashboard** → **API Keys**
3. Click **"+"** to create a new API key
4. Give it a name (e.g., "Prunarr Sidecar")
5. Copy the generated key and add it to your config

## Integration with Prunarr

### Configure Prunarr to Use the Sidecar

Update Prunarr's configuration to point to the sidecar endpoints:

**Base URL**: `http://jellyfin-sidecar:8090/api` (or your sidecar URL)

**API Endpoints**:
- Add items: `POST /api/leaving-soon/add`
- Remove items: `POST /api/leaving-soon/remove`
- Clear all: `POST /api/leaving-soon/clear`
- Status: `GET /api/status`
- Health: `GET /health`

**Authentication**: Include the API key (from `security.api_key` in config) in the `X-API-Key` header

### Example API Requests

**Check Status**:
```bash
curl http://localhost:8090/api/status \
  -H "X-API-Key: your-prunarr-api-key"
```

**Add Items**:
```bash
curl -X POST http://localhost:8090/api/leaving-soon/add \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-prunarr-api-key" \
  -d '{
    "items": [
      {
        "source_path": "/media/movies/Movie (2023)/Movie (2023).mkv",
        "deletion_date": "2024-12-31T00:00:00Z"
      }
    ]
  }'
```

**Remove Items**:
```bash
curl -X POST http://localhost:8090/api/leaving-soon/remove \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-prunarr-api-key" \
  -d '{
    "symlink_paths": ["/data/leaving-soon/Movie (2023)"]
  }'
```

**Clear All**:
```bash
curl -X POST http://localhost:8090/api/leaving-soon/clear \
  -H "X-API-Key: your-prunarr-api-key"
```

## Path Mapping Considerations

### Critical: Paths Must Match

The paths used by Prunarr must match how the sidecar sees them:

**Example Setup:**

Host filesystem:
```
/mnt/media/movies/Movie (2023)/Movie (2023).mkv
```

Docker volumes:
```yaml
jellyfin-sidecar:
  volumes:
    - /mnt/media/movies:/media/movies:ro
```

Prunarr should send:
```
/media/movies/Movie (2023)/Movie (2023).mkv
```

NOT:
```
/mnt/media/movies/Movie (2023)/Movie (2023).mkv  # Host path (wrong!)
```

### Handling Path Translations

If Prunarr runs with different path mappings, you may need to:

1. **Configure Prunarr** to use the sidecar's path perspective
2. **Add path translation** in Prunarr's configuration
3. **Ensure consistent volume mounts** across all containers

## Verification

### 1. Check Service Health

```bash
curl http://localhost:8090/health
```

Expected: `{"status":"healthy"}`

### 2. Check Jellyfin Connection

```bash
curl http://localhost:8090/api/status \
  -H "X-API-Key: your-api-key"
```

Expected:
```json
{
  "version": "1.0.0",
  "symlink_base_path": "/data/leaving-soon",
  "virtual_folder_name": "Leaving Soon",
  "jellyfin_connected": true
}
```

If `jellyfin_connected` is `false`:
- Check Jellyfin URL in config
- Verify Jellyfin API key is correct
- Ensure network connectivity between containers

### 3. Test Adding an Item

```bash
curl -X POST http://localhost:8090/api/leaving-soon/add \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{
    "items": [
      {
        "source_path": "/media/movies/TestMovie/TestMovie.mkv"
      }
    ]
  }'
```

Expected:
```json
{
  "success": true,
  "created_symlinks": ["/data/leaving-soon/TestMovie"],
  "errors": []
}
```

### 4. Verify in Jellyfin

1. Navigate to **Jellyfin** → **Libraries**
2. Look for the "Leaving Soon" library
3. The test movie should appear there after a library scan

## Troubleshooting

### Service Won't Start

**Check logs**:
```bash
docker-compose logs jellyfin-sidecar
```

**Common issues**:
- Config file not found or invalid JSON
- Port 8090 already in use (change in config and docker-compose)
- Volume mount issues

### Cannot Connect to Jellyfin

**Error**: `"jellyfin_connected": false`

**Solutions**:
- Verify Jellyfin URL (use container name in Docker: `http://jellyfin:8096`)
- Check Jellyfin API key is correct
- Ensure Jellyfin is running: `docker-compose ps`
- Check network: `docker-compose exec jellyfin-sidecar ping jellyfin`

### Symlink Creation Fails

**Error**: "permission denied" or "operation not permitted"

**Solutions**:
- Check volume permissions
- Ensure the sidecar has write access to `/data/leaving-soon`
- Verify source paths are readable
- Check that paths match between Prunarr and sidecar

### Virtual Folder Not Created

**Check**:
- Jellyfin API key has admin permissions
- Symlink base path exists and is accessible by Jellyfin
- Check Jellyfin logs for Virtual Folder creation errors

### Path Not Found Errors

**Error**: "no such file or directory"

**Cause**: Path mismatch between Prunarr and sidecar

**Solution**: Ensure volume mounts match and Prunarr sends paths from sidecar's perspective

## Updating the Service

### Docker:
```bash
cd /path/to/project
docker-compose build jellyfin-sidecar
docker-compose up -d jellyfin-sidecar
```

### From Source:
```bash
cd jellyfin-sidecar
git pull  # or copy updated files
go build -o jellyfin-sidecar ./cmd/main.go
sudo systemctl restart jellyfin-sidecar
```

## Uninstallation

### Docker:
```bash
docker-compose down
docker volume rm <project>_leaving-soon-data  # Optional: removes symlink volume
```

### From Source:
```bash
sudo systemctl stop jellyfin-sidecar
sudo systemctl disable jellyfin-sidecar
sudo rm /etc/systemd/system/jellyfin-sidecar.service
sudo rm /usr/local/bin/jellyfin-sidecar
sudo rm -rf /etc/jellyfin-sidecar
```

Manually remove "Leaving Soon" Virtual Folder from Jellyfin UI if desired.

## Advantages of the Sidecar Approach

- **Independent deployment**: Runs separately from Jellyfin
- **Easy debugging**: Separate logs and simpler troubleshooting
- **No Jellyfin restarts**: Update without restarting Jellyfin
- **Language flexibility**: Written in Go (faster, smaller footprint)
- **Containerized**: Easy Docker deployment

## Disadvantages

- **Additional container**: One more service to manage
- **Network overhead**: REST API calls between containers
- **Volume complexity**: Requires careful volume mapping
- **Path translation**: Must ensure path consistency

For a detailed comparison with the C# plugin approach, see `FEASIBILITY_REPORT.md`.
