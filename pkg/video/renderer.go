package video

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// VideoRenderer handles video composition with FFmpeg
type VideoRenderer struct {
	Width     int
	Height    int
	FPS       int
	OutputDir string
	TempDir   string

	// Timing statistics
	RenderTimings    []time.Duration
	MaxTimingSamples int
}

// Note: Removed TimingAdjustment constant - caused progressive timing drift
// Using accurate vocal onset timing instead

// VideoRenderOptions contains all parameters for video rendering
type VideoRenderOptions struct {
	// Audio
	AudioPath string
	Duration  float64

	// Images
	ImagePaths []ImageSegment

	// Lyrics
	LyricsData        []LyricLine
	VocalOnset        float64 // Offset for lyrics timing (in seconds)
	CrossfadeDuration float64 // Duration of crossfade between images (default 2.0s)
	EnableKaraoke     bool    // Enable word-by-word karaoke highlighting (default false)

	// Metadata
	Key    string
	Tempo  string
	BPM    float64
	Title  string
	Artist string

	// Output
	OutputPath string
}

// ImageSegment defines when each image should be displayed
type ImageSegment struct {
	ImagePath string
	StartTime float64 // seconds
	EndTime   float64 // seconds
}

// LyricLine defines a timed lyric line
type LyricLine struct {
	Text      string
	StartTime float64
	EndTime   float64
}

func NewVideoRenderer(outputDir string) *VideoRenderer {
	return &VideoRenderer{
		Width:            1920,
		Height:           1024,
		FPS:              30,
		OutputDir:        outputDir,
		TempDir:          filepath.Join(outputDir, "temp"),
		RenderTimings:    make([]time.Duration, 0),
		MaxTimingSamples: 5,
	}
}

// RenderVideo creates the final video composition
func (vr *VideoRenderer) RenderVideo(opts *VideoRenderOptions) (string, error) {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		vr.RenderTimings = append(vr.RenderTimings, duration)
		if len(vr.RenderTimings) > vr.MaxTimingSamples {
			vr.RenderTimings = vr.RenderTimings[1:]
		}
		log.Printf("Video rendering took: %.1fs", duration.Seconds())
	}()

	// Ensure temp and output directories exist
	if err := os.MkdirAll(vr.TempDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	if err := os.MkdirAll(vr.OutputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	log.Println("Step 1/5: Creating image slideshow...")
	slideshowPath, err := vr.createImageSlideshow(opts)
	if err != nil {
		return "", fmt.Errorf("failed to create slideshow: %w", err)
	}
	defer os.Remove(slideshowPath)

	log.Println("Step 2/5: Adding metadata overlay...")
	metadataPath, err := vr.addMetadataOverlay(slideshowPath, opts)
	if err != nil {
		return "", fmt.Errorf("failed to add metadata: %w", err)
	}
	defer os.Remove(metadataPath)

	log.Println("Step 2.5/5: Adding branding overlays...")
	brandingPath, err := vr.addBrandingOverlays(metadataPath, opts)
	if err != nil {
		return "", fmt.Errorf("failed to add branding: %w", err)
	}
	defer os.Remove(brandingPath)

	log.Println("Step 3/5: Adding lyrics overlay...")
	lyricsPath, err := vr.addLyricsOverlay(brandingPath, opts)
	if err != nil {
		return "", fmt.Errorf("failed to add lyrics: %w", err)
	}
	defer os.Remove(lyricsPath)

	log.Println("Step 4/5: Adding audio...")
	audioPath, err := vr.addAudio(lyricsPath, opts.AudioPath, opts.Duration)
	if err != nil {
		return "", fmt.Errorf("failed to add audio: %w", err)
	}
	defer os.Remove(audioPath)

	log.Println("Step 5/5: Encoding final video...")
	finalPath, err := vr.encodeFinalVideo(audioPath, opts.OutputPath)
	if err != nil {
		return "", fmt.Errorf("failed to encode final video: %w", err)
	}

	log.Printf("✓ Video rendered successfully: %s", finalPath)
	return finalPath, nil
}

// createImageSlideshow creates a video from timed image segments with crossfade transitions
func (vr *VideoRenderer) createImageSlideshow(opts *VideoRenderOptions) (string, error) {
	tempPath := filepath.Join(vr.TempDir, "slideshow.mp4")

	// If only one image, create a simple static video
	if len(opts.ImagePaths) == 1 {
		return vr.createStaticImageVideo(opts.ImagePaths[0].ImagePath, opts.Duration, tempPath)
	}

	// Set default crossfade duration
	crossfadeDuration := opts.CrossfadeDuration
	if crossfadeDuration <= 0 {
		crossfadeDuration = 2.0 // default 2 seconds
	}

	// Create individual video segments for each image, extending duration for crossfade overlap
	var segmentPaths []string
	for i, seg := range opts.ImagePaths {
		duration := seg.EndTime - seg.StartTime
		if duration <= 0 {
			log.Printf("Warning: invalid segment duration for image %d: %.2f", i, duration)
			continue
		}

		// Add crossfade duration to each segment (except last) for overlap
		if i < len(opts.ImagePaths)-1 {
			duration += crossfadeDuration
		}

		segmentPath := filepath.Join(vr.TempDir, fmt.Sprintf("segment_%d.mp4", i))

		// Create video segment for this image
		_, err := vr.createStaticImageVideo(seg.ImagePath, duration, segmentPath)
		if err != nil {
			return "", fmt.Errorf("failed to create segment %d: %w", i, err)
		}
		defer os.Remove(segmentPath)

		segmentPaths = append(segmentPaths, segmentPath)
	}

	// Apply crossfade transitions between segments using xfade filter
	if len(segmentPaths) == 1 {
		// Single image (shouldn't reach here, but handle anyway)
		cmd := exec.Command("ffmpeg", "-i", segmentPaths[0], "-c", "copy", "-y", tempPath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("ffmpeg copy failed: %w\nOutput: %s", err, string(output))
		}
	} else {
		// Multiple images - apply xfade transitions
		var inputs []string
		for _, path := range segmentPaths {
			inputs = append(inputs, "-i", path)
		}

		// Build xfade filter chain
		var filterParts []string
		currentLabel := "[0:v]"
		offset := opts.ImagePaths[0].EndTime - opts.ImagePaths[0].StartTime

		for i := 1; i < len(segmentPaths); i++ {
			nextLabel := fmt.Sprintf("[v%d]", i)
			if i == len(segmentPaths)-1 {
				nextLabel = "[outv]"
			}

			filterParts = append(filterParts,
				fmt.Sprintf("%s[%d:v]xfade=transition=fade:duration=%.2f:offset=%.2f%s",
					currentLabel, i, crossfadeDuration, offset, nextLabel))

			currentLabel = nextLabel
			if i < len(segmentPaths)-1 {
				offset += opts.ImagePaths[i].EndTime - opts.ImagePaths[i].StartTime
			}
		}

		filterComplex := strings.Join(filterParts, ";")

		args := append(inputs, "-filter_complex", filterComplex, "-map", "[outv]",
			"-c:v", "libx264", "-preset", "medium", "-crf", "23", "-pix_fmt", "yuv420p",
			"-r", fmt.Sprintf("%d", vr.FPS), "-y", tempPath)

		cmd := exec.Command("ffmpeg", args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("ffmpeg xfade failed: %w\nOutput: %s", err, string(output))
		}
	}

	return tempPath, nil
}

// createStaticImageVideo creates a video from a single image with specified duration
func (vr *VideoRenderer) createStaticImageVideo(imagePath string, duration float64, outputPath string) (string, error) {
	cmd := exec.Command("ffmpeg",
		"-loop", "1",
		"-i", imagePath,
		"-t", fmt.Sprintf("%.2f", duration),
		"-vf", fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2:black",
			vr.Width, vr.Height, vr.Width, vr.Height),
		"-c:v", "libx264",
		"-pix_fmt", "yuv420p",
		"-r", fmt.Sprintf("%d", vr.FPS),
		"-y",
		outputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ffmpeg static image failed: %w\nOutput: %s", err, string(output))
	}

	return outputPath, nil
}

// addMetadataOverlay adds metadata (Key, Tempo, BPM) to the top of video
func (vr *VideoRenderer) addMetadataOverlay(inputPath string, opts *VideoRenderOptions) (string, error) {
	tempPath := filepath.Join(vr.TempDir, "with_metadata.mp4")

	overlay := DefaultMetadataOverlay()
	filterStr := overlay.GetFFmpegDrawtextFilter(opts.Key, opts.Tempo, opts.BPM, vr.Width)

	if filterStr == "" {
		// No metadata to add, just copy
		return vr.copyVideo(inputPath, tempPath)
	}

	cmd := exec.Command("ffmpeg",
		"-i", inputPath,
		"-vf", filterStr,
		"-c:v", "libx264",
		"-preset", "medium",
		"-crf", "23",
		"-c:a", "copy",
		"-y",
		tempPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ffmpeg metadata overlay failed: %w\nOutput: %s", err, string(output))
	}

	return tempPath, nil
}

// addBrandingOverlays adds song title, copyright, and brand logo to video
func (vr *VideoRenderer) addBrandingOverlays(inputPath string, opts *VideoRenderOptions) (string, error) {
	tempPath := filepath.Join(vr.TempDir, "with_branding.mp4")

	// Check if artist logo exists
	logoPath := filepath.Join("storage", "branding", "artist-logo.png")
	logoExists := false
	if _, err := os.Stat(logoPath); err == nil {
		logoExists = true
	}

	// Build filter for title (bottom left), copyright (bottom center), and logo (bottom right)
	var filterParts []string

	// Song title - bottom left (Saira Condensed 64, white with shadow)
	// Position: 40px from left, 52px from bottom (raised 12px)
	titleFilter := fmt.Sprintf("drawtext=text='%s':x=40:y=h-92:fontsize=64:fontcolor=white:fontfile=/usr/share/fonts/truetype/dejavu/DejaVuSansCondensed-Bold.ttf:shadowcolor=black:shadowx=2:shadowy=2",
		escapeText(opts.Title))
	filterParts = append(filterParts, titleFilter)

	// Copyright - bottom center (Roboto 20, white with shadow)
	// Position: centered horizontally, 20px from bottom
	copyright := "All content Copyright 2017-2026 Nlaak Studios"
	copyrightFilter := fmt.Sprintf(",drawtext=text='%s':x=(w-text_w)/2:y=h-30:fontsize=20:fontcolor=white:fontfile=/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf:shadowcolor=black:shadowx=1:shadowy=1",
		escapeText(copyright))
	filterParts = append(filterParts, copyrightFilter)

	filterStr := strings.Join(filterParts, "")

	// Build FFmpeg command with logo overlay if it exists
	var cmd *exec.Cmd
	if logoExists {
		// Use overlay filter to add logo (150x150, bottom-right, 20px margins)
		// Note: At this stage, there's no audio yet (added later in addAudio step)
		cmd = exec.Command("ffmpeg",
			"-i", inputPath,
			"-i", logoPath,
			"-filter_complex",
			fmt.Sprintf("[0:v]%s[v1];[1:v]scale=150:150[logo];[v1][logo]overlay=W-w-20:H-h-20[vout]", filterStr),
			"-map", "[vout]",
			"-c:v", "libx264",
			"-preset", "medium",
			"-crf", "23",
			"-y",
			tempPath,
		)
	} else {
		// No logo, just text overlays
		cmd = exec.Command("ffmpeg",
			"-i", inputPath,
			"-vf", filterStr,
			"-c:v", "libx264",
			"-preset", "medium",
			"-crf", "23",
			"-y",
			tempPath,
		)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ffmpeg branding overlay failed: %w\nOutput: %s", err, string(output))
	}

	return tempPath, nil
}

// addLyricsOverlay adds word-by-word karaoke lyrics with preview line
func (vr *VideoRenderer) addLyricsOverlay(inputPath string, opts *VideoRenderOptions) (string, error) {
	tempPath := filepath.Join(vr.TempDir, "with_lyrics.mp4")

	if len(opts.LyricsData) == 0 {
		// No lyrics, just copy
		return vr.copyVideo(inputPath, tempPath)
	}

	// Apply vocal onset offset to all lyrics timing
	vocalOnset := opts.VocalOnset
	if vocalOnset < 0 {
		vocalOnset = 0
	}

	log.Printf("Building word-by-word karaoke for %d lyric lines", len(opts.LyricsData))

	// Build comprehensive filter with all lyrics rendered together
	centerY := vr.Height / 2
	previewY := centerY + 80 // Preview line below active line

	var filterParts []string

	for i, lyric := range opts.LyricsData {
		// Use accurate timing: vocal onset + lyric times (no manual adjustment)
		startTime := lyric.StartTime + vocalOnset
		endTime := lyric.EndTime + vocalOnset
		duration := endTime - startTime

		// Estimate word-level timing by dividing duration evenly
		words := strings.Fields(lyric.Text)
		if len(words) == 0 {
			continue
		}

		wordDuration := duration / float64(len(words))

		fullText := escapeText(lyric.Text)

		// SOLUTION: Use expression to calculate fixed X position where centered text starts
		// Then render both white and yellow left-aligned from that position
		// Expression: X_start = (width - full_text_width) / 2
		// In FFmpeg: We render full white centered, then yellow cumulative from same start X

		// Base layer: Full white text, centered by calculating start position
		// Use x=(w-text_w)/2 which centers the full line
		baseFilter := fmt.Sprintf("drawtext=text='%s':x=(w-text_w)/2:y=%d:fontsize=52:fontcolor=white:fontfile=/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf:shadowcolor=black@0.8:shadowx=3:shadowy=3:enable=between(t\\,%.2f\\,%.2f)",
			fullText, centerY, startTime, endTime)
		filterParts = append(filterParts, baseFilter)

		// Karaoke highlighting (optional)
		if opts.EnableKaraoke {
			// Karaoke layer: Build cumulative yellow text word by word
			// Use same X calculation as white text to ensure alignment
			// Key: Don't center the cumulative text itself - position it at the same X as full white text
			//
			// Approach: Calculate fixed X position using full text width, apply to all yellow renders
			// FFmpeg limitation: Can't reference text_w of other drawtext in same filter
			//
			// Working solution: Render full line ghost in yellow to get positioning, then clip
			// Or: Use fixed X value calculated from estimated full line width
			//
			// Best practical solution: Estimate full line width, calculate center X, render cumulative from there
			estimatedCharWidth := 28.0 // DejaVu Sans Bold 52pt average
			estimatedFullWidth := float64(len(lyric.Text)) * estimatedCharWidth
			baseXPos := (float64(vr.Width) - estimatedFullWidth) / 2.0

			for wordIdx := range words {
				wordStartTime := startTime + float64(wordIdx)*wordDuration
				wordEndTime := wordStartTime + wordDuration

				cumulativeWords := words[:wordIdx+1]
				cumulativeText := escapeText(strings.Join(cumulativeWords, " "))

				// Render cumulative text at fixed base X (left-aligned from center point)
				wordHighlight := fmt.Sprintf("drawtext=text='%s':fontfile=/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf:fontsize=52:x=%.0f:y=%d:fontcolor=yellow:enable=between(t\\,%.2f\\,%.2f)",
					cumulativeText, baseXPos, centerY, wordStartTime, wordEndTime)
				filterParts = append(filterParts, wordHighlight)
			}
		}

		// Preview line (next line at 50% opacity)
		if i < len(opts.LyricsData)-1 {
			nextLyric := opts.LyricsData[i+1]
			escapedNextText := escapeText(nextLyric.Text)
			previewFilter := fmt.Sprintf("drawtext=text='%s':x=(w-text_w)/2:y=%d:fontsize=44:fontcolor=white@0.5:fontfile=/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf:enable=between(t\\,%.2f\\,%.2f)",
				escapedNextText, previewY, startTime, endTime)
			filterParts = append(filterParts, previewFilter)
		}
	}

	// Add progress indicator for intro (non-vocal sections)
	if vocalOnset > 2.0 {
		// Position at 25% from bottom (centered)
		progressBarY := int(float64(vr.Height) * 0.75)
		progressWidth := 600
		progressFilter := fmt.Sprintf("drawbox=x=(w-%d)/2:y=%d:w=%d*min(1\\,t/%.2f):h=6:color=yellow:enable=lt(t\\,%.2f)",
			progressWidth, progressBarY, progressWidth, vocalOnset, vocalOnset)
		filterParts = append(filterParts, progressFilter)

		countdownFilter := fmt.Sprintf("drawtext=text='Starting in %%{eif\\:max(0\\,%.2f-t)\\:d}s':x=(w-text_w)/2:y=%d:fontsize=36:fontcolor=yellow:fontfile=/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf:enable=lt(t\\,%.2f)",
			vocalOnset, progressBarY-40, vocalOnset)
		filterParts = append(filterParts, countdownFilter)
	}

	filterStr := strings.Join(filterParts, ",")

	cmd := exec.Command("ffmpeg",
		"-i", inputPath,
		"-vf", filterStr,
		"-c:v", "libx264",
		"-preset", "medium",
		"-crf", "23",
		"-c:a", "copy",
		"-y",
		tempPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ffmpeg lyrics overlay failed: %w\nOutput: %s", err, string(output))
	}

	return tempPath, nil
}

// addAudio adds audio to the video
func (vr *VideoRenderer) addAudio(videoPath, audioPath string, duration float64) (string, error) {
	tempPath := filepath.Join(vr.TempDir, "with_audio.mp4")

	cmd := exec.Command("ffmpeg",
		"-i", videoPath,
		"-i", audioPath,
		"-c:v", "copy",
		"-c:a", "aac",
		"-b:a", "192k",
		"-shortest",
		"-y",
		tempPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ffmpeg add audio failed: %w\nOutput: %s", err, string(output))
	}

	return tempPath, nil
}

// encodeFinalVideo encodes the final video with optimized settings
func (vr *VideoRenderer) encodeFinalVideo(inputPath, outputPath string) (string, error) {
	cmd := exec.Command("ffmpeg",
		"-i", inputPath,
		"-c:v", "libx264",
		"-preset", "medium",
		"-crf", "20",
		"-c:a", "aac",
		"-b:a", "192k",
		"-movflags", "+faststart",
		"-y",
		outputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ffmpeg final encode failed: %w\nOutput: %s", err, string(output))
	}

	return outputPath, nil
}

// copyVideo copies a video file
func (vr *VideoRenderer) copyVideo(inputPath, outputPath string) (string, error) {
	cmd := exec.Command("ffmpeg",
		"-i", inputPath,
		"-c", "copy",
		"-y",
		outputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ffmpeg copy failed: %w\nOutput: %s", err, string(output))
	}

	return outputPath, nil
}

// escapeText escapes special characters for FFmpeg drawtext
func (vr *VideoRenderer) escapeText(text string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		"'", "\\'",
		":", "\\:",
		"%", "\\%",
	)
	return replacer.Replace(text)
}

// GetAverageRenderTime returns the average video render time
func (vr *VideoRenderer) GetAverageRenderTime() time.Duration {
	if len(vr.RenderTimings) == 0 {
		return 0
	}
	var total time.Duration
	for _, t := range vr.RenderTimings {
		total += t
	}
	return total / time.Duration(len(vr.RenderTimings))
}

// sanitizeText removes problematic characters that cause FFmpeg issues
// Keeps: letters, numbers, spaces, comma, period, question mark, exclamation, dash, parentheses
func sanitizeText(text string) string {
	var result strings.Builder
	for _, r := range text {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z': // letters
			result.WriteRune(r)
		case r >= '0' && r <= '9': // numbers
			result.WriteRune(r)
		case r == ' ', r == ',', r == '.', r == '?', r == '!', r == '-', r == '(', r == ')':
			result.WriteRune(r)
			// Skip all other characters including quotes, apostrophes, colons, semicolons, etc.
		}
	}
	return result.String()
}

// escapeText escapes special characters for FFmpeg drawtext filter
func escapeText(text string) string {
	// First sanitize to remove problematic characters
	text = sanitizeText(text)
	// Then escape remaining special chars for FFmpeg
	text = strings.ReplaceAll(text, "\\", "\\\\")
	text = strings.ReplaceAll(text, "'", "\\'")
	text = strings.ReplaceAll(text, ":", "\\:")
	return text
}
