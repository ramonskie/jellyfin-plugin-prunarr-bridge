# Integration Tests

Automated Go integration tests for the Jellyfin OxiCleanarr Plugin.

## Overview

These tests verify the complete plugin functionality including:
- Plugin loading and version verification
- Symlink creation via API
- Symlink listing with the new `SymlinkNames` field (v3.2.1+)
- Symlink removal via API
- Batch operations
- Authentication and authorization

## Quick Start

```bash
# 1. Build plugin (from project root)
./build.sh

# 2. Run tests (everything is automatic!)
cd tests/integration
go test -v ./...

# Tests automatically:
# - Start Docker Compose environment (if not running)
# - Wait for Jellyfin to be ready
# - Run all tests
# - Clean up:
#   - Stop and remove Docker containers
#   - Remove created directories (jellyfin-config, jellyfin-cache, leaving-soon-data)
#   - Remove all test symlinks
```

## Requirements

- Docker and docker-compose (automatically managed by tests)
- Go 1.21+
- Built plugin DLL (run `./build.sh` from project root)

## Test Environment

The tests automatically manage:
- **Docker Compose:** Started automatically if not running (from `tests/assets/docker-compose.yml`)
- **Jellyfin URL:** http://localhost:8096
- **Admin user:** admin / adminpass (auto-created by tests)
- **Test media:** `tests/assets/test-media/movies/Test Movie (2024)/`
- **Symlink dir:** `tests/assets/leaving-soon-data/` (created at runtime)

## Running Tests

### Full test suite

```bash
cd tests/integration
go test -v ./...
```

### Run specific test

```bash
cd tests/integration
go test -v -run TestSymlinkLifecycle
```

### Keep environment running for debugging

To keep the environment up after tests complete (useful for debugging):

```bash
cd tests/integration
export OXICLEANARR_KEEP_FILES=1  # Skip automatic cleanup
go test -v ./...

# Docker environment stays running
# Access Jellyfin at http://localhost:8096
# Username: admin
# Password: adminpass

# Manually clean up when done:
cd ../assets
docker-compose down -v
rm -rf jellyfin-config jellyfin-cache leaving-soon-data
```

## Test Structure

### `jellyfin_helpers.go`

Helper functions for Jellyfin interaction:
- `JellyfinClient` - HTTP client with authentication
- `WaitForReady()` - Waits for Jellyfin to start
- `NeedsSetup()` - Checks if setup wizard is needed
- `CompleteSetupWizard()` - Automates initial setup
- `Authenticate()` - Logs in and gets API token
- `SetupJellyfinForTest()` - Complete automated setup

### `plugin_test.go`

Integration test suite:
- `TestPluginLoad` - Verifies plugin is loaded and responding
- `TestSymlinkLifecycle` - Tests full CRUD workflow:
  - Create symlink
  - List symlinks (verifies `SymlinkNames` field)
  - Remove symlink
  - List empty directory
- `TestMultipleSymlinks` - Tests batch operations

## Test Features

### Automated Setup

Tests automatically handle the entire lifecycle:
- **Before tests:**
  - Start Docker Compose environment (if not already running)
  - Wait for Jellyfin to be ready
  - Complete the setup wizard if needed
  - Create admin user
  - Authenticate and get API token
- **After tests:**
  - Stop and remove Docker containers
  - Clean up all created directories and files

### What Tests Do

When you run `go test`, the following happens automatically:
- Start Docker containers (if not already running)
- Bind to port 8096
- Create test directories and files
- Run all integration tests
- Clean up everything (unless `OXICLEANARR_KEEP_FILES=1`)

### Automatic Cleanup

**Tests automatically clean up everything by default:**

1. **During each test** (via `t.Cleanup()`):
   - Removes all created symlinks

2. **After all tests complete** (via `TestMain`):
   - Stops and removes Docker containers (`docker-compose down -v`)
   - Removes runtime directories:
     - `tests/assets/jellyfin-config`
     - `tests/assets/jellyfin-cache`
     - `tests/assets/leaving-soon-data`

**To skip cleanup** (for debugging):
```bash
export OXICLEANARR_KEEP_FILES=1
```

This preserves:
- Docker containers (keep running)
- All created directories
- All test symlinks

When `OXICLEANARR_KEEP_FILES=1` is set, tests log manual cleanup commands.

## Verifying v3.2.1 Features

The tests specifically verify the new `SymlinkNames` field added in v3.2.1:

```go
// List response includes SymlinkNames array
{
  "Symlinks": [...],
  "Count": 1,
  "SymlinkNames": ["movie.mkv"],  // ← New in v3.2.1
  "Message": "Found 1 symlink(s)"
}
```

See `TestSymlinkLifecycle/ListSymlinks` for the verification.

## Troubleshooting

### Tests fail with "connection refused"

Jellyfin hasn't started yet. The test runner waits up to 60 seconds.

```bash
# Check container status
docker ps
docker logs jellyfin-test
```

### Tests fail with "401 Unauthorized"

Authentication failed. Check:
- Setup wizard completed successfully
- Admin credentials are correct (admin/adminpass)

### Symlinks not created

Check:
- Test media exists: `test-media/movies/Test Movie (2024)/Test Movie (2024).mkv`
- Volume permissions (SELinux: volumes need `:z` flag)
- Container logs: `docker logs jellyfin-test`

### Plugin not loaded

```bash
# Check plugin is present
docker exec jellyfin-test ls -la /config/plugins/OxiCleanarr/

# Check plugin version
curl http://localhost:8096/api/oxicleanarr/status
```

## CI/CD Integration

To run in CI (fully automated):

```yaml
- name: Build Plugin
  run: ./build.sh

- name: Run Integration Tests
  run: |
    cd tests/integration
    go test -v ./...
```

No need to manually start/stop Docker - tests handle everything!

## Development

### Adding New Tests

1. Add test function to `plugin_test.go`:

```go
func TestNewFeature(t *testing.T) {
    client, err := SetupJellyfinForTest(t, JellyfinURL, AdminUsername, AdminPassword)
    require.NoError(t, err)
    
    // Your test code
}
```

2. Run tests:

```bash
cd tests/integration
go test -v -run TestNewFeature
```

### Helper Functions

Add shared helpers to `jellyfin_helpers.go`:

```go
func (jc *JellyfinClient) CustomOperation() error {
    resp, err := jc.DoRequest("POST", "/api/custom", payload)
    // ...
}
```

## Comparison with OxiCleanarr Tests

These tests follow the same pattern as OxiCleanarr's integration tests:
- Environment-gated with env variable
- Automated Jellyfin setup wizard completion
- API key management
- Cleanup between tests
- Polling with timeouts

Key differences:
- Focused on plugin API endpoints only
- No Radarr/Sonarr setup needed
- Simpler test media requirements
- Tests plugin-specific features like `SymlinkNames`

## References

- [API Documentation](../../API.md)
- [Test Results v3.2.1](../TEST_RESULTS_v3.2.1.md)
- [OxiCleanarr Integration Tests](https://github.com/ramonskie/oxicleanarr/tree/main/test/integration)
