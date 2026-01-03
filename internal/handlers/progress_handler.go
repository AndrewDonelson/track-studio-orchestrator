package handlers

import (
	"io"
	"log"
	"time"

	"github.com/AndrewDonelson/track-studio-orchestrator/internal/database"
	"github.com/AndrewDonelson/track-studio-orchestrator/internal/services"
	"github.com/gin-gonic/gin"
)

// ProgressHandler handles progress streaming
type ProgressHandler struct {
	broadcaster *services.ProgressBroadcaster
	queueRepo   *database.QueueRepository
}

// NewProgressHandler creates a new progress handler
func NewProgressHandler(broadcaster *services.ProgressBroadcaster, queueRepo *database.QueueRepository) *ProgressHandler {
	return &ProgressHandler{
		broadcaster: broadcaster,
		queueRepo:   queueRepo,
	}
}

// StreamProgress streams progress updates via Server-Sent Events
func (h *ProgressHandler) StreamProgress(c *gin.Context) {
	// Set headers for SSE
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// Subscribe to progress updates
	clientChan := h.broadcaster.Subscribe()
	defer h.broadcaster.Unsubscribe(clientChan)

	// Create a channel for client disconnect
	clientGone := c.Request.Context().Done()

	// Send initial connection confirmation
	c.Writer.Write([]byte("data: {\"message\":\"connected\",\"timestamp\":\"" + time.Now().Format(time.RFC3339) + "\"}\n\n"))
	c.Writer.Flush()

	// Stream updates
	for {
		select {
		case <-clientGone:
			log.Println("Client disconnected from progress stream")
			return
		case update := <-clientChan:
			// Format and send SSE event
			data := services.FormatSSE(update)
			if data != "" {
				_, err := c.Writer.Write([]byte(data))
				if err != nil {
					if err != io.EOF {
						log.Printf("Error writing SSE data: %v", err)
					}
					return
				}
				c.Writer.Flush()
			}
		case <-time.After(30 * time.Second):
			// Send keepalive ping every 30 seconds
			c.Writer.Write([]byte(": keepalive\n\n"))
			c.Writer.Flush()
		}
	}
}

// StreamQueueProgress streams progress for a specific queue item
func (h *ProgressHandler) StreamQueueProgress(c *gin.Context) {
	queueID := c.Param("id")

	// Set headers for SSE
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// Subscribe to progress updates
	clientChan := h.broadcaster.Subscribe()
	defer h.broadcaster.Unsubscribe(clientChan)

	// Create a channel for client disconnect
	clientGone := c.Request.Context().Done()

	// Send initial connection confirmation
	c.Writer.Write([]byte("data: {\"message\":\"connected\",\"queue_id\":\"" + queueID + "\",\"timestamp\":\"" + time.Now().Format(time.RFC3339) + "\"}\n\n"))
	c.Writer.Flush()

	// Stream updates (filter by queue ID)
	for {
		select {
		case <-clientGone:
			log.Printf("Client disconnected from queue %s progress stream", queueID)
			return
		case update := <-clientChan:
			// Only send updates for this specific queue item
			if update.QueueID == 0 || c.Param("id") == string(rune(update.QueueID)) {
				data := services.FormatSSE(update)
				if data != "" {
					_, err := c.Writer.Write([]byte(data))
					if err != nil {
						if err != io.EOF {
							log.Printf("Error writing SSE data: %v", err)
						}
						return
					}
					c.Writer.Flush()
				}
			}
		case <-time.After(30 * time.Second):
			// Send keepalive ping
			c.Writer.Write([]byte(": keepalive\n\n"))
			c.Writer.Flush()
		}
	}
}

// GetStats returns broadcaster statistics
func (h *ProgressHandler) GetStats(c *gin.Context) {
	c.JSON(200, gin.H{
		"connected_clients": h.broadcaster.ClientCount(),
		"timestamp":         time.Now(),
	})
}
