package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/AndrewDonelson/track-studio-orchestrator/config"
	"github.com/AndrewDonelson/track-studio-orchestrator/internal/database"
	"github.com/AndrewDonelson/track-studio-orchestrator/internal/models"
	"github.com/AndrewDonelson/track-studio-orchestrator/internal/utils"
	"github.com/gin-gonic/gin"
)

// SongHandler handles song-related requests
type SongHandler struct {
	repo   *database.SongRepository
	config *config.Config
}

// NewSongHandler creates a new song handler
func NewSongHandler(repo *database.SongRepository, cfg *config.Config) *SongHandler {
	return &SongHandler{
		repo:   repo,
		config: cfg,
	}
}

// GetAll returns all songs
func (h *SongHandler) GetAll(c *gin.Context) {
	songs, err := h.repo.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"songs": songs})
}

// GetByID returns a song by ID
func (h *SongHandler) GetByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	song, err := h.repo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if song == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Song not found"})
		return
	}

	c.JSON(http.StatusOK, song)
}

// Create creates a new song
func (h *SongHandler) Create(c *gin.Context) {
	var song models.Song
	if err := c.ShouldBindJSON(&song); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.repo.Create(&song); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, song)
}

// Update updates an existing song
func (h *SongHandler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var song models.Song
	if err := c.ShouldBindJSON(&song); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	song.ID = id
	if err := h.repo.Update(&song); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, song)
}

// Delete deletes a song
func (h *SongHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	if err := h.repo.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Song deleted"})
}

// ValidateAudioPaths validates that audio files exist and suggests fixes
func (h *SongHandler) ValidateAudioPaths(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	song, err := h.repo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if song == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Song not found"})
		return
	}

	result := gin.H{
		"song_id": song.ID,
		"title":   song.Title,
		"valid":   true,
		"issues":  []string{},
	}

	// Check vocals stem using convention-based path
	if vocalPath := utils.GetSongVocalPath(song.ID); vocalPath != "" {
		result["vocals_ok"] = fmt.Sprintf("song_%d/vocal", song.ID)
	}

	// Check music stem using convention-based path
	if musicPath := utils.GetSongMusicPath(song.ID); musicPath != "" {
		result["music_ok"] = fmt.Sprintf("song_%d/music", song.ID)
	}

	// Check mixed audio using convention-based path
	if mixedPath := utils.GetSongMixedPath(song.ID); mixedPath != "" {
		result["mixed_ok"] = fmt.Sprintf("song_%d/mixed", song.ID)
	}

	// Check if any audio exists
	if !utils.HasSongAudio(song.ID) {
		result["valid"] = false
		result["error"] = "No audio files found for this song"
	}

	c.JSON(http.StatusOK, result)
}

// GetRenderLog returns the render log for a song
func (h *SongHandler) GetRenderLog(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	// Build log file path: /storage/logs/{song_id}/log.txt
	logPath := filepath.Join(h.config.LogsPath, fmt.Sprintf("%d", id), "log.txt")

	// Check if log exists
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "No render log found for this song",
			"path":  logPath,
		})
		return
	}

	// Read log file
	content, err := os.ReadFile(logPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to read log: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"song_id": id,
		"log":     string(content),
		"path":    logPath,
	})
}
