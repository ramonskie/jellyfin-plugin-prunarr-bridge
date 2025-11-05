# Jellyfin Integration POC for Prunarr

This repository contains proof-of-concept implementations for integrating Prunarr with Jellyfin to manage "Leaving Soon" media visibility through symlinks and Virtual Folders.

## Problem

Prunarr currently requires direct filesystem access to:
1. Create symlinks to media files that are approaching deletion
2. Manage Jellyfin's Virtual Folders to display "Leaving Soon" content

This creates deployment complexity with Docker volume mappings and path translation issues.

## Solution

Move symlink and Virtual Folder management into Jellyfin's filesystem context, exposing HTTP APIs that Prunarr can call.

## Two Approaches Implemented

### 1. C# Server Plugin (Jellyfin.Plugin.PrunarrBridge)

A native Jellyfin plugin that runs inside Jellyfin's process.

**Location**: `Jellyfin.Plugin.PrunarrBridge/`

**Key Features**:
- Native Jellyfin plugin using internal APIs
- Direct access to `ILibraryManager` for Virtual Folders
- Exposes REST endpoints on Jellyfin's port
- No additional containers needed

**Installation**: See [INSTALLATION.md](INSTALLATION.md)

### 2. Go Sidecar Service (Recommended)

An independent Go service that communicates with Jellyfin via REST API.

**Location**: `jellyfin-sidecar/`

**Key Features**:
- Standalone Docker container
- Uses Jellyfin's public REST API
- Independent deployment and updates
- Easy monitoring and debugging

**Installation**: See [INSTALLATION_SIDECAR.md](INSTALLATION_SIDECAR.md)

## Quick Start (Go Sidecar - Recommended)

```bash
# 1. Copy example configuration
cp jellyfin-sidecar/config.example.json jellyfin-sidecar/config.json

# 2. Edit configuration with your Jellyfin API key
nano jellyfin-sidecar/config.json

# 3. Update docker-compose.yml with your media paths
nano docker-compose.yml

# 4. Start services
docker-compose up -d

# 5. Test the service
curl http://localhost:8090/health
```

## API Endpoints

Both implementations expose the same REST API:

### Add Items to "Leaving Soon"
```bash
POST /api/leaving-soon/add
Content-Type: application/json
X-API-Key: your-api-key

{
  "items": [
    {
      "source_path": "/media/movies/Movie (2023)/Movie (2023).mkv",
      "deletion_date": "2024-12-31T00:00:00Z"
    }
  ]
}
```

### Remove Items
```bash
POST /api/leaving-soon/remove
Content-Type: application/json
X-API-Key: your-api-key

{
  "symlink_paths": ["/data/leaving-soon/Movie (2023)"]
}
```

### Clear All Items
```bash
POST /api/leaving-soon/clear
X-API-Key: your-api-key
```

### Get Status
```bash
GET /api/status
X-API-Key: your-api-key
```

## Documentation

- **[FEASIBILITY_REPORT.md](FEASIBILITY_REPORT.md)** - Comprehensive comparison and recommendation
- **[INSTALLATION.md](INSTALLATION.md)** - C# plugin installation guide
- **[INSTALLATION_SIDECAR.md](INSTALLATION_SIDECAR.md)** - Go sidecar installation guide

## Recommendation

**Use the Go Sidecar Service** for most deployments.

The feasibility report recommends the Go sidecar approach because it:
- ✅ Easier to deploy (Docker-based)
- ✅ Easier to maintain (independent updates)
- ✅ Easier to debug (separate logs)
- ✅ Safer (isolated failure domain)
- ✅ More future-proof (uses stable public APIs)

See [FEASIBILITY_REPORT.md](FEASIBILITY_REPORT.md) for detailed analysis.

## Project Structure

```
.
├── Jellyfin.Plugin.PrunarrBridge/    # C# Plugin
│   ├── Api/
│   │   └── PrunarrController.cs      # REST API endpoints
│   ├── Configuration/
│   │   └── PluginConfiguration.cs    # Plugin settings
│   ├── Services/
│   │   ├── SymlinkManager.cs         # Symlink operations
│   │   └── VirtualFolderManager.cs   # Virtual Folder management
│   ├── Plugin.cs                     # Plugin entry point
│   └── PluginServiceRegistrator.cs   # Dependency injection
│
├── jellyfin-sidecar/                 # Go Sidecar Service
│   ├── cmd/
│   │   └── main.go                   # Entry point
│   ├── internal/
│   │   ├── api/
│   │   │   └── server.go             # HTTP API server
│   │   ├── config/
│   │   │   └── config.go             # Configuration
│   │   ├── jellyfin/
│   │   │   └── client.go             # Jellyfin API client
│   │   └── symlink/
│   │       └── manager.go            # Symlink operations
│   ├── Dockerfile                    # Container image
│   ├── config.example.json           # Example configuration
│   └── go.mod                        # Go module definition
│
├── docker-compose.yml                # Example deployment
├── FEASIBILITY_REPORT.md             # Comparative analysis
├── INSTALLATION.md                   # C# plugin guide
├── INSTALLATION_SIDECAR.md           # Go sidecar guide
└── README.md                         # This file
```

## Development Status

- ✅ C# Plugin - Complete POC
- ✅ Go Sidecar - Complete POC
- ✅ Documentation - Complete
- ⏳ Integration with Prunarr - Pending
- ⏳ Docker Hub publication - Pending
- ⏳ Community testing - Pending

## Requirements

### For C# Plugin
- .NET SDK 7.0+
- Jellyfin 10.8.0+
- Access to Jellyfin's plugin directory

### For Go Sidecar
- Docker and Docker Compose (recommended)
- OR Go 1.21+ for building from source
- Jellyfin server with API access

## Contributing

This is a proof-of-concept for evaluating integration approaches. Once the recommended approach is selected and integrated into Prunarr, contributions will be welcome.

## Architecture Diagrams

### Current Prunarr Architecture (Problem)
```
┌─────────┐     Volume     ┌──────────┐
│ Prunarr │────Mapping─────│ Jellyfin │
└─────────┘                └──────────┘
     │                           │
     │ Create symlinks           │
     │ Path translation          │
     └──────────┬────────────────┘
                │
         ┌──────▼──────┐
         │   Media     │
         │ Filesystem  │
         └─────────────┘
```

### Proposed Architecture (Solution)
```
┌─────────┐    HTTP API    ┌──────────────┐    Jellyfin API    ┌──────────┐
│ Prunarr │───────────────▶│   Sidecar    │───────────────────▶│ Jellyfin │
└─────────┘                └──────────────┘                     └──────────┘
                                  │                                   │
                                  │ Shared Volume                     │
                                  └───────────┬───────────────────────┘
                                              │
                                       ┌──────▼──────┐
                                       │  Symlinks   │
                                       │  Directory  │
                                       └─────────────┘
```

## License

This is a proof-of-concept project. License to be determined when integrated into Prunarr.

## Related Projects

- **Prunarr** - Media cleanup tool for Plex/Jellyfin/Emby
- **Jellyfin** - Free software media system

## Support

For questions or issues:
1. Review the [FEASIBILITY_REPORT.md](FEASIBILITY_REPORT.md)
2. Check installation guides
3. Review troubleshooting sections in installation docs

## Next Steps

1. **Integration**: Integrate Go sidecar into Prunarr codebase
2. **Testing**: End-to-end testing with real Jellyfin instances
3. **Distribution**: Publish Docker image to Docker Hub
4. **Documentation**: Update Prunarr docs with new deployment method
5. **Community**: Beta testing with Prunarr users
