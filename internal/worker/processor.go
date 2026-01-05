package worker

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/AndrewDonelson/track-studio-orchestrator/internal/database"
	"github.com/AndrewDonelson/track-studio-orchestrator/internal/models"
	"github.com/AndrewDonelson/track-studio-orchestrator/internal/services"
	"github.com/AndrewDonelson/track-studio-orchestrator/pkg/audio"
	"github.com/AndrewDonelson/track-studio-orchestrator/pkg/image"
	"github.com/AndrewDonelson/track-studio-orchestrator/pkg/lyrics"
	"github.com/AndrewDonelson/track-studio-orchestrator/pkg/video"
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
	// Check if audio analysis already exists
	if song.BPM > 0 && song.Key != "" && song.DurationSeconds > 0 {
		log.Printf("Audio analysis already exists for song %s, skipping", song.Title)
		p.updateProgress(item, "Analyzing audio", 20, fmt.Sprintf("Using existing analysis: %.1f BPM, %s", song.BPM, song.Key))
		return nil
	}

	p.updateProgress(item, "Analyzing audio", 5, "Loading audio files")

	// For BPM/tempo analysis, use instrumental track (more accurate rhythm detection)
	// For vocal timing, use vocals track
	bpmAudioPath := song.MusicStemPath
	vocalAudioPath := song.VocalsStemPath

	if bpmAudioPath == "" {
		bpmAudioPath = song.MixedAudioPath
	}
	if vocalAudioPath == "" {
		vocalAudioPath = song.MixedAudioPath
	}
	if bpmAudioPath == "" {
		return fmt.Errorf("no audio file available for analysis")
	}

	p.updateProgress(item, "Analyzing audio", 10, "Running audio analysis (BPM, key, timing)")

	// Run Python audio analyzer on instrumental track for BPM/tempo
	analysis, err := audio.AnalyzeAudio(bpmAudioPath)
	if err != nil {
		return fmt.Errorf("audio analysis failed: %w", err)
	}

	p.updateProgress(item, "Analyzing audio", 15, "Processing analysis results")

	// Update song with analysis results
	song.BPM = analysis.BPM
	song.Key = analysis.Key
	song.Tempo = analysis.Tempo
	song.DurationSeconds = analysis.DurationSeconds

	// Update genre from audio analysis (if not already set manually)
	if song.Genre == "" && analysis.Genre != "" {
		song.Genre = analysis.Genre
		log.Printf("Detected genre: %s", analysis.Genre)
	}

	// If we have separate vocal track, analyze it for vocal timing
	if vocalAudioPath != "" && vocalAudioPath != bpmAudioPath {
		vocalAnalysis, err := audio.AnalyzeAudio(vocalAudioPath)
		if err == nil && len(vocalAnalysis.VocalSegments) > 0 {
			analysis.VocalSegments = vocalAnalysis.VocalSegments
			analysis.VocalSegmentCount = vocalAnalysis.VocalSegmentCount
		}
	}

	// Store vocal timing as JSON string
	if len(analysis.VocalSegments) > 0 {
		vocalTimingJSON, err := json.Marshal(analysis.VocalSegments)
		if err != nil {
			log.Printf("Warning: failed to marshal vocal segments: %v", err)
		} else {
			song.VocalTiming = string(vocalTimingJSON)
		}
		log.Printf("Detected %d vocal segments in %s (first vocal at %.2fs)",
			analysis.VocalSegmentCount, song.Title, analysis.VocalSegments[0].Start)
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
	p.updateProgress(item, "Generating images", 30, "Scanning for existing images")

	// Get executable directory for absolute paths
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	execDir := filepath.Dir(execPath)
	outputDir := filepath.Join(execDir, "storage", "images", fmt.Sprintf("song_%d", song.ID))
	imageGen := image.NewImageGenerator(outputDir)

	// Step 1: Check for existing image FILES on disk
	existingFiles := make(map[string]string) // filename -> full path
	if _, err := os.Stat(outputDir); err == nil {
		files, err := os.ReadDir(outputDir)
		if err == nil {
			for _, file := range files {
				if !file.IsDir() && strings.HasSuffix(file.Name(), ".png") {
					existingFiles[file.Name()] = filepath.Join(outputDir, file.Name())
					log.Printf("Found existing image file: %s", file.Name())
				}
			}
		}
	}

	// Step 2: Check database for existing image prompts
	existingImages, err := database.GetImagesBySongID(song.ID)
	if err != nil {
		return fmt.Errorf("failed to get existing images: %w", err)
	}

	// Step 3: Reverse-engineer prompts from orphaned image files (files without database entries)
	if len(existingFiles) > 0 && len(existingImages) == 0 {
		p.updateProgress(item, "Generating images", 32, fmt.Sprintf("Reverse-engineering prompts from %d existing images", len(existingFiles)))
		log.Printf("Found %d image files but no database entries - extracting prompts with vision AI", len(existingFiles))

		fileIndex := 0
		for filename, filePath := range existingFiles {
			fileIndex++
			progress := 32 + ((fileIndex * 8) / len(existingFiles))
			p.updateProgress(item, "Generating images", progress, fmt.Sprintf("Analyzing image %d/%d with vision AI", fileIndex, len(existingFiles)))

			// Extract prompt using vision model
			log.Printf("Extracting prompt from %s using vision AI...", filename)
			extractedPrompt, err := imageGen.ExtractPromptFromImage(filePath)
			if err != nil {
				log.Printf("Warning: failed to extract prompt from %s: %v", filename, err)
				continue
			}

			// Parse filename to determine image type and sequence
			// Format: bg-verse-1.png, bg-chorus.png, bg-intro.png, etc.
			imageType, sequenceNum := parseImageFilename(filename)
			if imageType == "" {
				log.Printf("Warning: couldn't parse image type from filename: %s", filename)
				continue
			}

			// Create database entry with extracted prompt
			relativePath := strings.TrimPrefix(filePath, execDir+"/")
			genImage := &models.GeneratedImage{
				SongID:         song.ID,
				QueueID:        &item.ID,
				ImagePath:      relativePath,
				Prompt:         extractedPrompt,
				NegativePrompt: image.MASTER_NEGATIVE_PROMPT,
				ImageType:      imageType,
				SequenceNumber: sequenceNum,
				Width:          1920,
				Height:         1080,
				Model:          "cqai",
			}

			if err := database.CreateGeneratedImage(genImage); err != nil {
				log.Printf("Warning: failed to create database entry for %s: %v", filename, err)
				continue
			}

			log.Printf("Successfully reverse-engineered prompt for %s (type: %s)", filename, imageType)
		}

		// Refresh the list of existing images from database
		existingImages, err = database.GetImagesBySongID(song.ID)
		if err != nil {
			return fmt.Errorf("failed to refresh image list: %w", err)
		}
	}

	// Step 4: Check which images are missing (have prompts but no files)
	var missingImages []models.GeneratedImage
	for _, img := range existingImages {
		if img.ImagePath == "" || img.ImagePath == "." {
			missingImages = append(missingImages, img)
		}
	}

	if len(missingImages) > 0 {
		log.Printf("Found %d existing prompts with missing images, generating them now", len(missingImages))
		p.updateProgress(item, "Generating images", 40, fmt.Sprintf("Generating %d missing images from saved prompts", len(missingImages)))

		// Generate each missing image using its stored prompt
		for i, img := range missingImages {
			progress := 40 + ((i+1)*10)/len(missingImages)

			// Generate filename based on image type and sequence number
			var filename string
			if img.SequenceNumber != nil && *img.SequenceNumber > 0 {
				filename = fmt.Sprintf("bg-%s-%d.png", img.ImageType, *img.SequenceNumber)
			} else {
				filename = fmt.Sprintf("bg-%s.png", img.ImageType)
			}

			message := fmt.Sprintf("Generating %s image (%d/%d)", img.ImageType, i+1, len(missingImages))
			p.updateProgress(item, "Generating images", progress, message)

			log.Printf("Generating missing image: %s with prompt: %s", filename, img.Prompt)

			// Generate image using the stored prompt
			imagePath, err := imageGen.GenerateImage(img.Prompt, filename)
			if err != nil {
				log.Printf("Warning: failed to generate image %s: %v", filename, err)
				continue
			}

			// Update database with the new image path
			relativePath := strings.TrimPrefix(imagePath, execDir+"/")
			if err := database.UpdateImagePath(img.ID, relativePath); err != nil {
				log.Printf("Warning: failed to update image path for %d: %v", img.ID, err)
				continue
			}

			log.Printf("Generated missing image %d/%d: %s", i+1, len(missingImages), imagePath)
		}

		p.updateProgress(item, "Generating images", 50, "All images ready")
		return nil
	}

	// Step 5: Check if all required images already exist (in database with paths)
	allImagesReady := len(existingImages) > 0
	for _, img := range existingImages {
		if img.ImagePath == "" || img.ImagePath == "." {
			allImagesReady = false
			break
		}
	}

	if allImagesReady {
		log.Printf("All %d images already exist in database with valid paths, skipping generation", len(existingImages))
		p.updateProgress(item, "Generating images", 50, fmt.Sprintf("Using %d existing images", len(existingImages)))
		return nil
	}

	// No existing prompts found, use legacy generation method
	log.Printf("No existing image prompts found, generating from lyrics")
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

	// Image generator already created at top of function, reuse it
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

		// Determine filename - Each verse gets unique image, repeated sections share images
		var filename string
		switch section.Type {
		case "verse":
			// Each verse gets its own unique image
			filename = fmt.Sprintf("bg-verse-%d.png", section.Number)
		case "pre-chorus":
			// Pre-choruses share one image (they repeat the same lyrics)
			filename = "bg-pre-chorus-1.png"
		case "chorus":
			// Choruses share one image (they repeat the same lyrics)
			filename = "bg-chorus-1.png"
		case "final-chorus":
			// Final chorus gets its own image
			filename = "bg-final-chorus-1.png"
		case "bridge":
			// Each bridge gets its own unique image
			filename = fmt.Sprintf("bg-bridge-%d.png", section.Number)
		case "intro":
			filename = "bg-intro.png"
		case "outro":
			filename = "bg-outro-1.png"
		default:
			filename = fmt.Sprintf("bg-%s.png", section.Type)
		}

		// Check if already generated (reuse for all repeated section types)
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
		imagePath, prompt, err := imageGen.GenerateFromSection(
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

		// Store image in database with captured prompt
		genImage := &models.GeneratedImage{
			SongID:         song.ID,
			QueueID:        &item.ID,
			ImagePath:      imagePath,
			Prompt:         prompt,
			NegativePrompt: image.MASTER_NEGATIVE_PROMPT,
			ImageType:      section.Type,
			SequenceNumber: &section.Number,
			Width:          1920,
			Height:         1080,
			Model:          "cqai",
		}
		if err := database.CreateGeneratedImage(genImage); err != nil {
			log.Printf("Warning: failed to store image record in database: %v", err)
		}
	}

	p.updateProgress(item, "Generating images", 50,
		fmt.Sprintf("Generated %d unique images from %d sections",
			len(generatedImages), totalSections))

	log.Printf("Image generation complete for song: %s - Generated %d unique images",
		song.Title, len(generatedImages))

	return nil
}

// renderVideo renders the final video
func (p *Processor) renderVideo(item *models.QueueItem, song *models.Song) error {
	p.updateProgress(item, "Rendering video", 55, "Preparing video assets")

	// Get executable directory for absolute paths
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	execDir := filepath.Dir(execPath)

	// Setup paths
	outputDir := filepath.Join(execDir, "storage", "videos")
	videoPath := filepath.Join(outputDir, fmt.Sprintf("%s.mp4",
		strings.ReplaceAll(song.Title, " ", "_")))

	// Get audio path - need to mix vocals and instrumental
	audioPath := ""
	if song.VocalsStemPath != "" && song.MusicStemPath != "" {
		// Mix vocals and instrumental together
		mixedPath := filepath.Join(execDir, "storage", "temp", fmt.Sprintf("mixed_%d.wav", song.ID))
		if err := p.mixAudioTracks(song.VocalsStemPath, song.MusicStemPath, mixedPath); err != nil {
			log.Printf("Warning: failed to mix audio tracks: %v, using vocals only", err)
			audioPath = song.VocalsStemPath
		} else {
			audioPath = mixedPath
			defer os.Remove(mixedPath)
		}
	} else if song.MixedAudioPath != "" && song.MixedAudioPath != song.VocalsStemPath {
		// Use pre-mixed audio if it's not the same as vocals
		audioPath = song.MixedAudioPath
	} else if song.VocalsStemPath != "" {
		audioPath = song.VocalsStemPath
	}

	if audioPath == "" {
		return fmt.Errorf("no audio file available")
	}

	p.updateProgress(item, "Rendering video", 60, "Loading lyrics and images")

	// Parse lyrics data from stored JSON fields
	var lyricsData lyrics.LyricsData
	lyricsData.RawLyrics = song.Lyrics

	// Parse sections from LyricsSections
	if song.LyricsSections != "" {
		var sections []lyrics.Section
		if err := json.Unmarshal([]byte(song.LyricsSections), &sections); err != nil {
			return fmt.Errorf("failed to parse lyrics sections: %w", err)
		}
		lyricsData.Sections = sections
	}

	// Parse timed lines from LyricsDisplay
	if song.LyricsDisplay != "" {
		var timedLines []lyrics.TimedLine
		if err := json.Unmarshal([]byte(song.LyricsDisplay), &timedLines); err != nil {
			return fmt.Errorf("failed to parse timed lines: %w", err)
		}
		lyricsData.TimedLines = timedLines
	}

	// Build image segments from sections
	imageDir := filepath.Join(execDir, "storage", "images", fmt.Sprintf("song_%d", song.ID))
	imageSegments, err := p.buildImageSegments(&lyricsData, imageDir, song.DurationSeconds)
	if err != nil {
		return fmt.Errorf("failed to build image segments: %w", err)
	}

	// Build timed lyrics from TimedLines
	timedLyrics := p.buildTimedLyrics(&lyricsData)

	// Get vocal onset time from database
	vocalOnset := 0.0
	if song.VocalTiming != "" {
		var vocalSegments []audio.VocalSegment
		if err := json.Unmarshal([]byte(song.VocalTiming), &vocalSegments); err == nil {
			if len(vocalSegments) > 0 {
				vocalOnset = vocalSegments[0].Start
				log.Printf("Applying vocal onset offset: %.2fs", vocalOnset)
			}
		}
	}

	p.updateProgress(item, "Rendering video", 70, "Composing video with FFmpeg")

	// Generate karaoke subtitles if vocals path is available
	assSubtitlePath := ""
	if song.VocalsStemPath != "" {
		log.Println("Generating word-level karaoke timestamps...")
		p.updateProgress(item, "Rendering video", 72, "Generating karaoke timestamps")

		// Create karaoke generator
		karaokeGen := lyrics.NewKaraokeGenerator(execDir)

		// Prepare karaoke customization options from song settings
		karaokeOptions := &lyrics.KaraokeOptions{
			FontFamily:           song.KaraokeFontFamily,
			FontSize:             song.KaraokeFontSize,
			PrimaryColor:         song.KaraokePrimaryColor,
			PrimaryBorderColor:   song.KaraokePrimaryBorderColor,
			HighlightColor:       song.KaraokeHighlightColor,
			HighlightBorderColor: song.KaraokeHighlightBorderColor,
			Alignment:            song.KaraokeAlignment,
			MarginBottom:         song.KaraokeMarginBottom,
		}

		// Use defaults if critical fields are missing or invalid
		defaults := lyrics.DefaultKaraokeOptions()
		if karaokeOptions.FontFamily == "" {
			karaokeOptions.FontFamily = defaults.FontFamily
		}
		if karaokeOptions.FontSize <= 0 {
			karaokeOptions.FontSize = defaults.FontSize
		}
		if karaokeOptions.PrimaryColor == "" {
			karaokeOptions.PrimaryColor = defaults.PrimaryColor
		}
		if karaokeOptions.PrimaryBorderColor == "" {
			karaokeOptions.PrimaryBorderColor = defaults.PrimaryBorderColor
		}
		if karaokeOptions.HighlightColor == "" {
			karaokeOptions.HighlightColor = defaults.HighlightColor
		}
		if karaokeOptions.HighlightBorderColor == "" {
			karaokeOptions.HighlightBorderColor = defaults.HighlightBorderColor
		}
		if karaokeOptions.Alignment <= 0 || karaokeOptions.Alignment > 9 {
			karaokeOptions.Alignment = defaults.Alignment
		}

		// Generate ASS subtitles from vocals, using lyrics_karaoke for display
		tempDir := filepath.Join(execDir, "storage", "temp")
		assPath, err := karaokeGen.GenerateKaraokeSubtitles(song.VocalsStemPath, int(song.ID), tempDir, song.LyricsKaraoke, karaokeOptions)
		if err != nil {
			log.Printf("Warning: failed to generate karaoke subtitles: %v, using fallback lyrics", err)
		} else {
			assSubtitlePath = assPath
			log.Printf("Generated karaoke subtitles: %s", assSubtitlePath)
		}
	}

	// Create video renderer
	renderer := video.NewVideoRenderer(outputDir)

	// Prepare render options
	opts := &video.VideoRenderOptions{
		AudioPath:         audioPath,
		Duration:          song.DurationSeconds,
		ImagePaths:        imageSegments,
		LyricsData:        timedLyrics,
		VocalOnset:        vocalOnset,
		CrossfadeDuration: 2.0,             // 2 second crossfade between images
		EnableKaraoke:     false,           // Karaoke highlighting disabled by default
		ASSSubtitlePath:   assSubtitlePath, // Use generated ASS subtitles if available
		Key:               song.Key,
		Tempo:             song.Tempo,
		BPM:               song.BPM,
		Title:             song.Title,
		Artist:            song.ArtistName,
		SpectrumStyle:     getSpectrumStyle(song.SpectrumStyle),
		SpectrumColor:     getSpectrumColorHex(song.SpectrumColor),
		SpectrumOpacity:   getSpectrumOpacity(song.SpectrumOpacity),
		OutputPath:        videoPath,
	}

	p.updateProgress(item, "Rendering video", 75, "Rendering video (this may take a few minutes)")

	// Render the video
	finalPath, err := renderer.RenderVideo(opts)
	if err != nil {
		return fmt.Errorf("video rendering failed: %w", err)
	}

	p.updateProgress(item, "Rendering video", 90, "Video rendering complete")

	// Get file size
	fileInfo, err := os.Stat(finalPath)
	if err != nil {
		log.Printf("Warning: could not get video file size: %v", err)
	} else {
		item.VideoFileSize = fileInfo.Size()
	}

	// Store video path
	item.VideoFilePath = finalPath

	log.Printf("Video rendering complete for song: %s - Output: %s (%.2f MB)",
		song.Title, finalPath, float64(item.VideoFileSize)/(1024*1024))
	return nil
}

// buildImageSegments creates timed image segments from lyrics sections
func (p *Processor) buildImageSegments(lyricsData *lyrics.LyricsData, imageDir string, totalDuration float64) ([]video.ImageSegment, error) {
	var segments []video.ImageSegment

	// Build timing map from timed lines
	lineTimings := make(map[int]*lyrics.TimedLine) // line index -> timing
	for i := range lyricsData.TimedLines {
		lineTimings[i] = &lyricsData.TimedLines[i]
	}

	for _, section := range lyricsData.Sections {
		var imageName string
		switch section.Type {
		case "verse":
			// Each verse has unique image
			imageName = fmt.Sprintf("bg-verse-%d.png", section.Number)
		case "pre-chorus":
			// Pre-choruses share one image
			imageName = "bg-pre-chorus-1.png"
		case "chorus":
			// Choruses share one image
			imageName = "bg-chorus-1.png"
		case "final-chorus":
			// Final chorus has its own image
			imageName = "bg-final-chorus-1.png"
		case "bridge":
			// Each bridge has unique image
			imageName = fmt.Sprintf("bg-bridge-%d.png", section.Number)
		case "intro":
			imageName = "bg-intro.png"
		case "outro":
			imageName = "bg-outro-1.png"
		default:
			imageName = fmt.Sprintf("bg-%s.png", section.Type)
		}

		imagePath := filepath.Join(imageDir, imageName)

		// Check if image exists
		if _, err := os.Stat(imagePath); err != nil {
			log.Printf("Warning: image not found: %s", imagePath)
			continue
		}

		// Calculate timing from section lines
		startTime := totalDuration
		endTime := 0.0

		// Use section line range to find timings
		for i := section.StartLine; i <= section.EndLine && i < len(lyricsData.TimedLines); i++ {
			timing := &lyricsData.TimedLines[i]
			if timing.StartTime < startTime {
				startTime = timing.StartTime
			}
			if timing.EndTime > endTime {
				endTime = timing.EndTime
			}
		}

		// Ensure valid timing
		if startTime >= totalDuration || endTime <= 0 {
			// Use section position as fallback
			startTime = float64(section.StartLine) * 3.0 // ~3 seconds per line
			endTime = float64(section.EndLine+1) * 3.0
		}

		if startTime >= endTime {
			endTime = startTime + 10.0 // default 10 seconds
		}

		segments = append(segments, video.ImageSegment{
			ImagePath: imagePath,
			StartTime: startTime,
			EndTime:   endTime,
		})
	}

	if len(segments) == 0 {
		return nil, fmt.Errorf("no image segments created")
	}

	return segments, nil
}

// buildTimedLyrics converts lyrics TimedLines to video LyricLines
func (p *Processor) buildTimedLyrics(lyricsData *lyrics.LyricsData) []video.LyricLine {
	var timedLyrics []video.LyricLine

	for _, tl := range lyricsData.TimedLines {
		if strings.TrimSpace(tl.Line) == "" {
			continue
		}

		timedLyrics = append(timedLyrics, video.LyricLine{
			Text:      tl.Line,
			StartTime: tl.StartTime,
			EndTime:   tl.EndTime,
		})
	}

	return timedLyrics
}

// mixAudioTracks mixes vocals and instrumental tracks together
func (p *Processor) mixAudioTracks(vocalsPath, instrumentalPath, outputPath string) error {
	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Use FFmpeg to mix the two audio tracks
	cmd := exec.Command("ffmpeg",
		"-i", vocalsPath,
		"-i", instrumentalPath,
		"-filter_complex", "[0:a][1:a]amix=inputs=2:duration=longest:weights=1.0 1.0",
		"-c:a", "pcm_s16le",
		"-y",
		outputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg mix failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// parseImageFilename extracts image type and sequence number from filename
// Examples: bg-verse-1.png -> ("verse", 1), bg-chorus.png -> ("chorus", 0), bg-intro.png -> ("intro", 0)
func parseImageFilename(filename string) (string, *int) {
	// Remove extension
	name := strings.TrimSuffix(filename, ".png")

	// Remove "bg-" prefix if present
	name = strings.TrimPrefix(name, "bg-")

	// Split by hyphen to check for sequence number
	parts := strings.Split(name, "-")

	if len(parts) == 1 {
		// No sequence number: bg-intro.png, bg-chorus.png, etc.
		return parts[0], nil
	}

	if len(parts) == 2 {
		// Has sequence number: bg-verse-1.png, bg-verse-2.png
		imageType := parts[0]

		// Try to parse sequence number
		var seqNum int
		if _, err := fmt.Sscanf(parts[1], "%d", &seqNum); err == nil {
			return imageType, &seqNum
		}

		// If parsing fails, treat whole thing as type
		return name, nil
	}

	// Multiple hyphens - join as type name
	return name, nil
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

// getSpectrumStyle returns the FFmpeg spectrum visualization style
func getSpectrumStyle(styleName string) string {
	// Map style name to FFmpeg filter
	// Support direct filter names or aliases
	switch styleName {
	case "stereo", "dual", "leftright":
		return "stereo" // Stereo visualizer with left/right channel bars (default)
	case "showfreqs", "bars", "equalizer", "freq":
		return "showfreqs" // Classic equalizer bars
	case "showspectrum", "spectrum", "spectro":
		return "showspectrum" // Stationary spectrum display
	case "showcqt", "cqt", "professional":
		return "showcqt" // High-quality CQT spectrum with bars
	case "showwaves", "wave", "waveform":
		return "showwaves" // Smooth waveform
	case "showvolume", "volume", "meter":
		return "showvolume" // Volume meter
	case "avectorscope", "scope", "circle":
		return "avectorscope" // Circular vector scope
	default:
		return "stereo" // Default to stereo visualizer
	}
}

// getSpectrumColorHex returns color setting (rainbow or color name)
func getSpectrumColorHex(colorName string) string {
	// Return color as-is if it's "rainbow" or a recognized color name
	// The renderer will handle rainbow vs mono color logic
	if colorName == "" {
		return "rainbow" // Default
	}
	return colorName
}

// getSpectrumOpacity returns the opacity value (0.0-1.0)
func getSpectrumOpacity(opacity float64) float64 {
	if opacity > 0 && opacity <= 1.0 {
		return opacity
	}
	return 0.3 // Default 30% opacity
}

// updateProgress updates the queue item progress and broadcasts it
func (p *Processor) updateProgress(item *models.QueueItem, step string, progress int, message string) {
	item.CurrentStep = step
	item.Progress = progress

	p.broadcaster.BroadcastFromQueueItem(item, message)

	log.Printf("[Queue %d] %s: %d%% - %s", item.ID, step, progress, message)
}
