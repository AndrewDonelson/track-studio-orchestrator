package handlers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/AndrewDonelson/track-studio-orchestrator/internal/database"
	"github.com/AndrewDonelson/track-studio-orchestrator/internal/services/ai"
	"github.com/AndrewDonelson/track-studio-orchestrator/internal/utils"
	"github.com/AndrewDonelson/track-studio-orchestrator/pkg/audio"
	"github.com/gin-gonic/gin"
)

// AudioHandler handles audio analysis requests
type AudioHandler struct {
	songRepo *database.SongRepository
	aiClient *ai.Client
}

// NewAudioHandler creates a new audio handler
func NewAudioHandler(songRepo *database.SongRepository, aiClient *ai.Client) *AudioHandler {
	return &AudioHandler{
		songRepo: songRepo,
		aiClient: aiClient,
	}
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

	// Get audio file path using convention (prefer instrumental for BPM)
	audioPath := utils.GetSongAudioPath(id)
	if audioPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No audio file available for analysis. Please upload audio files first."})
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

	// Perform AI metadata enrichment (if AI client is configured)
	var enrichment interface{} = nil
	if h.aiClient != nil {
		log.Printf("Enriching metadata for song %d after analysis", id)
		enrich, err := h.aiClient.EnrichSongMetadata(song)
		if err != nil {
			log.Printf("Warning: Failed to enrich metadata: %v", err)
			// Don't fail the whole request, just log and continue
		} else {
			// Save enrichment to database
			if err := h.songRepo.UpdateMetadataEnrichment(id, enrich); err != nil {
				log.Printf("Warning: Failed to save enrichment: %v", err)
			} else {
				enrichment = enrich
				log.Printf("Successfully enriched metadata for song %d", id)
			}
		}
	}

	// Return the updated song with analysis results
	response := gin.H{
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
	}

	if enrichment != nil {
		response["enrichment"] = enrichment
	}

	c.JSON(http.StatusOK, response)
}
