package handlers

import (
	"net/http"
	"strconv"

	"github.com/AndrewDonelson/track-studio-orchestrator/internal/database"
	"github.com/AndrewDonelson/track-studio-orchestrator/internal/models"

	"github.com/gin-gonic/gin"
)

type ImageHandler struct {
}

func NewImageHandler() *ImageHandler {
	return &ImageHandler{}
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

	// TODO: Trigger image regeneration with the updated prompt
	// This would call the image generation service with the new prompt

	c.JSON(http.StatusAccepted, gin.H{
		"message":  "Image regeneration queued",
		"image_id": strconv.Itoa(imageID),
	})
}
