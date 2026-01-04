package handlers

import (
	"net/http"
	"os"
	"strconv"

	"github.com/AndrewDonelson/track-studio-orchestrator/internal/database"
	"github.com/AndrewDonelson/track-studio-orchestrator/pkg/audio"
	"github.com/gin-gonic/gin"
)

// AudioHandler handles audio analysis requests
type AudioHandler struct {
	songRepo *database.SongRepository
}

// NewAudioHandler creates a new audio handler
func NewAudioHandler(songRepo *database.SongRepository) *AudioHandler {
	return &AudioHandler{songRepo: songRepo}
}

// AnalyzeSong performs audio analysis on a song's audio files
func (h *AudioHandler) AnalyzeSong(c *gin.Context) {
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

	// Determine which audio file to analyze (prefer instrumental for BPM)
	audioPath := song.MusicStemPath
	if audioPath == "" {
		audioPath = song.VocalsStemPath
	}
	if audioPath == "" {
		audioPath = song.MixedAudioPath
	}

	if audioPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No audio file available for analysis"})
		return
	}

	// Check if file exists
	if _, err := os.Stat(audioPath); os.IsNotExist(err) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Audio file not found. Please upload audio files first."})
		return
	}

	// Perform audio analysis
	analysis, err := audio.AnalyzeAudio(audioPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Audio analysis failed: " + err.Error()})
		return
	}

	// Update song with analysis results
	song.DurationSeconds = analysis.DurationSeconds
	song.BPM = analysis.BPM
	song.Key = analysis.Key
	song.Tempo = analysis.Tempo
	if song.Genre == "" && analysis.Genre != "" {
		song.Genre = analysis.Genre
	}

	// Save updated song
	if err := h.songRepo.Update(song); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update song: " + err.Error()})
		return
	}

	// Return the updated song with analysis results
	c.JSON(http.StatusOK, gin.H{
		"song": song,
		"analysis": gin.H{
			"duration_seconds":    analysis.DurationSeconds,
			"bpm":                 analysis.BPM,
			"key":                 analysis.Key,
			"tempo":               analysis.Tempo,
			"genre":               analysis.Genre,
			"beat_count":          analysis.BeatCount,
			"vocal_segment_count": analysis.VocalSegmentCount,
		},
	})
}
