package handlers

import (
	"net/http"
	"strconv"

	"github.com/AndrewDonelson/track-studio-orchestrator/internal/database"
	"github.com/AndrewDonelson/track-studio-orchestrator/internal/models"
	"github.com/AndrewDonelson/track-studio-orchestrator/internal/services"
	"github.com/gin-gonic/gin"
)

// QueueHandler handles queue-related requests
type QueueHandler struct {
	repo        *database.QueueRepository
	broadcaster *services.ProgressBroadcaster
}

// NewQueueHandler creates a new queue handler
func NewQueueHandler(repo *database.QueueRepository, broadcaster *services.ProgressBroadcaster) *QueueHandler {
	return &QueueHandler{
		repo:        repo,
		broadcaster: broadcaster,
	}
}

// GetAll returns all queue items
func (h *QueueHandler) GetAll(c *gin.Context) {
	items, err := h.repo.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"queue": items})
}

// GetByID returns a queue item by ID
func (h *QueueHandler) GetByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	item, err := h.repo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if item == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Queue item not found"})
		return
	}

	c.JSON(http.StatusOK, item)
}

// Create adds a song to the queue
func (h *QueueHandler) Create(c *gin.Context) {
	var req struct {
		SongID   int `json:"song_id" binding:"required"`
		Priority int `json:"priority"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	item := &models.QueueItem{
		SongID:   req.SongID,
		Status:   models.StatusQueued,
		Priority: req.Priority,
	}

	if err := h.repo.Create(item); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Broadcast queue item creation
	h.broadcaster.BroadcastFromQueueItem(item, "Queue item created")

	c.JSON(http.StatusCreated, item)
}

// Update updates a queue item
func (h *QueueHandler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var item models.QueueItem
	if err := c.ShouldBindJSON(&item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	item.ID = id
	if err := h.repo.Update(&item); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Broadcast queue item update
	h.broadcaster.BroadcastFromQueueItem(&item, "Queue item updated")

	c.JSON(http.StatusOK, item)
}

// Delete removes a queue item and cleans up associated files
func (h *QueueHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	// Get the item first to broadcast cancellation
	item, err := h.repo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if item == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Queue item not found"})
		return
	}

	// Delete from database
	if err := h.repo.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Broadcast cancellation
	h.broadcaster.BroadcastFromQueueItem(item, "Queue item cancelled")

	c.JSON(http.StatusOK, gin.H{"message": "Queue item deleted"})
}

// GetNext returns the next pending queue item
func (h *QueueHandler) GetNext(c *gin.Context) {
	item, err := h.repo.GetNextPending()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if item == nil {
		c.JSON(http.StatusOK, gin.H{"item": nil, "message": "No pending items"})
		return
	}

	c.JSON(http.StatusOK, item)
}
