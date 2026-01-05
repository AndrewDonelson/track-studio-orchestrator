package database

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"

	"github.com/AndrewDonelson/track-studio-orchestrator/internal/models"
)

// SettingsRepository handles settings data operations
type SettingsRepository struct {
	db *sql.DB
}

// NewSettingsRepository creates a new settings repository
func NewSettingsRepository(db *sql.DB) *SettingsRepository {
	return &SettingsRepository{db: db}
}

// Get retrieves the application settings (always ID = 1)
func (r *SettingsRepository) Get() (*models.Settings, error) {
	query := `
		SELECT id, master_prompt, master_negative_prompt, brand_logo_path, data_storage_path, created_at, updated_at
		FROM settings
		WHERE id = 1
	`

	var settings models.Settings
	err := r.db.QueryRow(query).Scan(
		&settings.ID,
		&settings.MasterPrompt,
		&settings.MasterNegativePrompt,
		&settings.BrandLogoPath,
		&settings.DataStoragePath,
		&settings.CreatedAt,
		&settings.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		// Create default settings if they don't exist
		return r.createDefault()
	}

	if err != nil {
		return nil, err
	}

	return &settings, nil
}

// Update updates the application settings
func (r *SettingsRepository) Update(settings *models.Settings) error {
	// Expand ~ to home directory
	dataPath := settings.DataStoragePath
	if strings.HasPrefix(dataPath, "~/") {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			dataPath = filepath.Join(homeDir, dataPath[2:])
		}
	}

	query := `
		UPDATE settings
		SET master_prompt = ?,
		    master_negative_prompt = ?,
		    brand_logo_path = ?,
		    data_storage_path = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = 1
	`

	_, err := r.db.Exec(query,
		settings.MasterPrompt,
		settings.MasterNegativePrompt,
		settings.BrandLogoPath,
		dataPath,
	)

	return err
}

// createDefault creates default settings
func (r *SettingsRepository) createDefault() (*models.Settings, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "/tmp"
	}
	defaultPath := filepath.Join(homeDir, "track-studio-data")

	query := `
		INSERT INTO settings (id, data_storage_path)
		VALUES (1, ?)
		ON CONFLICT(id) DO UPDATE SET data_storage_path = excluded.data_storage_path
	`

	_, err = r.db.Exec(query, defaultPath)
	if err != nil {
		return nil, err
	}

	return r.Get()
}

// GetDataPath returns the expanded data storage path
func (r *SettingsRepository) GetDataPath() (string, error) {
	settings, err := r.Get()
	if err != nil {
		return "", err
	}

	dataPath := settings.DataStoragePath
	if strings.HasPrefix(dataPath, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return dataPath, nil
		}
		dataPath = filepath.Join(homeDir, dataPath[2:])
	}

	return dataPath, nil
}
