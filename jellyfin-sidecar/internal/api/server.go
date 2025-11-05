package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/prunarr/jellyfin-sidecar/internal/config"
	"github.com/prunarr/jellyfin-sidecar/internal/jellyfin"
	"github.com/prunarr/jellyfin-sidecar/internal/symlink"
)

// Server represents the API server
type Server struct {
	config         *config.Config
	symlinkManager *symlink.Manager
	jellyfinClient *jellyfin.Client
	httpServer     *http.Server
}

// NewServer creates a new API server
func NewServer(cfg *config.Config) *Server {
	return &Server{
		config:         cfg,
		symlinkManager: symlink.NewManager(cfg.Symlink.BasePath),
		jellyfinClient: jellyfin.NewClient(cfg.Jellyfin.URL, cfg.Jellyfin.APIKey),
	}
}

// AddItemsRequest represents the request to add items
type AddItemsRequest struct {
	Items []MediaItem `json:"items"`
}

// MediaItem represents a media item to be added
type MediaItem struct {
	SourcePath   string     `json:"source_path"`
	DeletionDate *time.Time `json:"deletion_date,omitempty"`
}

// AddItemsResponse represents the response for adding items
type AddItemsResponse struct {
	Success         bool     `json:"success"`
	CreatedSymlinks []string `json:"created_symlinks"`
	Errors          []string `json:"errors,omitempty"`
}

// RemoveItemsRequest represents the request to remove items
type RemoveItemsRequest struct {
	SymlinkPaths []string `json:"symlink_paths"`
}

// RemoveItemsResponse represents the response for removing items
type RemoveItemsResponse struct {
	Success         bool     `json:"success"`
	RemovedSymlinks []string `json:"removed_symlinks"`
	Errors          []string `json:"errors,omitempty"`
}

// StatusResponse represents the status response
type StatusResponse struct {
	Version           string `json:"version"`
	SymlinkBasePath   string `json:"symlink_base_path"`
	VirtualFolderName string `json:"virtual_folder_name"`
	JellyfinConnected bool   `json:"jellyfin_connected"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// Start starts the API server
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Register routes
	mux.HandleFunc("/api/leaving-soon/add", s.authMiddleware(s.handleAddItems))
	mux.HandleFunc("/api/leaving-soon/remove", s.authMiddleware(s.handleRemoveItems))
	mux.HandleFunc("/api/leaving-soon/clear", s.authMiddleware(s.handleClearItems))
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/health", s.handleHealth)

	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Starting server on %s", addr)
	return s.httpServer.ListenAndServe()
}

// authMiddleware validates API key
func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			apiKey = r.URL.Query().Get("api_key")
		}

		if s.config.Security.APIKey != "" && apiKey != s.config.Security.APIKey {
			writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
			return
		}

		next(w, r)
	}
}

// handleAddItems handles adding items to the "Leaving Soon" library
func (s *Server) handleAddItems(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "method not allowed"})
		return
	}

	var req AddItemsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	if len(req.Items) == 0 {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "no items provided"})
		return
	}

	var createdSymlinks []string
	var errors []string

	for _, item := range req.Items {
		symlinkPath, err := s.symlinkManager.CreateSymlink(item.SourcePath)
		if err != nil {
			log.Printf("Failed to create symlink for %s: %v", item.SourcePath, err)
			errors = append(errors, fmt.Sprintf("%s: %v", item.SourcePath, err))
			continue
		}
		createdSymlinks = append(createdSymlinks, symlinkPath)
	}

	// Ensure virtual folder exists and trigger refresh
	if len(createdSymlinks) > 0 {
		err := s.jellyfinClient.EnsureVirtualFolder(
			s.config.Symlink.VirtualFolderName,
			s.config.Symlink.CollectionType,
			s.config.Symlink.BasePath,
		)
		if err != nil {
			log.Printf("Failed to ensure virtual folder: %v", err)
			errors = append(errors, fmt.Sprintf("virtual folder: %v", err))
		} else {
			// Trigger library scan
			if err := s.jellyfinClient.RefreshLibrary(); err != nil {
				log.Printf("Failed to refresh library: %v", err)
				errors = append(errors, fmt.Sprintf("library refresh: %v", err))
			}
		}
	}

	writeJSON(w, http.StatusOK, AddItemsResponse{
		Success:         true,
		CreatedSymlinks: createdSymlinks,
		Errors:          errors,
	})
}

// handleRemoveItems handles removing items from the "Leaving Soon" library
func (s *Server) handleRemoveItems(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "method not allowed"})
		return
	}

	var req RemoveItemsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	if len(req.SymlinkPaths) == 0 {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "no symlink paths provided"})
		return
	}

	var removedSymlinks []string
	var errors []string

	for _, path := range req.SymlinkPaths {
		if err := s.symlinkManager.RemoveSymlink(path); err != nil {
			log.Printf("Failed to remove symlink %s: %v", path, err)
			errors = append(errors, fmt.Sprintf("%s: %v", path, err))
			continue
		}
		removedSymlinks = append(removedSymlinks, path)
	}

	writeJSON(w, http.StatusOK, RemoveItemsResponse{
		Success:         true,
		RemovedSymlinks: removedSymlinks,
		Errors:          errors,
	})
}

// handleClearItems handles clearing all items from the "Leaving Soon" library
func (s *Server) handleClearItems(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "method not allowed"})
		return
	}

	if err := s.symlinkManager.ClearSymlinks(); err != nil {
		log.Printf("Failed to clear symlinks: %v", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "All symlinks cleared",
	})
}

// handleStatus handles status requests
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	// Test Jellyfin connection
	connected := false
	if _, err := s.jellyfinClient.GetVirtualFolders(); err == nil {
		connected = true
	}

	writeJSON(w, http.StatusOK, StatusResponse{
		Version:           "1.0.0",
		SymlinkBasePath:   s.config.Symlink.BasePath,
		VirtualFolderName: s.config.Symlink.VirtualFolderName,
		JellyfinConnected: connected,
	})
}

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "healthy",
	})
}

// writeJSON writes a JSON response
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
