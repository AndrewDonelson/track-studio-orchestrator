package handlers

import (
	"net/http"
	"strconv"

	"github.com/AndrewDonelson/track-studio-orchestrator/internal/database"
	"github.com/gin-gonic/gin"
)

type VideoHandler struct {
	repo *database.VideoRepository
}

func NewVideoHandler(repo *database.VideoRepository) *VideoHandler {
	return &VideoHandler{repo: repo}
}

// GetAll returns all videos
func (h *VideoHandler) GetAll(c *gin.Context) {
	videos, err := h.repo.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, videos)
}

// GetBySongID returns all videos for a specific song
func (h *VideoHandler) GetBySongID(c *gin.Context) {
	songID, err := strconv.Atoi(c.Param("songId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid song ID"})
		return
	}

	videos, err := h.repo.GetBySongID(songID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, videos)
}

// Delete marks a video as deleted
func (h *VideoHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	if err := h.repo.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Video deleted"})
}
