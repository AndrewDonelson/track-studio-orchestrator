package services

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/AndrewDonelson/track-studio-orchestrator/internal/models"
)

// ProgressUpdate represents a progress update event
type ProgressUpdate struct {
	QueueID      int       `json:"queue_id"`
	SongID       int       `json:"song_id"`
	Status       string    `json:"status"`
	CurrentStep  string    `json:"current_step"`
	Progress     int       `json:"progress"`
	Message      string    `json:"message"`
	ErrorMessage string    `json:"error_message,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
}

// ProgressBroadcaster manages SSE connections for live progress updates
type ProgressBroadcaster struct {
	clients map[chan ProgressUpdate]bool
	mutex   sync.RWMutex
}

// NewProgressBroadcaster creates a new progress broadcaster
func NewProgressBroadcaster() *ProgressBroadcaster {
	return &ProgressBroadcaster{
		clients: make(map[chan ProgressUpdate]bool),
	}
}

// Subscribe adds a new client to receive progress updates
func (pb *ProgressBroadcaster) Subscribe() chan ProgressUpdate {
	pb.mutex.Lock()
	defer pb.mutex.Unlock()

	client := make(chan ProgressUpdate, 10)
	pb.clients[client] = true
	log.Printf("Client subscribed to progress updates. Total clients: %d", len(pb.clients))
	return client
}

// Unsubscribe removes a client from receiving updates
func (pb *ProgressBroadcaster) Unsubscribe(client chan ProgressUpdate) {
	pb.mutex.Lock()
	defer pb.mutex.Unlock()

	if _, ok := pb.clients[client]; ok {
		delete(pb.clients, client)
		close(client)
		log.Printf("Client unsubscribed from progress updates. Total clients: %d", len(pb.clients))
	}
}

// Broadcast sends a progress update to all connected clients
func (pb *ProgressBroadcaster) Broadcast(update ProgressUpdate) {
	pb.mutex.RLock()
	defer pb.mutex.RUnlock()

	update.Timestamp = time.Now()
	
	for client := range pb.clients {
		select {
		case client <- update:
			// Successfully sent
		default:
			// Client buffer full, skip
			log.Printf("Warning: Client buffer full, skipping update for queue_id=%d", update.QueueID)
		}
	}

	log.Printf("Progress update broadcast: queue_id=%d, step=%s, progress=%d%%", 
		update.QueueID, update.CurrentStep, update.Progress)
}

// BroadcastFromQueueItem converts a queue item to progress update and broadcasts
func (pb *ProgressBroadcaster) BroadcastFromQueueItem(item *models.QueueItem, message string) {
	update := ProgressUpdate{
		QueueID:      item.ID,
		SongID:       item.SongID,
		Status:       item.Status,
		CurrentStep:  item.CurrentStep,
		Progress:     item.Progress,
		Message:      message,
		ErrorMessage: item.ErrorMessage,
	}
	pb.Broadcast(update)
}

// ClientCount returns the number of connected clients
func (pb *ProgressBroadcaster) ClientCount() int {
	pb.mutex.RLock()
	defer pb.mutex.RUnlock()
	return len(pb.clients)
}

// FormatSSE formats a progress update as Server-Sent Event
func FormatSSE(update ProgressUpdate) string {
	data, err := json.Marshal(update)
	if err != nil {
		log.Printf("Error marshaling SSE data: %v", err)
		return ""
	}
	return "data: " + string(data) + "\n\n"
}
