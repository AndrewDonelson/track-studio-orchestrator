package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// Config holds all application configuration
type Config struct {
	Environment string
	ServerPort  int
	DBPath      string

	// Storage paths
	StoragePath string
	SongsPath   string
	VideosPath  string
	TempPath    string

	// CQAI settings
	CQAIURL    string
	LLMModel   string
	ImageModel string

	// Image generation settings
	ImageWidth  int
	ImageHeight int
	ImageSteps  int
}

// LoadConfig loads configuration based on environment
func LoadConfig() *Config {
	env := os.Getenv("TRACK_STUDIO_ENV")
	if env == "" {
		env = "development"
	}

	var cfg Config
	cfg.Environment = env

	if env == "production" {
		// Production paths (on mule)
		cfg.ServerPort = 8080
		cfg.DBPath = "/home/andrew/trackstudio/orchestrator/data/trackstudio.db"
		cfg.StoragePath = "/home/andrew/trackstudio/orchestrator/storage"
	} else {
		// Development paths
		cfg.ServerPort = 8080
		homeDir, _ := os.UserHomeDir()
		basePath := filepath.Join(homeDir, "Development", "Fullstack-Projects", "TrackStudio", "track-studio-orchestrator")
		cfg.DBPath = filepath.Join(basePath, "data", "trackstudio.db")
		cfg.StoragePath = filepath.Join(basePath, "storage")
	}

	// Derived storage paths
	cfg.SongsPath = filepath.Join(cfg.StoragePath, "songs")
	cfg.VideosPath = filepath.Join(cfg.StoragePath, "videos")
	cfg.TempPath = filepath.Join(cfg.StoragePath, "temp")

	// CQAI configuration
	cfg.CQAIURL = "http://cqai.nlaakstudios"
	cfg.LLMModel = "qwen2.5:7b"
	cfg.ImageModel = "z-image-nsfw"

	// Image generation settings (verified working)
	cfg.ImageWidth = 1920
	cfg.ImageHeight = 1024
	cfg.ImageSteps = 25

	fmt.Printf("Loaded configuration for environment: %s\n", env)
	return &cfg
}
