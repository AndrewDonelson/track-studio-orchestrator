package worker

import (
	"fmt"
	"log"
	"time"

	"github.com/AndrewDonelson/track-studio-orchestrator/internal/database"
	"github.com/AndrewDonelson/track-studio-orchestrator/internal/models"
	"github.com/AndrewDonelson/track-studio-orchestrator/internal/services"
)

// Processor handles the actual video processing pipeline
type Processor struct {
	songRepo    *database.SongRepository
	broadcaster *services.ProgressBroadcaster
}

// NewProcessor creates a new processor
func NewProcessor(
	songRepo *database.SongRepository,
	broadcaster *services.ProgressBroadcaster,
) *Processor {
	return &Processor{
		songRepo:    songRepo,
		broadcaster: broadcaster,
	}
}

// Process executes the full video generation pipeline
func (p *Processor) Process(item *models.QueueItem, song *models.Song) error {
	log.Printf("Starting processing pipeline for song: %s", song.Title)

	// Phase 1: Audio Analysis (0-20%)
	if err := p.analyzeAudio(item, song); err != nil {
		return fmt.Errorf("audio analysis failed: %w", err)
	}

	// Phase 2: Lyrics Processing (20-30%)
	if err := p.processLyrics(item, song); err != nil {
		return fmt.Errorf("lyrics processing failed: %w", err)
	}

	// Phase 3: Image Generation (30-50%)
	if err := p.generateImages(item, song); err != nil {
		return fmt.Errorf("image generation failed: %w", err)
	}

	// Phase 4: Video Rendering (50-90%)
	if err := p.renderVideo(item, song); err != nil {
		return fmt.Errorf("video rendering failed: %w", err)
	}

	// Phase 5: YouTube Upload (90-100%)
	if err := p.uploadToYouTube(item, song); err != nil {
		return fmt.Errorf("youtube upload failed: %w", err)
	}

	return nil
}

// analyzeAudio performs audio analysis
func (p *Processor) analyzeAudio(item *models.QueueItem, song *models.Song) error {
	p.updateProgress(item, "Analyzing audio", 5, "Loading audio files")
	time.Sleep(500 * time.Millisecond) // Simulate work

	p.updateProgress(item, "Analyzing audio", 10, "Detecting BPM and key")
	time.Sleep(500 * time.Millisecond)

	p.updateProgress(item, "Analyzing audio", 15, "Analyzing vocal timing")
	time.Sleep(500 * time.Millisecond)

	p.updateProgress(item, "Analyzing audio", 20, "Audio analysis complete")

	log.Printf("Audio analysis complete for song: %s", song.Title)
	return nil
}

// processLyrics processes and times the lyrics
func (p *Processor) processLyrics(item *models.QueueItem, song *models.Song) error {
	p.updateProgress(item, "Processing lyrics", 22, "Parsing lyrics")
	time.Sleep(300 * time.Millisecond)

	p.updateProgress(item, "Processing lyrics", 25, "Aligning lyrics with audio")
	time.Sleep(500 * time.Millisecond)

	p.updateProgress(item, "Processing lyrics", 30, "Lyrics processing complete")

	log.Printf("Lyrics processing complete for song: %s", song.Title)
	return nil
}

// generateImages generates background images via CQAI
func (p *Processor) generateImages(item *models.QueueItem, song *models.Song) error {
	imageCount := 5 // Generate 5 background images

	for i := 1; i <= imageCount; i++ {
		progress := 30 + (i * 4) // 30-50%
		message := fmt.Sprintf("Generating image %d of %d", i, imageCount)
		p.updateProgress(item, "Generating images", progress, message)
		time.Sleep(1 * time.Second) // Simulate CQAI API call
	}

	p.updateProgress(item, "Generating images", 50, "All images generated")

	log.Printf("Image generation complete for song: %s", song.Title)
	return nil
}

// renderVideo renders the final video
func (p *Processor) renderVideo(item *models.QueueItem, song *models.Song) error {
	p.updateProgress(item, "Rendering video", 55, "Preparing video assets")
	time.Sleep(500 * time.Millisecond)

	p.updateProgress(item, "Rendering video", 60, "Creating video timeline")
	time.Sleep(500 * time.Millisecond)

	p.updateProgress(item, "Rendering video", 70, "Adding audio waveform")
	time.Sleep(800 * time.Millisecond)

	p.updateProgress(item, "Rendering video", 75, "Overlaying lyrics")
	time.Sleep(800 * time.Millisecond)

	p.updateProgress(item, "Rendering video", 80, "Adding branding")
	time.Sleep(500 * time.Millisecond)

	p.updateProgress(item, "Rendering video", 85, "Encoding final video")
	time.Sleep(1 * time.Second)

	p.updateProgress(item, "Rendering video", 90, "Video rendering complete")

	// Store video path
	item.VideoFilePath = fmt.Sprintf("storage/videos/%s_%d.mp4", song.Title, item.ID)
	item.VideoFileSize = 125000000 // 125MB placeholder

	log.Printf("Video rendering complete for song: %s", song.Title)
	return nil
}

// uploadToYouTube uploads the video to YouTube
func (p *Processor) uploadToYouTube(item *models.QueueItem, song *models.Song) error {
	p.updateProgress(item, "Uploading to YouTube", 92, "Preparing upload")
	time.Sleep(500 * time.Millisecond)

	p.updateProgress(item, "Uploading to YouTube", 95, "Uploading video")
	time.Sleep(1 * time.Second)

	p.updateProgress(item, "Uploading to YouTube", 98, "Setting metadata")
	time.Sleep(300 * time.Millisecond)

	p.updateProgress(item, "Uploading to YouTube", 100, "Upload complete")

	log.Printf("YouTube upload complete for song: %s", song.Title)
	return nil
}

// updateProgress updates the queue item progress and broadcasts it
func (p *Processor) updateProgress(item *models.QueueItem, step string, progress int, message string) {
	item.CurrentStep = step
	item.Progress = progress

	p.broadcaster.BroadcastFromQueueItem(item, message)

	log.Printf("[Queue %d] %s: %d%% - %s", item.ID, step, progress, message)
}
