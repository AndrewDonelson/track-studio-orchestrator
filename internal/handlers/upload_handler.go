package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/AndrewDonelson/track-studio-orchestrator/internal/database"
	"github.com/gin-gonic/gin"
)

// UploadHandler handles file upload requests
type UploadHandler struct {
	songRepo *database.SongRepository
}

// NewUploadHandler creates a new upload handler
func NewUploadHandler(songRepo *database.SongRepository) *UploadHandler {
	return &UploadHandler{songRepo: songRepo}
}

// UploadAudio handles audio file uploads for a song
func (h *UploadHandler) UploadAudio(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid song ID"})
		return
	}

	// Get song from database
	song, err := h.songRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if song == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Song not found"})
		return
	}

	// Create storage directory for this song's audio files
	audioDir := filepath.Join("storage", "audio", fmt.Sprintf("song_%d", id))
	if err := os.MkdirAll(audioDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create storage directory: " + err.Error()})
		return
	}

	var updatedPaths = make(map[string]string)

	// Handle vocals file upload
	vocalsFile, vocalsHeader, err := c.Request.FormFile("vocals")
	if err == nil {
		defer vocalsFile.Close()

		// Determine file extension
		ext := filepath.Ext(vocalsHeader.Filename)
		if ext == "" {
			ext = ".mp3" // default
		}

		// Save vocals file
		vocalsPath := filepath.Join(audioDir, "vocals"+ext)
		destFile, err := os.Create(vocalsPath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save vocals file: " + err.Error()})
			return
		}
		defer destFile.Close()

		if _, err := io.Copy(destFile, vocalsFile); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to write vocals file: " + err.Error()})
			return
		}

		// Get absolute path
		absPath, _ := filepath.Abs(vocalsPath)
		song.VocalsStemPath = absPath
		updatedPaths["vocals"] = absPath
	}

	// Handle music/instrumental file upload
	musicFile, musicHeader, err := c.Request.FormFile("music")
	if err == nil {
		defer musicFile.Close()

		// Determine file extension
		ext := filepath.Ext(musicHeader.Filename)
		if ext == "" {
			ext = ".mp3" // default
		}

		// Save music file
		musicPath := filepath.Join(audioDir, "music"+ext)
		destFile, err := os.Create(musicPath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save music file: " + err.Error()})
			return
		}
		defer destFile.Close()

		if _, err := io.Copy(destFile, musicFile); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to write music file: " + err.Error()})
			return
		}

		// Get absolute path
		absPath, _ := filepath.Abs(musicPath)
		song.MusicStemPath = absPath
		updatedPaths["music"] = absPath
	}

	// Check if at least one file was uploaded
	if len(updatedPaths) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No audio files provided. Include 'vocals' and/or 'music' in the form data."})
		return
	}

	// Update song in database
	if err := h.songRepo.Update(song); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update song: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "Audio files uploaded successfully",
		"song":           song,
		"uploaded_paths": updatedPaths,
	})
}
