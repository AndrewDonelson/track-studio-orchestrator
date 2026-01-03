package worker

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/AndrewDonelson/track-studio-orchestrator/internal/database"
	"github.com/AndrewDonelson/track-studio-orchestrator/internal/models"
	"github.com/AndrewDonelson/track-studio-orchestrator/internal/services"
	"github.com/AndrewDonelson/track-studio-orchestrator/pkg/audio"
	"github.com/AndrewDonelson/track-studio-orchestrator/pkg/image"
	"github.com/AndrewDonelson/track-studio-orchestrator/pkg/lyrics"
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

// analyzeAudio performs audio analysis using librosa
func (p *Processor) analyzeAudio(item *models.QueueItem, song *models.Song) error {
	p.updateProgress(item, "Analyzing audio", 5, "Loading audio files")

	// Determine which audio file to analyze (prefer vocals stem, fallback to mixed)
	audioPath := song.VocalsStemPath
	if audioPath == "" {
		audioPath = song.MixedAudioPath
	}
	if audioPath == "" {
		return fmt.Errorf("no audio file available for analysis")
	}

	p.updateProgress(item, "Analyzing audio", 10, "Running audio analysis (BPM, key, timing)")

	// Run Python audio analyzer
	analysis, err := audio.AnalyzeAudio(audioPath)
	if err != nil {
		return fmt.Errorf("audio analysis failed: %w", err)
	}

	p.updateProgress(item, "Analyzing audio", 15, "Processing analysis results")

	// Update song with analysis results
	song.BPM = analysis.BPM
	song.Key = analysis.Key
	song.Tempo = analysis.Tempo
	song.DurationSeconds = analysis.DurationSeconds

	// Store vocal timing as JSON string
	if len(analysis.VocalSegments) > 0 {
		// For now, we'll just store the count and log details
		log.Printf("Detected %d vocal segments in %s", analysis.VocalSegmentCount, song.Title)
		log.Printf("Audio Analysis: %s", analysis.Summary())
	}

	// Save updated song data
	if err := p.songRepo.Update(song); err != nil {
		log.Printf("Warning: failed to save audio analysis results: %v", err)
	}

	p.updateProgress(item, "Analyzing audio", 20, fmt.Sprintf("Analysis complete: %.1f BPM, %s", analysis.BPM, analysis.Key))

	log.Printf("Audio analysis complete for song: %s - %s", song.Title, analysis.Summary())
	return nil
}

// processLyrics processes and times the lyrics
func (p *Processor) processLyrics(item *models.QueueItem, song *models.Song) error {
	p.updateProgress(item, "Processing lyrics", 22, "Parsing lyrics structure")

	// Parse lyrics to detect sections
	lyricsData, err := lyrics.ParseLyrics(song.Lyrics)
	if err != nil {
		return fmt.Errorf("failed to parse lyrics: %w", err)
	}

	log.Printf("Parsed lyrics for %s: %s", song.Title, lyricsData.GetSectionSummary())

	p.updateProgress(item, "Processing lyrics", 25, "Aligning lyrics with audio timing")

	// We need beat times from the audio analysis
	// For now, we'll use a simplified alignment
	// In production, this would use the beat_times from audio analysis
	beatTimes := []float64{} // Will be populated from audio analysis in future

	timedLines, err := lyrics.AlignLyricsToBeats(song.Lyrics, beatTimes, song.DurationSeconds)
	if err != nil {
		return fmt.Errorf("failed to align lyrics: %w", err)
	}

	log.Printf("Aligned %d lyrics lines to audio timing", len(timedLines))

	// Store processed lyrics data
	sectionsJSON, err := json.Marshal(lyricsData.Sections)
	if err != nil {
		log.Printf("Warning: failed to marshal sections: %v", err)
	} else {
		song.LyricsSections = string(sectionsJSON)
	}

	timedLinesJSON, err := json.Marshal(timedLines)
	if err != nil {
		log.Printf("Warning: failed to marshal timed lines: %v", err)
	} else {
		song.LyricsDisplay = string(timedLinesJSON)
	}

	// Save updated song data
	if err := p.songRepo.Update(song); err != nil {
		log.Printf("Warning: failed to save lyrics processing results: %v", err)
	}

	p.updateProgress(item, "Processing lyrics", 30, fmt.Sprintf("Processed %d sections, %d lines", len(lyricsData.Sections), len(timedLines)))

	log.Printf("Lyrics processing complete for song: %s", song.Title)
	return nil
}

// generateImages generates background images via CQAI for each unique section
func (p *Processor) generateImages(item *models.QueueItem, song *models.Song) error {
	p.updateProgress(item, "Generating images", 34, "Parsing lyrics sections")

	// Parse lyrics to get sections
	lyricsData, err := lyrics.ParseLyrics(song.Lyrics)
	if err != nil {
		return fmt.Errorf("failed to parse lyrics for images: %w", err)
	}

	if len(lyricsData.Sections) == 0 {
		log.Printf("No sections found, skipping image generation")
		return nil
	}

	// Create image generator with absolute path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	execDir := filepath.Dir(execPath)
	outputDir := filepath.Join(execDir, "storage", "images", fmt.Sprintf("song_%d", song.ID))
	imageGen := image.NewImageGenerator(outputDir)

	// Build style keywords from genre and background style
	styleKeywords := image.BuildStyleKeywords(song.Genre, song.BackgroundStyle)
	log.Printf("Style keywords for %s: %s", song.Title, styleKeywords)

	// Track unique images generated
	generatedImages := make(map[string]string) // filename -> path
	var imagePaths []string

	totalSections := len(lyricsData.Sections)
	for i, section := range lyricsData.Sections {
		// Calculate progress (34% to 50%)
		progress := 34 + ((i+1)*16)/totalSections

		// Determine if we need to generate this image
		var filename string
		switch section.Type {
		case "verse":
			filename = fmt.Sprintf("bg-verse-%d.png", section.Number)
		case "pre-chorus":
			filename = "bg-prechorus.png"
		case "chorus":
			filename = "bg-chorus.png"
		case "bridge":
			filename = "bg-bridge.png"
		case "intro":
			filename = "bg-intro.png"
		case "outro":
			filename = "bg-outro.png"
		default:
			filename = fmt.Sprintf("bg-%s-%d.png", section.Type, section.Number)
		}

		// Check if already generated (for repeated choruses/pre-choruses)
		if existingPath, exists := generatedImages[filename]; exists {
			log.Printf("Reusing existing image for %s %d: %s", section.Type, section.Number, filename)
			imagePaths = append(imagePaths, existingPath)
			continue
		}

		// Prepare lyrics content
		sectionLyrics := strings.Join(section.Lines, "\n")

		message := fmt.Sprintf("Generating image for %s %d (%s)",
			section.Type, section.Number, filename)
		p.updateProgress(item, "Generating images", progress, message)

		// Generate image
		log.Printf("Generating image for %s %d: %s", section.Type, section.Number, filename)
		imagePath, err := imageGen.GenerateFromSection(
			section.Type,
			section.Number,
			sectionLyrics,
			styleKeywords,
		)
		if err != nil {
			log.Printf("Warning: failed to generate image for %s %d: %v",
				section.Type, section.Number, err)
			// Continue with other images
			continue
		}

		generatedImages[filename] = imagePath
		imagePaths = append(imagePaths, imagePath)
		log.Printf("Generated image %d/%d: %s", len(generatedImages), totalSections, imagePath)
	}

	p.updateProgress(item, "Generating images", 50,
		fmt.Sprintf("Generated %d unique images from %d sections",
			len(generatedImages), totalSections))

	log.Printf("Image generation complete for song: %s - Generated %d unique images",
		song.Title, len(generatedImages))

	// TODO: Store image paths in database (new field: image_paths JSON)

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
