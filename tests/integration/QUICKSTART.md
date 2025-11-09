# Go Integration Tests - Quick Reference

## Run Tests (Fully Automated)

```bash
# 1. Build plugin
./build.sh

# 2. Run tests (everything automated)
cd tests/integration
go test -v ./...

# That's it! Tests automatically:
# - Start Docker Compose (if not running)
# - Setup Jellyfin
# - Run all tests
# - Clean up containers and files
```

## Run Specific Test

```bash
cd tests/integration
go test -v -run TestSymlinkLifecycle
```

## Keep Environment for Debugging

```bash
# Skip automatic cleanup
cd tests/integration
export OXICLEANARR_KEEP_FILES=1
go test -v ./...

# Docker environment stays running
# Access at http://localhost:8096 (admin/adminpass)

# Manually clean up later:
cd ../assets
docker-compose down -v
rm -rf jellyfin-config jellyfin-cache leaving-soon-data
```

## Test Structure

```
tests/
├── assets/
│   ├── docker-compose.yml        # Docker config (auto-started)
│   └── test-media/               # Test files
├── integration/
│   ├── go.mod                    # Go module
│   ├── jellyfin_helpers.go       # Jellyfin automation
│   ├── plugin_test.go            # Test suite with auto setup/cleanup
│   ├── README.md                 # Full documentation
│   └── QUICKSTART.md             # This file
```

## Test Coverage

### Test Functions
1. **TestPluginLoad** - Verifies plugin loaded, version 3.2.1
2. **TestSymlinkLifecycle** - Complete CRUD workflow
   - Create symlink
   - List symlinks (with SymlinkNames field)
   - Remove symlink
   - List empty directory
3. **TestMultipleSymlinks** - Batch operations

### Endpoints Tested
- `GET /api/oxicleanarr/status` (unauthenticated)
- `POST /api/oxicleanarr/symlinks/add`
- `GET /api/oxicleanarr/symlinks/list`
- `POST /api/oxicleanarr/symlinks/remove`

### Features Verified
- ✅ Automated Docker Compose startup/shutdown
- ✅ Plugin loading and version
- ✅ Automated Jellyfin setup wizard
- ✅ Authentication with API tokens
- ✅ Symlink creation on filesystem
- ✅ **New: SymlinkNames array field (v3.2.1)**
- ✅ Symlink removal
- ✅ Empty directory handling
- ✅ Automatic cleanup of test environment

## Key Files

### jellyfin_helpers.go
Helper functions for Jellyfin interaction:
- `JellyfinClient` - HTTP client with auth
- `WaitForReady()` - Wait for Jellyfin start
- `CompleteSetupWizard()` - Automate initial setup
- `Authenticate()` - Get API token
- `SetupJellyfinForTest()` - Complete setup

### plugin_test.go
Integration test suite with automatic setup/teardown via `TestMain`.

## Troubleshooting

### Connection refused
Wait for Jellyfin to start (up to 60 seconds):
```bash
docker logs jellyfin-test -f
curl http://localhost:8096/health
```

### Plugin not loaded
```bash
docker exec jellyfin-test ls -la /config/plugins/OxiCleanarr/
curl http://localhost:8096/api/oxicleanarr/status
```

### Compilation errors (IDE only)
The tests compile successfully with `go build` and `go test`. IDE warnings about missing packages can be ignored.

## Development

### Add new test
```go
func TestNewFeature(t *testing.T) {
    client, err := SetupJellyfinForTest(t, JellyfinURL, AdminUsername, AdminPassword)
    require.NoError(t, err)
    
    // Test code here
}
```

### Add helper function
```go
func (jc *JellyfinClient) NewOperation() error {
    resp, err := jc.DoRequest("POST", "/api/path", payload)
    // ...
}
```

## CI/CD

```yaml
- name: Build Plugin
  run: ./build.sh

- name: Run Integration Tests
  run: |
    cd tests/integration
    go test -v ./...
```

Tests handle Docker Compose automatically - no extra CI setup needed!

## Documentation

- Full guide: `tests/integration/README.md`
- Session notes: `tests/SESSION_SUMMARY_2025-11-09.md`
- API docs: `API.md`

## Stats

- **Total test code:** 665 lines (Go)
- **Test functions:** 3 (with 5 subtests)
- **Helper functions:** 8
- **Endpoints covered:** 4 of 6 (67%)
