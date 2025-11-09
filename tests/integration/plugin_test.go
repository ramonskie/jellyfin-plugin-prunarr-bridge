package integration

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	JellyfinURL   = "http://localhost:8096"
	AdminUsername = "admin"
	AdminPassword = "adminpass"
	// Container paths (as seen from inside Jellyfin Docker container)
	ContainerMediaDir   = "/media/movies"
	ContainerSymlinkDir = "/data/leaving-soon"
	TestMovieFile       = "Test Movie (2024)/Test Movie (2024).mkv"
	// Host paths (for file verification outside container)
	AssetsDir      = "../assets"
	HostSymlinkDir = "../assets/leaving-soon-data"
)

// shouldKeepFiles returns true if cleanup should be skipped
func shouldKeepFiles() bool {
	return os.Getenv("OXICLEANARR_KEEP_FILES") == "1"
}

// BuildPlugin builds the plugin DLL using dotnet build
func BuildPlugin() error {
	fmt.Println("Building plugin...")

	// Get project root (two directories up from tests/integration)
	projectRoot, err := filepath.Abs("../..")
	if err != nil {
		return fmt.Errorf("failed to get project root: %w", err)
	}

	pluginDir := filepath.Join(projectRoot, "Jellyfin.Plugin.OxiCleanarr")
	buildDir := filepath.Join(projectRoot, "build")

	// Clean previous build
	os.RemoveAll(buildDir)
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return fmt.Errorf("failed to create build directory: %w", err)
	}

	// Restore dependencies
	fmt.Println("  Restoring dependencies...")
	cmd := exec.Command("dotnet", "restore")
	cmd.Dir = pluginDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to restore dependencies: %w", err)
	}

	// Build plugin
	fmt.Println("  Building plugin DLL...")
	cmd = exec.Command("dotnet", "build", "--configuration", "Release", "--output", buildDir)
	cmd.Dir = pluginDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to build plugin: %w", err)
	}

	fmt.Println("Plugin build complete!")
	return nil
}

// InstallPluginToJellyfin copies the built plugin DLL to Jellyfin's plugins directory
// This is called BEFORE starting Docker, and creates the directory structure
func InstallPluginToJellyfin() error {
	fmt.Println("Installing plugin to Jellyfin...")

	projectRoot, err := filepath.Abs("../..")
	if err != nil {
		return fmt.Errorf("failed to get project root: %w", err)
	}

	buildDir := filepath.Join(projectRoot, "build")
	assetsDir := filepath.Join(projectRoot, "tests", "assets")
	pluginDir := filepath.Join(assetsDir, "jellyfin-config", "plugins", "OxiCleanarr")

	// Create plugin directory structure
	fmt.Println("  Creating plugin directory structure...")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		return fmt.Errorf("failed to create plugin directory: %w", err)
	}

	// Copy all DLL files from build directory
	entries, err := os.ReadDir(buildDir)
	if err != nil {
		return fmt.Errorf("failed to read build directory: %w", err)
	}

	copiedCount := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) == ".dll" {
			srcPath := filepath.Join(buildDir, entry.Name())
			dstPath := filepath.Join(pluginDir, entry.Name())

			// Read source file
			data, err := os.ReadFile(srcPath)
			if err != nil {
				return fmt.Errorf("failed to read %s: %w", entry.Name(), err)
			}

			// Write to destination
			if err := os.WriteFile(dstPath, data, 0644); err != nil {
				return fmt.Errorf("failed to write %s: %w", entry.Name(), err)
			}

			fmt.Printf("  Copied: %s\n", entry.Name())
			copiedCount++
		}
	}

	if copiedCount == 0 {
		return fmt.Errorf("no DLL files found in build directory")
	}

	fmt.Printf("Plugin installed! (%d files copied)\n", copiedCount)
	return nil
}

// StartDockerEnvironment starts the Docker Compose environment
func StartDockerEnvironment() error {
	absAssetsDir, err := filepath.Abs(AssetsDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute assets dir: %w", err)
	}

	fmt.Println("Starting Docker Compose environment...")

	// Run docker-compose up -d
	cmd := exec.Command("docker-compose", "up", "-d")
	cmd.Dir = absAssetsDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start Docker Compose: %w", err)
	}

	fmt.Println("Docker Compose environment started")
	return nil
}

// IsDockerEnvironmentRunning checks if the Jellyfin container is running
func IsDockerEnvironmentRunning() bool {
	cmd := exec.Command("docker", "ps", "--filter", "name=jellyfin-test", "--format", "{{.Names}}")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}

	return len(output) > 0 && string(output) != ""
}

// TestMain runs before all tests and handles global setup/cleanup
func TestMain(m *testing.M) {
	var code int

	// Setup environment
	fmt.Println("============================================================")
	fmt.Println("Starting Integration Test Environment")
	fmt.Println("============================================================")

	// Stop any existing Docker environment first to ensure clean state
	tmpT := &testing.T{}
	if IsDockerEnvironmentRunning() {
		fmt.Println("Stopping existing Docker environment...")
		CleanupDockerEnvironment(tmpT)
		// Wait for cleanup to complete
		time.Sleep(2 * time.Second)
	}

	// Always clean up old directories before starting (forced cleanup for fresh state)
	fmt.Println("Cleaning up old test directories...")
	cleanupTestDirectoriesForced(tmpT)

	// Build plugin
	if err := BuildPlugin(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to build plugin: %v\n", err)
		os.Exit(1)
	}

	// Install plugin to Jellyfin plugins directory (before Docker starts)
	if err := InstallPluginToJellyfin(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to install plugin: %v\n", err)
		os.Exit(1)
	}

	// Start Docker Compose environment
	if err := StartDockerEnvironment(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start Docker environment: %v\n", err)
		os.Exit(1)
	}

	// Give Docker a moment to initialize
	fmt.Println("Waiting for Docker containers to initialize...")
	time.Sleep(2 * time.Second)

	fmt.Println("============================================================")

	// Run tests
	code = m.Run()

	// Global cleanup after all tests complete
	fmt.Println("============================================================")
	fmt.Println("Cleaning Up Test Environment")
	fmt.Println("============================================================")

	// Create a temporary test for cleanup logging
	t := &testing.T{}
	CleanupAll(t)

	fmt.Println("============================================================")
	fmt.Println("Integration Tests Complete")
	fmt.Println("============================================================")

	os.Exit(code)
}

// TestIntegration runs all integration tests in sequence with fail-fast
func TestIntegration(t *testing.T) {
	// Setup Jellyfin once for all tests
	t.Logf("Setting up Jellyfin for testing...")
	client, err := SetupJellyfinForTest(t, JellyfinURL, AdminUsername, AdminPassword)
	if err != nil {
		t.Fatalf("Failed to setup Jellyfin (fail-fast): %v", err)
	}
	t.Logf("Jellyfin setup complete - UserID: %s", client.UserID)

	// Cleanup test environment at the end
	t.Cleanup(func() {
		t.Logf("Cleaning up test symlinks...")
		CleanupTestSymlinks(t, client)
	})

	// Test 1: Plugin Status (unauthenticated endpoint)
	t.Run("PluginStatus", func(t *testing.T) {
		t.Logf("Testing plugin status endpoint (unauthenticated)...")

		resp, err := http.Get(JellyfinURL + "/api/oxicleanarr/status")
		if err != nil {
			t.Fatalf("Failed to call status endpoint (fail-fast): %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Status endpoint returned %d, expected 200 (fail-fast)", resp.StatusCode)
		}

		var status struct {
			Version string `json:"Version"`
		}

		err = json.NewDecoder(resp.Body).Decode(&status)
		if err != nil {
			t.Fatalf("Failed to decode status response (fail-fast): %v", err)
		}

		t.Logf("✓ Status endpoint accessible without authentication")
		t.Logf("  Plugin version: %s", status.Version)
		assert.NotEmpty(t, status.Version, "Version should not be empty")
		assert.Contains(t, status.Version, "3.2.1", "Expected v3.2.1")

		// Also verify that status endpoint works WITH authentication
		t.Logf("Testing status endpoint with authentication...")
		resp2, err := client.DoRequest("GET", "/api/oxicleanarr/status", nil)
		if err != nil {
			t.Fatalf("Failed to call status endpoint with auth (fail-fast): %v", err)
		}
		defer resp2.Body.Close()

		if resp2.StatusCode != http.StatusOK {
			t.Fatalf("Status endpoint with auth returned %d, expected 200 (fail-fast)", resp2.StatusCode)
		}

		var status2 struct {
			Version string `json:"Version"`
		}

		err = json.NewDecoder(resp2.Body).Decode(&status2)
		if err != nil {
			t.Fatalf("Failed to decode status response with auth (fail-fast): %v", err)
		}

		assert.Equal(t, status.Version, status2.Version, "Version should be the same with and without auth")
		t.Logf("✓ Status endpoint also works with authentication")
	})

	// Test 1a: Verify other endpoints require authentication
	t.Run("AuthenticationRequired", func(t *testing.T) {
		t.Logf("Testing that non-status endpoints require authentication...")

		// Try to list symlinks without authentication
		resp, err := http.Get(fmt.Sprintf("%s/api/oxicleanarr/symlinks/list?directory=%s", JellyfinURL, ContainerSymlinkDir))
		if err != nil {
			t.Fatalf("Failed to call list endpoint (fail-fast): %v", err)
		}
		defer resp.Body.Close()

		// Should return 401 Unauthorized or 403 Forbidden
		if resp.StatusCode == http.StatusOK {
			t.Fatalf("List endpoint should require authentication but returned 200 (fail-fast)")
		}

		assert.True(t, resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden,
			"Expected 401 or 403, got %d", resp.StatusCode)

		t.Logf("✓ List endpoint correctly requires authentication (got %d)", resp.StatusCode)

		// Verify it works WITH authentication
		resp2, err := client.DoRequest("GET", fmt.Sprintf("/api/oxicleanarr/symlinks/list?directory=%s", ContainerSymlinkDir), nil)
		if err != nil {
			t.Fatalf("Failed to call list endpoint with auth (fail-fast): %v", err)
		}
		defer resp2.Body.Close()

		if resp2.StatusCode != http.StatusOK {
			t.Fatalf("List endpoint with auth returned %d, expected 200 (fail-fast)", resp2.StatusCode)
		}

		t.Logf("✓ List endpoint works correctly with authentication")
	})

	// Test 1b: Verify Plugin Installation via /Plugins API
	t.Run("VerifyPluginInstalled", func(t *testing.T) {
		t.Logf("Verifying plugin is installed via /Plugins API...")

		plugins, err := client.GetInstalledPlugins()
		if err != nil {
			t.Fatalf("Failed to query installed plugins (fail-fast): %v", err)
		}

		t.Logf("Found %d installed plugins", len(plugins))

		// Look for our OxiCleanarr Bridge plugin
		var oxiPlugin *Plugin
		for i := range plugins {
			t.Logf("  Plugin: %s (Version: %s)", plugins[i].Name, plugins[i].Version)
			if plugins[i].Name == "OxiCleanarr Bridge" {
				oxiPlugin = &plugins[i]
			}
		}

		if oxiPlugin == nil {
			t.Fatalf("OxiCleanarr Bridge plugin not found in installed plugins list (fail-fast)")
		}

		assert.Equal(t, "OxiCleanarr Bridge", oxiPlugin.Name, "Plugin name should match")
		assert.Contains(t, oxiPlugin.Version, "3.2.1", "Plugin version should match")
		t.Logf("✓ OxiCleanarr plugin verified: %s v%s", oxiPlugin.Name, oxiPlugin.Version)
	})

	// Use container paths for API calls (as seen from inside Docker container)
	sourceFile := filepath.Join(ContainerMediaDir, TestMovieFile)
	symlinkDir := ContainerSymlinkDir

	t.Logf("Container source file: %s", sourceFile)
	t.Logf("Container symlink directory: %s", symlinkDir)

	// Verify test media exists on host filesystem
	hostTestMediaPath := filepath.Join(AssetsDir, "test-media/movies", TestMovieFile)
	if _, err := os.Stat(hostTestMediaPath); os.IsNotExist(err) {
		t.Fatalf("Test media file not found on host: %s (fail-fast)", hostTestMediaPath)
	}

	// Test 2: Create Symlink
	// Test 2: Create Symlink
	t.Run("CreateSymlink", func(t *testing.T) {
		t.Logf("Testing symlink creation...")

		// Create symlink via API using container paths
		payload := map[string]interface{}{
			"items": []map[string]string{
				{
					"sourcePath":      sourceFile,
					"targetDirectory": symlinkDir,
				},
			},
		}

		resp, err := client.DoRequest("POST", "/api/oxicleanarr/symlinks/add", payload)
		if err != nil {
			t.Fatalf("Failed to call add symlink endpoint (fail-fast): %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Add symlink returned %d, expected 200 (fail-fast)", resp.StatusCode)
		}

		var addResponse struct {
			Success         bool     `json:"Success"`
			CreatedSymlinks []string `json:"CreatedSymlinks"`
			Errors          []string `json:"Errors"`
		}

		body, _ := io.ReadAll(resp.Body)
		t.Logf("Add response: %s", string(body))

		err = json.Unmarshal(body, &addResponse)
		if err != nil {
			t.Fatalf("Failed to decode add response (fail-fast): %v", err)
		}

		if !addResponse.Success {
			t.Fatalf("Symlink creation failed (fail-fast): %v", addResponse.Errors)
		}

		assert.Len(t, addResponse.CreatedSymlinks, 1, "Should create 1 symlink")
		assert.Empty(t, addResponse.Errors, "Should have no errors")

		// Plugin returns container paths in response
		expectedSymlinkContainer := filepath.Join(symlinkDir, "Test Movie (2024).mkv")
		assert.Contains(t, addResponse.CreatedSymlinks, expectedSymlinkContainer, "Should create symlink with correct path")

		// Verify symlink exists on host filesystem
		hostSymlinkPath := filepath.Join(HostSymlinkDir, "Test Movie (2024).mkv")
		symlinkInfo, err := os.Lstat(hostSymlinkPath)
		if err != nil {
			t.Fatalf("Symlink should exist on filesystem (fail-fast): %v", err)
		}
		assert.NotEqual(t, 0, symlinkInfo.Mode()&os.ModeSymlink, "File should be a symlink")

		t.Logf("✓ Symlink created successfully: %s", hostSymlinkPath)
	})

	// Test 3: List Symlinks
	// Test 3: List Symlinks
	t.Run("ListSymlinks", func(t *testing.T) {
		t.Logf("Testing symlink listing...")

		// List symlinks via API using container path
		resp, err := client.DoRequest("GET", fmt.Sprintf("/api/oxicleanarr/symlinks/list?directory=%s", symlinkDir), nil)
		if err != nil {
			t.Fatalf("Failed to call list symlinks endpoint (fail-fast): %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("List symlinks returned %d, expected 200 (fail-fast)", resp.StatusCode)
		}

		var listResponse struct {
			Symlinks []struct {
				Path   string `json:"Path"`
				Target string `json:"Target"`
				Name   string `json:"Name"`
			} `json:"Symlinks"`
			Count        int      `json:"Count"`
			SymlinkNames []string `json:"SymlinkNames"`
			Message      string   `json:"Message"`
		}

		body, _ := io.ReadAll(resp.Body)
		t.Logf("List response: %s", string(body))

		err = json.Unmarshal(body, &listResponse)
		if err != nil {
			t.Fatalf("Failed to decode list response (fail-fast): %v", err)
		}

		assert.Equal(t, 1, listResponse.Count, "Should list 1 symlink")
		assert.Len(t, listResponse.Symlinks, 1, "Should have 1 symlink in array")
		assert.Len(t, listResponse.SymlinkNames, 1, "Should have 1 symlink name")

		// Verify SymlinkNames field (new in v3.2.1)
		assert.Contains(t, listResponse.SymlinkNames, "Test Movie (2024).mkv", "SymlinkNames should contain filename")

		// Verify full symlink details (plugin returns container paths)
		symlink := listResponse.Symlinks[0]
		assert.Equal(t, "Test Movie (2024).mkv", symlink.Name, "Symlink name should match")
		assert.Contains(t, symlink.Path, symlinkDir, "Symlink path should be in target directory")
		assert.Equal(t, sourceFile, symlink.Target, "Symlink target should match source file")

		t.Logf("✓ Symlink listed successfully")
		t.Logf("  Path: %s", symlink.Path)
		t.Logf("  Target: %s", symlink.Target)
		t.Logf("  Name: %s", symlink.Name)
	})

	// Test 4: Remove Symlink
	t.Run("RemoveSymlink", func(t *testing.T) {
		t.Logf("Testing symlink removal...")

		// Use container path for API call
		expectedSymlinkContainer := filepath.Join(symlinkDir, "Test Movie (2024).mkv")
		// Use host path for filesystem verification
		expectedSymlinkHost := filepath.Join(HostSymlinkDir, "Test Movie (2024).mkv")

		// Remove symlink via API
		payload := map[string]interface{}{
			"symlinkPaths": []string{expectedSymlinkContainer},
		}

		resp, err := client.DoRequest("POST", "/api/oxicleanarr/symlinks/remove", payload)
		if err != nil {
			t.Fatalf("Failed to call remove symlink endpoint (fail-fast): %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Remove symlink returned %d, expected 200 (fail-fast)", resp.StatusCode)
		}

		var removeResponse struct {
			Success         bool     `json:"Success"`
			RemovedSymlinks []string `json:"RemovedSymlinks"`
			Errors          []string `json:"Errors"`
		}

		body, _ := io.ReadAll(resp.Body)
		t.Logf("Remove response: %s", string(body))

		err = json.Unmarshal(body, &removeResponse)
		if err != nil {
			t.Fatalf("Failed to decode remove response (fail-fast): %v", err)
		}

		if !removeResponse.Success {
			t.Fatalf("Symlink removal failed (fail-fast): %v", removeResponse.Errors)
		}

		assert.Len(t, removeResponse.RemovedSymlinks, 1, "Should remove 1 symlink")
		assert.Empty(t, removeResponse.Errors, "Should have no errors")

		// Verify symlink no longer exists on host filesystem
		_, err = os.Lstat(expectedSymlinkHost)
		assert.True(t, os.IsNotExist(err), "Symlink should not exist after removal")

		t.Logf("✓ Symlink removed successfully")
	})

	// Test 5: List Empty Directory
	t.Run("ListEmptyDirectory", func(t *testing.T) {
		t.Logf("Testing list on empty directory...")

		// List symlinks in now-empty directory using container path
		resp, err := client.DoRequest("GET", fmt.Sprintf("/api/oxicleanarr/symlinks/list?directory=%s", symlinkDir), nil)
		if err != nil {
			t.Fatalf("Failed to call list symlinks endpoint (fail-fast): %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("List symlinks returned %d, expected 200 (fail-fast)", resp.StatusCode)
		}

		var listResponse struct {
			Symlinks     []interface{} `json:"Symlinks"`
			Count        int           `json:"Count"`
			SymlinkNames []string      `json:"SymlinkNames"`
			Message      string        `json:"Message"`
		}

		body, _ := io.ReadAll(resp.Body)
		t.Logf("List empty response: %s", string(body))

		err = json.Unmarshal(body, &listResponse)
		if err != nil {
			t.Fatalf("Failed to decode list response (fail-fast): %v", err)
		}

		assert.Equal(t, 0, listResponse.Count, "Should list 0 symlinks")
		assert.Empty(t, listResponse.Symlinks, "Symlinks array should be empty")
		assert.Empty(t, listResponse.SymlinkNames, "SymlinkNames array should be empty")
		assert.Contains(t, listResponse.Message, "No symlinks found", "Message should indicate no symlinks")

		t.Logf("✓ Empty directory listed successfully")
	})

	// Test 6: Multiple Symlinks (batch operations)
	t.Run("MultipleSymlinks", func(t *testing.T) {
		t.Logf("Testing creation and listing of multiple symlinks...")

		// Define multiple test movies
		testMovies := []struct {
			name string
			path string
		}{
			{"Test Movie (2024).mkv", "Test Movie (2024)/Test Movie (2024).mkv"},
			{"Action Movie (2023).mkv", "Action Movie (2023)/Action Movie (2023).mkv"},
			{"Comedy Movie (2022).mkv", "Comedy Movie (2022)/Comedy Movie (2022).mkv"},
		}

		// Create multiple symlinks in a single batch request
		var items []map[string]string
		for _, movie := range testMovies {
			items = append(items, map[string]string{
				"sourcePath":      filepath.Join(ContainerMediaDir, movie.path),
				"targetDirectory": symlinkDir,
			})
		}

		payload := map[string]interface{}{
			"items": items,
		}

		t.Logf("Creating %d symlinks in batch...", len(testMovies))
		resp, err := client.DoRequest("POST", "/api/oxicleanarr/symlinks/add", payload)
		if err != nil {
			t.Fatalf("Failed to call add symlink endpoint (fail-fast): %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("Add symlink returned %d, expected 200 (fail-fast). Response: %s", resp.StatusCode, string(body))
		}

		var addResponse struct {
			Success         bool     `json:"Success"`
			CreatedSymlinks []string `json:"CreatedSymlinks"`
			Errors          []string `json:"Errors"`
		}

		body, _ := io.ReadAll(resp.Body)
		err = json.Unmarshal(body, &addResponse)
		if err != nil {
			t.Fatalf("Failed to decode add response (fail-fast): %v", err)
		}

		if !addResponse.Success {
			t.Fatalf("Batch symlink creation failed (fail-fast): %v", addResponse.Errors)
		}

		assert.Len(t, addResponse.CreatedSymlinks, len(testMovies), "Should create %d symlinks", len(testMovies))
		assert.Empty(t, addResponse.Errors, "Should have no errors")
		t.Logf("✓ Created %d symlinks successfully", len(addResponse.CreatedSymlinks))

		// Wait for filesystem sync
		time.Sleep(100 * time.Millisecond)

		// List and verify all symlinks using container path
		t.Logf("Listing symlinks to verify all were created...")
		resp, err = client.DoRequest("GET", fmt.Sprintf("/api/oxicleanarr/symlinks/list?directory=%s", symlinkDir), nil)
		if err != nil {
			t.Fatalf("Failed to call list symlinks endpoint (fail-fast): %v", err)
		}
		defer resp.Body.Close()

		var listResponse struct {
			Symlinks []struct {
				Path   string `json:"Path"`
				Target string `json:"Target"`
				Name   string `json:"Name"`
			} `json:"Symlinks"`
			Count        int      `json:"Count"`
			SymlinkNames []string `json:"SymlinkNames"`
			Message      string   `json:"Message"`
		}

		body, _ = io.ReadAll(resp.Body)
		t.Logf("List response: %s", string(body))
		err = json.Unmarshal(body, &listResponse)
		if err != nil {
			t.Fatalf("Failed to decode list response (fail-fast): %v", err)
		}

		// Verify count
		assert.Equal(t, len(testMovies), listResponse.Count, "Should list %d symlinks", len(testMovies))
		assert.Len(t, listResponse.Symlinks, len(testMovies), "Symlinks array should have %d entries", len(testMovies))
		assert.Len(t, listResponse.SymlinkNames, len(testMovies), "SymlinkNames should have %d entries", len(testMovies))

		// Verify each expected symlink is present
		for _, movie := range testMovies {
			assert.Contains(t, listResponse.SymlinkNames, movie.name, "Should contain symlink: %s", movie.name)
		}

		// Verify symlinks exist on host filesystem
		for _, movie := range testMovies {
			hostSymlinkPath := filepath.Join(HostSymlinkDir, movie.name)
			symlinkInfo, err := os.Lstat(hostSymlinkPath)
			if err != nil {
				t.Errorf("Symlink should exist on filesystem: %s (error: %v)", hostSymlinkPath, err)
				continue
			}
			assert.NotEqual(t, 0, symlinkInfo.Mode()&os.ModeSymlink, "File should be a symlink: %s", movie.name)
		}

		t.Logf("✓ Listed and verified %d symlinks", listResponse.Count)
		t.Logf("  Symlink names: %v", listResponse.SymlinkNames)

		// Verify detailed metadata for each symlink
		for i, symlink := range listResponse.Symlinks {
			t.Logf("  [%d] Name: %s", i+1, symlink.Name)
			t.Logf("      Path: %s", symlink.Path)
			t.Logf("      Target: %s", symlink.Target)
			assert.Contains(t, symlink.Path, symlinkDir, "Symlink path should be in target directory")
			assert.Contains(t, symlink.Target, ContainerMediaDir, "Symlink target should be in media directory")
		}
	})
}

// CleanupTestSymlinks removes all test symlinks
func CleanupTestSymlinks(t *testing.T, client *JellyfinClient) {
	if shouldKeepFiles() {
		t.Logf("Skipping symlink cleanup (OXICLEANARR_KEEP_FILES=1)")
		return
	}

	// List all symlinks using container path
	resp, err := client.DoRequest("GET", fmt.Sprintf("/api/oxicleanarr/symlinks/list?directory=%s", ContainerSymlinkDir), nil)
	if err != nil {
		t.Logf("Warning: Failed to list symlinks during cleanup: %v", err)
		return
	}
	defer resp.Body.Close()

	var listResponse struct {
		Symlinks []struct {
			Path string `json:"Path"`
		} `json:"Symlinks"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&listResponse); err != nil {
		t.Logf("Warning: Failed to decode list response during cleanup: %v", err)
		return
	}

	if len(listResponse.Symlinks) == 0 {
		t.Logf("No symlinks to clean up")
		return
	}

	// Remove all symlinks
	var paths []string
	for _, symlink := range listResponse.Symlinks {
		paths = append(paths, symlink.Path)
	}

	payload := map[string]interface{}{
		"symlinkPaths": paths,
	}

	resp, err = client.DoRequest("POST", "/api/oxicleanarr/symlinks/remove", payload)
	if err != nil {
		t.Logf("Warning: Failed to remove symlinks during cleanup: %v", err)
		return
	}
	defer resp.Body.Close()

	t.Logf("Cleaned up %d symlink(s)", len(paths))
}

// CleanupDockerEnvironment stops and removes Docker containers
func CleanupDockerEnvironment(t *testing.T) {
	if shouldKeepFiles() {
		t.Logf("Skipping Docker cleanup (OXICLEANARR_KEEP_FILES=1)")
		return
	}

	t.Logf("Stopping Docker environment...")

	absAssetsDir, err := filepath.Abs(AssetsDir)
	if err != nil {
		t.Logf("Warning: Failed to get absolute assets dir: %v", err)
		return
	}

	// Run docker-compose down
	cmd := exec.Command("docker-compose", "down", "-v")
	cmd.Dir = absAssetsDir
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Logf("Warning: Failed to stop Docker environment: %v\nOutput: %s", err, string(output))
		return
	}

	t.Logf("Docker environment stopped and removed")
}

// CleanupTestDirectories removes directories created during tests
func CleanupTestDirectories(t *testing.T) {
	if shouldKeepFiles() {
		t.Logf("Skipping directory cleanup (OXICLEANARR_KEEP_FILES=1)")
		return
	}

	cleanupTestDirectoriesForced(t)
}

// cleanupTestDirectoriesForced removes directories without checking flags (for startup cleanup)
func cleanupTestDirectoriesForced(t *testing.T) {
	absAssetsDir, err := filepath.Abs(AssetsDir)
	if err != nil {
		t.Logf("Warning: Failed to get absolute assets dir: %v", err)
		return
	}

	dirsToRemove := []string{
		filepath.Join(absAssetsDir, "jellyfin-config"),
		filepath.Join(absAssetsDir, "jellyfin-cache"),
		filepath.Join(absAssetsDir, "leaving-soon-data"),
	}

	for _, dir := range dirsToRemove {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		// Try to remove - if permission denied, try with sudo
		if err := os.RemoveAll(dir); err != nil {
			// Try with sudo if regular removal fails
			cmd := exec.Command("sudo", "rm", "-rf", dir)
			if sudoErr := cmd.Run(); sudoErr != nil {
				t.Logf("Warning: Failed to remove directory %s: %v (sudo also failed: %v)", dir, err, sudoErr)
				continue
			}
		}

		t.Logf("Removed directory: %s", dir)
	}
}

// CleanupAll performs complete cleanup of test environment
func CleanupAll(t *testing.T) {
	if shouldKeepFiles() {
		t.Logf("OXICLEANARR_KEEP_FILES=1 - Keeping all test files and containers")
		t.Logf("To manually clean up:")
		t.Logf("  cd tests/assets && docker-compose down -v")
		t.Logf("  rm -rf tests/assets/jellyfin-config tests/assets/jellyfin-cache tests/assets/leaving-soon-data")
		return
	}

	CleanupDockerEnvironment(t)
	CleanupTestDirectories(t)
	t.Logf("Complete cleanup finished")
}
