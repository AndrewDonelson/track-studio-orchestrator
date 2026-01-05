package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/AndrewDonelson/track-studio-orchestrator/config"
	"github.com/AndrewDonelson/track-studio-orchestrator/internal/database"
	"github.com/AndrewDonelson/track-studio-orchestrator/internal/handlers"
	"github.com/AndrewDonelson/track-studio-orchestrator/internal/services"
	"github.com/AndrewDonelson/track-studio-orchestrator/internal/utils"
	"github.com/AndrewDonelson/track-studio-orchestrator/internal/worker"
	"github.com/gin-gonic/gin"
)

func main() {
	fmt.Println("Track Studio Orchestrator")
	fmt.Println("Copyright 2017-2026 Nlaak Studios")

	// Load configuration
	cfg := config.LoadConfig()
	log.Printf("Environment: %s", cfg.Environment)
	log.Printf("Server port: %d", cfg.ServerPort)
	log.Printf("Data path: %s", cfg.DBPath)

	// Ensure data directories exist
	if err := utils.EnsureDataDirectories(); err != nil {
		log.Fatalf("Failed to create data directories: %v", err)
	}
	log.Printf("Data directories verified")

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

	// Create repositories
	songRepo := database.NewSongRepository(database.DB)
	queueRepo := database.NewQueueRepository(database.DB)
	videoRepo := database.NewVideoRepository(database.DB)
	settingsRepo := database.NewSettingsRepository(database.DB)

	// Create progress broadcaster for live updates
	broadcaster := services.NewProgressBroadcaster()

	// Create handlers
	songHandler := handlers.NewSongHandler(songRepo)
	queueHandler := handlers.NewQueueHandler(queueRepo, broadcaster)
	progressHandler := handlers.NewProgressHandler(broadcaster, queueRepo)
	imageHandler := handlers.NewImageHandler()
	audioHandler := handlers.NewAudioHandler(songRepo)
	uploadHandler := handlers.NewUploadHandler(songRepo)
	dashboardHandler := handlers.NewDashboardHandler(database.DB)
	videoHandler := handlers.NewVideoHandler(videoRepo)
	settingsHandler := handlers.NewSettingsHandler(settingsRepo)

	// Create and start queue worker
	queueWorker := worker.NewWorker(queueRepo, songRepo, broadcaster, 5*time.Second)
	go queueWorker.Start()
	log.Println("Queue worker started (polling every 5 seconds)")

	// Create Gin router
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// CORS middleware - MUST be first
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Add("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Add("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Add("Access-Control-Allow-Headers", "Content-Type, Authorization, Cache-Control, Accept")
		c.Writer.Header().Add("Access-Control-Expose-Headers", "Content-Type, Cache-Control, Connection")
		c.Writer.Header().Add("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(200)
			return
		}

		c.Next()
	})

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "track-studio-orchestrator",
		})
	})

	// Serve static files from new data directory
	videosPath := utils.GetVideosPath()
	router.Static("/videos", videosPath)
	log.Printf("Serving videos from: %s", videosPath)

	// Serve static image files
	imagesPath := utils.GetImagesPath()
	router.Static("/images", imagesPath)
	log.Printf("Serving images from: %s", imagesPath)

	// API v1 group
	v1 := router.Group("/api/v1")
	{
		// Dashboard endpoint
		v1.GET("/dashboard", dashboardHandler.GetDashboard)

		// Songs endpoints
		songs := v1.Group("/songs")
		{
			songs.GET("", songHandler.GetAll)
			songs.GET("/:id", songHandler.GetByID)
			songs.POST("", songHandler.Create)
			songs.PUT("/:id", songHandler.Update)
			songs.DELETE("/:id", songHandler.Delete)

			// Validation endpoint
			songs.GET("/:id/validate-paths", songHandler.ValidateAudioPaths)

			// Image endpoints for songs
			songs.GET("/:id/images", imageHandler.GetImagesBySong)
			songs.POST("/:id/images", imageHandler.CreateImagePrompt)
			songs.DELETE("/:id/images", imageHandler.DeleteImagesBySong)

			// Audio analysis endpoint
			songs.POST("/:id/analyze", audioHandler.AnalyzeSong) // Audio upload endpoint
			songs.POST("/:id/upload-audio", uploadHandler.UploadAudio)
		}

		// Images endpoints
		images := v1.Group("/images")
		{
			images.POST("/generate-prompt", imageHandler.GeneratePromptFromLyrics)
			images.PUT("/:id/prompt", imageHandler.UpdateImagePrompt)
			images.POST("/:id/regenerate", imageHandler.RegenerateImage)
		}

		// Queue endpoints
		queue := v1.Group("/queue")
		{
			queue.GET("", queueHandler.GetAll)
			queue.POST("", queueHandler.Create)
			queue.GET("/next", queueHandler.GetNext)
			queue.GET("/:id", queueHandler.GetByID)
			queue.PUT("/:id", queueHandler.Update)
			queue.DELETE("/:id", queueHandler.Delete)
			queue.PUT("/:id/flag", queueHandler.UpdateFlag)
		}

		// Progress streaming endpoints (SSE)
		progress := v1.Group("/progress")
		{
			progress.GET("/stream", progressHandler.StreamProgress)
			progress.GET("/stream/:id", progressHandler.StreamQueueProgress)
			progress.GET("/stats", progressHandler.GetStats)
		}

		// Videos endpoints
		videos := v1.Group("/videos")
		{
			videos.GET("", videoHandler.GetAll)
			videos.GET("/song/:songId", videoHandler.GetBySongID)
			videos.DELETE("/:id", videoHandler.Delete)
		}

		// Settings endpoints
		v1.GET("/settings", settingsHandler.Get)
		v1.POST("/settings", settingsHandler.Update)

		// Albums endpoints (placeholder)
		albums := v1.Group("/albums")
		{
			albums.GET("", func(c *gin.Context) {
				c.JSON(200, gin.H{"albums": []interface{}{}})
			})
		}

		// Artists endpoints (placeholder)
		artists := v1.Group("/artists")
		{
			artists.GET("", func(c *gin.Context) {
				c.JSON(200, gin.H{"artists": []interface{}{}})
			})
		}
	}

	// Start server in goroutine
	addr := fmt.Sprintf(":%d", cfg.ServerPort)
	log.Printf("Starting server on %s", addr)

	go func() {
		if err := router.Run(addr); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down gracefully...")

	// Stop worker
	queueWorker.Stop()

	// Close database
	database.Close()

	log.Println("Shutdown complete")
}
