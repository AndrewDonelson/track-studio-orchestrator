package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/AndrewDonelson/track-studio-orchestrator/internal/database"
	"github.com/AndrewDonelson/track-studio-orchestrator/internal/utils"
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
	songAudioDir := fmt.Sprintf("song_%d", id)
	audioDir := filepath.Join(utils.GetAudioPath(), songAudioDir)
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

		// Remove any existing vocal files with different extensions
		for _, oldExt := range []string{".wav", ".mp3", ".flac", ".m4a"} {
			oldPath := filepath.Join(audioDir, "vocal"+oldExt)
			if oldExt != ext {
				os.Remove(oldPath) // Ignore errors if file doesn't exist
			}
		}

		// Save vocals file with absolute path
		vocalsPath := filepath.Join(audioDir, "vocal"+ext)
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

		// File saved successfully
		updatedPaths["vocals"] = vocalsPath
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

		// Remove any existing music files with different extensions
		for _, oldExt := range []string{".wav", ".mp3", ".flac", ".m4a"} {
			oldPath := filepath.Join(audioDir, "music"+oldExt)
			if oldExt != ext {
				os.Remove(oldPath) // Ignore errors if file doesn't exist
			}
		}

		// Save music file with absolute path
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

		// File saved successfully
		updatedPaths["music"] = musicPath
	}

	// Check if at least one file was uploaded
	// Check if at least one file was uploaded
	if len(updatedPaths) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No audio files provided. Include 'vocals' and/or 'music' in the form data."})
		return
	}

	// No need to update database - paths are convention-based

	c.JSON(http.StatusOK, gin.H{
		"message":        "Audio files uploaded successfully",
		"song_id":        id,
		"uploaded_paths": updatedPaths,
	})
}
