package jellyfin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client handles communication with Jellyfin API
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new Jellyfin API client
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// VirtualFolder represents a Jellyfin virtual folder
type VirtualFolder struct {
	Name           string   `json:"Name"`
	Locations      []string `json:"Locations"`
	CollectionType string   `json:"CollectionType"`
}

// LibraryOptions represents library configuration options
type LibraryOptions struct {
	EnablePhotos                 bool `json:"EnablePhotos"`
	EnableRealtimeMonitor        bool `json:"EnableRealtimeMonitor"`
	EnableChapterImageExtraction bool `json:"EnableChapterImageExtraction"`
	EnableInternetProviders      bool `json:"EnableInternetProviders"`
	SaveLocalMetadata            bool `json:"SaveLocalMetadata"`
}

// AddVirtualFolderRequest represents the request to create a virtual folder
type AddVirtualFolderRequest struct {
	LibraryOptions LibraryOptions `json:"LibraryOptions"`
}

// GetVirtualFolders retrieves all virtual folders
func (c *Client) GetVirtualFolders() ([]VirtualFolder, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/Library/VirtualFolders", c.baseURL), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Emby-Token", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get virtual folders: %s - %s", resp.Status, string(body))
	}

	var folders []VirtualFolder
	if err := json.NewDecoder(resp.Body).Decode(&folders); err != nil {
		return nil, err
	}

	return folders, nil
}

// CreateVirtualFolder creates a new virtual folder
func (c *Client) CreateVirtualFolder(name, collectionType string) error {
	reqBody := AddVirtualFolderRequest{
		LibraryOptions: LibraryOptions{
			EnablePhotos:                 false,
			EnableRealtimeMonitor:        true,
			EnableChapterImageExtraction: false,
			EnableInternetProviders:      true,
			SaveLocalMetadata:            false,
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/Library/VirtualFolders?name=%s&collectionType=%s&refreshLibrary=true",
		c.baseURL, name, collectionType)

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("X-Emby-Token", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create virtual folder: %s - %s", resp.Status, string(body))
	}

	return nil
}

// AddMediaPath adds a path to an existing virtual folder
func (c *Client) AddMediaPath(folderName, path string) error {
	url := fmt.Sprintf("%s/Library/VirtualFolders/Paths?name=%s&path=%s&refreshLibrary=true",
		c.baseURL, folderName, path)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("X-Emby-Token", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to add media path: %s - %s", resp.Status, string(body))
	}

	return nil
}

// RefreshLibrary triggers a library scan
func (c *Client) RefreshLibrary() error {
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/Library/Refresh", c.baseURL), nil)
	if err != nil {
		return err
	}

	req.Header.Set("X-Emby-Token", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to refresh library: %s - %s", resp.Status, string(body))
	}

	return nil
}

// EnsureVirtualFolder ensures the virtual folder exists and has the correct path
func (c *Client) EnsureVirtualFolder(name, collectionType, path string) error {
	folders, err := c.GetVirtualFolders()
	if err != nil {
		return fmt.Errorf("failed to get virtual folders: %w", err)
	}

	// Check if folder exists
	var exists bool
	var hasPath bool
	for _, folder := range folders {
		if folder.Name == name {
			exists = true
			for _, location := range folder.Locations {
				if location == path {
					hasPath = true
					break
				}
			}
			break
		}
	}

	// Create folder if it doesn't exist
	if !exists {
		if err := c.CreateVirtualFolder(name, collectionType); err != nil {
			return fmt.Errorf("failed to create virtual folder: %w", err)
		}
		time.Sleep(2 * time.Second) // Wait for folder creation
	}

	// Add path if it doesn't exist
	if !hasPath {
		if err := c.AddMediaPath(name, path); err != nil {
			return fmt.Errorf("failed to add media path: %w", err)
		}
	}

	return nil
}
