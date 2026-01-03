package worker

import (
	"context"
	"log"
	"time"

	"github.com/AndrewDonelson/track-studio-orchestrator/internal/database"
	"github.com/AndrewDonelson/track-studio-orchestrator/internal/models"
	"github.com/AndrewDonelson/track-studio-orchestrator/internal/services"
)

// Worker processes queue items
type Worker struct {
	queueRepo    *database.QueueRepository
	songRepo     *database.SongRepository
	broadcaster  *services.ProgressBroadcaster
	processor    *Processor
	pollInterval time.Duration
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewWorker creates a new queue worker
func NewWorker(
	queueRepo *database.QueueRepository,
	songRepo *database.SongRepository,
	broadcaster *services.ProgressBroadcaster,
	pollInterval time.Duration,
) *Worker {
	processor := NewProcessor(songRepo, broadcaster)
	ctx, cancel := context.WithCancel(context.Background())

	return &Worker{
		queueRepo:    queueRepo,
		songRepo:     songRepo,
		broadcaster:  broadcaster,
		processor:    processor,
		pollInterval: pollInterval,
		ctx:          ctx,
		cancel:       cancel,
	}
}

// Start begins processing queue items
func (w *Worker) Start() {
	log.Println("Queue worker started")

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	// Process immediately on start
	w.processNext()

	// Then process on interval
	for {
		select {
		case <-w.ctx.Done():
			log.Println("Queue worker stopped")
			return
		case <-ticker.C:
			w.processNext()
		}
	}
}

// Stop gracefully stops the worker
func (w *Worker) Stop() {
	log.Println("Stopping queue worker...")
	w.cancel()
}

// processNext processes the next pending queue item
func (w *Worker) processNext() {
	// Get next pending item
	item, err := w.queueRepo.GetNextPending()
	if err != nil {
		log.Printf("Error getting next pending item: %v", err)
		return
	}

	if item == nil {
		// No items to process
		return
	}

	log.Printf("Processing queue item %d (song %d)", item.ID, item.SongID)

	// Get song details
	song, err := w.songRepo.GetByID(item.SongID)
	if err != nil {
		log.Printf("Error getting song %d: %v", item.SongID, err)
		w.failQueueItem(item, "Failed to load song data")
		return
	}
	if song == nil {
		log.Printf("Song %d not found", item.SongID)
		w.failQueueItem(item, "Song not found")
		return
	}

	// Mark as processing
	now := time.Now()
	item.Status = models.StatusProcessing
	item.StartedAt = &now
	item.Progress = 0
	item.CurrentStep = "Starting"
	if err := w.queueRepo.Update(item); err != nil {
		log.Printf("Error updating queue item: %v", err)
		return
	}

	// Broadcast start
	w.broadcaster.BroadcastFromQueueItem(item, "Processing started")

	// Process the item
	if err := w.processor.Process(item, song); err != nil {
		log.Printf("Error processing queue item %d: %v", item.ID, err)
		w.failQueueItem(item, err.Error())
		return
	}

	// Mark as completed
	completed := time.Now()
	item.Status = models.StatusCompleted
	item.CompletedAt = &completed
	item.Progress = 100
	item.CurrentStep = "Completed"
	if err := w.queueRepo.Update(item); err != nil {
		log.Printf("Error updating completed queue item: %v", err)
		return
	}

	// Broadcast completion
	w.broadcaster.BroadcastFromQueueItem(item, "Processing completed successfully")
	log.Printf("Queue item %d completed successfully", item.ID)
}

// failQueueItem marks a queue item as failed
func (w *Worker) failQueueItem(item *models.QueueItem, errorMsg string) {
	item.Status = models.StatusFailed
	item.ErrorMessage = errorMsg
	item.RetryCount++
	completed := time.Now()
	item.CompletedAt = &completed

	if err := w.queueRepo.Update(item); err != nil {
		log.Printf("Error updating failed queue item: %v", err)
		return
	}

	w.broadcaster.BroadcastFromQueueItem(item, "Processing failed")
	log.Printf("Queue item %d failed: %s", item.ID, errorMsg)
}
