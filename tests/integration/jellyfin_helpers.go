package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
)

const (
	DefaultMaxRetries = 60
	DefaultRetryDelay = 2 * time.Second
)

// JellyfinClient handles Jellyfin API interactions for testing
type JellyfinClient struct {
	BaseURL  string
	Username string
	Password string
	APIKey   string
	UserID   string
	client   *http.Client
	t        *testing.T
}

// NewJellyfinClient creates a new Jellyfin client for testing
func NewJellyfinClient(t *testing.T, baseURL, username, password string) *JellyfinClient {
	return &JellyfinClient{
		BaseURL:  baseURL,
		Username: username,
		Password: password,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		t: t,
	}
}

// WaitForReady waits for Jellyfin to be accessible
func (jc *JellyfinClient) WaitForReady() error {
	jc.t.Logf("Waiting for Jellyfin to be ready at %s...", jc.BaseURL)

	for i := 0; i < DefaultMaxRetries; i++ {
		// Try health endpoint
		resp, err := jc.client.Get(jc.BaseURL + "/health")
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				jc.t.Logf("Jellyfin is ready!")
				return nil
			}
		}

		// Fallback to public info endpoint
		resp, err = jc.client.Get(jc.BaseURL + "/System/Info/Public")
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				jc.t.Logf("Jellyfin is ready!")
				return nil
			}
		}

		if (i+1)%10 == 0 {
			jc.t.Logf("Still waiting... (%d/%d)", i+1, DefaultMaxRetries)
		}
		time.Sleep(DefaultRetryDelay)
	}

	return fmt.Errorf("jellyfin failed to start after %v", time.Duration(DefaultMaxRetries)*DefaultRetryDelay)
}

// NeedsSetup returns true if setup wizard needs to be completed
func (jc *JellyfinClient) NeedsSetup() (bool, error) {
	jc.t.Logf("Checking if setup wizard is needed...")

	// Check if we can get the startup User endpoint (means wizard not completed)
	resp, err := jc.client.Get(jc.BaseURL + "/Startup/User")
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			jc.t.Logf("Setup wizard needs to be completed")
			return true, nil
		}
	}

	// If User endpoint fails, check public system info (means setup complete)
	resp, err = jc.client.Get(jc.BaseURL + "/System/Info/Public")
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			jc.t.Logf("Setup wizard already completed")
			return false, nil
		}
	}

	return false, fmt.Errorf("unable to determine setup status")
}

// CompleteSetupWizard automates the Jellyfin setup wizard
func (jc *JellyfinClient) CompleteSetupWizard() error {
	jc.t.Logf("Completing setup wizard...")

	// Step 1: Create admin user
	createUserPayload := map[string]interface{}{
		"Name":     jc.Username,
		"Password": jc.Password,
	}

	body, _ := json.Marshal(createUserPayload)
	req, _ := http.NewRequest("POST", jc.BaseURL+"/Startup/User", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := jc.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create admin user: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	jc.t.Logf("Admin user created")

	// Step 2: Complete the wizard
	req, _ = http.NewRequest("POST", jc.BaseURL+"/Startup/Complete", nil)
	resp, err = jc.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to complete wizard: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to complete wizard: status %d", resp.StatusCode)
	}

	jc.t.Logf("Setup wizard completed")

	// Wait a moment for Jellyfin to finalize setup
	time.Sleep(2 * time.Second)

	return nil
}

// Authenticate logs in and gets an API key
func (jc *JellyfinClient) Authenticate() error {
	jc.t.Logf("Authenticating as %s...", jc.Username)

	// Login to get access token
	loginPayload := map[string]interface{}{
		"Username": jc.Username,
		"Pw":       jc.Password,
	}

	body, _ := json.Marshal(loginPayload)
	req, _ := http.NewRequest("POST", jc.BaseURL+"/Users/authenticatebyname", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Emby-Authorization", `MediaBrowser Client="IntegrationTest", Device="TestRunner", DeviceId="test-device", Version="1.0.0"`)

	resp, err := jc.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to authenticate: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var authResponse struct {
		User struct {
			ID string `json:"Id"`
		} `json:"User"`
		AccessToken string `json:"AccessToken"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&authResponse); err != nil {
		return fmt.Errorf("failed to decode auth response: %w", err)
	}

	jc.UserID = authResponse.User.ID
	jc.APIKey = authResponse.AccessToken

	jc.t.Logf("Authenticated - UserID: %s, Token: %s...", jc.UserID, jc.APIKey[:8])

	return nil
}

// CreateAPIKey creates a permanent API key for testing
func (jc *JellyfinClient) CreateAPIKey(appName string) (string, error) {
	jc.t.Logf("Creating API key for %s...", appName)

	req, _ := http.NewRequest("POST", jc.BaseURL+"/Auth/Keys?App="+appName, nil)
	req.Header.Set("X-MediaBrowser-Token", jc.APIKey)

	resp, err := jc.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to create API key: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to create API key: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	// Get the created key
	req, _ = http.NewRequest("GET", jc.BaseURL+"/Auth/Keys", nil)
	req.Header.Set("X-MediaBrowser-Token", jc.APIKey)

	resp, err = jc.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get API keys: %w", err)
	}
	defer resp.Body.Close()

	var keysResponse struct {
		Items []struct {
			AccessToken string `json:"AccessToken"`
			AppName     string `json:"AppName"`
		} `json:"Items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&keysResponse); err != nil {
		return "", fmt.Errorf("failed to decode keys response: %w", err)
	}

	for _, item := range keysResponse.Items {
		if item.AppName == appName {
			jc.t.Logf("API key created: %s...", item.AccessToken[:8])
			return item.AccessToken, nil
		}
	}

	return "", fmt.Errorf("API key not found after creation")
}

// SetupForTest performs complete Jellyfin setup for integration testing
func SetupJellyfinForTest(t *testing.T, baseURL, username, password string) (*JellyfinClient, error) {
	client := NewJellyfinClient(t, baseURL, username, password)

	// Wait for Jellyfin to be ready
	if err := client.WaitForReady(); err != nil {
		return nil, err
	}

	// Check if setup is needed
	needsSetup, err := client.NeedsSetup()
	if err != nil {
		return nil, err
	}

	// Complete setup wizard if needed
	if needsSetup {
		if err := client.CompleteSetupWizard(); err != nil {
			return nil, err
		}
	}

	// Authenticate
	if err := client.Authenticate(); err != nil {
		return nil, err
	}

	return client, nil
}

// DoRequest performs an authenticated HTTP request
func (jc *JellyfinClient) DoRequest(method, path string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, jc.BaseURL+path, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-MediaBrowser-Token", jc.APIKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return jc.client.Do(req)
}

// GetInstalledPlugins queries the Jellyfin /Plugins endpoint to get all installed plugins
func (jc *JellyfinClient) GetInstalledPlugins() ([]Plugin, error) {
	resp, err := jc.DoRequest("GET", "/Plugins", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query plugins: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get plugins: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var plugins []Plugin
	if err := json.NewDecoder(resp.Body).Decode(&plugins); err != nil {
		return nil, fmt.Errorf("failed to decode plugins response: %w", err)
	}

	return plugins, nil
}

// Plugin represents a Jellyfin plugin
type Plugin struct {
	Name        string `json:"Name"`
	Version     string `json:"Version"`
	ID          string `json:"Id"`
	Description string `json:"Description"`
	Status      string `json:"Status"`
}
