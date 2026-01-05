package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/AndrewDonelson/track-studio-orchestrator/internal/database"
	"github.com/AndrewDonelson/track-studio-orchestrator/internal/services/ai"
	"github.com/gin-gonic/gin"
)

type EnrichmentHandler struct {
	songRepo *database.SongRepository
	aiClient *ai.Client
}

func NewEnrichmentHandler(songRepo *database.SongRepository, aiClient *ai.Client) *EnrichmentHandler {
	return &EnrichmentHandler{
		songRepo: songRepo,
		aiClient: aiClient,
	}
}

// EnrichSongMetadata enriches a single song with AI-generated metadata
func (h *EnrichmentHandler) EnrichSongMetadata(c *gin.Context) {
	songID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid song ID"})
		return
	}

	// Get the song
	song, err := h.songRepo.GetByID(songID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Song not found"})
		return
	}

	// Check if already enriched (unless force_refresh is true)
	type EnrichRequest struct {
		ForceRefresh bool `json:"force_refresh"`
	}
	var req EnrichRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Default to false if no body provided
		req.ForceRefresh = false
	}

	if song.MetadataEnrichedAt != nil && !req.ForceRefresh {
		c.JSON(http.StatusOK, gin.H{
			"message": "Song already enriched (use force_refresh: true to re-enrich)",
			"song_id": songID,
		})
		return
	}

	log.Printf("Enriching metadata for song %d: %s", songID, song.Title)

	// Call AI to generate metadata
	enrichment, err := h.aiClient.EnrichSongMetadata(song)
	if err != nil {
		log.Printf("Error enriching song %d: %v", songID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to enrich metadata: %v", err)})
		return
	}

	// Update the database
	if err := h.songRepo.UpdateMetadataEnrichment(songID, enrichment); err != nil {
		log.Printf("Error saving enrichment for song %d: %v", songID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save enrichment"})
		return
	}

	log.Printf("Successfully enriched metadata for song %d", songID)

	c.JSON(http.StatusOK, gin.H{
		"message":    "Metadata enriched successfully",
		"song_id":    songID,
		"enrichment": enrichment,
	})
}

// EnrichBatch enriches multiple songs in batch
func (h *EnrichmentHandler) EnrichBatch(c *gin.Context) {
	type BatchRequest struct {
		SongIDs      []int `json:"song_ids"`
		ForceRefresh bool  `json:"force_refresh"`
	}

	var req BatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if len(req.SongIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No song IDs provided"})
		return
	}

	results := make([]map[string]interface{}, 0)
	successCount := 0
	errorCount := 0

	for _, songID := range req.SongIDs {
		// Get the song
		song, err := h.songRepo.GetByID(songID)
		if err != nil {
			errorCount++
			results = append(results, map[string]interface{}{
				"song_id": songID,
				"status":  "error",
				"message": "Song not found",
			})
			continue
		}

		// Skip if already enriched (unless force_refresh)
		if song.MetadataEnrichedAt != nil && !req.ForceRefresh {
			results = append(results, map[string]interface{}{
				"song_id": songID,
				"status":  "skipped",
				"message": "Already enriched",
			})
			continue
		}

		// Call AI to generate metadata
		enrichment, err := h.aiClient.EnrichSongMetadata(song)
		if err != nil {
			log.Printf("Error enriching song %d: %v", songID, err)
			errorCount++
			results = append(results, map[string]interface{}{
				"song_id": songID,
				"status":  "error",
				"message": fmt.Sprintf("AI enrichment failed: %v", err),
			})
			continue
		}

		// Update the database
		if err := h.songRepo.UpdateMetadataEnrichment(songID, enrichment); err != nil {
			log.Printf("Error saving enrichment for song %d: %v", songID, err)
			errorCount++
			results = append(results, map[string]interface{}{
				"song_id": songID,
				"status":  "error",
				"message": "Failed to save enrichment",
			})
			continue
		}

		successCount++
		results = append(results, map[string]interface{}{
			"song_id": songID,
			"status":  "success",
			"title":   song.Title,
		})

		log.Printf("Successfully enriched song %d: %s", songID, song.Title)
	}

	c.JSON(http.StatusOK, gin.H{
		"total":   len(req.SongIDs),
		"success": successCount,
		"errors":  errorCount,
		"results": results,
	})
}

// GetEnrichmentStatus returns the enrichment status for all songs
func (h *EnrichmentHandler) GetEnrichmentStatus(c *gin.Context) {
	songs, err := h.songRepo.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch songs"})
		return
	}

	enrichedCount := 0
	unenrichedSongs := make([]map[string]interface{}, 0)

	for _, song := range songs {
		if song.MetadataEnrichedAt != nil {
			enrichedCount++
		} else {
			unenrichedSongs = append(unenrichedSongs, map[string]interface{}{
				"id":     song.ID,
				"title":  song.Title,
				"artist": song.ArtistName,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"total_songs":      len(songs),
		"enriched_count":   enrichedCount,
		"unenriched_count": len(unenrichedSongs),
		"unenriched_songs": unenrichedSongs,
	})
}
