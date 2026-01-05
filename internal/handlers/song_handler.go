package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/AndrewDonelson/track-studio-orchestrator/internal/database"
	"github.com/AndrewDonelson/track-studio-orchestrator/internal/models"
	"github.com/gin-gonic/gin"
)

// SongHandler handles song-related requests
type SongHandler struct {
	repo *database.SongRepository
}

// NewSongHandler creates a new song handler
func NewSongHandler(repo *database.SongRepository) *SongHandler {
	return &SongHandler{repo: repo}
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

	// Check vocals stem
	if song.VocalsStemPath != "" {
		if _, err := os.Stat(song.VocalsStemPath); os.IsNotExist(err) {
			result["valid"] = false
			result["vocals_missing"] = song.VocalsStemPath

			// Try to find similar files
			if suggested := findSimilarFile(song.VocalsStemPath); suggested != "" {
				result["vocals_suggested"] = suggested
			}
		} else {
			result["vocals_ok"] = song.VocalsStemPath
		}
	}

	// Check music stem
	if song.MusicStemPath != "" {
		if _, err := os.Stat(song.MusicStemPath); os.IsNotExist(err) {
			result["valid"] = false
			result["music_missing"] = song.MusicStemPath

			// Try to find similar files
			if suggested := findSimilarFile(song.MusicStemPath); suggested != "" {
				result["music_suggested"] = suggested
			}
		} else {
			result["music_ok"] = song.MusicStemPath
		}
	}

	// Check mixed audio
	if song.MixedAudioPath != "" {
		if _, err := os.Stat(song.MixedAudioPath); os.IsNotExist(err) {
			result["valid"] = false
			result["mixed_missing"] = song.MixedAudioPath

			if suggested := findSimilarFile(song.MixedAudioPath); suggested != "" {
				result["mixed_suggested"] = suggested
			}
		} else {
			result["mixed_ok"] = song.MixedAudioPath
		}
	}

	c.JSON(http.StatusOK, result)
}

// findSimilarFile attempts to find a similar file in the same directory
func findSimilarFile(missingPath string) string {
	dir := filepath.Dir(missingPath)
	baseName := filepath.Base(missingPath)
	ext := filepath.Ext(baseName)
	nameWithoutExt := strings.TrimSuffix(baseName, ext)

	// Check if directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return ""
	}

	// List files in directory
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	// Look for files with similar names
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		fileName := entry.Name()

		// Check if it has the same extension and contains similar name parts
		if filepath.Ext(fileName) == ext {
			// Case-insensitive substring match
			if strings.Contains(strings.ToLower(fileName), strings.ToLower(nameWithoutExt)) ||
				strings.Contains(strings.ToLower(nameWithoutExt), strings.ToLower(strings.TrimSuffix(fileName, ext))) {
				return filepath.Join(dir, fileName)
			}
		}
	}

	return ""
}
