package symlink

import (
	"fmt"
	"os"
	"path/filepath"
)

// Manager handles symlink operations
type Manager struct {
	basePath string
}

// NewManager creates a new symlink manager
func NewManager(basePath string) *Manager {
	return &Manager{
		basePath: basePath,
	}
}

// CreateSymlink creates a symlink to the source file in the base path
func (m *Manager) CreateSymlink(sourcePath string) (string, error) {
	// Verify source exists
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return "", fmt.Errorf("source file does not exist: %s", sourcePath)
	}

	// Ensure base path exists
	if err := os.MkdirAll(m.basePath, 0755); err != nil {
		return "", fmt.Errorf("failed to create base path: %w", err)
	}

	// Generate symlink path
	fileName := filepath.Base(sourcePath)
	symlinkPath := filepath.Join(m.basePath, fileName)

	// Remove existing symlink if present
	if _, err := os.Lstat(symlinkPath); err == nil {
		if err := os.Remove(symlinkPath); err != nil {
			return "", fmt.Errorf("failed to remove existing symlink: %w", err)
		}
	}

	// Create symlink
	if err := os.Symlink(sourcePath, symlinkPath); err != nil {
		return "", fmt.Errorf("failed to create symlink: %w", err)
	}

	return symlinkPath, nil
}

// RemoveSymlink removes a symlink
func (m *Manager) RemoveSymlink(symlinkPath string) error {
	// Verify it's a symlink
	info, err := os.Lstat(symlinkPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Already doesn't exist
		}
		return fmt.Errorf("failed to stat symlink: %w", err)
	}

	if info.Mode()&os.ModeSymlink == 0 {
		return fmt.Errorf("path is not a symlink: %s", symlinkPath)
	}

	// Remove symlink
	if err := os.Remove(symlinkPath); err != nil {
		return fmt.Errorf("failed to remove symlink: %w", err)
	}

	return nil
}

// ClearSymlinks removes all symlinks from the base path
func (m *Manager) ClearSymlinks() error {
	if _, err := os.Stat(m.basePath); os.IsNotExist(err) {
		return nil // Directory doesn't exist, nothing to clear
	}

	entries, err := os.ReadDir(m.basePath)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		path := filepath.Join(m.basePath, entry.Name())
		info, err := os.Lstat(path)
		if err != nil {
			continue
		}

		// Only remove symlinks
		if info.Mode()&os.ModeSymlink != 0 {
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("failed to remove symlink %s: %w", path, err)
			}
		}
	}

	return nil
}

// ListSymlinks returns all symlinks in the base path
func (m *Manager) ListSymlinks() ([]string, error) {
	if _, err := os.Stat(m.basePath); os.IsNotExist(err) {
		return []string{}, nil
	}

	entries, err := os.ReadDir(m.basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var symlinks []string
	for _, entry := range entries {
		path := filepath.Join(m.basePath, entry.Name())
		info, err := os.Lstat(path)
		if err != nil {
			continue
		}

		if info.Mode()&os.ModeSymlink != 0 {
			symlinks = append(symlinks, path)
		}
	}

	return symlinks, nil
}
