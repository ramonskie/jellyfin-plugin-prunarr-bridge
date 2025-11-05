package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/prunarr/jellyfin-sidecar/internal/api"
	"github.com/prunarr/jellyfin-sidecar/internal/config"
)

var (
	version   = "1.0.0"
	buildTime = "unknown"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "/etc/jellyfin-sidecar/config.json", "Path to configuration file")
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	if *showVersion {
		log.Printf("Jellyfin Sidecar Service v%s (built: %s)", version, buildTime)
		os.Exit(0)
	}

	// Load configuration
	log.Printf("Loading configuration from %s", *configPath)
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Validate configuration
	if err := validateConfig(cfg); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	log.Printf("Starting Jellyfin Sidecar Service v%s", version)
	log.Printf("Jellyfin URL: %s", cfg.Jellyfin.URL)
	log.Printf("Symlink Base Path: %s", cfg.Symlink.BasePath)
	log.Printf("Virtual Folder Name: %s", cfg.Symlink.VirtualFolderName)

	// Create API server
	server := api.NewServer(cfg)

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := server.Start(); err != nil {
			errChan <- err
		}
	}()

	// Wait for shutdown signal or error
	select {
	case sig := <-sigChan:
		log.Printf("Received signal %v, shutting down gracefully...", sig)
	case err := <-errChan:
		log.Fatalf("Server error: %v", err)
	}

	log.Println("Shutdown complete")
}

func validateConfig(cfg *config.Config) error {
	if cfg.Jellyfin.URL == "" {
		log.Fatal("jellyfin.url is required")
	}
	if cfg.Jellyfin.APIKey == "" {
		log.Fatal("jellyfin.api_key is required")
	}
	if cfg.Symlink.BasePath == "" {
		log.Fatal("symlink.base_path is required")
	}
	return nil
}
