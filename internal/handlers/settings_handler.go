package handlers

import (
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/AndrewDonelson/track-studio-orchestrator/internal/database"
	"github.com/AndrewDonelson/track-studio-orchestrator/internal/models"
	"github.com/AndrewDonelson/track-studio-orchestrator/internal/utils"
	"github.com/gin-gonic/gin"
)

// SettingsHandler handles settings-related requests
type SettingsHandler struct {
	repo *database.SettingsRepository
}

// NewSettingsHandler creates a new settings handler
func NewSettingsHandler(repo *database.SettingsRepository) *SettingsHandler {
	return &SettingsHandler{repo: repo}
}

// Get returns the application settings
func (h *SettingsHandler) Get(c *gin.Context) {
	settings, err := h.repo.Get()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, settings)
}

// Update updates the application settings
func (h *SettingsHandler) Update(c *gin.Context) {
	var settings models.Settings
	if err := c.ShouldBindJSON(&settings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Force ID to 1 (singleton settings)
	settings.ID = 1

	if err := h.repo.Update(&settings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return updated settings
	updated, err := h.repo.Get()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, updated)
}

// UploadLogo handles brand logo uploads
func (h *SettingsHandler) UploadLogo(c *gin.Context) {
	// Create branding directory
	brandingDir := filepath.Join(utils.GetDataPath(), "branding")
	if err := os.MkdirAll(brandingDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create branding directory: " + err.Error()})
		return
	}

	// Get uploaded file
	file, header, err := c.Request.FormFile("logo")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}
	defer file.Close()

	// Validate file type
	ext := filepath.Ext(header.Filename)
	if ext != ".png" && ext != ".jpg" && ext != ".jpeg" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only PNG and JPG files are allowed"})
		return
	}

	// Save as artist-logo.png (overwrite existing)
	logoPath := filepath.Join(brandingDir, "artist-logo.png")
	destFile, err := os.Create(logoPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save logo: " + err.Error()})
		return
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to write logo: " + err.Error()})
		return
	}

	// Update settings with logo path
	settings, err := h.repo.Get()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	settings.BrandLogoPath = "branding/artist-logo.png"
	if err := h.repo.Update(settings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Logo uploaded successfully",
		"path":    settings.BrandLogoPath,
	})
}
