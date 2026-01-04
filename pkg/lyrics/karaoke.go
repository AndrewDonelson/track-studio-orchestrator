package lyrics

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

// KaraokeOptions holds customization settings for karaoke subtitles
type KaraokeOptions struct {
	FontFamily           string
	FontSize             int
	PrimaryColor         string
	PrimaryBorderColor   string
	HighlightColor       string
	HighlightBorderColor string
	Alignment            int
	MarginBottom         int
}

// DefaultKaraokeOptions returns default karaoke settings
func DefaultKaraokeOptions() *KaraokeOptions {
	return &KaraokeOptions{
		FontFamily:           "Arial",
		FontSize:             96,
		PrimaryColor:         "4169E1", // Royal Blue
		PrimaryBorderColor:   "FFFFFF", // White
		HighlightColor:       "FFD700", // Gold
		HighlightBorderColor: "FFFFFF", // White
		Alignment:            5,        // Center
		MarginBottom:         0,
	}
}

// KaraokeGenerator handles word-level timestamp generation and ASS subtitle creation
type KaraokeGenerator struct {
	PythonPath   string
	ScriptsDir   string
	WhisperModel string
	VenvPath     string
}

// WhisperResult contains the full transcription result
type WhisperResult struct {
	Segments []WhisperSegment `json:"segments"`
	Language string           `json:"language,omitempty"`
}

// NewKaraokeGenerator creates a new karaoke generator instance
func NewKaraokeGenerator(orchestratorRoot string) *KaraokeGenerator {
	// Detect venv path - check multiple locations
	venvPaths := []string{
		// If orchestratorRoot is the bin/ directory
		filepath.Join(orchestratorRoot, "../../.venv/bin/python"),
		// If orchestratorRoot is the project root
		filepath.Join(orchestratorRoot, "../.venv/bin/python"),
		// Direct project root
		filepath.Join(orchestratorRoot, ".venv/bin/python"),
	}

	venvPath := "python3" // fallback
	for _, path := range venvPaths {
		if _, err := os.Stat(path); err == nil {
			venvPath = path
			break
		}
	}

	// Scripts directory should also be found relative to bin/
	scriptsDirs := []string{
		filepath.Join(orchestratorRoot, "python-scripts"),
		filepath.Join(orchestratorRoot, "../python-scripts"),
	}

	scriptsDir := ""
	for _, dir := range scriptsDirs {
		if _, err := os.Stat(dir); err == nil {
			scriptsDir = dir
			break
		}
	}

	return &KaraokeGenerator{
		PythonPath:   venvPath,
		ScriptsDir:   scriptsDir,
		WhisperModel: "base", // Use "base" for faster processing, "large-v3" for best quality
		VenvPath:     venvPath,
	}
}

// GenerateTimestamps generates word-level timestamps from vocals track
func (kg *KaraokeGenerator) GenerateTimestamps(vocalsPath string, outputJSON string) (*WhisperResult, error) {
	log.Printf("Generating word-level timestamps from: %s", vocalsPath)

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outputJSON), 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	cmd := exec.Command(
		kg.PythonPath,
		filepath.Join(kg.ScriptsDir, "generate_timestamps_faster.py"),
		"--vocals", vocalsPath,
		"--output", outputJSON,
		"--model", kg.WhisperModel,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("timestamp generation failed: %w\nOutput: %s", err, string(output))
	}

	log.Printf("Faster-Whisper output:\n%s", string(output))

	// Load and return the result
	var result WhisperResult
	data, err := os.ReadFile(outputJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to read timestamps: %w", err)
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse timestamps: %w", err)
	}

	totalWords := 0
	for _, seg := range result.Segments {
		totalWords += len(seg.Words)
	}
	log.Printf("Generated %d segments with %d words total", len(result.Segments), totalWords)

	return &result, nil
}

// GenerateASSFile generates an ASS subtitle file with karaoke effects
// If lyricsKaraoke is provided, uses actual lyrics instead of Whisper transcription
func (kg *KaraokeGenerator) GenerateASSFile(timestampsJSON string, outputASS string, lyricsKaraoke string, options *KaraokeOptions) error {
	log.Printf("Generating ASS subtitles from: %s", timestampsJSON)

	if options == nil {
		options = DefaultKaraokeOptions()
	}

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outputASS), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Prepare command arguments
	cmdArgs := []string{
		filepath.Join(kg.ScriptsDir, "generate_karaoke_ass.py"),
		"--timestamps", timestampsJSON,
		"--output", outputASS,
		"--font-family", options.FontFamily,
		"--font-size", fmt.Sprintf("%d", options.FontSize),
		"--primary-color", options.PrimaryColor,
		"--primary-border-color", options.PrimaryBorderColor,
		"--highlight-color", options.HighlightColor,
		"--highlight-border-color", options.HighlightBorderColor,
		"--alignment", fmt.Sprintf("%d", options.Alignment),
		"--margin-bottom", fmt.Sprintf("%d", options.MarginBottom),
	}

	// If lyrics_karaoke is provided, write to temp file and pass to script
	if lyricsKaraoke != "" {
		lyricsFile := filepath.Join(filepath.Dir(outputASS), "lyrics_temp.txt")
		if err := os.WriteFile(lyricsFile, []byte(lyricsKaraoke), 0644); err != nil {
			log.Printf("Warning: failed to write lyrics file: %v", err)
		} else {
			cmdArgs = append(cmdArgs, "--lyrics", lyricsFile)
			defer os.Remove(lyricsFile) // Clean up temp file
		}
	}

	cmd := exec.Command(kg.PythonPath, cmdArgs...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ASS generation failed: %w\nOutput: %s", err, string(output))
	}

	log.Printf("ASS generation output:\n%s", string(output))
	return nil
}

// GenerateKaraokeSubtitles is the complete pipeline: vocals → timestamps → ASS
// If lyricsKaraoke is provided, uses actual lyrics for display instead of Whisper transcription
func (kg *KaraokeGenerator) GenerateKaraokeSubtitles(vocalsPath string, songID int, workingDir string, lyricsKaraoke string, options *KaraokeOptions) (string, error) {
	// Define output paths
	timestampsJSON := filepath.Join(workingDir, fmt.Sprintf("song_%d_timestamps.json", songID))
	assPath := filepath.Join(workingDir, fmt.Sprintf("song_%d_karaoke.ass", songID))

	// Step 1: Generate timestamps (uses Whisper for timing only)
	_, err := kg.GenerateTimestamps(vocalsPath, timestampsJSON)
	if err != nil {
		return "", fmt.Errorf("failed to generate timestamps: %w", err)
	}

	// Step 2: Generate ASS file (with actual lyrics if provided)
	err = kg.GenerateASSFile(timestampsJSON, assPath, lyricsKaraoke, options)
	if err != nil {
		return "", fmt.Errorf("failed to generate ASS file: %w", err)
	}

	log.Printf("Successfully generated karaoke subtitles: %s", assPath)
	return assPath, nil
}
