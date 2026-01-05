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
	ASSSubtitlePath   string  // Path to ASS subtitle file for karaoke (optional)

	// Metadata
	Key    string
	Tempo  string
	BPM    float64
	Title  string
	Artist string

	// Spectrum Analyzer
	SpectrumStyle   string  // "showwaves", "showfreqs", "showspectrum", etc.
	SpectrumColor   string  // Color for spectrum (hex or color name)
	SpectrumOpacity float64 // Opacity for spectrum overlay (0.0-1.0)

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
	slideshowPath := filepath.Join(vr.TempDir, "slideshow.mp4")
	if err := vr.createImageSlideshow(opts, slideshowPath); err != nil {
		return "", fmt.Errorf("failed to create slideshow: %w", err)
	}
	defer os.Remove(slideshowPath)

	log.Println("Step 2/5: Adding spectrum analyzer overlay...")
	spectrumPath, err := vr.addSpectrumAnalyzer(slideshowPath, opts)
	if err != nil {
		return "", fmt.Errorf("failed to add spectrum analyzer: %w", err)
	}
	defer os.Remove(spectrumPath)

	log.Println("Step 3/5: Adding metadata and branding overlays...")
	metadataPath, err := vr.addMetadataOverlays(spectrumPath, opts)
	if err != nil {
		return "", fmt.Errorf("failed to add metadata: %w", err)
	}
	defer os.Remove(metadataPath)

	log.Println("Step 4/5: Adding lyrics overlay...")
	lyricsPath, err := vr.addLyricsOverlay(metadataPath, opts)
	if err != nil {
		return "", fmt.Errorf("failed to add lyrics: %w", err)
	}
	defer os.Remove(lyricsPath)

	log.Println("Step 5/5: Adding audio and encoding final video...")
	finalPath, err := vr.addAudioAndEncode(lyricsPath, opts.AudioPath, opts.Duration, opts.OutputPath)
	if err != nil {
		return "", fmt.Errorf("failed to encode final video: %w", err)
	}

	log.Printf("✓ Video rendered successfully: %s", finalPath)
	return finalPath, nil
}

// addMetadataOverlays adds metadata text and logo to video (after spectrum analyzer)
func (vr *VideoRenderer) addMetadataOverlays(inputPath string, opts *VideoRenderOptions) (string, error) {
	tempPath := filepath.Join(vr.TempDir, "with_metadata.mp4")

	// Build comprehensive filter for metadata + branding
	var filterParts []string

	// Top bar - Yellow/Gold text (Saira Condensed 48pt)
	// KEY (Top-Left, aligned left, 20px from edges)
	if opts.Key != "" {
		keyFilter := fmt.Sprintf("drawtext=text='KEY\\\\: %s':x=20:y=20:fontsize=48:fontcolor=0xFFD700:fontfile=/usr/share/fonts/truetype/dejavu/DejaVuSansCondensed-Bold.ttf:shadowcolor=black@0.7:shadowx=2:shadowy=2",
			escapeText(opts.Key))
		filterParts = append(filterParts, keyFilter)
	}

	// TEMPO (Top-Center, aligned center)
	if opts.Tempo != "" {
		tempoFilter := fmt.Sprintf("drawtext=text='%s':x=(w-text_w)/2:y=20:fontsize=48:fontcolor=0xFFD700:fontfile=/usr/share/fonts/truetype/dejavu/DejaVuSansCondensed-Bold.ttf:shadowcolor=black@0.7:shadowx=2:shadowy=2",
			escapeText(opts.Tempo))
		filterParts = append(filterParts, tempoFilter)
	}

	// BPM (Top-Right, aligned right, 20px from edge)
	if opts.BPM > 0 {
		bpmFilter := fmt.Sprintf("drawtext=text='BPM\\\\: %.0f':x=w-text_w-20:y=20:fontsize=48:fontcolor=0xFFD700:fontfile=/usr/share/fonts/truetype/dejavu/DejaVuSansCondensed-Bold.ttf:shadowcolor=black@0.7:shadowx=2:shadowy=2",
			opts.BPM)
		filterParts = append(filterParts, bpmFilter)
	}

	// Bottom bar - Title (yellow/gold), Copyright (white), Logo (image overlay)
	// Song title - bottom left (Saira Condensed 64, yellow/gold)
	// Position: 20px from left, 96px from bottom (raised 16px)
	titleFilter := fmt.Sprintf("drawtext=text='%s':x=20:y=h-96:fontsize=64:fontcolor=0xFFD700:fontfile=/usr/share/fonts/truetype/dejavu/DejaVuSansCondensed-Bold.ttf:shadowcolor=black@0.7:shadowx=2:shadowy=2",
		escapeText(opts.Title))
	filterParts = append(filterParts, titleFilter)

	// Copyright - bottom center (Roboto 20, white)
	// Position: centered horizontally, 25px from bottom
	copyright := "All content Copyright 2017-2026 Nlaak Studios"
	copyrightFilter := fmt.Sprintf("drawtext=text='%s':x=(w-text_w)/2:y=h-25:fontsize=20:fontcolor=white:fontfile=/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf:shadowcolor=black@0.7:shadowx=1:shadowy=1",
		escapeText(copyright))
	filterParts = append(filterParts, copyrightFilter)

	filterStr := strings.Join(filterParts, ",")

	// Check if artist logo exists for overlay
	logoPath := filepath.Join("storage", "branding", "artist-logo.png")
	logoExists := false
	if _, err := os.Stat(logoPath); err == nil {
		logoExists = true
	}

	var cmd *exec.Cmd
	if logoExists {
		// Use filter_complex to add text overlays + logo overlay (256x256 with 70% opacity, bottom-right, 20px margins)
		cmd = exec.Command("ffmpeg",
			"-i", inputPath,
			"-i", logoPath,
			"-filter_complex",
			fmt.Sprintf("[0:v]%s[v1];[1:v]scale=256:256,format=rgba,colorchannelmixer=aa=0.7[logo];[v1][logo]overlay=W-w-20:H-h-20[vout]", filterStr),
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
		return "", fmt.Errorf("ffmpeg metadata overlay failed: %w\nOutput: %s", err, string(output))
	}

	return tempPath, nil
}

// createBasicVideo creates slideshow with all static overlays (metadata, title, copyright, logo)
// DEPRECATED: Use createImageSlideshow + addMetadataOverlays separately for better layering
func (vr *VideoRenderer) createBasicVideo(opts *VideoRenderOptions) (string, error) {
	tempPath := filepath.Join(vr.TempDir, "base_with_overlays.mp4")

	// Step 1: Create image slideshow
	slideshowPath := filepath.Join(vr.TempDir, "slideshow.mp4")
	if err := vr.createImageSlideshow(opts, slideshowPath); err != nil {
		return "", fmt.Errorf("failed to create slideshow: %w", err)
	}
	defer os.Remove(slideshowPath)

	// Step 2: Add all static overlays in one pass
	// Build comprehensive filter for metadata + branding
	var filterParts []string

	// Top bar - Yellow/Gold text (Saira Condensed 48pt)
	// KEY (Top-Left, aligned left, 20px from edges)
	if opts.Key != "" {
		keyFilter := fmt.Sprintf("drawtext=text='KEY\\\\: %s':x=20:y=20:fontsize=48:fontcolor=0xFFD700:fontfile=/usr/share/fonts/truetype/dejavu/DejaVuSansCondensed-Bold.ttf:shadowcolor=black@0.7:shadowx=2:shadowy=2",
			escapeText(opts.Key))
		filterParts = append(filterParts, keyFilter)
	}

	// TEMPO (Top-Center, aligned center)
	if opts.Tempo != "" {
		tempoFilter := fmt.Sprintf("drawtext=text='%s':x=(w-text_w)/2:y=20:fontsize=48:fontcolor=0xFFD700:fontfile=/usr/share/fonts/truetype/dejavu/DejaVuSansCondensed-Bold.ttf:shadowcolor=black@0.7:shadowx=2:shadowy=2",
			escapeText(opts.Tempo))
		filterParts = append(filterParts, tempoFilter)
	}

	// BPM (Top-Right, aligned right, 20px from edge)
	if opts.BPM > 0 {
		bpmFilter := fmt.Sprintf("drawtext=text='BPM\\\\: %.0f':x=w-text_w-20:y=20:fontsize=48:fontcolor=0xFFD700:fontfile=/usr/share/fonts/truetype/dejavu/DejaVuSansCondensed-Bold.ttf:shadowcolor=black@0.7:shadowx=2:shadowy=2",
			opts.BPM)
		filterParts = append(filterParts, bpmFilter)
	}

	// Bottom bar - Title (yellow/gold), Copyright (white), Logo (image overlay)
	// Song title - bottom left (Saira Condensed 64, yellow/gold)
	// Position: 20px from left, 96px from bottom (raised 16px)
	titleFilter := fmt.Sprintf("drawtext=text='%s':x=20:y=h-96:fontsize=64:fontcolor=0xFFD700:fontfile=/usr/share/fonts/truetype/dejavu/DejaVuSansCondensed-Bold.ttf:shadowcolor=black@0.7:shadowx=2:shadowy=2",
		escapeText(opts.Title))
	filterParts = append(filterParts, titleFilter)

	// Copyright - bottom center (Roboto 20, white)
	// Position: centered horizontally, 25px from bottom
	copyright := "All content Copyright 2017-2026 Nlaak Studios"
	copyrightFilter := fmt.Sprintf("drawtext=text='%s':x=(w-text_w)/2:y=h-25:fontsize=20:fontcolor=white:fontfile=/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf:shadowcolor=black@0.7:shadowx=1:shadowy=1",
		escapeText(copyright))
	filterParts = append(filterParts, copyrightFilter)

	filterStr := strings.Join(filterParts, ",")

	// Check if artist logo exists for overlay
	logoPath := filepath.Join("storage", "branding", "artist-logo.png")
	logoExists := false
	if _, err := os.Stat(logoPath); err == nil {
		logoExists = true
	}

	var cmd *exec.Cmd
	if logoExists {
		// Use filter_complex to add text overlays + logo overlay (256x256 with 70% opacity, bottom-right, 20px margins)
		cmd = exec.Command("ffmpeg",
			"-i", slideshowPath,
			"-i", logoPath,
			"-filter_complex",
			fmt.Sprintf("[0:v]%s[v1];[1:v]scale=256:256,format=rgba,colorchannelmixer=aa=0.7[logo];[v1][logo]overlay=W-w-20:H-h-20[vout]", filterStr),
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
			"-i", slideshowPath,
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
		return "", fmt.Errorf("ffmpeg basic video creation failed: %w\nOutput: %s", err, string(output))
	}

	return tempPath, nil
}

// addSpectrumAnalyzer adds audio spectrum visualization overlay
func (vr *VideoRenderer) addSpectrumAnalyzer(inputPath string, opts *VideoRenderOptions) (string, error) {
	tempPath := filepath.Join(vr.TempDir, "spectrum_"+filepath.Base(opts.OutputPath))

	// Default spectrum settings if not specified
	spectrumStyle := opts.SpectrumStyle
	if spectrumStyle == "" {
		spectrumStyle = "stereo" // Default to stereo spectrum visualizer
	}

	spectrumColor := opts.SpectrumColor
	if spectrumColor == "" {
		spectrumColor = "charcoal" // Default charcoal gray
	}

	spectrumOpacity := opts.SpectrumOpacity
	if spectrumOpacity == 0 {
		spectrumOpacity = 0.3 // Default 30% opacity
	}

	// Determine if using rainbow or mono color
	useRainbow := (spectrumColor == "rainbow")
	monoColorHex := "0x00FFFF" // Default bright cyan

	// Map color names to bright hex values for spectrum visualization
	if !useRainbow {
		colorMap := map[string]string{
			"charcoal": "0x808080", // Medium gray (brighter than 0x303030)
			"cyan":     "0x00FFFF", // Bright cyan
			"blue":     "0x0080FF", // Bright blue
			"red":      "0xFF0000", // Bright red
			"green":    "0x00FF00", // Bright green
			"yellow":   "0xFFFF00", // Bright yellow
			"magenta":  "0xFF00FF", // Bright magenta
			"white":    "0xFFFFFF", // White
			"orange":   "0xFF8000", // Bright orange
			"purple":   "0x8000FF", // Bright purple
			"pink":     "0xFF00FF", // Bright pink (magenta)
			"gold":     "0xFFD700", // Gold
		}

		if hex, ok := colorMap[spectrumColor]; ok {
			monoColorHex = hex
		}
	}

	// Build spectrum visualization filter based on style
	var spectrumFilter string
	var filterComplex string

	switch spectrumStyle {
	case "showwaves":
		// Smooth waveform
		if useRainbow {
			// Rainbow gradient waveform
			spectrumFilter = fmt.Sprintf("[1:a]showwaves=s=%dx%d:mode=cline:colors=red|orange|yellow|green|cyan|blue|violet:scale=sqrt,format=rgba,colorchannelmixer=aa=%.2f[spectrum]",
				vr.Width, vr.Height, spectrumOpacity)
		} else {
			// Mono color waveform with explicit hex color
			spectrumFilter = fmt.Sprintf("[1:a]showwaves=s=%dx%d:mode=cline:colors=%s:scale=sqrt,format=rgba,colorchannelmixer=aa=%.2f[spectrum]",
				vr.Width, vr.Height, monoColorHex, spectrumOpacity)
		}

	case "showfreqs", "bars", "equalizer":
		// Frequency spectrum bars (classic equalizer bars) - vertical bars dancing with music
		if useRainbow {
			// Rainbow gradient bars
			spectrumFilter = fmt.Sprintf("[1:a]showfreqs=s=%dx%d:mode=bar:fscale=log:ascale=sqrt:win_size=4096:colors=red|orange|yellow|green|cyan|blue|violet,format=rgba,colorchannelmixer=aa=%.2f[spectrum]",
				vr.Width, vr.Height, spectrumOpacity)
		} else {
			// Mono color bars with explicit hex color for brightness
			spectrumFilter = fmt.Sprintf("[1:a]showfreqs=s=%dx%d:mode=bar:fscale=log:ascale=sqrt:win_size=4096:colors=%s,format=rgba,colorchannelmixer=aa=%.2f[spectrum]",
				vr.Width, vr.Height, monoColorHex, spectrumOpacity)
		}

	case "showspectrum", "spectrum":
		// Full spectrum visualization (stationary, not scrolling)
		if useRainbow {
			// Rainbow gradient spectrum
			spectrumFilter = fmt.Sprintf("[1:a]showspectrum=s=%dx%d:slide=replace:color=rainbow:scale=sqrt:saturation=3,format=rgba,colorchannelmixer=aa=%.2f[spectrum]",
				vr.Width, vr.Height, spectrumOpacity)
		} else {
			// Mono color spectrum
			spectrumFilter = fmt.Sprintf("[1:a]showspectrum=s=%dx%d:slide=replace:color=intensity:scale=sqrt,format=rgba,colorchannelmixer=aa=%.2f[spectrum]",
				vr.Width, vr.Height, spectrumOpacity)
		}

	case "showcqt", "cqt":
		// High-quality Constant Q Transform spectrum with bars
		// Frequency range: 50Hz to 20kHz
		// CQT has built-in colorization, opacity applied after
		spectrumFilter = fmt.Sprintf("[1:a]showcqt=s=%dx%d:fps=30:bar_h=%d:sono_h=0:bar_t=%.2f:basefreq=50:endfreq=20000,format=rgba[spectrum]",
			vr.Width, vr.Height, vr.Height/3, spectrumOpacity)

	case "showvolume":
		// Volume meter
		spectrumFilter = fmt.Sprintf("[1:a]showvolume=w=%d:h=%d:b=4:f=%.2f,format=rgba,colorchannelmixer=aa=%.2f[spectrum]",
			vr.Width/4, vr.Height/10, spectrumOpacity, spectrumOpacity)

	case "avectorscope":
		// Circular vector scope (stereo field visualization)
		spectrumFilter = fmt.Sprintf("[1:a]avectorscope=s=%dx%d:zoom=1.5:draw=line,format=rgba,colorchannelmixer=aa=%.2f[spectrum]",
			vr.Width, vr.Height, spectrumOpacity)

	case "stereo", "":
		// Stereo spectrum visualizer - left/right channel bars on edges growing inward
		barWidth := 300               // How far bars extend inward from edge
		visualizerHeight := vr.Height // Full height (1024)

		var colorParam string

		if useRainbow {
			colorParam = ":colors=red|orange|yellow|green|cyan|blue|violet"
		} else {
			colorParam = ":colors=white"
		}

		// Left channel: transpose=2 (90° CCW), then hflip so bars grow INWARD (rightward)
		// Bars positioned at left edge (x=0), extending toward center
		leftChain := fmt.Sprintf("s=%dx%d:mode=bar:fscale=log:ascale=log%s,transpose=2,hflip,format=yuva420p,colorchannelmixer=aa=%.2f",
			visualizerHeight, barWidth, colorParam, spectrumOpacity)

		// Right channel: transpose=1 (90° CW), hflip for inward growth, vflip to match left frequency orientation
		// Bars positioned at right edge (x=W-w), extending toward center, low freqs at bottom
		rightChain := fmt.Sprintf("s=%dx%d:mode=bar:fscale=log:ascale=log%s,transpose=1,hflip,vflip,format=yuva420p,colorchannelmixer=aa=%.2f",
			visualizerHeight, barWidth, colorParam, spectrumOpacity)

		if !useRainbow {
			leftChain += ",eq=saturation=0"
			rightChain += ",eq=saturation=0"
		}

		spectrumFilter = fmt.Sprintf(
			"[1:a]channelsplit=channel_layout=stereo[L][R];"+
				"[L]showfreqs=%s[left_vis];"+
				"[R]showfreqs=%s[right_vis]",
			leftChain, rightChain)

		// Now overlay bars at edges: left at x=0, right at x=W-w
		filterComplex = fmt.Sprintf(
			"%s;"+
				"[0:v][left_vis]overlay=0:0[v1];"+
				"[v1][right_vis]overlay=W-w:0[outv]",
			spectrumFilter) // Skip the default overlay logic below
		goto applyFilter

	default:
		// Fallback: Simple waveform at bottom
		waveHeight := vr.Height / 4
		spectrumFilter = fmt.Sprintf("[1:a]showwaves=s=%dx%d:mode=cline:colors=%s:rate=25,format=rgba,colorchannelmixer=aa=%.2f[spectrum]",
			vr.Width, waveHeight, monoColorHex, spectrumOpacity)
	}

	// Determine overlay position (stereo mode jumps here directly)
	if filterComplex == "" {
		if spectrumStyle == "showfreqs" || spectrumStyle == "bars" || spectrumStyle == "equalizer" {
			// Position at bottom of screen
			waveHeight := vr.Height / 4
			yPosition := vr.Height - waveHeight
			filterComplex = fmt.Sprintf("%s;[0:v][spectrum]overlay=0:%d[outv]", spectrumFilter, yPosition)
		} else {
			// Default: fullscreen overlay
			filterComplex = fmt.Sprintf("%s;[0:v][spectrum]overlay=0:0[outv]", spectrumFilter)
		}
	}

applyFilter:
	cmd := exec.Command("ffmpeg",
		"-i", inputPath,
		"-i", opts.AudioPath,
		"-filter_complex", filterComplex,
		"-map", "[outv]",
		"-map", "1:a",
		"-c:v", "libx264",
		"-c:a", "aac",
		"-b:a", "192k",
		"-preset", "medium",
		"-crf", "23",
		"-t", fmt.Sprintf("%.2f", opts.Duration),
		"-y",
		tempPath,
	)

	// DEBUG: Log the exact FFmpeg command
	log.Printf("[SPECTRUM DEBUG] Filter: %s", filterComplex)
	log.Printf("[SPECTRUM DEBUG] Full command: ffmpeg -i %s -i %s -filter_complex '%s' -map '[outv]' -map '1:a' -c:v libx264 -c:a aac -b:a 192k -preset medium -crf 23 -t %.2f -y %s",
		inputPath, opts.AudioPath, filterComplex, opts.Duration, tempPath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ffmpeg spectrum analyzer failed: %w\nOutput: %s", err, string(output))
	}

	colorMode := "rainbow"
	if !useRainbow {
		colorMode = spectrumColor
	}
	log.Printf("Added %s spectrum analyzer (%s, %.0f%% opacity)", spectrumStyle, colorMode, spectrumOpacity*100)
	return tempPath, nil
}

// getColorHex converts color name to hex value for FFmpeg
func getColorHex(colorName string) string {
	colors := map[string]string{
		"charcoal": "0x303030", // rgb(48,48,48)
		"cyan":     "0x00FFFF",
		"blue":     "0x0000FF",
		"red":      "0xFF0000",
		"green":    "0x00FF00",
		"yellow":   "0xFFFF00",
		"magenta":  "0xFF00FF",
		"white":    "0xFFFFFF",
		"orange":   "0xFFA500",
		"purple":   "0x800080",
		"pink":     "0xFFC0CB",
		"gold":     "0xFFD700",
	}

	if hex, ok := colors[colorName]; ok {
		return hex
	}
	return "0x303030" // Default charcoal
}

// createImageSlideshow creates a video from timed image segments with crossfade transitions
func (vr *VideoRenderer) createImageSlideshow(opts *VideoRenderOptions, outputPath string) error {
	tempPath := outputPath

	// If only one image, create a simple static video
	if len(opts.ImagePaths) == 1 {
		_, err := vr.createStaticImageVideo(opts.ImagePaths[0].ImagePath, opts.Duration, tempPath)
		return err
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
			return fmt.Errorf("failed to create segment %d: %w", i, err)
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
			return fmt.Errorf("ffmpeg copy failed: %w\nOutput: %s", err, string(output))
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
			return fmt.Errorf("ffmpeg xfade failed: %w\nOutput: %s", err, string(output))
		}
	}

	return nil
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

	// If ASS subtitle file is provided, use it for karaoke
	if opts.ASSSubtitlePath != "" && fileExists(opts.ASSSubtitlePath) {
		log.Printf("Using ASS karaoke subtitles: %s", opts.ASSSubtitlePath)
		return vr.addASSSubtitles(inputPath, opts.ASSSubtitlePath, tempPath)
	}

	if len(opts.LyricsData) == 0 {
		// No lyrics, just copy
		return vr.copyVideo(inputPath, tempPath)
	}

	// Apply vocal onset offset to all lyrics timing
	vocalOnset := opts.VocalOnset
	if vocalOnset < 0 {
		vocalOnset = 0
	}

	log.Printf("Building multi-line lyrics display for %d lyric lines", len(opts.LyricsData))

	// Break long lyrics into display lines
	type DisplayLine struct {
		Text      string
		StartTime float64
		EndTime   float64
		LineIndex int // Which lyric line this came from
	}

	var displayLines []DisplayLine
	maxCharsPerLine := 38 // Max characters before breaking (reduced from 45 to prevent clipping)

	for i, lyric := range opts.LyricsData {
		text := lyric.Text
		startTime := lyric.StartTime + vocalOnset
		endTime := lyric.EndTime + vocalOnset

		// Check if line needs breaking
		if len(text) <= maxCharsPerLine {
			displayLines = append(displayLines, DisplayLine{
				Text:      text,
				StartTime: startTime,
				EndTime:   endTime,
				LineIndex: i,
			})
		} else {
			// Try to break at comma ANYWHERE in the text (not just middle 30-70%)
			commaPos := -1
			// Find the LAST comma before maxCharsPerLine
			for idx := min(len(text)-1, maxCharsPerLine); idx > 0; idx-- {
				if text[idx] == ',' {
					commaPos = idx
					break
				}
			}
			// If no comma in first maxChars, try ANY comma
			if commaPos < 0 {
				for idx, ch := range text {
					if ch == ',' {
						commaPos = idx
						break
					}
				}
			}

			duration := endTime - startTime
			if commaPos > 0 && commaPos < len(text)-1 {
				// Break at comma
				line1 := strings.TrimSpace(text[:commaPos+1])
				line2 := strings.TrimSpace(text[commaPos+1:])

				// Check if line2 is still too long, recursively break it
				if len(line2) > maxCharsPerLine {
					// Split the time proportionally
					line1Ratio := float64(len(line1)) / float64(len(text))
					line1Time := startTime + duration*line1Ratio

					displayLines = append(displayLines, DisplayLine{
						Text:      line1,
						StartTime: startTime,
						EndTime:   line1Time,
						LineIndex: i,
					})

					// Recursively process line2 by adding it back to processing
					// For now, just split at midpoint
					midPoint := len(line2) / 2
					subLine1 := strings.TrimSpace(line2[:midPoint])
					subLine2 := strings.TrimSpace(line2[midPoint:])
					midTime := line1Time + (endTime-line1Time)*0.5

					displayLines = append(displayLines, DisplayLine{
						Text:      subLine1,
						StartTime: line1Time,
						EndTime:   midTime,
						LineIndex: i,
					})
					displayLines = append(displayLines, DisplayLine{
						Text:      subLine2,
						StartTime: midTime,
						EndTime:   endTime,
						LineIndex: i,
					})
				} else {
					// Simple two-line break
					line1Ratio := float64(len(line1)) / float64(len(text))
					midTime := startTime + duration*line1Ratio

					displayLines = append(displayLines, DisplayLine{
						Text:      line1,
						StartTime: startTime,
						EndTime:   midTime,
						LineIndex: i,
					})
					displayLines = append(displayLines, DisplayLine{
						Text:      line2,
						StartTime: midTime,
						EndTime:   endTime,
						LineIndex: i,
					})
				}
			} else {
				// Break at last space before max chars (fixed bounds check)
				breakPos := -1
				for idx := min(maxCharsPerLine-1, len(text)-1); idx > 0; idx-- {
					if text[idx] == ' ' {
						breakPos = idx
						break
					}
				}
				if breakPos <= 0 {
					// Force break at maxCharsPerLine if no space found
					breakPos = maxCharsPerLine
				}
				line1 := strings.TrimSpace(text[:breakPos])
				line2 := strings.TrimSpace(text[breakPos:])
				midTime := startTime + duration*0.5

				displayLines = append(displayLines, DisplayLine{
					Text:      line1,
					StartTime: startTime,
					EndTime:   midTime,
					LineIndex: i,
				})
				displayLines = append(displayLines, DisplayLine{
					Text:      line2,
					StartTime: midTime,
					EndTime:   endTime,
					LineIndex: i,
				})
			}
		}
	}

	// Build filter for multi-line display with scrolling
	// Y positions for 4 lines (center screen, avoid top/bottom bars)
	centerY := vr.Height / 2
	lineSpacing := 80
	line1Y := centerY - lineSpacing   // Active line (100% opacity)
	line2Y := centerY                 // Next line (50% opacity)
	line3Y := centerY + lineSpacing   // Future line (30% opacity)
	line4Y := centerY + lineSpacing*2 // Future line (10% opacity)

	var filterParts []string

	// Render each display line at all 4 positions with appropriate timing and opacity
	for i, line := range displayLines {
		escapedText := escapeText(line.Text)

		// Position 1: Active line (100% opacity, blue with white border)
		filter1 := fmt.Sprintf("drawtext=text='%s':x=(w-text_w)/2:y=%d:fontsize=64:fontcolor=0x4169E1:fontfile=/usr/share/fonts/truetype/dejavu/DejaVuSansCondensed-Bold.ttf:borderw=3:bordercolor=white:enable=between(t\\,%.2f\\,%.2f)",
			escapedText, line1Y, line.StartTime, line.EndTime)
		filterParts = append(filterParts, filter1)

		// Position 2: Next line (50% opacity) - show NEXT line (i+1) while current is active
		if i < len(displayLines)-1 {
			nextLine := displayLines[i+1]
			nextEscapedText := escapeText(nextLine.Text)
			filter2 := fmt.Sprintf("drawtext=text='%s':x=(w-text_w)/2:y=%d:fontsize=64:fontcolor=0x4169E1@0.5:fontfile=/usr/share/fonts/truetype/dejavu/DejaVuSansCondensed-Bold.ttf:borderw=3:bordercolor=white@0.5:enable=between(t\\,%.2f\\,%.2f)",
				nextEscapedText, line2Y, line.StartTime, line.EndTime)
			filterParts = append(filterParts, filter2)
		}

		// Position 3: Future line (30% opacity) - show line i+2 while current is active
		if i < len(displayLines)-2 {
			next2Line := displayLines[i+2]
			next2EscapedText := escapeText(next2Line.Text)
			filter3 := fmt.Sprintf("drawtext=text='%s':x=(w-text_w)/2:y=%d:fontsize=64:fontcolor=0x4169E1@0.3:fontfile=/usr/share/fonts/truetype/dejavu/DejaVuSansCondensed-Bold.ttf:borderw=3:bordercolor=white@0.3:enable=between(t\\,%.2f\\,%.2f)",
				next2EscapedText, line3Y, line.StartTime, line.EndTime)
			filterParts = append(filterParts, filter3)
		}

		// Position 4: Future line (10% opacity) - show line i+3 while current is active
		if i < len(displayLines)-3 {
			next3Line := displayLines[i+3]
			next3EscapedText := escapeText(next3Line.Text)
			filter4 := fmt.Sprintf("drawtext=text='%s':x=(w-text_w)/2:y=%d:fontsize=64:fontcolor=0x4169E1@0.1:fontfile=/usr/share/fonts/truetype/dejavu/DejaVuSansCondensed-Bold.ttf:borderw=3:bordercolor=white@0.1:enable=between(t\\,%.2f\\,%.2f)",
				next3EscapedText, line4Y, line.StartTime, line.EndTime)
			filterParts = append(filterParts, filter4)
		}
	}

	// Add progress indicator for intro (non-vocal sections)
	if vocalOnset > 2.0 {
		// Position at 25% from bottom (centered)
		progressBarY := int(float64(vr.Height) * 0.75)
		progressWidth := 600
		progressFilter := fmt.Sprintf("drawbox=x=(w-%d)/2:y=%d:w=%d*min(1\\,t/%.2f):h=6:color=0xFFD700:enable=lt(t\\,%.2f)",
			progressWidth, progressBarY, progressWidth, vocalOnset, vocalOnset)
		filterParts = append(filterParts, progressFilter)

		countdownFilter := fmt.Sprintf("drawtext=text='Starting in %%{eif\\:max(0\\,%.2f-t)\\:d}s':x=(w-text_w)/2:y=%d:fontsize=36:fontcolor=0xFFD700:fontfile=/usr/share/fonts/truetype/dejavu/DejaVuSansCondensed-Bold.ttf:shadowcolor=black@0.7:shadowx=2:shadowy=2:enable=lt(t\\,%.2f)",
			vocalOnset, progressBarY-40, vocalOnset)
		filterParts = append(filterParts, countdownFilter)
	}

	filterStr := strings.Join(filterParts, ",")

	// For very long filter strings (many lyrics), write to file to avoid ARG_MAX limit
	var cmd *exec.Cmd
	if len(filterStr) > 100000 { // ~100KB threshold
		// Write filter to temporary file
		filterFile, err := os.CreateTemp("", "ffmpeg-filter-*.txt")
		if err != nil {
			return "", fmt.Errorf("failed to create filter file: %w", err)
		}
		defer os.Remove(filterFile.Name())
		defer filterFile.Close()

		if _, err := filterFile.WriteString(filterStr); err != nil {
			return "", fmt.Errorf("failed to write filter file: %w", err)
		}
		filterFile.Close()

		log.Printf("Using filter file (filter length: %d bytes) for lyrics overlay", len(filterStr))
		cmd = exec.Command("ffmpeg",
			"-i", inputPath,
			"-filter_complex_script", filterFile.Name(),
			"-c:v", "libx264",
			"-preset", "medium",
			"-crf", "23",
			"-y",
			tempPath,
		)
	} else {
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
		return "", fmt.Errorf("ffmpeg lyrics overlay failed: %w\nOutput: %s", err, string(output))
	}

	return tempPath, nil
}

// addAudio adds audio to the video
// addAudioAndEncode adds audio and encodes final video in one step
func (vr *VideoRenderer) addAudioAndEncode(videoPath, audioPath string, duration float64, outputPath string) (string, error) {
	cmd := exec.Command("ffmpeg",
		"-i", videoPath,
		"-i", audioPath,
		"-c:v", "libx264",
		"-preset", "medium",
		"-crf", "23",
		"-c:a", "aac",
		"-b:a", "192k",
		"-shortest",
		"-y",
		outputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ffmpeg add audio and encode failed: %w\nOutput: %s", err, string(output))
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

// addASSSubtitles adds ASS karaoke subtitles to the video with logo overlay
func (vr *VideoRenderer) addASSSubtitles(inputPath, assPath, outputPath string) (string, error) {
	log.Printf("Adding ASS subtitles from: %s", assPath)

	// Check if artist logo exists for overlay
	logoPath := filepath.Join("storage", "branding", "artist-logo.png")
	logoExists := false
	if _, err := os.Stat(logoPath); err == nil {
		logoExists = true
	}

	var cmd *exec.Cmd
	if logoExists {
		// Use filter_complex to add ASS subtitles + logo overlay (256x256 with 70% opacity, bottom-right, 20px margins)
		cmd = exec.Command("ffmpeg",
			"-i", inputPath,
			"-i", logoPath,
			"-filter_complex",
			fmt.Sprintf("[0:v]subtitles=%s[v1];[1:v]scale=256:256,format=rgba,colorchannelmixer=aa=0.7[logo];[v1][logo]overlay=W-w-20:H-h-20[vout]", assPath),
			"-map", "[vout]",
			"-c:v", "libx264",
			"-preset", "medium",
			"-crf", "23",
			"-y",
			outputPath,
		)
	} else {
		// No logo, just ASS subtitles
		cmd = exec.Command("ffmpeg",
			"-i", inputPath,
			"-vf", fmt.Sprintf("subtitles=%s", assPath),
			"-c:v", "libx264",
			"-preset", "medium",
			"-crf", "23",
			"-y",
			outputPath,
		)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ffmpeg ASS subtitle overlay failed: %w\nOutput: %s", err, string(output))
	}

	log.Println("✓ ASS karaoke subtitles added successfully")
	return outputPath, nil
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
