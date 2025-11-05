# Installation Guide: Jellyfin C# Plugin for Prunarr

This guide explains how to install and configure the Jellyfin.Plugin.PrunarrBridge C# plugin.

## Overview

The C# plugin approach embeds the symlink management functionality directly into Jellyfin as a native plugin. This eliminates the need for a separate sidecar container but requires building and installing a C# .NET plugin.

## Prerequisites

- Jellyfin server (version 10.8.0 or newer)
- .NET SDK 7.0 or newer (for building the plugin)
- Access to Jellyfin's plugin directory
- Prunarr configured and running

## Building the Plugin

### 1. Install .NET SDK

If you don't have the .NET SDK installed:

```bash
# Ubuntu/Debian
wget https://dot.net/v1/dotnet-install.sh -O dotnet-install.sh
chmod +x dotnet-install.sh
./dotnet-install.sh --channel 7.0

# Or use your package manager
apt-get install dotnet-sdk-7.0
```

### 2. Build the Plugin

```bash
cd Jellyfin.Plugin.PrunarrBridge
dotnet restore
dotnet build --configuration Release
```

The compiled plugin will be located at:
```
bin/Release/net7.0/Jellyfin.Plugin.PrunarrBridge.dll
```

## Installation

### Method 1: Manual Installation (Docker)

1. **Locate your Jellyfin config directory** (usually mounted as a volume in Docker)

2. **Create the plugin directory**:
   ```bash
   mkdir -p /path/to/jellyfin/config/plugins/PrunarrBridge
   ```

3. **Copy the plugin DLL**:
   ```bash
   cp bin/Release/net7.0/Jellyfin.Plugin.PrunarrBridge.dll \
      /path/to/jellyfin/config/plugins/PrunarrBridge/
   ```

4. **Restart Jellyfin**:
   ```bash
   docker restart jellyfin
   ```

### Method 2: Manual Installation (Bare Metal)

1. **Locate Jellyfin's plugin directory**:
   - Linux: `/var/lib/jellyfin/plugins/`
   - Windows: `C:\ProgramData\Jellyfin\Server\plugins\`
   - macOS: `/var/lib/jellyfin/plugins/`

2. **Create the plugin directory**:
   ```bash
   mkdir -p /var/lib/jellyfin/plugins/PrunarrBridge
   ```

3. **Copy the plugin DLL**:
   ```bash
   cp bin/Release/net7.0/Jellyfin.Plugin.PrunarrBridge.dll \
      /var/lib/jellyfin/plugins/PrunarrBridge/
   ```

4. **Restart Jellyfin service**:
   ```bash
   sudo systemctl restart jellyfin
   ```

## Configuration

### 1. Access Plugin Settings

1. Log into Jellyfin as an administrator
2. Navigate to **Dashboard** → **Plugins**
3. Find **Prunarr Bridge** in the plugin list
4. Click on it to access settings

### 2. Configure Plugin Settings

Configure the following settings:

- **Symlink Base Path**: Directory where symlinks will be created (e.g., `/data/leaving-soon`)
  - This path must be accessible by Jellyfin
  - Jellyfin must have write permissions
  
- **Virtual Folder Name**: Name of the Virtual Folder (default: "Leaving Soon")
  - This will appear as a library in Jellyfin
  
- **Collection Type**: Type of media collection
  - `movies`: For movies only
  - `tvshows`: For TV shows only
  - `mixed`: For both (default)

- **API Key**: Secret key for authenticating Prunarr requests
  - Generate a secure random string
  - This will be used by Prunarr to authenticate

### 3. Save and Verify

1. Click **Save** to apply settings
2. Check Jellyfin logs for any errors:
   ```bash
   # Docker
   docker logs jellyfin
   
   # Systemd
   journalctl -u jellyfin -f
   ```

## Integration with Prunarr

### Configure Prunarr to Use the Plugin

Update Prunarr's configuration to point to the plugin endpoints:

**Base URL**: `http://jellyfin:8096/api/prunarr` (or your Jellyfin URL)

**API Endpoints**:
- Add items: `POST /api/prunarr/leaving-soon/add`
- Remove items: `POST /api/prunarr/leaving-soon/remove`
- Clear all: `POST /api/prunarr/leaving-soon/clear`
- Status: `GET /api/prunarr/status`

**Authentication**: Include the API key in the `X-API-Key` header

### Example API Request

```bash
curl -X POST http://jellyfin:8096/api/prunarr/leaving-soon/add \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key-here" \
  -d '{
    "items": [
      {
        "source_path": "/media/movies/Movie (2023)/Movie (2023).mkv",
        "deletion_date": "2024-12-31T00:00:00Z"
      }
    ]
  }'
```

## Volume Mapping Considerations

### Docker Setup

When using Docker, ensure proper volume mapping:

```yaml
services:
  jellyfin:
    volumes:
      # Config directory (where plugin is installed)
      - ./jellyfin-config:/config
      # Your actual media
      - /path/to/movies:/media/movies:ro
      - /path/to/tvshows:/media/tvshows:ro
      # Directory for symlinks (must be writable)
      - leaving-soon-data:/data/leaving-soon
```

**Important**: The symlink base path in the plugin configuration must match the path as Jellyfin sees it inside the container (e.g., `/data/leaving-soon`, not the host path).

## Verification

### 1. Check Plugin Status

```bash
curl http://jellyfin:8096/api/prunarr/status \
  -H "X-API-Key: your-api-key-here"
```

Expected response:
```json
{
  "version": "1.0.0",
  "symlink_base_path": "/data/leaving-soon",
  "virtual_folder_name": "Leaving Soon",
  "jellyfin_connected": true
}
```

### 2. Test Adding an Item

```bash
curl -X POST http://jellyfin:8096/api/prunarr/leaving-soon/add \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key-here" \
  -d '{
    "items": [
      {
        "source_path": "/media/movies/TestMovie/TestMovie.mkv"
      }
    ]
  }'
```

### 3. Verify in Jellyfin

1. Navigate to **Jellyfin** → **Libraries**
2. You should see the "Leaving Soon" library
3. The test movie should appear there

## Troubleshooting

### Plugin Not Loading

- Check Jellyfin logs for errors
- Verify the plugin DLL is in the correct directory
- Ensure .NET 7.0 runtime is available
- Try restarting Jellyfin

### Symlink Creation Fails

- Verify write permissions on the symlink base path
- Check that source paths exist and are accessible
- Ensure paths use the correct format (as seen by Jellyfin)

### Virtual Folder Not Created

- Check Jellyfin logs for ILibraryManager errors
- Verify the symlink base path exists
- Ensure Jellyfin has scan permissions

### API Authentication Fails

- Verify the API key matches in both plugin config and Prunarr
- Check that the `X-API-Key` header is being sent
- Try using the query parameter: `?api_key=your-key`

## Uninstallation

To remove the plugin:

1. Delete the plugin directory:
   ```bash
   rm -rf /path/to/jellyfin/config/plugins/PrunarrBridge
   ```

2. Restart Jellyfin

3. Optionally, remove the "Leaving Soon" Virtual Folder from Jellyfin's UI

## Advantages of the Plugin Approach

- **No additional container**: Runs directly in Jellyfin's process
- **Direct API access**: Uses internal ILibraryManager APIs
- **Simpler deployment**: One less service to manage
- **Native integration**: Appears as a standard Jellyfin plugin

## Disadvantages

- **Requires building**: Must compile C# code
- **Jellyfin restarts**: Updates require restarting Jellyfin
- **Debugging complexity**: Harder to debug than standalone service
- **Version coupling**: Plugin must be compatible with Jellyfin version

For a comparison with the Go sidecar approach, see `FEASIBILITY_REPORT.md`.
