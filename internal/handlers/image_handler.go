package handlers

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/AndrewDonelson/track-studio-orchestrator/internal/database"
	"github.com/AndrewDonelson/track-studio-orchestrator/internal/models"
	"github.com/AndrewDonelson/track-studio-orchestrator/internal/utils"
	"github.com/AndrewDonelson/track-studio-orchestrator/pkg/image"

	"github.com/gin-gonic/gin"
)

type ImageHandler struct {
	settingsRepo *database.SettingsRepository
}

func NewImageHandler(settingsRepo *database.SettingsRepository) *ImageHandler {
	return &ImageHandler{
		settingsRepo: settingsRepo,
	}
}

// GetImagesBySong returns all images for a song
func (h *ImageHandler) GetImagesBySong(c *gin.Context) {
	songID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid song ID"})
		return
	}

	images, err := database.GetImagesBySongID(songID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if images == nil {
		images = []models.GeneratedImage{}
	}

	c.JSON(http.StatusOK, images)
}

// CreateImagePrompt creates a new image record with just a prompt (no actual image yet)
func (h *ImageHandler) CreateImagePrompt(c *gin.Context) {
	songID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid song ID"})
		return
	}

	var req models.GeneratedImage
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Ensure song_id matches the URL parameter
	req.SongID = songID

	// Create the image record
	id, err := database.CreateImagePrompt(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	req.ID = id
	c.JSON(http.StatusCreated, req)
}

// DeleteImagesBySong deletes all images for a song
func (h *ImageHandler) DeleteImagesBySong(c *gin.Context) {
	songID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid song ID"})
		return
	}

	if err := database.DeleteImagesBySongID(songID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "All images deleted successfully"})
}

// UpdateImagePrompt updates the prompt for an image
func (h *ImageHandler) UpdateImagePrompt(c *gin.Context) {
	imageID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid image ID"})
		return
	}

	var req struct {
		Prompt         string `json:"prompt"`
		NegativePrompt string `json:"negative_prompt"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := database.UpdateImagePrompt(imageID, req.Prompt, req.NegativePrompt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Image prompt updated"})
}

// RegenerateImage triggers regeneration of a specific image
func (h *ImageHandler) RegenerateImage(c *gin.Context) {
	imageID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid image ID"})
		return
	}

	image, err := database.GetImageByID(imageID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if image == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Image not found"})
		return
	}

	// Regeneration happens in a goroutine to avoid blocking
	go h.regenerateImageAsync(image)

	c.JSON(http.StatusAccepted, gin.H{
		"message":  "Image regeneration started",
		"image_id": strconv.Itoa(imageID),
	})
}

// regenerateImageAsync regenerates an image in the background
func (h *ImageHandler) regenerateImageAsync(img *models.GeneratedImage) {
	log.Printf("Starting image regeneration for ID %d", img.ID)

	// Load settings for master prompts
	settings, err := h.settingsRepo.Get()
	if err != nil {
		log.Printf("Warning: failed to load settings: %v, using defaults", err)
	}

	// Setup image generator with the correct output directory
	outputDir := filepath.Join(utils.GetImagesPath(), fmt.Sprintf("song_%d", img.SongID))
	imageGen := image.NewImageGenerator(outputDir)

	// Set master prompts from settings if available
	if settings != nil {
		if settings.MasterPrompt != "" {
			imageGen.MasterPrompt = settings.MasterPrompt
		}
		if settings.MasterNegativePrompt != "" {
			imageGen.MasterNegative = settings.MasterNegativePrompt
		}
	}

	// Generate filename based on image type if path is empty
	var filename string
	if img.ImagePath != "" && img.ImagePath != "." {
		filename = filepath.Base(img.ImagePath)
		// Delete old image file if it exists
		fullPath := filepath.Join(utils.GetDataPath(), img.ImagePath)
		if err := os.Remove(fullPath); err != nil {
			log.Printf("Warning: failed to delete old image file %s: %v", fullPath, err)
		}
	} else {
		// Generate filename from image type, including sequence number if present
		if img.SequenceNumber != nil && *img.SequenceNumber > 0 {
			filename = fmt.Sprintf("bg-%s-%d.png", img.ImageType, *img.SequenceNumber)
		} else {
			filename = fmt.Sprintf("bg-%s.png", img.ImageType)
		}
		log.Printf("No existing image path, using generated filename: %s", filename)
	}

	// Generate new image with the updated prompt and custom negative prompt
	log.Printf("Regenerating image %s with prompt: %s", filename, img.Prompt)
	negPrompt := ""
	if img.NegativePrompt != nil {
		negPrompt = *img.NegativePrompt
		log.Printf("Custom negative prompt: %s", negPrompt)
	}
	newPath, err := imageGen.GenerateImageWithNegative(img.Prompt, negPrompt, filename)
	if err != nil {
		log.Printf("Error regenerating image: %v", err)
		return
	}

	log.Printf("Image regenerated successfully: %s", newPath)

	// Update database with the relative path from data directory
	dataPath := utils.GetDataPath()
	relativePath := strings.TrimPrefix(newPath, dataPath+"/")
	if err := database.UpdateImagePath(img.ID, relativePath); err != nil {
		log.Printf("Error updating image path in database: %v", err)
		return
	}

	log.Printf("Database updated with path: %s", relativePath)
}

// GeneratePromptFromLyrics generates an image prompt from lyrics using LLM
func (h *ImageHandler) GeneratePromptFromLyrics(c *gin.Context) {
	var req struct {
		Lyrics          string `json:"lyrics"`
		SectionType     string `json:"section_type"`
		Genre           string `json:"genre"`
		BackgroundStyle string `json:"background_style"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Lyrics == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Lyrics text is required"})
		return
	}

	// Load settings for master prompts
	settings, err := h.settingsRepo.Get()
	if err != nil {
		log.Printf("Warning: failed to load settings: %v, using defaults", err)
	}

	// Create temporary image generator just for prompt enhancement
	imageGen := image.NewImageGenerator("")

	// Set master prompts from settings if available
	if settings != nil {
		if settings.MasterPrompt != "" {
			imageGen.MasterPrompt = settings.MasterPrompt
		}
		if settings.MasterNegativePrompt != "" {
			imageGen.MasterNegative = settings.MasterNegativePrompt
		}
	}

	// Build style keywords
	styleKeywords := image.BuildStyleKeywords(req.Genre, req.BackgroundStyle)

	// Use the LLM to enhance the prompt based on lyrics
	log.Printf("Generating prompt for %s section from lyrics", req.SectionType)
	enhancedPrompt, promptErr := imageGen.EnhancePromptWithLLM(req.SectionType, req.Lyrics, styleKeywords)
	if promptErr != nil {
		log.Printf("Error generating prompt: %v", promptErr)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate prompt: " + promptErr.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"prompt":          enhancedPrompt,
		"negative_prompt": "",
	})
}
