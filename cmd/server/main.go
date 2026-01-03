package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/AndrewDonelson/track-studio-orchestrator/config"
	"github.com/AndrewDonelson/track-studio-orchestrator/internal/database"
	"github.com/gin-gonic/gin"
)

func main() {
	fmt.Println("Track Studio Orchestrator")
	fmt.Println("Copyright 2017-2026 Nlaak Studios")

	// Load configuration
	cfg := config.LoadConfig()
	log.Printf("Environment: %s", cfg.Environment)
	log.Printf("Server port: %d", cfg.ServerPort)

	// Initialize database
	if err := database.InitDB(cfg.DBPath); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// Apply schema if database is new
	if _, err := os.Stat(cfg.DBPath); err == nil {
		schemaPath := filepath.Join("scripts", "schema.sql")
		if err := database.ExecSchema(schemaPath); err != nil {
			log.Printf("Warning: Failed to apply schema: %v", err)
		}
	}

	// Create Gin router
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "track-studio-orchestrator",
		})
	})

	// API v1 group
	v1 := router.Group("/api/v1")
	{
		// Songs endpoints
		songs := v1.Group("/songs")
		{
			songs.GET("", getSongs)
			songs.GET("/:id", getSong)
			songs.POST("", createSong)
			songs.PUT("/:id", updateSong)
			songs.DELETE("/:id", deleteSong)
		}

		// Queue endpoints
		queue := v1.Group("/queue")
		{
			queue.GET("", getQueue)
			queue.POST("", addToQueue)
			queue.GET("/:id", getQueueItem)
			queue.PUT("/:id", updateQueueItem)
		}

		// Albums endpoints
		albums := v1.Group("/albums")
		{
			albums.GET("", getAlbums)
			albums.GET("/:id", getAlbum)
			albums.POST("", createAlbum)
		}

		// Artists endpoints
		artists := v1.Group("/artists")
		{
			artists.GET("", getArtists)
			artists.GET("/:id", getArtist)
			artists.POST("", createArtist)
		}
	}

	// Start server
	addr := fmt.Sprintf(":%d", cfg.ServerPort)
	log.Printf("Starting server on %s", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// Placeholder handlers - to be implemented
func getSongs(c *gin.Context) {
	c.JSON(200, gin.H{"songs": []interface{}{}})
}

func getSong(c *gin.Context) {
	c.JSON(200, gin.H{"song": nil})
}

func createSong(c *gin.Context) {
	c.JSON(201, gin.H{"created": true})
}

func updateSong(c *gin.Context) {
	c.JSON(200, gin.H{"updated": true})
}

func deleteSong(c *gin.Context) {
	c.JSON(200, gin.H{"deleted": true})
}

func getQueue(c *gin.Context) {
	c.JSON(200, gin.H{"queue": []interface{}{}})
}

func addToQueue(c *gin.Context) {
	c.JSON(201, gin.H{"queued": true})
}

func getQueueItem(c *gin.Context) {
	c.JSON(200, gin.H{"item": nil})
}

func updateQueueItem(c *gin.Context) {
	c.JSON(200, gin.H{"updated": true})
}

func getAlbums(c *gin.Context) {
	c.JSON(200, gin.H{"albums": []interface{}{}})
}

func getAlbum(c *gin.Context) {
	c.JSON(200, gin.H{"album": nil})
}

func createAlbum(c *gin.Context) {
	c.JSON(201, gin.H{"created": true})
}

func getArtists(c *gin.Context) {
	c.JSON(200, gin.H{"artists": []interface{}{}})
}

func getArtist(c *gin.Context) {
	c.JSON(200, gin.H{"artist": nil})
}

func createArtist(c *gin.Context) {
	c.JSON(201, gin.H{"created": true})
}
