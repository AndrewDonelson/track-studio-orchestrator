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

// Timing adjustment constant (user feedback: offset is 1.5s too slow)
const TimingAdjustment = -1.5

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

	// Build filter for title (bottom left), copyright (bottom center), and logo (bottom right)
	var filterParts []string

	// Song title - bottom left (Saira Condensed 64, white with shadow)
	// Position: 40px from left, 40px from bottom
	titleFilter := fmt.Sprintf("drawtext=text='%s':x=40:y=h-80:fontsize=64:fontcolor=white:fontfile=/usr/share/fonts/truetype/dejavu/DejaVuSansCondensed-Bold.ttf:shadowcolor=black:shadowx=2:shadowy=2",
		escapeText(opts.Title))
	filterParts = append(filterParts, titleFilter)

	// Copyright - bottom center (Roboto 20, white with shadow)
	// Position: centered horizontally, 20px from bottom
	copyright := "All content Copyright 2017-2026 Nlaak Studios"
	copyrightFilter := fmt.Sprintf(",drawtext=text='%s':x=(w-text_w)/2:y=h-30:fontsize=20:fontcolor=white:fontfile=/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf:shadowcolor=black:shadowx=1:shadowy=1",
		escapeText(copyright))
	filterParts = append(filterParts, copyrightFilter)

	filterStr := strings.Join(filterParts, "")

	// If there's a brand logo, we need to overlay it separately using movie filter
	// For now, text only (logo would require movie filter which is more complex)
	// TODO: Add logo overlay if brand_logo_path is available

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
		return "", fmt.Errorf("ffmpeg branding overlay failed: %w\nOutput: %s", err, string(output))
	}

	return tempPath, nil
}

// addLyricsOverlay adds timed lyrics with karaoke-style highlighting to the center of video
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

	log.Printf("Applying vocal onset offset of %.2fs to %d lyric lines", vocalOnset, len(opts.LyricsData))

	// Build drawtext filters for each lyric line with timing and karaoke effect
	// Process each line individually to avoid escaping issues
	centerY := vr.Height / 2

	currentInput := inputPath

	for i, lyric := range opts.LyricsData {
		var nextPath string
		if i == len(opts.LyricsData)-1 {
			nextPath = tempPath
		} else {
			nextPath = filepath.Join(vr.TempDir, fmt.Sprintf("lyrics_step_%d.mp4", i))
		}

		// Apply vocal onset offset with timing adjustment
		startTime := lyric.StartTime + vocalOnset + TimingAdjustment
		endTime := lyric.EndTime + vocalOnset + TimingAdjustment
		duration := endTime - startTime

		// Write text to a temporary file to avoid escaping issues
		textFile := filepath.Join(vr.TempDir, fmt.Sprintf("lyric_%d.txt", i))
		if err := os.WriteFile(textFile, []byte(lyric.Text), 0644); err != nil {
			return "", fmt.Errorf("failed to write lyric text file: %w", err)
		}
		defer os.Remove(textFile)

		// Build karaoke-style filter with color transition effect
		// Since FFmpeg doesn't support dynamic color expressions in drawtext,
		// we'll use two text layers: white base + yellow overlay with alpha fade
		enableStr := fmt.Sprintf("between(t,%.2f,%.2f)", startTime, endTime)

		// Calculate alpha transition for the yellow overlay (0 to 1 over duration)
		// This creates a fade-in effect for the yellow color over the white base
		alphaTransition := fmt.Sprintf("if(lt(t,%.2f),0,if(gt(t,%.2f),1,min(1,(t-%.2f)/%.2f)))",
			startTime, endTime, startTime, duration)

		// Layer 1: White text with shadow (base layer, always visible)
		baseFilter := fmt.Sprintf("drawtext=textfile='%s':x=(w-text_w)/2:y=%d:fontsize=52:fontcolor=white:fontfile=/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf:shadowcolor=black:shadowx=3:shadowy=3:enable='%s'",
			textFile, centerY, enableStr)

		// Layer 2: Yellow overlay that fades in over time (karaoke effect)
		karaokeFilter := fmt.Sprintf(",drawtext=textfile='%s':x=(w-text_w)/2:y=%d:fontsize=52:fontcolor=yellow:fontfile=/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf:enable='%s':alpha='%s'",
			textFile, centerY, enableStr, alphaTransition)

		filterStr := baseFilter + karaokeFilter

		cmd := exec.Command("ffmpeg",
			"-i", currentInput,
			"-vf", filterStr,
			"-c:v", "libx264",
			"-preset", "ultrafast",
			"-crf", "23",
			"-c:a", "copy",
			"-y",
			nextPath,
		)

		output, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("ffmpeg lyrics overlay failed at line %d ('%s'): %w\nOutput: %s", i, lyric.Text, err, string(output))
		}

		// Clean up intermediate file
		if currentInput != inputPath {
			os.Remove(currentInput)
		}

		currentInput = nextPath
	}

	return tempPath, nil
} // addAudio adds audio to the video
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

// escapeText escapes special characters for FFmpeg drawtext filter
func escapeText(text string) string {
	// Escape single quotes, colons, and backslashes for FFmpeg
	text = strings.ReplaceAll(text, "\\", "\\\\")
	text = strings.ReplaceAll(text, "'", "\\'")
	text = strings.ReplaceAll(text, ":", "\\:")
	return text
}
