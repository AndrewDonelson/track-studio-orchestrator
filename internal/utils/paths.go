package utils

import (
	"os"
	"path/filepath"
	"strings"
)

// GetDataPath returns the configured data storage path
// It expands ~ to home directory and uses ~/track-studio-data as default
func GetDataPath() string {
	// Try to get from environment variable first
	dataPath := os.Getenv("TRACK_STUDIO_DATA_PATH")

	if dataPath == "" {
		// Default to ~/track-studio-data
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "/tmp/track-studio-data"
		}
		dataPath = filepath.Join(homeDir, "track-studio-data")
	}

	// Expand ~ if present
	if strings.HasPrefix(dataPath, "~/") {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			dataPath = filepath.Join(homeDir, dataPath[2:])
		}
	}

	return dataPath
}

// GetImagesPath returns the images storage directory
func GetImagesPath() string {
	return filepath.Join(GetDataPath(), "images")
}

// GetVideosPath returns the videos storage directory
func GetVideosPath() string {
	return filepath.Join(GetDataPath(), "videos")
}

// GetAudioPath returns the audio storage directory
func GetAudioPath() string {
	return filepath.Join(GetDataPath(), "audio")
}

// GetTempPath returns the temporary files directory
func GetTempPath() string {
	return filepath.Join(GetDataPath(), "temp")
}

// GetBrandingPath returns the branding assets directory
func GetBrandingPath() string {
	return filepath.Join(GetDataPath(), "branding")
}

// EnsureDataDirectories creates all necessary data directories if they don't exist
func EnsureDataDirectories() error {
	dirs := []string{
		GetImagesPath(),
		GetVideosPath(),
		GetAudioPath(),
		GetTempPath(),
		GetBrandingPath(),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return nil
}
